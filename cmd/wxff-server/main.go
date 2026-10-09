// wxff-server 微信备份工具后端服务
package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nicols/wxff/internal/archive"
	"github.com/nicols/wxff/internal/decryptedb"
	"github.com/nicols/wxff/internal/procwin"
	_ "modernc.org/sqlite"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
)

func (s *Server) findWechatWindow() uintptr {
	// 找 Weixin.exe 的主窗口
	_, err := procwin.FindMainWeixinPID()
	if err != nil {
		return 0
	}
	// 简化：后面再实现 EnumWindows 回调
	return 0
}

type Server struct {
	dataRoot   string
	outDir     string
	account    string // 当前账号 wxid
	keys       map[string][]byte
	imageKey   string
	xorKey     int
	cdb        *sql.DB
	mdb        *sql.DB
	sdb        *sql.DB
	plainDir   string
	mu         sync.Mutex
	progress   BackupProgress
	wechatHwnd uintptr
}

type BackupProgress struct {
	Running    bool   `json:"running"`
	Total      int    `json:"total"`
	Done       int    `json:"done"`
	Current    string `json:"current"`
	CurrentCnt int    `json:"current_cnt"`
	Finished   bool   `json:"finished"`
	Error      string `json:"error,omitempty"`
}

func main() {
	dataRoot := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6`
	outDir := `F:\wxff\data\backup`
	plainDir := filepath.Join(outDir, "_plain")
	os.MkdirAll(plainDir, 0755)

	// 从 dataRoot 提账号 wxid，并解析账号自身昵称，顶层目录用「昵称_wxid」。
	account := resolveAccountDir(dataRoot)

	s := &Server{
		dataRoot: dataRoot,
		outDir:   outDir,
		account:  account,
		plainDir: plainDir,
	}

	http.Handle("/", http.FileServer(http.Dir(`F:\wxff\cmd\wxff-server\web`)))
	http.HandleFunc("/api/accounts", s.handleAccounts)
	http.HandleFunc("/api/init", s.handleInit)
	http.HandleFunc("/api/contacts", s.handleContacts)
	http.HandleFunc("/api/backup", s.handleBackup)
	http.HandleFunc("/api/progress", s.handleProgress)

	fmt.Println("服务启动: http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, "xwechat_files")

	type AccountInfo struct {
		wxid     string
		nick     string
		dataRoot string
		mtime    time.Time
		ok       bool
	}
	var accounts []AccountInfo
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "wxid_") {
			continue
		}
		dbStorage := filepath.Join(root, name, "db_storage")
		if _, err := os.Stat(dbStorage); err != nil {
			continue
		}
		wxid := name
		if idx := strings.LastIndex(name, "_"); idx > 0 && len(name)-idx-1 == 4 {
			wxid = name[:idx]
		}
		// 找 keys
		nick := ""
		ok := false
		contactDb := filepath.Join(dbStorage, "contact", "contact.db")
		if keys, err := loadKeysFromDir(dbStorage); err == nil {
			if key, has := keys["contact/contact.db"]; has {
				plainPath := filepath.Join(os.TempDir(), "contact_"+wxid+".db")
				if decryptedb.DecryptFile(contactDb, plainPath, key) == nil {
					if db, err := sql.Open("sqlite", plainPath); err == nil {
						db.QueryRow("SELECT nick_name FROM contact WHERE username=?", wxid).Scan(&nick)
						db.Close()
						ok = true
					}
					os.Remove(plainPath)
				}
			}
		}
		// 没密钥的直接跳过，不显示
		if !ok {
			continue
		}
		// 找最新修改时间
		maxTime := time.Time{}
		filepath.Walk(dbStorage, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if info.ModTime().After(maxTime) {
				maxTime = info.ModTime()
			}
			return nil
		})
		accounts = append(accounts, AccountInfo{wxid, nick, filepath.Join(root, name), maxTime, ok})
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].mtime.After(accounts[j].mtime) })

	type Account struct {
		Wxid     string `json:"wxid"`
		Nick     string `json:"nick"`
		DataRoot string `json:"data_root"`
		Recent   bool   `json:"recent"`
		Ok       bool   `json:"ok"`
	}
	var list []Account
	for i, a := range accounts {
		list = append(list, Account{a.wxid, a.nick, a.dataRoot, i == 0, a.ok})
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "accounts": list})
}

func (s *Server) handleInit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// 允许前端指定 data_root（多账号选择）；未指定则自动选最近使用的账号。
	var req struct {
		DataRoot string `json:"data_root"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}
	bestDir := req.DataRoot
	if bestDir == "" {
		// 自动找当前微信数据目录（最近修改的）
		home, _ := os.UserHomeDir()
		root := filepath.Join(home, "xwechat_files")
		entries, _ := os.ReadDir(root)
		var bestTime time.Time
		for _, e := range entries {
			if !e.IsDir() || !strings.HasPrefix(e.Name(), "wxid_") {
				continue
			}
			dbStorage := filepath.Join(root, e.Name(), "db_storage")
			if _, err := os.Stat(dbStorage); err != nil {
				continue
			}
			info, _ := e.Info()
			if info.ModTime().After(bestTime) {
				bestTime = info.ModTime()
				bestDir = filepath.Join(root, e.Name())
			}
		}
	}
	if bestDir == "" {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "没找到微信数据目录"})
		return
	}
	s.dataRoot = bestDir
	leaf := filepath.Base(bestDir)
	account := leaf
	if idx := strings.LastIndex(leaf, "_"); idx > 0 && len(leaf)-idx-1 == 4 {
		account = leaf[:idx]
	}
	s.account = account

	// 自动读 keys.txt
	keys, err := loadKeys(`F:\wxff\data\keys.txt`)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "读 keys.txt 失败: " + err.Error()})
		return
	}
	s.keys = keys

	// 自动推 image key
	imageKey, xorKey, err := deriveImageKey(bestDir)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "推导 image key 失败: " + err.Error()})
		return
	}
	s.imageKey = imageKey
	s.xorKey = xorKey

	// 自动解密 db
	plainDir := filepath.Join(s.outDir, "_plain")
	os.MkdirAll(plainDir, 0755)
	need := []string{"contact/contact.db", "message/message_0.db", "sns/sns.db"}
	for _, rel := range need {
		key := keys[rel]
		src := filepath.Join(bestDir, "db_storage", filepath.FromSlash(rel))
		dst := filepath.Join(plainDir, strings.ReplaceAll(rel, "/", "__"))
		if _, err := os.Stat(dst); err != nil {
			if err := decryptedb.DecryptFile(src, dst, key); err != nil {
				json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "解密 " + rel + ": " + err.Error()})
				return
			}
		}
	}

	// 打开 db
	s.cdb, _ = sql.Open("sqlite", filepath.Join(plainDir, "contact__contact.db"))
	s.mdb, _ = sql.Open("sqlite", filepath.Join(plainDir, "message__message_0.db"))
	s.sdb, _ = sql.Open("sqlite", filepath.Join(plainDir, "sns__sns.db"))

	// 查本账号昵称（用于前端清晰显示「当前账号」）
	accountNick := ""
	if s.cdb != nil {
		s.cdb.QueryRow("SELECT nick_name FROM contact WHERE username=?", account).Scan(&accountNick)
	}

	json.NewEncoder(w).Encode(map[string]any{
		"ok":           true,
		"account":      account,
		"account_nick": accountNick,
		"data_root":    bestDir,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pid, err := procwin.FindMainWeixinPID()
	wechatRunning := err == nil
	wechatPID := 0
	if wechatRunning {
		wechatPID = int(pid)
		if s.wechatHwnd == 0 {
			s.wechatHwnd = s.findWechatWindow()
		}
	}
	hasKeys := len(s.keys) > 0
	hasImageKey := s.imageKey != ""
	json.NewEncoder(w).Encode(map[string]any{
		"wechat_running":  wechatRunning,
		"wechat_pid":      wechatPID,
		"wechat_hwnd":     s.wechatHwnd,
		"db_keys_ready":   hasKeys,
		"image_key_ready": hasImageKey,
		"image_key":       s.imageKey,
	})
}

func (s *Server) handleExtractKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	dataRoot := r.URL.Query().Get("data_root")
	if dataRoot == "" {
		dataRoot = s.dataRoot
	}
	s.dataRoot = dataRoot
	// 重算 account（顶层目录用「昵称_wxid」）
	s.account = resolveAccountDir(dataRoot)
	// 1. 扫 db keys
	// TODO: 调用 keyextract 逻辑
	// 先从 keys.txt 读
	keys, err := loadKeys(`F:\wxff\data\keys.txt`)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.keys = keys

	// 2. 推 image key
	imageKey, xorKey, err := deriveImageKey(s.dataRoot)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.imageKey = imageKey
	s.xorKey = xorKey

	// 3. 解密 db
	need := []string{"contact/contact.db", "message/message_0.db", "sns/sns.db"}
	for _, rel := range need {
		key := s.keys[rel]
		src := filepath.Join(s.dataRoot, "db_storage", filepath.FromSlash(rel))
		dst := filepath.Join(s.plainDir, strings.ReplaceAll(rel, "/", "__"))
		if _, err := os.Stat(dst); err != nil {
			if err := decryptedb.DecryptFile(src, dst, key); err != nil {
				json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "解密 " + rel + ": " + err.Error()})
				return
			}
		}
	}

	// 4. 打开 db
	s.cdb, _ = sql.Open("sqlite", filepath.Join(s.plainDir, "contact__contact.db"))
	s.mdb, _ = sql.Open("sqlite", filepath.Join(s.plainDir, "message__message_0.db"))
	s.sdb, _ = sql.Open("sqlite", filepath.Join(s.plainDir, "sns__sns.db"))

	json.NewEncoder(w).Encode(map[string]any{
		"ok":        true,
		"db_count":  len(s.keys),
		"image_key": imageKey,
		"xor_key":   fmt.Sprintf("0x%02X", xorKey),
	})
}

func (s *Server) handleContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cdb == nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "请先选账号初始化"})
		return
	}

	// 1. 一次性算所有聊天统计
	msgTables, _ := s.mdb.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
	type chatStats struct{ text, img, file, video int }
	statsMap := map[string]chatStats{}
	for msgTables.Next() {
		var t string
		msgTables.Scan(&t)
		var st chatStats
		s.mdb.QueryRow("SELECT sum(CASE WHEN local_type=1 THEN 1 ELSE 0 END), sum(CASE WHEN local_type=3 THEN 1 ELSE 0 END), sum(CASE WHEN local_type=49 THEN 1 ELSE 0 END), sum(CASE WHEN local_type=43 THEN 1 ELSE 0 END) FROM "+t).Scan(&st.text, &st.img, &st.file, &st.video)
		statsMap[strings.TrimPrefix(t, "Msg_")] = st
	}
	msgTables.Close()

	// 2. 一次性算所有朋友圈统计
	snsMap := map[string]struct{ count, img, video int }{}
	snsRows, _ := s.sdb.Query("SELECT user_name, count(*), sum(CASE WHEN content LIKE '%<thumb%' THEN 1 ELSE 0 END) FROM SnsTimeLine GROUP BY user_name")
	for snsRows.Next() {
		var u string
		var cnt, img int
		snsRows.Scan(&u, &cnt, &img)
		snsMap[u] = struct{ count, img, video int }{cnt, img, 0}
	}
	snsRows.Close()

	// 3. 查好友
	rows, _ := s.cdb.Query("SELECT username, nick_name, remark, alias, small_head_url FROM contact WHERE username NOT LIKE '%@chatroom' AND username NOT LIKE 'gh_%' ORDER BY quan_pin")
	defer rows.Close()
	type Contact struct {
		UserName  string `json:"wxid"`
		NickName  string `json:"nick"`
		Remark    string `json:"remark"`
		Alias     string `json:"alias"`
		HeadUrl   string `json:"head_url"`
		ChatText  int    `json:"chat_text"`
		ChatImg   int    `json:"chat_img"`
		ChatFile  int    `json:"chat_file"`
		ChatVideo int    `json:"chat_video"`
		SnsCount  int    `json:"sns_count"`
		SnsImg    int    `json:"sns_img"`
		SnsVideo  int    `json:"sns_video"`
	}
	var list []Contact
	for rows.Next() {
		var c Contact
		rows.Scan(&c.UserName, &c.NickName, &c.Remark, &c.Alias, &c.HeadUrl)
		sum := md5.Sum([]byte(c.UserName))
		m := hex.EncodeToString(sum[:])
		stats := statsMap[m]
		c.ChatText = stats.text
		c.ChatImg = stats.img
		c.ChatFile = stats.file
		c.ChatVideo = stats.video
		sns := snsMap[c.UserName]
		c.SnsCount = sns.count
		c.SnsImg = sns.img
		if c.ChatText+c.ChatImg+c.ChatFile+c.ChatVideo > 0 || c.SnsCount > 0 {
			list = append(list, c)
		}
	}
	// 当前账号昵称（前端顶部显示「当前账号」）
	accountNick := ""
	if s.cdb != nil {
		s.cdb.QueryRow("SELECT nick_name FROM contact WHERE username=?", s.account).Scan(&accountNick)
	}
	json.NewEncoder(w).Encode(map[string]any{
		"ok":           true,
		"account":      s.account,
		"account_nick": accountNick,
		"contacts":     list,
	})
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Users     []string `json:"users"`
		ChatTypes []string `json:"chat_types"`
		SnsTypes  []string `json:"sns_types"`
		From      int64    `json:"from"`
		To        int64    `json:"to"`
		OutDir    string   `json:"out_dir"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	chatTypes := map[string]bool{}
	for _, t := range req.ChatTypes {
		chatTypes[t] = true
	}
	snsTypes := map[string]bool{}
	for _, t := range req.SnsTypes {
		snsTypes[t] = true
	}

	s.mu.Lock()
	s.progress = BackupProgress{
		Running: true,
		Total:   len(req.Users),
	}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.progress.Finished = true
			s.progress.Running = false
			s.mu.Unlock()
		}()

		cfg := archive.Config{
			DataRoot:  s.dataRoot,
			OutDir:    req.OutDir,
			ImageKey:  s.imageKey,
			XORKey:    s.xorKey,
			Types:     chatTypes,
			SnsTypes:  snsTypes,
			StartTime: req.From,
			EndTime:   req.To,
			Account:   s.account,
		}

		for i, wxid := range req.Users {
			s.mu.Lock()
			s.progress.Done = i
			s.progress.Current = wxid
			s.mu.Unlock()

			var nick string
			s.cdb.QueryRow("SELECT nick_name FROM contact WHERE username=?", wxid).Scan(&nick)
			// 导出聊天记录
			if len(chatTypes) > 0 {
				if err := cfg.ExportContact(wxid, nick, s.mdb); err != nil {
					continue
				}
			}
			// 导出朋友圈
			if len(snsTypes) > 0 {
				cfg.ExportMoments(wxid, nick, s.sdb, s.cdb)
			}
		}
	}()

	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.mu.Lock()
	defer s.mu.Unlock()
	json.NewEncoder(w).Encode(s.progress)
}

func (s *Server) handleAutoScroll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Wxid  string `json:"wxid"`
		Times int    `json:"times"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Times == 0 {
		req.Times = 30
	}
	// 查昵称
	var nick string
	s.cdb.QueryRow("SELECT nick_name FROM contact WHERE username=?", req.Wxid).Scan(&nick)
	// 调用 Python 脚本，传昵称
	cmd := exec.Command(`C:\Python310\python.exe`, `F:\wxff\tools\auto_scroll.py`, nick, fmt.Sprintf("%d", req.Times))
	if err := cmd.Start(); err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	go cmd.Wait()
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "msg": "开始刷朋友圈，期间不要动鼠标键盘"})
}

func loadKeysFromDir(dbRoot string) (map[string][]byte, error) {
	// 从全局 keys.txt 读
	data, err := os.ReadFile(`F:\wxff\data\keys.txt`)
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

// resolveAccountDir 解析账号顶层目录名：优先「昵称_wxid」，昵称为空时退回 wxid。
// 账号自身昵称存在 contact 表（username=账号wxid，local_type=1），
// 用中文昵称做顶层目录比裸 wxid 直观。解密失败/查不到时退回 wxid。
func resolveAccountDir(dataRoot string) string {
	leaf := filepath.Base(dataRoot)
	wxid := leaf
	if idx := strings.LastIndex(leaf, "_"); idx > 0 && len(leaf)-idx-1 == 4 {
		wxid = leaf[:idx]
	}
	nick := ""
	contactDb := filepath.Join(dataRoot, "db_storage", "contact", "contact.db")
	if keys, err := loadKeysFromDir(filepath.Join(dataRoot, "db_storage")); err == nil {
		if key, has := keys["contact/contact.db"]; has {
			plainPath := filepath.Join(os.TempDir(), "acct_"+wxid+".db")
			if decryptedb.DecryptFile(contactDb, plainPath, key) == nil {
				if db, err := sql.Open("sqlite", plainPath); err == nil {
					db.QueryRow("SELECT nick_name FROM contact WHERE username=?", wxid).Scan(&nick)
					db.Close()
				}
				os.Remove(plainPath)
			}
		}
	}
	return archive.AccountDirName(nick, wxid)
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
	}
	radiumRoot := filepath.Join(home, "AppData", "Roaming", "Tencent", "xwechat", "radium", "ilink")
	if entries, err := os.ReadDir(radiumRoot); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(radiumRoot, e.Name(), "kvcomm"))
			}
		}
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "key_") {
				rest := strings.TrimPrefix(e.Name(), "key_")
				code := strings.Split(rest, "_")[0]
				sum := md5.Sum([]byte(code + wxid))
				return fmt.Sprintf("%x", sum)[:16], 0xE6, nil
			}
		}
	}
	return "", 0, fmt.Errorf("kvcomm 里没找到 key_*.statistic")
}
