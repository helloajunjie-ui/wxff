// Package archive 把解密后的 db 导出成结构化归档目录。
package archive

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"  // 注册 gif 解码器
	_ "image/jpeg" // 注册 jpeg 解码器
	_ "image/png"  // 注册 png 解码器
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/nicols/wxff/internal/imgkey"
)

type Config struct {
	DataRoot  string
	OutDir    string
	ImageKey  string
	XORKey    int
	Types     map[string]bool // 聊天记录类型
	SnsTypes  map[string]bool // 朋友圈类型
	StartTime int64
	EndTime   int64
	Account   string
	// SnsMinEdge 朋友圈图片最小长边（像素）。缓存里缩略图与原图混存，
	// 长边小于该值的图片会被归入「_缩略图/」子目录，避免与原图混淆。
	// 0 表示不过滤（全部当作原图）。
	SnsMinEdge int
}

func (c *Config) MsgRoot() string { return filepath.Join(c.DataRoot, "msg") }

// accountDir 返回某个账号的归档根目录：<输出目录>/<账号目录名>/
// 多账号时每个账号一个顶层目录，互不混淆。
func (c *Config) accountDir() string {
	return filepath.Join(c.OutDir, c.Account)
}

// AccountDirName 生成账号顶层目录名：优先「昵称_wxid」，昵称为空时退回 wxid。
// 账号自身的昵称存在 contact 表里（username=账号wxid，local_type=1），
// 用中文昵称做顶层目录，比裸 wxid 直观得多。
// 形如：彼岸轮回_wxid_mpvlokr4ycnu22
func AccountDirName(nick, wxid string) string {
	name := sanitize(nick)
	if name == "" {
		return sanitize(wxid)
	}
	name = truncateRunes(name, 40)
	return name + "_" + sanitize(wxid)
}

// chatDir 返回某账号下某好友的聊天记录目录：
// <输出目录>/<账号wxid>/聊天记录/<好友>/
// 好友之间完全隔离，各自一个目录。
func (c *Config) chatDir(nick, wxid string) string {
	return filepath.Join(c.accountDir(), "聊天记录", friendDirName(nick, wxid))
}

// momentsDir 返回某账号下某好友的朋友圈目录：
// <输出目录>/<账号wxid>/朋友圈/<好友>/
func (c *Config) momentsDir(nick, wxid string) string {
	return filepath.Join(c.accountDir(), "朋友圈", friendDirName(nick, wxid))
}

// momentsMediaDir 返回某账号下朋友圈「媒体缓存」目录：
// <输出目录>/<账号wxid>/朋友圈/_媒体缓存/
// 朋友圈图片缓存文件名是微信内部不透明 id，无法归属到具体发布者，
// 因此整账号只导出一次，所有发布者共用。
func (c *Config) momentsMediaDir() string {
	return filepath.Join(c.accountDir(), "朋友圈", "_媒体缓存")
}

// friendDirName 生成好友目录名：优先备注/昵称，为空用 wxid，
// 并做文件系统非法字符替换。为保证好友之间不重名，末尾追加 wxid 短哈希。
func friendDirName(nick, wxid string) string {
	name := sanitize(nick)
	if name == "" {
		name = sanitize(wxid)
	}
	// 限制目录名长度：部分 wxid（如 v3_...@stranger_...）极长，
	// 直接用作目录名会超出 Windows 路径长度上限，这里按「字符」截断。
	name = truncateRunes(name, 40)
	// wxid 为空（如全局朋友圈缓存）时不追加哈希，保持目录名干净。
	if wxid == "" {
		return name
	}
	// 追加 wxid 的短哈希，避免同名好友互相覆盖（好友隔离）。
	sum := md5.Sum([]byte(wxid))
	return name + "_" + hex.EncodeToString(sum[:])[:8]
}

// shortHash 返回 wxid 的 8 位短哈希，用于目录名去重与陌生人命名。
func shortHash(wxid string) string {
	sum := md5.Sum([]byte(wxid))
	return hex.EncodeToString(sum[:])[:8]
}

// isStrangerWxid 判断是否为「陌生人」wxid。
// 微信里非好友/未加好友的人，wxid 形如
// v3_020b3826fd0301...@stranger（长达 200+ 字符），
// 这类人在 contact 表里查不到昵称，直接用 wxid 做目录名完全不可读。
func isStrangerWxid(wxid string) bool {
	return strings.Contains(wxid, "@stranger") || strings.HasPrefix(wxid, "v3_")
}

// posterDirName 生成朋友圈「发布者」目录名。
// 与 friendDirName 的区别：
//   - 加序号前缀（seq），按动态数降序，让用户一眼看出主次（01_、02_…）；
//   - 陌生人（v3_...@stranger）不显示超长 wxid，改用「陌生人_<短哈希>」。
func posterDirName(seq int, nick, wxid string) string {
	name := sanitize(nick)
	if name == "" {
		if isStrangerWxid(wxid) {
			name = "陌生人"
		} else {
			name = sanitize(wxid)
		}
	}
	name = truncateRunes(name, 40)
	base := name + "_" + shortHash(wxid)
	if seq > 0 {
		return fmt.Sprintf("%02d_%s", seq, base)
	}
	return base
}

// truncateRunes 按 Unicode 字符（而非字节）截断字符串，避免截断多字节字符。
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// ExportContact 导出一个好友的聊天记录。
// 若该好友没有消息表（从未聊过天），直接跳过，不创建空目录。
func (c *Config) ExportContact(wxid string, nick string, mdb *sql.DB) error {
	sum := md5.Sum([]byte(wxid))
	table := "Msg_" + hex.EncodeToString(sum[:])

	// 先确认消息表存在：微信只为「有聊天记录」的好友建 Msg_<md5(wxid)> 表，
	// 通讯录里大量好友从未聊过，没有对应表，直接跳过（不报错、不建空目录）。
	var exists int
	if err := mdb.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&exists); err != nil || exists == 0 {
		return nil
	}

	// 路径：备份根/账号wxid/聊天记录/好友名_wxid短哈希/
	// 好友之间完全隔离，各自一个目录。
	baseDir := c.chatDir(nick, wxid)
	os.MkdirAll(baseDir, 0755)

	rows, err := mdb.Query("SELECT local_id, local_type, create_time, message_content, packed_info_data FROM " + table + " ORDER BY create_time ASC")
	if err != nil {
		return fmt.Errorf("查询 %s: %w", table, err)
	}
	defer rows.Close()

	// 建立图片索引：扫描 msg/attach/<md5(wxid)>/<年-月>/Img/ 下的 .dat 文件。
	// 文件名（去掉 _t/_h/_W 后缀）就是图片标识，与消息 packed_info_data 里的 32 位 hex 对应。
	//
	// 注意：微信 4.x 主图 .dat 解密后可能是私有 "wxgf" 容器（内含 HEVC/AVIF），
	// 无法直接用标准解码器打开；而 _t.dat（缩略图）通常是标准 JPEG。
	// 因此这里对每个候选解密并检测格式，优先选择能直接解码的标准图片（JPEG/PNG）。
	talkerMd5 := hex.EncodeToString(sum[:])
	attachRoot := filepath.Join(c.DataRoot, "msg", "attach", talkerMd5)
	imgFileMap := map[string]string{} // 32hex 标识 -> .dat 路径
	imgRank := map[string]int{}       // 32hex 标识 -> 当前最优变体的像素数（越大越好）
	filepath.Walk(attachRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		base := filepath.Base(p)
		if !strings.HasSuffix(base, ".dat") {
			return nil
		}
		stem := strings.TrimSuffix(base, ".dat")
		// 去掉 _t（缩略图）/ _h（高清）/ _W 等后缀，得到 32 位标识
		for {
			changed := false
			for _, suf := range []string{"_t", "_h", "_W"} {
				if strings.HasSuffix(stem, suf) {
					stem = strings.TrimSuffix(stem, suf)
					changed = true
				}
			}
			if !changed {
				break
			}
		}
		if len(stem) != 32 {
			return nil
		}
		key := strings.ToLower(stem)

		// 解密并**完整解码**验证：只有能完整解码的标准图片才可用。
		// 关键：必须用 image.Decode（真正解码像素），不能用 image.DecodeConfig
		// （只读文件头）。微信 4.x 的 _h.dat 大图常常文件头正常（DecodeConfig 通过）
		// 但正文损坏（Decode 失败），若用 DecodeConfig 会把这种坏图选进索引，
		// 导出时再被 Decode 拦下，导致大量图片"索引里有、导出时没了"。
		//
		// 同一标识可能有多个变体：main（原图）/ _h（高清）/ _t（缩略图）/ _W。
		// 优先选择**分辨率最大**的变体（原图 > 高清 > 缩略图），
		// 而不是"第一个能解码的"，否则可能只导出缩略图。
		// 用像素数（宽×高）作为排名，越大越好；不能完整解码的跳过。
		data, e := os.ReadFile(p)
		if e != nil || !imgkey.IsWechatV4DAT(data) {
			return nil
		}
		dec, e2 := imgkey.DecryptDAT(data, c.ImageKey, c.XORKey)
		if e2 != nil || len(dec) < 4 {
			return nil
		}
		img, _, derr := image.Decode(bytes.NewReader(dec))
		if derr != nil {
			// 不能完整解码（如 wxgf 私有容器、损坏的 _h PNG），跳过。
			return nil
		}
		b := img.Bounds()
		area := b.Dx() * b.Dy()
		if old, ok := imgRank[key]; !ok || area > old {
			imgRank[key] = area
			imgFileMap[key] = p
		}
		return nil
	})
	fmt.Printf("  [调试] 当前好友图片索引: %d 张\n", len(imgFileMap))

	// 建立文件索引：扫描 msg/file/<年-月>/ 与 msg/migrate/File/<年-月>/ 下的真实文件。
	// 文件消息（type 49，含 appattach）的 XML 里有 <title>（原始文件名），
	// 用它去匹配这里的文件名即可定位到本地文件并拷贝。
	// 注意：msg/file 是**全局**的（不按好友隔离），同名文件可能属于不同好友，
	// 因此这里只做「文件名 -> 路径」映射，导出时按文件名取。
	fileMap := map[string]string{} // 原始文件名 -> 本地路径
	for _, sub := range []string{
		filepath.Join(c.DataRoot, "msg", "file"),
		filepath.Join(c.DataRoot, "msg", "migrate", "File"),
	} {
		filepath.Walk(sub, func(p string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			fileMap[filepath.Base(p)] = p
			return nil
		})
	}
	fmt.Printf("  [调试] 当前文件索引: %d 个\n", len(fileMap))

	out, _ := os.Create(filepath.Join(baseDir, "messages.jsonl"))
	defer out.Close()

	count := 0
	imgCount := 0
	fileCount := 0
	for rows.Next() {
		var id, mtype, ctime int64
		var content, packed []byte
		rows.Scan(&id, &mtype, &ctime, &content, &packed)

		if c.StartTime > 0 && ctime < c.StartTime {
			continue
		}
		if c.EndTime > 0 && ctime > c.EndTime {
			continue
		}

		text := string(content)
		// 所有消息都先尝试 zstd 解压
		if len(text) >= 4 && text[0] == 0x28 && text[1] == 0xb5 && text[2] == 0x2f && text[3] == 0xfd {
			r, err := zstd.NewReader(nil)
			if err == nil {
				dst := make([]byte, 0, len(content)*10)
				out, err := r.DecodeAll(content, dst)
				r.Close()
				if err == nil {
					text = string(out)
				}
			}
		}
		// 注意：图片消息（type=3）的 message_content 可能为空，
		// 图片标识在 packed_info_data 里，所以不能在这里因 text=="" 跳过，
		// 否则会漏掉大量图片。空文本检查放到各 case 内部按需处理。
		msgTime := time.Unix(ctime, 0)
		msgTimeStr := msgTime.Format("2006-01-02 15:04:05")
		monthDir := msgTime.Format("2006-01") // 按年-月分目录

		// 关键：微信 4.x 的 local_type 是 64 位，高位是「子类型/标志位」，
		// 真实消息类型在**低 16 位**。例如文件消息是 49，但实际存储为
		// 25769803825 / 8589934592049 / 21474836529 等（低位都是 49）。
		// 若用 mtype 精确匹配，会漏掉大量带标志的消息（本机实测漏 469 条文件消息）。
		baseType := int(mtype & 0xFFFF)
		switch baseType {
		case 1:
			if !c.Types["text"] || text == "" {
				continue
			}
			fmt.Fprintf(out, `{"time":"%s","ts":%d,"type":"text","text":%q}`+"\n", msgTimeStr, ctime, text)
			count++
		case 3:
			if !c.Types["image"] {
				continue
			}
			// 图片标识在 packed_info_data 里（形如 ..." <32位hex>），
			// 与 msg/attach/<md5(wxid)>/<年-月>/Img/<32位hex>.dat 的文件名对应。
			imgHash := extractImgHash(packed)
			if imgHash == "" {
				continue
			}
			imgPath, ok := imgFileMap[imgHash]
			if !ok {
				continue
			}
			if data, err := os.ReadFile(imgPath); err == nil {
				if decImg, err := imgkey.DecryptDAT(data, c.ImageKey, c.XORKey); err == nil {
					// 再次确认能完整解码，避免写入 wxgf 等无效图片。
					if _, _, derr := image.Decode(bytes.NewReader(decImg)); derr != nil {
						continue
					}
					ext := ".jpg"
					if len(decImg) >= 4 && decImg[0] == 0x89 {
						ext = ".png"
					} else if len(decImg) >= 3 && decImg[0] == 0x47 && decImg[1] == 0x49 && decImg[2] == 0x46 {
						ext = ".gif"
					}
					// 图片存到 月/images/ 目录
					imgDir := filepath.Join(baseDir, "media", monthDir, "images")
					os.MkdirAll(imgDir, 0755)
					name := fmt.Sprintf("%d_%s%s", ctime, imgHash[:8], ext)
					os.WriteFile(filepath.Join(imgDir, name), decImg, 0644)
					relPath := fmt.Sprintf("media/%s/images/%s", monthDir, name)
					fmt.Fprintf(out, `{"time":"%s","ts":%d,"type":"image","file":%q}`+"\n", msgTimeStr, ctime, relPath)
					count++
					imgCount++
				}
			}
		case 34:
			// 语音消息：正文是 <voicemsg .../>，音频数据不在消息表里，
			// 而是存在 media_0.db 的 VoiceInfo 表（SILK v3 格式）。
			// 这里只记录元数据（时长/格式），音频文件由 ExportVoices 统一导出。
			if !c.Types["voice"] {
				continue
			}
			dur := parseVoiceLength(text)
			fmt.Fprintf(out, `{"time":"%s","ts":%d,"type":"voice","dur":%d}`+"\n", msgTimeStr, ctime, dur)
			count++
		case 43:
			// 视频消息：本地**不存原视频**（只有 CDN 地址 + 缩略图）。
			// 这里记录时长与 CDN 地址，缩略图若能找到则一并导出。
			if !c.Types["video"] {
				continue
			}
			dur := parseVideoPlayLength(text)
			cdn := parseVideoCDNURL(text)
			rec := map[string]any{
				"time": msgTimeStr,
				"ts":   ctime,
				"type": "video",
				"dur":  dur,
			}
			if cdn != "" {
				rec["cdn"] = cdn
			}
			b, _ := json.Marshal(rec)
			out.Write(b)
			out.Write([]byte("\n"))
			count++
		case 49:
			if text == "" {
				continue
			}
			titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
			m := titleRe.FindStringSubmatch(text)
			title := ""
			if len(m) > 1 {
				title = m[1]
			}
			// 关键：type 49 是「appmsg」大类，包含文件/链接/红包/小程序/引用等。
			// 不能只看是否含 <appattach>（链接、引用、小程序也常带一个空的 appattach），
			// 必须看 <appmsg> 里的 <type> 子类型：
			//   6  = 文件（唯一需要拷贝本地文件的类型）
			//   5  = 链接   57 = 引用   33/36 = 小程序   8 = 表情   19 = 聊天记录
			//   51 = 视频号   62 = 其他
			// 实测本机 364 条含 appattach 的消息里，只有 41 条是 type=6（文件），
			// 其余 323 条是链接/引用/小程序等，若按 appattach 判断会全部误判成文件。
			appType := parseAppMsgType(text)
			if appType == 6 {
				// 文件消息：尝试从 msg/file 索引里找到真实文件并拷贝。
				if !c.Types["file"] {
					continue
				}
				rec := map[string]any{
					"time": msgTimeStr,
					"ts":   ctime,
					"type": "file",
					"name": title,
				}
				if src, ok := fileMap[title]; ok {
					if data, e := os.ReadFile(src); e == nil {
						fileDir := filepath.Join(baseDir, "media", monthDir, "files")
						os.MkdirAll(fileDir, 0755)
						dst := filepath.Join(fileDir, title)
						if os.WriteFile(dst, data, 0644) == nil {
							rec["file"] = fmt.Sprintf("media/%s/files/%s", monthDir, title)
							fileCount++
						}
					}
				}
				b, _ := json.Marshal(rec)
				out.Write(b)
				out.Write([]byte("\n"))
			} else {
				// 链接/红包/小程序/引用等：只记标题。
				if !c.Types["text"] {
					continue
				}
				fmt.Fprintf(out, `{"time":"%s","ts":%d,"type":"link","title":%q}`+"\n", msgTimeStr, ctime, title)
			}
			count++
		}
	}
	fmt.Printf("  [OK] 导出 %d 条消息（其中图片 %d 张，文件 %d 个）\n", count, imgCount, fileCount)
	// 生成 index.html
	generateChatHTML(baseDir)
	return nil
}

func generateChatHTML(dir string) {
	data, _ := os.ReadFile(filepath.Join(dir, "messages.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	html := `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>聊天记录</title><style>
body{font-family:"Microsoft YaHei",sans-serif;max-width:800px;margin:0 auto;padding:20px;background:#f5f5f5}
.msg{margin:12px 0;padding:10px 14px;border-radius:8px;background:#fff;box-shadow:0 1px 2px rgba(0,0,0,0.1)}
.msg .time{font-size:12px;color:#999;margin-bottom:4px}
.msg .text{font-size:15px;line-height:1.5}
.msg img{max-width:300px;border-radius:4px;margin-top:8px}
</style></head><body>`
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 简单解析 JSON
		var m map[string]any
		json.Unmarshal([]byte(line), &m)
		html += `<div class="msg"><div class="time">` + fmt.Sprintf("%v", m["time"]) + `</div>`
		switch m["type"] {
		case "text", "link":
			html += `<div class="text">` + fmt.Sprintf("%v", m["text"]) + `</div>`
		case "image":
			html += `<img src="` + fmt.Sprintf("%v", m["file"]) + `">`
		case "file":
			html += `<div class="text">📎 ` + fmt.Sprintf("%v", m["name"]) + `</div>`
		}
		html += `</div>`
	}
	html += `</body></html>`
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(html), 0644)
}

// SnsPoster 是一个朋友圈发布者。
type SnsPoster struct {
	Wxid  string
	Nick  string
	Posts int
}

// ListSnsPosters 列出本机缓存里所有朋友圈发布者（含昵称与动态数）。
// cdb 用于把发布者 wxid 映射成昵称/备注，可为 nil。
func ListSnsPosters(sdb, cdb *sql.DB) ([]SnsPoster, error) {
	rows, err := sdb.Query(`SELECT user_name, count(*) FROM SnsTimeLine GROUP BY user_name ORDER BY count(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SnsPoster
	for rows.Next() {
		var p SnsPoster
		rows.Scan(&p.Wxid, &p.Posts)
		if cdb != nil {
			var nick, remark string
			cdb.QueryRow("SELECT nick_name, remark FROM contact WHERE username=?", p.Wxid).Scan(&nick, &remark)
			if remark != "" {
				p.Nick = remark
			} else {
				p.Nick = nick
			}
		}
		list = append(list, p)
	}
	return list, nil
}

// ExportMoments 导出朋友圈。
//
// 说明（实测结论，2026-10-09）：
//   - 朋友圈正文存在 sns.db 的 SnsTimeLine 表，user_name 是「发布者」wxid，
//     只包含本机实际浏览过的动态（本机缓存），并非某个好友的全部朋友圈。
//   - 朋友圈图片缓存在 cache/<年-月>/Sns/Img/<2hex>/<30hex>（无扩展名），
//     文件名是微信内部不透明 id，无法与 XML 里的 md5/url/id 建立对应关系。
//   - 因此：
//     1) 图片缓存整账号只导出一次到 朋友圈/_媒体缓存/<年-月>/（所有发布者共用）；
//     2) 动态按「发布者」隔离：每个发布者一个目录 朋友圈/<昵称>_<hash>/，
//     内含 posts.json + index.html。
//
// 参数 wxid 为「发布者过滤」：
//   - 为空：导出全部发布者，每人一个目录；
//   - 非空：只导出该发布者，目录名用传入的 nick。
func (c *Config) ExportMoments(wxid string, nick string, sdb, cdb *sql.DB) error {
	// 朋友圈媒体与聊天媒体完全分开（不同顶层目录）。
	// ---- 1. 导出朋友圈图片缓存（整账号一次，按月份）----
	// 缓存路径：cache/<年-月>/Sns/Img/<2hex>/<30hex>（无扩展名）。
	// 文件名是 30 位 hex 的不透明 id，无法归属到具体发布者，故整账号共用。
	mediaRoot := c.momentsMediaDir()
	os.MkdirAll(mediaRoot, 0755)
	snsImgRoot := filepath.Join(c.DataRoot, "cache")
	imgCount := 0
	thumbCount := 0
	filepath.Walk(snsImgRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		// 只看 Sns/Img 目录下的文件
		if !strings.Contains(p, `\Sns\Img\`) {
			return nil
		}
		base := filepath.Base(p)
		// 去掉可能的扩展名（视频缩略图带 .jpg）
		stem := base
		if i := strings.LastIndex(stem, "."); i > 0 {
			stem = stem[:i]
		}
		// 图片缓存文件名是 30 位 hex
		if len(stem) != 30 {
			return nil
		}
		data, e := os.ReadFile(p)
		if e != nil || !imgkey.IsWechatV4DAT(data) {
			return nil
		}
		dec, e2 := imgkey.DecryptDAT(data, c.ImageKey, c.XORKey)
		if e2 != nil || len(dec) < 4 {
			return nil
		}
		// 完整解码验证，只保留标准图片（JPEG/PNG/GIF），跳过损坏流。
		cfg, _, derr := image.DecodeConfig(bytes.NewReader(dec))
		if derr != nil {
			return nil
		}
		// 从路径中取「年-月」：cache/<年-月>/Sns/Img/...
		// 归档时按年分组：_媒体缓存/<年>/<月>/，避免顶层堆 65 个月份目录。
		rel, rerr := filepath.Rel(snsImgRoot, p)
		year, month := "unknown", "unknown"
		if rerr == nil {
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) >= 1 {
				ym := parts[0] // 形如 2021-06
				if i := strings.Index(ym, "-"); i > 0 {
					year = ym[:i]
					month = ym[i+1:]
				} else {
					year = ym
				}
			}
		}
		ext := ".jpg"
		if len(dec) >= 4 && dec[0] == 0x89 {
			ext = ".png"
		} else if len(dec) >= 3 && dec[0] == 0x47 && dec[1] == 0x49 && dec[2] == 0x46 {
			ext = ".gif"
		}
		// 缩略图与原图混存：长边小于阈值的归入「_缩略图/」子目录，避免混淆。
		sub := ""
		long := cfg.Width
		if cfg.Height > long {
			long = cfg.Height
		}
		if c.SnsMinEdge > 0 && long < c.SnsMinEdge {
			sub = "_缩略图"
			thumbCount++
		} else {
			imgCount++
		}
		imgDir := filepath.Join(mediaRoot, sub, year, month)
		os.MkdirAll(imgDir, 0755)
		os.WriteFile(filepath.Join(imgDir, stem+ext), dec, 0644)
		return nil
	})
	if c.SnsMinEdge > 0 {
		fmt.Printf("  [OK] 朋友圈图片缓存导出 %d 张原图（长边>=%d，按 年/月 归档到 _媒体缓存/）\n", imgCount, c.SnsMinEdge)
		fmt.Printf("  [OK] 另有 %d 张缩略图（长边<%d）归入 _媒体缓存/_缩略图/\n", thumbCount, c.SnsMinEdge)
	} else {
		fmt.Printf("  [OK] 朋友圈图片缓存导出 %d 张（按 年/月 归档到 _媒体缓存/）\n", imgCount)
	}

	// ---- 1b. 导出朋友圈视频（本地缓存里有真实 mp4）----
	// 缓存路径：cache/<年-月>/Sns/Video/<2hex>/<30hex>.mp4（另有同名 .jpg 封面）。
	// 与图片一样，文件名是不透明 id，无法归属到具体发布者，整账号共用。
	videoCount := 0
	filepath.Walk(snsImgRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.Contains(p, `\Sns\Video\`) {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(p), ".mp4") {
			return nil
		}
		rel, rerr := filepath.Rel(snsImgRoot, p)
		year, month := "unknown", "unknown"
		if rerr == nil {
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) >= 1 {
				ym := parts[0]
				if i := strings.Index(ym, "-"); i > 0 {
					year = ym[:i]
					month = ym[i+1:]
				} else {
					year = ym
				}
			}
		}
		dir := filepath.Join(mediaRoot, "_视频", year, month)
		os.MkdirAll(dir, 0755)
		data, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		os.WriteFile(filepath.Join(dir, filepath.Base(p)), data, 0644)
		videoCount++
		return nil
	})
	fmt.Printf("  [OK] 朋友圈视频导出 %d 个（按 年/月 归档到 _媒体缓存/_视频/）\n", videoCount)

	// ---- 2. 按发布者导出动态 ----
	// 收集要导出的发布者列表（含动态数，用于排序与序号前缀）。
	type poster struct {
		wxid, nick string
		posts      int
	}
	var posters []poster
	if wxid != "" {
		posters = append(posters, poster{wxid, nick, 0})
	} else {
		rows, err := sdb.Query(`SELECT user_name, count(*) FROM SnsTimeLine GROUP BY user_name ORDER BY count(*) DESC`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var u string
			var n int
			rows.Scan(&u, &n)
			posters = append(posters, poster{u, "", n})
		}
		rows.Close()
	}

	// 先解析所有发布者的昵称，供目录命名与 README 索引使用。
	for i := range posters {
		if posters[i].nick != "" {
			continue
		}
		if cdb != nil {
			var nick, remark string
			cdb.QueryRow("SELECT nick_name, remark FROM contact WHERE username=?", posters[i].wxid).Scan(&nick, &remark)
			if remark != "" {
				posters[i].nick = remark
			} else {
				posters[i].nick = nick
			}
		}
	}

	// README 索引：列出所有发布者（序号/昵称/动态数/目录名），当索引用。
	var idx []snsIdxRow

	totalPosts := 0
	for i, ps := range posters {
		// 目录名：序号前缀（按动态数降序）+ 昵称 + 短哈希；
		// 陌生人（v3_...@stranger）用「陌生人_<短哈希>」代替超长 wxid。
		seq := i + 1
		dirName := posterDirName(seq, ps.nick, ps.wxid)
		dir := filepath.Join(c.accountDir(), "朋友圈", dirName)
		os.MkdirAll(dir, 0755)
		idx = append(idx, snsIdxRow{seq, ps.nick, dirName, ps.posts})

		rows, err := sdb.Query("SELECT tid, user_name, content FROM SnsTimeLine WHERE user_name=? ORDER BY tid ASC", ps.wxid)
		if err != nil {
			return err
		}
		out, _ := os.Create(filepath.Join(dir, "posts.json"))
		count := 0
		fmt.Fprintln(out, "[")
		first := true
		for rows.Next() {
			var tid int64
			var user, content string
			rows.Scan(&tid, &user, &content)

			timeRe := regexp.MustCompile(`<createTime>(\d+)</createTime>`)
			ct := int64(0)
			if m := timeRe.FindStringSubmatch(content); len(m) > 1 {
				fmt.Sscanf(m[1], "%d", &ct)
			}
			if c.StartTime > 0 && ct < c.StartTime {
				continue
			}
			if c.EndTime > 0 && ct > c.EndTime {
				continue
			}
			descRe := regexp.MustCompile(`<contentDesc>([^<]*)</contentDesc>`)
			m := descRe.FindStringSubmatch(content)
			text := ""
			if len(m) > 1 {
				text = m[1]
			}

			// 媒体明细：每条 <media> 提取 type（2=图片,6=视频）、尺寸、时长、md5。
			mediaItems := parseSnsMedia(content)
			// 评论与点赞。
			comments := parseSnsComments(content)
			likes := parseSnsLikes(content)

			if !first {
				fmt.Fprintln(out, ",")
			}
			first = false
			rec := map[string]any{
				"time":     ct,
				"user":     user,
				"text":     text,
				"media":    mediaItems,
				"comments": comments,
				"likes":    likes,
			}
			b, _ := json.Marshal(rec)
			out.Write(b)
			count++
		}
		fmt.Fprintln(out, "\n]")
		out.Close()
		rows.Close()
		generateSnsHTML(dir)
		totalPosts += count
	}

	// 生成 README 索引：一眼看清有哪些发布者、各多少条、目录叫什么。
	writeSnsReadme(filepath.Join(c.accountDir(), "朋友圈"), idx, totalPosts)

	fmt.Printf("  [OK] 朋友圈发布者 %d 人，动态共 %d 条（每人独立目录，已生成 README.txt 索引）\n", len(posters), totalPosts)
	return nil
}

// snsMedia 是一条朋友圈媒体（图片/视频）的元数据。
type snsMedia struct {
	Type   int    `json:"type"`   // 2=图片 6=视频
	Width  int    `json:"width"`  // 0 表示未知
	Height int    `json:"height"` // 0 表示未知
	Dur    int    `json:"dur"`    // 视频时长（秒），图片为 0
	MD5    string `json:"md5"`    // 服务端图片 md5（仅作标识，非本地文件名）
}

// parseSnsMedia 从朋友圈 content XML 中解析所有 <media> 项。
func parseSnsMedia(content string) []snsMedia {
	mediaRe := regexp.MustCompile(`(?s)<media>(.*?)</media>`)
	typeRe := regexp.MustCompile(`<type>(\d+)</type>`)
	sizeRe := regexp.MustCompile(`<size width="(\d+)" height="(\d+)"`)
	durRe := regexp.MustCompile(`<videoDuration>(\d+)</videoDuration>`)
	md5Re := regexp.MustCompile(`md5="([0-9a-f]{32})"`)
	var out []snsMedia
	for _, m := range mediaRe.FindAllStringSubmatch(content, -1) {
		body := m[1]
		var it snsMedia
		if t := typeRe.FindStringSubmatch(body); len(t) > 1 {
			fmt.Sscanf(t[1], "%d", &it.Type)
		}
		if s := sizeRe.FindStringSubmatch(body); len(s) > 2 {
			fmt.Sscanf(s[1], "%d", &it.Width)
			fmt.Sscanf(s[2], "%d", &it.Height)
		}
		if d := durRe.FindStringSubmatch(body); len(d) > 1 {
			fmt.Sscanf(d[1], "%d", &it.Dur)
		}
		if h := md5Re.FindStringSubmatch(body); len(h) > 1 {
			it.MD5 = h[1]
		}
		out = append(out, it)
	}
	return out
}

// snsComment 是一条评论。
type snsComment struct {
	Nick string `json:"nick"`
	Text string `json:"text"`
	Time int64  `json:"time"`
}

// parseSnsComments 从朋友圈 content XML 中解析评论列表。
func parseSnsComments(content string) []snsComment {
	blockRe := regexp.MustCompile(`(?s)<user_comment>(.*?)</user_comment>`)
	nickRe := regexp.MustCompile(`<nickname>([^<]*)</nickname>`)
	// 评论正文可能在 <content> 或 <comment_content> 中。
	textRe := regexp.MustCompile(`<content>([^<]*)</content>`)
	timeRe := regexp.MustCompile(`<create_time>(\d+)</create_time>`)
	var out []snsComment
	for _, m := range blockRe.FindAllStringSubmatch(content, -1) {
		body := m[1]
		var cm snsComment
		if n := nickRe.FindStringSubmatch(body); len(n) > 1 {
			cm.Nick = n[1]
		}
		if t := textRe.FindStringSubmatch(body); len(t) > 1 {
			cm.Text = t[1]
		}
		if ts := timeRe.FindStringSubmatch(body); len(ts) > 1 {
			fmt.Sscanf(ts[1], "%d", &cm.Time)
		}
		out = append(out, cm)
	}
	return out
}

// parseSnsLikes 从朋友圈 content XML 中解析点赞者昵称列表。
func parseSnsLikes(content string) []string {
	// 点赞在 <like_user_list> 内，但评论也在同一块；点赞项通常只有 username/nickname 无 user_comment。
	// 这里取 <like_user_list> 中所有 <nickname>，再排除评论块里的。
	likeRe := regexp.MustCompile(`(?s)<like_user_list>(.*?)</like_user_list>`)
	nickRe := regexp.MustCompile(`<nickname>([^<]*)</nickname>`)
	commentRe := regexp.MustCompile(`(?s)<user_comment>.*?</user_comment>`)
	var out []string
	for _, m := range likeRe.FindAllStringSubmatch(content, -1) {
		body := commentRe.ReplaceAllString(m[1], "")
		for _, n := range nickRe.FindAllStringSubmatch(body, -1) {
			if n[1] != "" {
				out = append(out, n[1])
			}
		}
	}
	return out
}

func generateSnsHTML(dir string) {
	data, _ := os.ReadFile(filepath.Join(dir, "posts.json"))
	var posts []map[string]any
	json.Unmarshal(data, &posts)
	html := `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>朋友圈</title><style>
body{font-family:"Microsoft YaHei",sans-serif;max-width:800px;margin:0 auto;padding:20px;background:#f5f5f5}
.note{margin:16px 0;padding:12px;border-radius:8px;background:#fff8e1;color:#8a6d3b;font-size:13px}
.post{margin:16px 0;padding:14px;border-radius:8px;background:#fff;box-shadow:0 1px 2px rgba(0,0,0,0.1)}
.post .time{font-size:12px;color:#999;margin-bottom:8px}
.post .user{font-size:12px;color:#576b95;margin-bottom:6px}
.post .text{font-size:15px;line-height:1.5;white-space:pre-wrap}
.post .media{font-size:12px;color:#999;margin-top:6px}
.post .cmt{font-size:13px;color:#333;margin-top:6px;padding:6px 8px;background:#f7f7f7;border-radius:6px}
.post .cmt .who{color:#576b95}
.post .like{font-size:13px;color:#576b95;margin-top:6px}
</style></head><body>`
	html += `<div class="note">说明：朋友圈图片/视频缓存文件名是微信内部不透明 id，无法与具体动态对应，` +
		`因此所有图片已按 年/月 统一导出到账号根目录下的 <b>朋友圈/_媒体缓存/<年>/<月>/</b>` +
		`（缩略图在 <b>_媒体缓存/_缩略图/<年>/<月>/</b>，视频在 <b>_媒体缓存/_视频/<年>/<月>/</b>）；` +
		`下方为该发布者的动态正文、媒体明细、评论与点赞。</div>`
	for _, p := range posts {
		html += `<div class="post"><div class="time">` + time.Unix(int64(p["time"].(float64)), 0).Format("2006-01-02 15:04") + `</div>`
		if u, ok := p["user"]; ok {
			html += `<div class="user">` + fmt.Sprintf("%v", u) + `</div>`
		}
		html += `<div class="text">` + fmt.Sprintf("%v", p["text"]) + `</div>`
		// 媒体明细：数组形式 [{type,width,height,dur,md5}]
		if arr, ok := p["media"].([]any); ok && len(arr) > 0 {
			parts := make([]string, 0, len(arr))
			for _, it := range arr {
				m, _ := it.(map[string]any)
				kind := "图片"
				if t, _ := m["type"].(float64); int(t) == 6 {
					kind = "视频"
				}
				w, _ := m["width"].(float64)
				h, _ := m["height"].(float64)
				parts = append(parts, fmt.Sprintf("%s %dx%d", kind, int(w), int(h)))
			}
			html += `<div class="media">媒体 ` + fmt.Sprintf("%d", len(arr)) + ` 项：` + strings.Join(parts, "，") + `</div>`
		}
		// 点赞
		if arr, ok := p["likes"].([]any); ok && len(arr) > 0 {
			names := make([]string, 0, len(arr))
			for _, it := range arr {
				names = append(names, fmt.Sprintf("%v", it))
			}
			html += `<div class="like">♥ ` + strings.Join(names, "、") + `</div>`
		}
		// 评论
		if arr, ok := p["comments"].([]any); ok && len(arr) > 0 {
			for _, it := range arr {
				m, _ := it.(map[string]any)
				html += `<div class="cmt"><span class="who">` + fmt.Sprintf("%v", m["nick"]) + `</span>：` + fmt.Sprintf("%v", m["text"]) + `</div>`
			}
		}
		html += `</div>`
	}
	html += `</body></html>`
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(html), 0644)
}

// snsIdxRow 是朋友圈 README 索引的一行：序号 / 昵称 / 目录名 / 动态数。
type snsIdxRow struct {
	seq  int
	nick string
	dir  string
	n    int
}

// writeSnsReadme 在「朋友圈/」目录下生成 README.txt 索引，
// 列出所有发布者（序号 / 昵称 / 动态数 / 目录名），方便一眼看清有哪些人。
func writeSnsReadme(root string, idx []snsIdxRow, totalPosts int) {
	var b strings.Builder
	b.WriteString("朋友圈归档索引\n")
	b.WriteString("========================================\n")
	b.WriteString(fmt.Sprintf("发布者共 %d 人，动态共 %d 条\n\n", len(idx), totalPosts))
	b.WriteString("说明：\n")
	b.WriteString("  - 每个发布者一个目录，目录名 = 序号_昵称_短哈希（按动态数降序）。\n")
	b.WriteString("  - 目录内 posts.json 是结构化动态，index.html 可直接浏览器打开。\n")
	b.WriteString("  - 图片缓存无法归属到具体动态，统一放在 _媒体缓存/<年>/<月>/。\n")
	b.WriteString("  - 缩略图（长边<720）单独放在 _媒体缓存/_缩略图/<年>/<月>/。\n")
	b.WriteString("  - 视频（本地缓存的 mp4）放在 _媒体缓存/_视频/<年>/<月>/。\n")
	b.WriteString("  - 陌生人（v3_...@stranger）在通讯录里查不到昵称，目录名用「陌生人_短哈希」。\n\n")
	b.WriteString("序号  动态数  目录名\n")
	b.WriteString("----  ------  ----------------------------------------\n")
	for _, r := range idx {
		nick := r.nick
		if nick == "" {
			nick = "(无昵称)"
		}
		b.WriteString(fmt.Sprintf("%02d    %6d  %s\n", r.seq, r.n, r.dir))
		_ = nick
	}
	os.WriteFile(filepath.Join(root, "README.txt"), []byte(b.String()), 0644)
}

// extractImgHash 从 packed_info_data 中提取 32 位 hex 图片标识。
// packed_info_data 是 protobuf 编码，其中某个字段的值为图片文件名（32 位 hex）。
func extractImgHash(packed []byte) string {
	if len(packed) == 0 {
		return ""
	}
	re := regexp.MustCompile(`[0-9a-f]{32}`)
	m := re.Find(packed)
	if m == nil {
		return ""
	}
	return string(m)
}

// parseVoiceLength 从语音消息 XML 中解析时长（毫秒）。
// 形如 <voicemsg voicelength="11554" .../>，voicelength 单位是毫秒。
func parseVoiceLength(xml string) int {
	re := regexp.MustCompile(`voicelength="(\d+)"`)
	if m := re.FindStringSubmatch(xml); len(m) > 1 {
		n := 0
		fmt.Sscanf(m[1], "%d", &n)
		return n
	}
	return 0
}

// parseVideoPlayLength 从视频消息 XML 中解析播放时长（秒）。
// 形如 <videomsg playlength="9" .../>。
func parseVideoPlayLength(xml string) int {
	re := regexp.MustCompile(`playlength="(\d+)"`)
	if m := re.FindStringSubmatch(xml); len(m) > 1 {
		n := 0
		fmt.Sscanf(m[1], "%d", &n)
		return n
	}
	return 0
}

// parseVideoCDNURL 从视频消息 XML 中提取 CDN 视频地址（仅作记录，本地无文件）。
func parseVideoCDNURL(xml string) string {
	re := regexp.MustCompile(`cdnvideourl="([^"]+)"`)
	if m := re.FindStringSubmatch(xml); len(m) > 1 {
		return m[1]
	}
	return ""
}

// parseAppMsgType 从 type 49（appmsg）消息 XML 中解析 <appmsg> 的 <type> 子类型。
// 常见取值：6=文件，5=链接，57=引用，33/36=小程序，8=表情，19=聊天记录，51=视频号。
// 只有 6 才是真正的文件消息（需要拷贝本地文件）。
func parseAppMsgType(xml string) int {
	// 优先匹配 <appmsg ...> ... <type>N</type>，避免误取其它嵌套 <type>。
	re := regexp.MustCompile(`(?s)<appmsg[^>]*>.*?<type>(\d+)</type>`)
	if m := re.FindStringSubmatch(xml); len(m) > 1 {
		n := 0
		fmt.Sscanf(m[1], "%d", &n)
		return n
	}
	// 退路：取第一个 <type>N</type>。
	re2 := regexp.MustCompile(`<type>(\d+)</type>`)
	if m := re2.FindStringSubmatch(xml); len(m) > 1 {
		n := 0
		fmt.Sscanf(m[1], "%d", &n)
		return n
	}
	return 0
}

// ExportVoices 导出聊天语音。
//
// 实测结论（2026-10-09）：
//   - 语音音频数据不在 message_0.db，而在 media_0.db 的 VoiceInfo 表：
//     VoiceInfo(chat_name_id, create_time, local_id, svr_id, voice_data, data_index)
//   - voice_data 是 SILK v3 格式（头部 \x02#!SILK_V3），微信/QQ 私有音频编码。
//   - chat_name_id 通过 media_0.db 的 Name2Id 表映射到 wxid：
//     Name2Id(rowid, user_name)，rowid 即 chat_name_id。
//
// 输出：<账号>/聊天记录/<好友>/media/<年-月>/voices/<时间戳>_<local_id>.silk
// 说明：SILK 无法用普通播放器直接播放，需转码为 MP3（见 TranscodeVoices）。
// 这里先原样导出 .silk，保证数据不丢；同时写一份 voices.jsonl 索引。
func (c *Config) ExportVoices(vdb *sql.DB, nickOf func(wxid string) string) error {
	// 1. 读 Name2Id：chat_name_id -> wxid
	name2id := map[int64]string{}
	rows, err := vdb.Query("SELECT rowid, user_name FROM Name2Id")
	if err != nil {
		return fmt.Errorf("查询 Name2Id: %w", err)
	}
	for rows.Next() {
		var id int64
		var un string
		rows.Scan(&id, &un)
		name2id[id] = un
	}
	rows.Close()

	// 2. 遍历 VoiceInfo，按好友分组写出
	type voiceRow struct {
		ct   int64
		lid  int64
		blob []byte
	}
	byWxid := map[string][]voiceRow{}
	vr, err := vdb.Query("SELECT chat_name_id, create_time, local_id, voice_data FROM VoiceInfo ORDER BY create_time ASC")
	if err != nil {
		return fmt.Errorf("查询 VoiceInfo: %w", err)
	}
	for vr.Next() {
		var cn, ct, lid int64
		var blob []byte
		vr.Scan(&cn, &ct, &lid, &blob)
		wxid := name2id[cn]
		if wxid == "" {
			wxid = fmt.Sprintf("unknown_%d", cn)
		}
		byWxid[wxid] = append(byWxid[wxid], voiceRow{ct, lid, blob})
	}
	vr.Close()

	total := 0
	for wxid, list := range byWxid {
		nick := ""
		if nickOf != nil {
			nick = nickOf(wxid)
		}
		baseDir := c.chatDir(nick, wxid)
		idxPath := filepath.Join(baseDir, "media", "voices.jsonl")
		os.MkdirAll(filepath.Dir(idxPath), 0755)
		idx, _ := os.Create(idxPath)
		for _, v := range list {
			monthDir := time.Unix(v.ct, 0).Format("2006-01")
			dir := filepath.Join(baseDir, "media", monthDir, "voices")
			os.MkdirAll(dir, 0755)
			name := fmt.Sprintf("%d_%d.silk", v.ct, v.lid)
			os.WriteFile(filepath.Join(dir, name), v.blob, 0644)
			if idx != nil {
				rel := fmt.Sprintf("media/%s/voices/%s", monthDir, name)
				fmt.Fprintf(idx, `{"time":"%s","ts":%d,"type":"voice","file":%q,"bytes":%d}`+"\n",
					time.Unix(v.ct, 0).Format("2006-01-02 15:04:05"), v.ct, rel, len(v.blob))
			}
			total++
		}
		if idx != nil {
			idx.Close()
		}
	}
	fmt.Printf("  [OK] 导出语音 %d 条（SILK v3，按好友/月份归档到 聊天记录/<好友>/media/<年-月>/voices/）\n", total)
	return nil
}

// TranscodeVoices 把导出目录下的 .silk 批量转成可播放的 .mp3。
//
// 依赖外部工具（缺一不可）：
//   - python + pilk（pip install pilk）：把 SILK v3 解码成 PCM
//   - ffmpeg：把 PCM 编码成 mp3
//
// 实现方式：调用仓库自带的 tools/silk2mp3.py 脚本，递归处理整个归档目录。
// 若 python / pilk / ffmpeg 任一缺失，则跳过转码（不报错），保留 .silk 原始文件。
// 转码成功后默认删除 .silk（脚本内行为），只留 .mp3。
func (c *Config) TranscodeVoices() error {
	script := filepath.Join("tools", "silk2mp3.py")
	if _, err := os.Stat(script); err != nil {
		fmt.Println("  [i] 未找到 tools/silk2mp3.py，跳过语音转码（保留 .silk）")
		return nil
	}
	python := findPython()
	if python == "" {
		fmt.Println("  [i] 未找到 python，跳过语音转码（保留 .silk）")
		return nil
	}
	// 先探测 pilk 是否可用
	chk := exec.Command(python, "-c", "import pilk")
	if err := chk.Run(); err != nil {
		fmt.Println("  [i] 未安装 pilk（pip install pilk），跳过语音转码（保留 .silk）")
		return nil
	}
	root := filepath.Join(c.accountDir(), "聊天记录")
	cmd := exec.Command(python, script, root)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("语音转码: %w", err)
	}
	return nil
}

// findPython 在 PATH 与常见安装位置里找一个可用的 python。
func findPython() string {
	for _, name := range []string{"python", "python3", "py"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	for _, p := range []string{
		`C:\Python310\python.exe`,
		`C:\Python313\python.exe`,
		`C:\Python312\python.exe`,
		`C:\Python311\python.exe`,
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func sanitize(s string) string {
	for _, ch := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
		s = strings.ReplaceAll(s, ch, "_")
	}
	return s
}
