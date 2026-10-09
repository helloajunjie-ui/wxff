# wxff · 微信本地备份工具

> 把 **PC 微信** 指定好友的 **聊天记录** 和 **朋友圈** 结构化备份到本地。
> 路线：解密本地数据库 + 必要时驱动微信浏览，**不注入、不 hook**。

目标版本：**微信 4.1.9.57（4.x）**，SQLCipher 4。

---

## 特性

- **数据库解密**：从 `Weixin.exe` 进程内存扫 SQLCipher 4 密钥，纯 Go AES-256-CBC 解密，只读不改原库。
- **聊天记录**：文字 / 图片 / 文件 / 语音 / 视频，按好友隔离导出，生成可浏览的 `index.html` + 结构化 `messages.jsonl`。
- **图片解密**：微信 V4 `.dat` 格式（AES-128-ECB + 尾部单字节 XOR），导出真实原图。
- **语音转码**：`media_0.db` 的 `VoiceInfo` 表（SILK v3）导出后经 `pilk` + `ffmpeg` 自动转成可播放 `.mp3`。
- **朋友圈**：按发布者隔离导出，缩略图 / 原图分离，视频缓存单独落盘。
- **多账号隔离**：每个微信账号一个顶层目录，用中文昵称一眼分清。
- **Web 前端**：`wxff-server` 提供账号 / 好友选择界面，勾选即备份。

---

## 归档结构

三条硬性要求：**① 多账号隔离；② 好友隔离；③ 聊天媒体与朋友圈媒体分开。**

```
<输出目录>/
└── <账号昵称>_<账号wxid>/                 ← ① 每个微信账号一个顶层目录
    │
    ├── 聊天记录/                          ← ③ 聊天媒体（与朋友圈完全分开）
    │   └── <好友备注或昵称>_<wxid短哈希>/   ← ② 好友隔离（同名好友不覆盖）
    │       ├── index.html                  ← 浏览器打开就能翻完整对话
    │       ├── messages.jsonl              ← 结构化数据（每行一条消息）
    │       └── media/
    │           ├── images/                 ← 聊天图片（按 年-月 分目录）
    │           ├── videos/
    │           ├── files/                  ← 收发的原文件
    │           └── voices/                 ← 语音（.silk + 转码后的 .mp3）
    │
    └── 朋友圈/                             ← ③ 朋友圈媒体（与聊天完全分开）
        ├── README.txt                      ← 索引：序号/动态数/目录名
        ├── _媒体缓存/                       ← 朋友圈图片缓存（整账号一次，按「年/月」两级）
        │   ├── <年>/<月>/                   ← 原图（长边 ≥ -sns-min，默认 720）
        │   └── _缩略图/<年>/<月>/            ← 缩略图（长边 < -sns-min），与原图分开
        └── <序号>_<发布者备注或昵称>_<wxid短哈希>/   ← ④ 按发布者隔离
            ├── index.html
            └── posts.json                  ← 该发布者的动态（时间、文字、媒体、评论、点赞）
```

> 解密后的临时 db **不在归档目录内**，放在 `<输出目录的父目录>/decrypted/`，避免污染归档结构。

---

## 构建

```bash
# 只构建主程序（tools/ 下有多个 package main，不能直接 go build ./...）
go build -o wxff.exe ./cmd/wxff

# 构建 Web 后端
go build -o wxff-server.exe ./cmd/wxff-server

# 构建全部正式命令
go build ./internal/... ./cmd/wxff/... ./cmd/wxff-server/...
```

依赖：Go 1.26+，`modernc.org/sqlite`（纯 Go，无需 CGO）。

---

## 使用

### 1. 提取数据库密钥

微信运行时，从进程内存提取 SQLCipher 4 密钥，写入 `keys.txt`：

```bash
go run ./cmd/keyextract
```

### 2. 命令行导出

```bash
# 导出所有好友的聊天记录
wxff.exe -all

# 导出指定好友
wxff.exe -user wxid_xxx,wxid_yyy

# 列出朋友圈发布者（昵称 + wxid + 动态数）
wxff.exe -sns-list

# 只导出指定发布者的朋友圈
wxff.exe -sns-user wxid_xxx

# 导出全部发布者的朋友圈
wxff.exe -moments
```

### 3. Web 前端

```bash
wxff-server.exe
# 浏览器打开 http://127.0.0.1:<port>
```

前端流程：选择账号 → 勾选好友 → 勾选内容 → 开始备份 → 一键打开归档目录。

---

## 命令行参数（wxff.exe）

| 参数 | 默认 | 说明 |
|---|---|---|
| `-data` | 自动探测 | 微信数据根目录（形如 `...\xwechat_files\<wxid>_<hash>`） |
| `-out` | `data/backup` | 归档输出目录 |
| `-user` | — | 好友 wxid，多个用逗号分隔 |
| `-all` | `false` | 导出所有好友（以 `message_0.db` 中存在 `Msg_<md5(wxid)>` 表为准） |
| `-type` | `text,image,file,voice,video` | 导出内容类型（逗号分隔）。含 `voice` 时导出语音并自动转 mp3 |
| `-moments` | `false` | 导出全部发布者的朋友圈 |
| `-sns-list` | `false` | 只列出朋友圈发布者（昵称 + wxid + 动态数），不导出 |
| `-sns-user` | — | 只导出指定发布者的朋友圈（wxid，可逗号分隔多个） |
| `-sns-min` | `720` | 朋友圈图片最小长边（像素）。小于此值的缩略图归入 `_缩略图/`；`0` = 不过滤 |
| `-from` / `-to` | `0` | 时间范围过滤（Unix 秒，0 = 不限） |
| `-list` | `false` | 列出最近会话 |
| `-imgkey` | 自动提取 | 手动指定图片 AES key（16 字符），微信未运行时用 |
| `-xorkey` | 自动提取 | 手动指定 XOR key（0-255），微信未运行时用 |

示例（微信未运行时手动指定图片密钥）：

```bash
wxff.exe -all -imgkey b30e5d7754486daf -xorkey 230
```

---

## 语音转码依赖

缺失则自动跳过，保留 `.silk` 原文件：

- `pip install pilk`（纯 Python SILK v3 解码器）
- `ffmpeg`（PCM → mp3）

转码脚本：[`tools/silk2mp3.py`](tools/silk2mp3.py:1)

---

## 技术要点

### 数据库布局（微信 4.x）

| 库 | 路径 | 说明 |
|---|---|---|
| 联系人 | `db_storage\contact\contact.db` | 好友列表、昵称、备注 |
| 消息 | `db_storage\message\message_0.db` | 聊天消息（按时间分片） |
| 媒体索引 | `db_storage\message\media_0.db` | **语音库**：`VoiceInfo` 表存 SILK v3 音频 |
| 朋友圈 | `db_storage\sns\sns.db` | `SnsTimeLine` 表 |
| 会话列表 | `db_storage\session\session.db` | 最近会话 |

### 关键实测结论

- **`local_type` 是 64 位**，真实消息类型在低 16 位（`local_type & 0xFFFF`）。
  分布：`1`=文本、`3`=图片、`34`=语音、`43`=视频、`49`=appmsg、`50`=通话、`10000`=系统消息。
- **type 49 是 appmsg 大类**，包含文件 / 链接 / 引用 / 小程序等。
  必须看 `<appmsg>` 里的 `<type>` 子类型：**`6` = 文件**（唯一需要拷贝本地文件的类型），
  `5` = 链接、`57` = 引用、`33/36` = 小程序、`8` = 表情、`19` = 聊天记录、`51` = 视频号。
  仅凭是否含 `<appattach>` 判断会大量误判。
- **聊天视频本地不落原文件**，消息 XML 里只有 CDN 地址 + 缩略图。
- **朋友圈缓存文件名是 30 位 hex 的不透明 id**，无法与 XML 里的 `md5`/`url`/`media id` 建立对应关系，
  因此按月份统一导出，不做「按 hash 挂到某条动态」。
- **朋友圈有物理边界**：`SnsTimeLine` 只存本机实际浏览过的动态，没浏览过的库里就没有。
- **朋友圈缓存是压缩版**，不等于发帖原图。

更多细节见 [`DESIGN.md`](DESIGN.md:1)。

---

## 风险与边界

- 工具读取的是**你自己账号在本机落库的数据**，不越权访问别人的。
- 微信每次大版本升级都可能让密钥提取失效；失效时需回退版本或等待更新。
- 朋友圈有物理边界：**没浏览过的动态库里就没有**。
- 备份期间微信窗口可能被短暂接管（朋友圈阶段），聊天记录阶段不需要动微信窗口。

---

## 项目结构

```
cmd/
  wxff/            # 主程序（命令行导出）
  wxff-server/     # Web 后端 + 前端页面
  keyextract/      # 从进程内存提取 SQLCipher 密钥
  imgkeyextract/   # 从进程内存提取图片 AES key
  decryptdb/       # 解密单个 db 文件
internal/
  dbkey/           # SQLCipher 4 密钥提取
  decryptedb/      # db 解密
  imgkey/          # 图片 .dat 解密
  procwin/         # Windows 进程内存访问封装
  archive/         # 归档导出（聊天 / 朋友圈 / 语音）
probe/             # 调试探针（一次性分析脚本，非正式代码）
tools/             # 辅助脚本（图片检查、语音转码等）
```

---

## 免责声明

本工具仅供**个人数据备份**用途，用于备份你自己账号在本机的聊天记录与朋友圈。
请遵守相关法律法规，勿用于侵犯他人隐私或任何非法用途。
