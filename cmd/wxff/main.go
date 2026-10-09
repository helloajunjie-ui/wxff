// wxff 微信备份工具
//
// 用法:
//
//	wxff list                                    列出最近会话
//	wxff dump -user wxid_xxx                     导出单个好友全部内容
//	wxff dump -user wxid_a,wxid_b                批量导出多个好友
//	wxff dump -user wxid_xxx -type text,image    只导出文字和图片
//	wxff dump -user wxid_xxx --moments           同时导出朋友圈
//	wxff dump -group                             导出所有好友
package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nicols/wxff/internal/archive"
	"github.com/nicols/wxff/internal/decryptedb"
	_ "modernc.org/sqlite"
)

func loadKeys(path string) (map[string][]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) < 3 {
			continue
		}
		key, err := hex.DecodeString(parts[2])
		if err != nil {
			continue
		}
		rel := strings.ReplaceAll(parts[0], "\\", "/")
		out[rel] = key
	}
	return out, nil
}

func deriveImageKey(dataRoot string) (string, int, error) {
	leaf := filepath.Base(dataRoot)
	wxid := leaf
	if idx := strings.LastIndex(leaf, "_"); idx > 0 && len(leaf)-idx-1 == 4 {
		wxid = leaf[:idx]
	}
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, "AppData", "Roaming", "Tencent", "xwechat", "net", "kvcomm"),
		filepath.Join(home, "AppData", "Roaming", "Tencent", "xwechat", "ilink", "kvcomm"),
	}
	radiumRoot := filepath.Join(home, "AppData", "Roaming", "Tencent", "xwechat", "radium", "ilink")
	if entries, err := os.ReadDir(radiumRoot); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(radiumRoot, e.Name(), "kvcomm"))
			}
		}
	}
	codeRe := regexp.MustCompile(`^key_(\d+)_`)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			m := codeRe.FindStringSubmatch(e.Name())
			if len(m) < 2 {
				continue
			}
			code := m[1]
			sum := md5.Sum([]byte(code + wxid))
			key := fmt.Sprintf("%x", sum)[:16]
			return key, 0xE6, nil
		}
	}
	return "", 0, fmt.Errorf("kvcomm 里没找到 key_*.statistic")
}

type Options struct {
	Users   []string        // 要导出的 wxid 列表
	Types   map[string]bool // text/image/file/video/moment
	Moments bool            // 是否导出朋友圈
}

func main() {
	dataRoot := flag.String("data", `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6`, "微信数据根")
	outDir := flag.String("out", `F:\wxff\data\backup`, "输出目录")
	userList := flag.String("user", "", "好友 wxid，多个用逗号分隔")
	allUsers := flag.Bool("all", false, "导出所有好友")
	typeList := flag.String("type", "text,image,file,voice,video", "内容类型: text,image,file,voice,video 逗号分隔")
	moments := flag.Bool("moments", false, "导出朋友圈（全部发布者，每人独立目录）")
	snsList := flag.Bool("sns-list", false, "列出朋友圈发布者（wxid/昵称/动态数）")
	snsUser := flag.String("sns-user", "", "只导出指定发布者的朋友圈（wxid，可逗号分隔）")
	snsMin := flag.Int("sns-min", 720, "朋友圈图片最小长边（像素），小于此值的缩略图归入 _缩略图/；0=不过滤")
	startTime := flag.Int64("from", 0, "起始时间戳（Unix秒，0=不限）")
	endTime := flag.Int64("to", 0, "结束时间戳（Unix秒，0=不限）")
	list := flag.Bool("list", false, "列出最近会话")
	imgKeyFlag := flag.String("imgkey", "", "手动指定 image key（16 字符），留空则自动推导")
	xorKeyFlag := flag.Int("xorkey", -1, "手动指定 xor key（0-255），留空则自动推导")
	flag.Parse()

	// 1. 推导 image key（可用 -imgkey/-xorkey 覆盖）
	var imageKey string
	var xorKey int
	if *imgKeyFlag != "" {
		imageKey = *imgKeyFlag
		xorKey = *xorKeyFlag
		if xorKey < 0 {
			xorKey = 0xE6
		}
		fmt.Printf("[OK] image_key=%s  xor_key=0x%02X（手动指定）\n", imageKey, xorKey)
	} else {
		var err error
		imageKey, xorKey, err = deriveImageKey(*dataRoot)
		if err != nil {
			fmt.Println("[!] 推导 image key 失败:", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] image_key=%s  xor_key=0x%02X\n", imageKey, xorKey)
	}

	// 2. 读 keys.txt
	keys, err := loadKeys(`F:\wxff\data\keys.txt`)
	if err != nil {
		fmt.Println("[!] 读 keys.txt 失败:", err)
		os.Exit(2)
	}

	// 3. 解密 db
	// 解密后的临时 db 放在归档目录「之外」的 decrypted/，避免污染归档结构。
	plainDir := filepath.Join(filepath.Dir(*outDir), "decrypted")
	os.MkdirAll(plainDir, 0755)
	need := []string{"contact/contact.db", "message/message_0.db", "sns/sns.db", "message/media_0.db"}
	decrypted := map[string]string{}
	for _, rel := range need {
		key := keys[rel]
		src := filepath.Join(*dataRoot, "db_storage", filepath.FromSlash(rel))
		dst := filepath.Join(plainDir, strings.ReplaceAll(rel, "/", "__"))
		if _, err := os.Stat(dst); err != nil {
			if err := decryptedb.DecryptFile(src, dst, key); err != nil {
				fmt.Println("[!] 解密失败", rel, err)
				continue
			}
		}
		decrypted[rel] = dst
	}

	cdb, _ := sql.Open("sqlite", decrypted["contact/contact.db"])
	defer cdb.Close()
	mdb, _ := sql.Open("sqlite", decrypted["message/message_0.db"])
	defer mdb.Close()
	sdb, _ := sql.Open("sqlite", decrypted["sns/sns.db"])
	defer sdb.Close()
	// media_0.db 存语音（VoiceInfo 表，SILK v3），可能不存在，允许为空。
	var vdb *sql.DB
	if p := decrypted["message/media_0.db"]; p != "" {
		vdb, _ = sql.Open("sqlite", p)
		if vdb != nil {
			defer vdb.Close()
		}
	}

	// 解析类型
	types := map[string]bool{}
	for _, t := range strings.Split(*typeList, ",") {
		types[strings.TrimSpace(t)] = true
	}

	if *list {
		listRecent(mdb, cdb)
		return
	}

	// 列出朋友圈发布者（wxid / 昵称 / 动态数），供 -sns-user 选择。
	if *snsList {
		posters, err := archive.ListSnsPosters(sdb, cdb)
		if err != nil {
			fmt.Println("[!] 读取朋友圈发布者失败:", err)
			os.Exit(1)
		}
		fmt.Printf("\n=== 朋友圈发布者 %d 人 ===\n", len(posters))
		for _, p := range posters {
			fmt.Printf("  %-30s %-24s posts=%d\n", p.Wxid, p.Nick, p.Posts)
		}
		return
	}

	// 确定要导出的好友列表。
	// 若只导出朋友圈（-sns-user / -moments），则允许不指定聊天好友。
	momentsOnly := *snsUser != "" || *moments
	var users []string
	if *allUsers {
		// 关键：不能按 contact.local_type 过滤。
		// 实测 contact.local_type 分布为 0=56 / 1=113 / 2=7 / 3=379 / 5=2 / 6=5，
		// 而**真正有聊天记录**的好友（有 Msg_<md5(username)> 表）大多是 local_type=1，
		// 用 local_type=0 会选到一批「没有聊天表」的联系人，导出结果为空。
		// 正确做法：直接以「message_0.db 里存在对应 Msg_ 表」为准，
		// 这样与 ExportContact 的表存在性检查完全一致，不会漏也不会多。
		msgTables := map[string]bool{}
		tr, _ := mdb.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
		for tr.Next() {
			var n string
			tr.Scan(&n)
			msgTables[n] = true
		}
		tr.Close()

		rows, _ := cdb.Query("SELECT username FROM contact")
		for rows.Next() {
			var u string
			rows.Scan(&u)
			if u == "" || strings.HasSuffix(u, "@chatroom") || strings.HasPrefix(u, "gh_") || u == "filehelper" {
				continue
			}
			sum := md5.Sum([]byte(u))
			if msgTables["Msg_"+hex.EncodeToString(sum[:])] {
				users = append(users, u)
			}
		}
		rows.Close()
	} else if *userList != "" {
		for _, u := range strings.Split(*userList, ",") {
			users = append(users, strings.TrimSpace(u))
		}
	} else if !momentsOnly {
		fmt.Println("请用 -user <wxid> 或 -all 指定要导出的好友")
		os.Exit(1)
	}

	if len(users) > 0 {
		fmt.Printf("\n=== 待导出 %d 个好友 ===\n", len(users))
	}

	snsTypes := map[string]bool{}
	if *moments {
		snsTypes["text"] = true
		snsTypes["image"] = true
	}
	// 账号 wxid 取数据根目录名（形如 wxid_xxx_dfd6），
	// 去掉末尾的 _<4位hash> 得到真实 wxid，再从 contact 表查账号自身昵称，
	// 顶层目录用「昵称_wxid」，比裸 wxid 直观。
	accountWxid := filepath.Base(*dataRoot)
	if idx := strings.LastIndex(accountWxid, "_"); idx > 0 && len(accountWxid)-idx-1 == 4 {
		accountWxid = accountWxid[:idx]
	}
	var accountNick string
	cdb.QueryRow("SELECT nick_name FROM contact WHERE username=?", accountWxid).Scan(&accountNick)
	account := archive.AccountDirName(accountNick, accountWxid)
	cfg := archive.Config{
		DataRoot:   *dataRoot,
		OutDir:     *outDir,
		ImageKey:   imageKey,
		XORKey:     xorKey,
		Types:      types,
		SnsTypes:   snsTypes,
		StartTime:  *startTime,
		EndTime:    *endTime,
		Account:    account,
		SnsMinEdge: *snsMin,
	}

	// 串行导出（图片解密 CPU 密集，不并行）
	for i, wxid := range users {
		var nick, remark string
		cdb.QueryRow("SELECT nick_name, remark FROM contact WHERE username=?", wxid).Scan(&nick, &remark)
		name := nick
		if remark != "" {
			name = remark
		}
		fmt.Printf("\n[%d/%d] %s (%s)\n", i+1, len(users), wxid, name)
		if err := cfg.ExportContact(wxid, name, mdb); err != nil {
			fmt.Println("  [!] 失败:", err)
			continue
		}
	}

	// 语音：音频数据在 media_0.db 的 VoiceInfo 表（SILK v3），
	// 与聊天消息分开存储，这里统一导出到各好友的 聊天记录/<好友>/media/<年-月>/voices/。
	if vdb != nil && types["voice"] {
		fmt.Printf("\n=== 导出聊天语音 ===\n")
		nickOf := func(wxid string) string {
			var nick, remark string
			cdb.QueryRow("SELECT nick_name, remark FROM contact WHERE username=?", wxid).Scan(&nick, &remark)
			if remark != "" {
				return remark
			}
			return nick
		}
		if err := cfg.ExportVoices(vdb, nickOf); err != nil {
			fmt.Println("  [!] 语音导出失败:", err)
		}
		// 把 .silk 转成可播放的 .mp3（依赖 python+pilk+ffmpeg，缺失则自动跳过）。
		fmt.Printf("\n=== 语音转码为 MP3 ===\n")
		if err := cfg.TranscodeVoices(); err != nil {
			fmt.Println("  [!] 语音转码失败:", err)
		}
	}

	// 朋友圈是「本机浏览过的动态」的全局缓存（SnsTimeLine 的 user_name 是发布者）。
	// 支持两种模式：
	//   -sns-user <wxid,...>：只导出指定发布者（每人独立目录）；
	//   -moments：导出全部发布者（每人独立目录）。
	// 图片缓存整账号只导出一次到 朋友圈/_媒体缓存/。
	if *snsUser != "" {
		fmt.Printf("\n=== 导出朋友圈（指定发布者）===\n")
		for _, u := range strings.Split(*snsUser, ",") {
			u = strings.TrimSpace(u)
			if u == "" {
				continue
			}
			var nick, remark string
			cdb.QueryRow("SELECT nick_name, remark FROM contact WHERE username=?", u).Scan(&nick, &remark)
			name := nick
			if remark != "" {
				name = remark
			}
			fmt.Printf("  [发布者] %s (%s)\n", u, name)
			if err := cfg.ExportMoments(u, name, sdb, cdb); err != nil {
				fmt.Println("  [!] 朋友圈失败:", err)
			}
		}
	} else if *moments {
		fmt.Printf("\n=== 导出朋友圈（全部发布者，每人独立目录）===\n")
		if err := cfg.ExportMoments("", "", sdb, cdb); err != nil {
			fmt.Println("  [!] 朋友圈失败:", err)
		}
	}
	fmt.Printf("\n[完成] 归档在 %s\n", *outDir)
}

func listRecent(mdb, cdb *sql.DB) {
	type s struct {
		md5  string
		last int64
		cnt  int
		wxid string
		nick string
	}
	md2wx := map[string]struct{ wxid, nick string }{}
	rows, _ := cdb.Query("SELECT username, nick_name, remark FROM contact")
	for rows.Next() {
		var u, n, r string
		rows.Scan(&u, &n, &r)
		sum := md5.Sum([]byte(u))
		name := n
		if r != "" {
			name = r + "(" + n + ")"
		}
		md2wx[hex.EncodeToString(sum[:])] = struct{ wxid, nick string }{u, name}
	}
	rows.Close()

	rows, _ = mdb.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
	list := []s{}
	for rows.Next() {
		var t string
		rows.Scan(&t)
		var last int64
		var cnt int
		mdb.QueryRow("SELECT max(create_time), count(*) FROM "+t).Scan(&last, &cnt)
		m := strings.TrimPrefix(t, "Msg_")
		wxid, nick := "", ""
		if info, ok := md2wx[m]; ok {
			wxid = info.wxid
			nick = info.nick
		}
		list = append(list, s{m, last, cnt, wxid, nick})
	}
	rows.Close()
	sort.Slice(list, func(i, j int) bool { return list[i].last > list[j].last })
	fmt.Println("\n=== 最近会话 ===")
	for i := 0; i < 20 && i < len(list); i++ {
		fmt.Printf("  %-30s %-25s msgs=%-5d\n", list[i].wxid, list[i].nick, list[i].cnt)
	}
}
