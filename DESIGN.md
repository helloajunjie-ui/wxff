# 微信本地备份工具 · 设计文档 v1.0（定稿）

> 范围：**PC 微信**，把指定好友的**聊天记录**和**朋友圈**结构化备份到本地。
> 路线：解密本地数据库 + 必要时自动驱动微信浏览，不注入不 hook。

---

## 1. 最终交付物（锁死）

一个本地工具，用户打开后做这几件事：

1. 工具自动探测本机微信（版本、数据目录、当前登录 wxid）。
2. 列出好友列表（从 MicroMsg.db 读），用户勾选**单人 / 多人 / 分组**。
3. 勾选要备份的内容：☑ 聊天记录（文字/图/视频/文件）  ☑ 朋友圈（文字/图）。
4. 点"开始"。工具自动跑：解密数据库 → 读消息/动态 → 归集媒体文件 → 生成归档。
5. 完成后一键打开归档目录。

**用户全程不需要手动打开微信刷朋友圈，不需要敲命令，不需要碰微信设置。**

## 2. 归档结构（不混）

三条硬性要求：**① 多账号隔离；② 好友隔离；③ 聊天媒体与朋友圈媒体分开。**

```
<输出目录>/
└── <账号昵称>_<账号wxid>/           ← ① 每个微信账号一个顶层目录，多账号互不混淆
    │
    ├── 聊天记录/                    ← ③ 聊天媒体（与朋友圈完全分开）
    │   └── <好友备注或昵称>_<wxid短哈希>/   ← ② 好友隔离（同名好友不覆盖）
    │       ├── index.html            ← 浏览器打开就能翻完整对话
    │       ├── messages.jsonl         ← 结构化数据（每行一条消息，供后续处理）
    │       └── media/
    │           ├── images/            ← 聊天里的图（按 年-月 分目录）
    │           ├── videos/
    │           ├── files/            ← 收发的原文件
    │           └── voices/            ← V1 先不导出语音，留空
    │
    └── 朋友圈/                       ← ③ 朋友圈媒体（与聊天完全分开）
        ├── README.txt                 ← ⑥ 索引：序号/动态数/目录名，一眼看清有哪些人
        ├── _媒体缓存/                 ← 朋友圈图片缓存（整账号一次，按「年/月」两级）
        │   ├── <年>/                  ← 先按年分组（避免顶层堆 60+ 个月份目录）
        │   │   └── <月>/              ← 原图（长边 ≥ -sns-min，默认 720）
        │   │       └── <30位hex>.jpg  ← 文件名=微信内部 30 位 hex id，无法归属到人
        │   └── _缩略图/               ← 缩略图（长边 < -sns-min），与原图分开
        │       └── <年>/<月>/
        └── <序号>_<发布者备注或昵称>_<wxid短哈希>/   ← ④ 按发布者隔离（每人独立目录）
            ├── index.html
            └── posts.json            ← 该发布者的动态（时间、文字、媒体明细、评论、点赞）
```

> 解密后的临时 db **不在归档目录内**，放在 `<输出目录的父目录>/decrypted/`，
> 避免 `_plain` 之类的中间产物污染归档结构。

- **① 多账号**：顶层目录名 = `账号昵称_wxid`（形如 `彼岸轮回_wxid_mpvlokr4ycnu22`）。
  账号自身昵称从 contact 表查（`username=账号wxid`，`local_type=1`），查不到时退回裸 wxid。
  多账号时各自一个顶层目录，用中文昵称一眼分清是哪个号。
- **② 好友隔离**：好友目录名 = `备注/昵称` + `_` + `md5(wxid)[:8]`，保证同名好友不互相覆盖。
- **③ 媒体分离**：聊天媒体在 `聊天记录/<好友>/media/`，朋友圈媒体在 `朋友圈/_媒体缓存/`，两套目录物理隔离。
- **④ 朋友圈按发布者隔离**：`SnsTimeLine.user_name` 是**发布者** wxid，因此每个发布者一个目录
  `朋友圈/<序号>_<昵称>_<md5(wxid)[:8]>/`，内含该发布者的 `posts.json` + `index.html`。
  可用 `-sns-list` 列出全部发布者，用 `-sns-user <wxid>` 只导出指定发布者。
- **⑤ 缩略图/原图分离（实测结论）**：微信朋友圈缓存 `cache/<年-月>/Sns/Img/` 里
  **缩略图与原图混存**，且都是独立的 30 位 hex 文件，无法从文件名区分。实测某账号 2834 张：
  长边 ≤320px 的缩略图 1465 张（如 276×276 有 872 张、150×300 有 116 张），
  长边 ≥720px 的大图 956 张。导出时用 `image.DecodeConfig` 读像素尺寸，
  长边 < `-sns-min`（默认 720）的归入 `_媒体缓存/_缩略图/`，其余留在 `_媒体缓存/<年>/<月>/`。
  设 `-sns-min 0` 可关闭过滤（全部当原图）。
- **⑥ 目录可读性（避免「目录太多头晕」）**：
  - **序号前缀**：发布者目录按动态数**降序**编号 `01_`、`02_`…，一眼看出谁最重要（实测 42 人）。
  - **陌生人命名**：`v3_...@stranger` 这类非好友在通讯录里查不到昵称，目录名统一为
    `陌生人_<md5(wxid)[:8]>`，不再显示 200+ 字符的裸 wxid。
  - **年份分组**：`_媒体缓存/` 顶层只放年份（`2021`…`2026`），月份收在年份下，
    避免顶层堆 60+ 个月份目录。
  - **README.txt 索引**：`朋友圈/README.txt` 列出全部发布者的序号/动态数/目录名，当索引用。
- **缓存是压缩版，非原始尺寸**：朋友圈 XML 里记录的 `size` 是服务端原图尺寸
  （实测有 1074×8464、1080×7081 等长截图），这些尺寸在本地缓存中**不存在**；
  本地缓存存的是压缩后的版本，因此导出的是「缓存里能找到的最大版本」，不等于发帖原图。
- 名字里不能用的文件系统字符自动替换为 `_`；目录名按字符截断到 40 字符以内
  （部分 wxid 形如 `v3_...@stranger_...` 长达 200+ 字符，直接做目录名会超出 Windows 路径上限）。

## 3. 技术路线

### 3.1 共用底座：数据库解密
- 从 `Weixin.exe` 进程内存扫 SQLCipher 密钥（参考 PyWxDump 开源实现）。
- 当前目标版本：**微信 4.1.9.57（4.x）**，数据根目录 `C:\Users\Administrator\xwechat_files\<wxid>_<hash>\`，SQLCipher 4。
- 解密出来的 db 落到临时目录，只读，不改原库（注意 WAL 文件要一起拷，否则丢最近数据）。
- 3.x 旧版（`WeChat Files\`）作为兼容目标，不在 M1 范围。

### 3.2 聊天记录备份（主战场，数据全在库里）
4.x 实际库布局（已探测本机）：
- 联系人：`db_storage\contact\contact.db`
- 消息：`db_storage\message\message_0.db ~ message_3.db`（按时间分片）
- 媒体索引：`db_storage\message\media_0.db`（111MB，**语音库**：`VoiceInfo` 表存 SILK v3 音频）
- 会话列表：`db_storage\session\session.db`
- 媒体文件本体：`msg\attach\`（图片）、`msg\file\`（文件，不加密），按 db 里的 key 找到并拷贝。
- **语音**：音频数据不在消息表里，而在 `media_0.db` 的 `VoiceInfo` 表（SILK v3），按 `chat_name_id` 归属好友。
- **聊天视频**：本地**不落原视频**，消息 XML 里只有 CDN 地址 + 缩略图（详见 9.8）。
- **这部分不需要驱动 UI**，库读出来就有，比朋友圈干净。

### 3.3 朋友圈备份（实测结论，2026-10-09）
- 主库：`db_storage\sns\sns.db`（本机 7.3MB）。
- 表 `SnsTimeLine`，列：`tid / user_name / content / pack_info_buf`。
  - `user_name` 是**发布者** wxid，不是被查看的好友。
  - `content` 是 `<SnsDataItem><TimelineObject>...` XML，含 `createTime`、`contentDesc`、`mediaList`。
- **物理边界（关键）**：`SnsTimeLine` 只存**本机实际浏览过**的动态（本机缓存），本机实测 **1326 条 / 42 个发布者**。
  没浏览过的朋友圈，库里没有；某个好友若没发过/没被浏览过，按 `user_name=好友` 查就是 0 条。
- **导出策略（按发布者隔离）**：因为 `user_name` 是发布者，所以**按发布者**导出，每人一个目录
  `朋友圈/<昵称>_<md5(wxid)[:8]>/`（`posts.json` + `index.html`）。
  - `-sns-list`：列出全部发布者（wxid / 昵称 / 动态数），供选择。
  - `-sns-user <wxid>`：只导出指定发布者（可逗号分隔多个）。
  - `-moments`：导出全部发布者，每人独立目录。
- **图片缓存**：`cache\<年-月>\Sns\Img\<2hex>\<30hex>`（无扩展名），视频在 `Sns\Video\<2hex>\<30hex>.mp4`（**本地有真实 mp4**）。
  - 文件名是 **30 位 hex** 的微信内部不透明 id，**无法**与 XML 里的 `md5`(32hex)/`url`/`media id` 建立对应关系（已穷举验证：md5(url)、md5(mediaID)、key 属性等全部 0 命中）。
  - 因此**不做「按 hash 把图挂到某条动态」**，而是把缓存里所有能完整解码的图片**按月份**统一导出到
    `朋友圈/_媒体缓存/<年-月>/`（整账号一次，所有发布者共用）。
  - 本机实测：2834 个 Img 文件，**2244 张可完整解码**（JPEG），其中 **633 张 ≥1000×1000**（最大 4000×1856），590 张 JPEG 流损坏跳过。
  - **视频缓存**：`cache\<年-月>\Sns\Video\<2hex>\<30hex>.mp4` 是**真实可播放的 mp4**，
    按 `年/月` 统一导出到 `朋友圈/_媒体缓存/_视频/<年>/<月>/`（本机实测 3 个）。
- 评论表待 schema dump 后确认。
- 自动驱动微信朋友圈面板（UIAutomation）用于「让微信把更多动态落库」，属于可选增强，不在 M1 核心。

## 4. 架构

```
Vue 前端 (127.0.0.1)
  ├─ 环境探测结果 / 好友多选 / 选项
  ├─ 任务进度 WebSocket
  └─ 归档浏览入口
        │ HTTP / WS
Go 后端
  ├─ env 模块：检测微信版本、路径、进程、登录态
  ├─ keygen 模块：扫内存拿 DB key（封装 PyWxDump 同款逻辑）
  ├─ db 模块：解密 Sns.db / MicroMsg.db / MSG*.db / MediaMSG*.db
  ├─ parser 模块：
  │    ├─ 聊天消息解析（XML / 各种消息类型：文本/图片/视频/文件/链接）
  │    └─ 朋友圈 protobuf 解析（TimeLineObject）
  ├─ media 模块：按 db 索引从 FileStorage 归集媒体，MD5 去重
  ├─ driver 模块：调用 Python 脚本自动浏览朋友圈
  ├─ archive 模块：生成 index.html / index.md / jsonl
  ├─ store：自己的 SQLite（好友、任务、增量锚点、错误清单）
  └─ queue：好友串行队列
        │ subprocess
Python driver (pywinauto)
  └─ browse_moments.py <wxid>
```

## 5. 用户操作流程（压到最少）

```
打开工具
  → ① 列出本机所有微信账号（昵称 + wxid + "最近使用"标记），点选一个
  → ② 顶部横幅显示「当前账号：昵称（wxid）」，下面列出该账号的好友
       每个好友清晰显示：昵称 / 备注 / wxid / alias / 聊天与朋友圈统计
       支持搜索、全选、全不选，勾人 + 勾要备份什么
  → ③ 点"开始备份"
  → 进度条："正在备份 张三 · 聊天记录 1234 条 · 朋友圈 45 条"
  → 完成，一键打开归档目录
```

**前端账号/好友选择交互（已实现）**：
- 步骤①「选择账号」：调 `GET /api/accounts`，返回本机所有可用账号
  （`{wxid, nick, data_root, recent, ok}`），卡片式展示，昵称大字、wxid 小字，
  最近使用的账号带「最近使用」徽标。点选后调 `POST /api/init {data_root}` 初始化该账号。
- 步骤②「选择好友」：顶部绿色横幅显示当前账号（昵称 + wxid），
  下方好友列表每项显示 `昵称`（备注不同时加「备注」标签）、`wxid · alias`、
  `💬 文字/图片/视频 | 📱 朋友圈 N 条 / M 张图`。支持按昵称/备注/wxid 搜索。
- 后端 `GET /api/contacts` 同时返回 `account` 与 `account_nick`，供前端显示当前账号。

失败处理：
- 微信没开 / 没登录 → 前端红字提示，不让继续。
- 没找到可用账号（无 keys.txt）→ 步骤①红字提示，不让继续。
- 某个好友的朋友圈被屏蔽/三天可见 → 自动跳过，记入失败清单，不阻塞其他人。
- 某张图匹配不上原图 → 继续跑，最后在报告里列"低分辨率 N 张"。

**命令行参数（wxff.exe）**：

| 参数 | 默认 | 说明 |
|---|---|---|
| `-data-root` | 自动探测 | 微信数据根目录（形如 `...\xwechat_files\<wxid>_<hash>`） |
| `-out` | `data/backup` | 归档输出目录 |
| `-sns-list` | — | 只列出朋友圈发布者（昵称 + wxid + 动态数），不导出 |
| `-sns-user` | — | 只导出指定发布者的朋友圈（wxid，可逗号分隔多个） |
| `-sns-min` | `720` | 朋友圈图片最小长边（像素）。长边小于此值的缩略图归入 `_媒体缓存/_缩略图/`；`0` = 不过滤（全部当原图） |
| `-moments` | — | 导出全部发布者的朋友圈 |
| `-start` / `-end` | — | 时间范围过滤（Unix 秒） |
| `-types` / `-sns-types` | — | 只导出指定消息类型 / 朋友圈类型 |
| `-imgkey` / `-xorkey` | 自动提取 | 手动指定图片 AES key / XOR key（微信未运行时用，如 `-imgkey b30e5d7754486daf -xorkey 230`） |
| `-type` | `text,image,file,voice,video` | 导出内容类型（逗号分隔）。含 `voice` 时导出语音并自动转 mp3 |

**语音转码依赖**（缺失则自动跳过，保留 `.silk`）：
- `pip install pilk`（纯 Python SILK v3 解码器）
- `ffmpeg`（PCM → mp3；本机 `D:\down\ffmpeg\bin\ffmpeg.exe`）
- 转码脚本：[`tools/silk2mp3.py`](tools/silk2mp3.py:1)

## 6. 自己拍板的默认决策（不再问）

| 项 | 决定 |
|---|---|
| 微信版本支持 | 3.x 和 4.x 都做，运行时自动探测，先适配你本机跑的那个 |
| 浏览驱动 | Python + pywinauto（V1），后续看稳定性再决定是否 Go 重写 |
| 拿不到原图 | 存缩略图，`low_res=true` 标注，不阻塞 |
| 归档格式 | **同时出 HTML 和 Markdown**：HTML 用来翻着看（图、视频、文件都能点），Markdown 用来纯文本检索和迁移 |
| 语音消息 | **已导出并转码**：`media_0.db` 的 `VoiceInfo` 表（SILK v3）按好友/月份落盘，再经 pilk+ffmpeg 转成可播放 `.mp3`（见 9.8） |
| 视频动态 | **已导出**：朋友圈视频（本地缓存 mp4）按 年/月 落盘；聊天视频本地无原文件，只记 CDN 地址 + 缩略图（见 9.8） |
| 增量 | 记每个 talker / 每个好友的最新游标，下次只补新的 |

## 7. 里程碑

| 阶段 | 目标 | 验收 |
|---|---|---|
| **M1** | 环境探测 + 拿 key + 解密一个 db + 把某个好友的聊天文字消息打印出来 | 控制台看到真实聊天文本 |
| **M2** | 聊天记录完整：图片/视频/文件正确归集到归档目录，生成 index.html | 浏览器打开能翻完整对话，图能看文件能下 |
| **M3** | 朋友圈链路：driver 自动进好友朋友圈页滚动，SnsInfo protobuf 解析成功，文字+图归档 | 单人朋友圈全自动跑完 |
| **M4** | Go 后端 + Vue 前端成型：多选、队列、进度、增量、失败清单 | 日常可用 |

## 8. 风险与边界（写在 README 里，不藏）

- 工具读取的是**你自己账号在本机落库的数据**，不越权访问别人的。
- 微信每次大版本升级都可能让密钥提取失效；失效时前端明确提示"当前微信版本暂不支持，请回退到 x.x 或等待更新"。
- 朋友圈有物理边界：**没浏览过的动态库里就没有**。工具会自动帮你浏览，但浏览本身有速度上限（每条约 1–2 秒），几百人的朋友圈全量刷会非常慢。
- 备份期间微信窗口会被 driver 短暂接管（朋友圈阶段），聊天记录阶段不需要动微信窗口。

---

## 9. M1 实战技术笔记（实测，2026-10-09）

> 本机环境：微信 **4.1.9.57**，进程 Weixin.exe，数据根 `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\`。

### 9.1 Key 提取（已跑通，23/23 db 全中）
- **不要用 YARA rule、不要 hook、不要 x'...' 字面量扫描**——4.1.9.57 这三条全废。
- 唯一靠谱：**salt neighborhood 扫描**。流程：
  1. 遍历 `db_storage\**\*.db`，读每个文件前 16 字节作为 salt。
  2. 在 Weixin.exe 进程内存里搜这个 16 字节 salt 的字节序列。
  3. salt 命中后，在它前后 ±64KB 范围内逐 32 字节滑动，把每个候选块当 **post-KDF enc_key** 试 HMAC 验证。
- **不要在 salt neighborhood 里试 password 模式**（256k PBKDF2），会卡死。只试 enc_key 快路径（2 次 PBKDF2）。
- HMAC 验证参数（SQLCipher 4）：
  - `macSalt = salt XOR 0x3A`
  - `macKey = PBKDF2-HMAC-SHA512(encKey, macSalt, iter=2, dkLen=32)`
  - HMAC-SHA512 输入：`page1[16:4032] + LE32(pageNum=1)`
  - 对比 `page1[4032:4096]`
- 项目结构（模块化，加新 scanner 不动主流程）：
  ```
  internal/
  ├── dbkey/          # 纯 Go，Scanner 接口可插拔
  │   ├── scan_salt.go    # 主力
  │   └── scan_xlit.go    # 兜底
  ├── procwin/        # Windows 进程内存访问
  └── decryptedb/     # 纯 Go SQLCipher 解密（不依赖 CGO）
  ```

### 9.2 DB 解密参数（纯 Go AES-256-CBC）
- PageSize=4096, Reserve=80（IV 16 + HMAC 64）
- 每个 page 独立解密：IV = `page[4016:4032]`，密文 = `page[0:4016]`（所有 page 都一样，包括 page 1）
- **Page 1 特殊处理**：解密 `[16:4016]`，然后把 `[0:16]` 写成 `"SQLite format 3\x00"`（salt 是加密时替换明文头用的，解密后要恢复）。reserve 区清零。
- 之前踩过的坑：page 1 把 `[0:16]` 保留成 salt → SQLite 不认；整段解密 `[0:4016]` → 出来不是 SQLite 头。正确做法就是上面那样。

### 9.3 sns.db（朋友圈）实测
- 表：`SnsTimeLine`（1291 行，朋友圈主表）、`SnsTopItem_1`（1127）、`SnsMessage_tmp3`（23）。
- `SnsTimeLine` 列：`tid / user_name / content(TEXT) / pack_info_buf(TEXT)`。
- **content 不是 protobuf，是 XML**！4.x 直接存可读 XML，比 3.x 好解析。
- 关键 XML 标签：
  - `<username>` 发布人 wxid
  - `<createTime>` Unix 时间戳
  - `<contentDesc>` 文字
  - `<mediaList><media><thumb>` 缩略图 URL（结尾 `/150`）
  - `<media><url>` 原图 URL（结尾 `/0`）
  - `<location latitude longitude>` 位置
  - `<user_comment>` / `<like_user_list>` 评论点赞
- CDN 图片 URL 带 token 鉴权，裸下 400。**备份时优先用本地 FileStorage 缓存，不从 CDN 拉**。

### 9.4 contact.db（联系人）实测
- 表 `contact`（562 行好友），关键列：
  - `username` wxid
  - `nick_name` 昵称
  - `remark` 备注
  - `alias` 微信号
  - `big_head_url` / `small_head_url` 头像
- 群聊：`chat_room`（7 个群）、`chatroom_member`（151 个群成员）。

### 9.5 message_0.db（聊天记录）实测
- 按会话分表：`Msg_<md5(wxid)>` 一张表，4.x 没 Name2Id 映射（user_name 列空）。需要自己算 md5 匹配。
- 表列：`local_id / local_type / create_time / source / message_content / packed_info_data`
- **消息内容是 zstd 压缩**的（magic `0x28 0xB5 0x2F 0xFD`）。
  - `source` 字段：zstd 压缩的 `<msgsource>` XML（消息元数据：签名、pua 等）
  - `message_content` 字段：消息正文（也是 zstd）
- **⚠️ `local_type` 是 64 位，真实类型在低 16 位**（`local_type & 0xFFFF`）。
  高位是「子类型/标志位」，例如文件消息是 49，但实际存储为
  `25769803825` / `8589934592049` / `21474836529` / `141733920817` 等（低位都是 49）。
  若用 `local_type` 精确匹配，会漏掉大量带标志的消息（本机实测漏 469 条文件消息）。
  低 16 位分布（本机实测）：1=17416、3=1174、34=2166、42=1、43=141、47=1304、48=2、49=469、50=37、10000=98。
- 最大单表 13430 条消息。
- **wxid → 表名映射**：`Msg_<md5(wxid) 小写 hex>`。已验证：
  - `md5("wxid_m6qaw8adu0w522") = 6d29a82ef6f54aee29353cc42d185b4f` → `Msg_6d29a82e...` ✓
  - `md5("wxid_i49id3jztuzq12") = 56a7cd8c...` → `Msg_56a7cd8c...` ✓
- 文本消息（local_type=1）：`message_content` 字段就是明文，直接可读。
- 非文本消息：`message_content` 是 XML（如 `<msg><emoji md5="..." len="...">`），按 local_type 解析。
- `source` 字段是 zstd 压缩的 `<msgsource>` 元数据（签名/pua/eggSeed），不是正文。
- **local_type 实测映射**（取 `local_type & 0xFFFF` 后的低 16 位）：
  - `1` = 文本（message_content 直接是明文）
  - `3` = 图片（XML：`<img aeskey="..." cdnthumburl="..." encryver="1" hdlength="..." length="...">`）
  - `34` = 语音（XML：`<msg><voicemsg voiceformat="4" voicelength="..." .../></msg>`，音频本体在 media_0.db）
  - `43` = 视频（XML：`<videomsg aeskey=... cdnvideourl=... playlength=... />`，本地无原文件）
  - `47` = 表情（XML：`<msg><emoji md5="..." len="...">`）
  - `49` = 应用消息 appmsg（XML：`<appmsg><title>...</title><type>N</type>...<appattach>...</appattach></appmsg>`，**N=6 才是文件**，详见 9.9）
  - `50` = 通话
  - `10000` = 系统消息（撤回/打招呼，XML：`<sysmsg type="revokemsg">...</sysmsg>`）
- **非文本消息的 message_content 也可能是 zstd 压缩的**（magic `0x28 0xB5 0x2F 0xFD`），解压后才是 XML。文本消息不压缩。
- 图片 XML 关键字段：`aeskey`、`cdnthumburl`/`cdnbigimgurl`（CDN 地址）、`encryver`、`length`/`hdlength`。
  **注意：图片 XML 里没有 `md5` 字段**，本地图片文件与消息的关联键是
  `packed_info_data` 里的 32 位 hex（见 9.7 节），不是 XML 里的 aeskey/md5。

### 9.7 媒体文件本地存储（M2 已解）
- 路径：`<数据根>\msg\attach\{md5(wxid)}\{YYYY-MM}\Img\{hash}.dat`
  - `{hash}.dat` 主图（**实测多为 `wxgf` 私有容器**，标准解码器打不开）
  - `{hash}_t.dat` 缩略图（**实测多为可解码的标准 JPEG**）
  - `{hash}_h.dat` 高清图（**实测多为微信自身存储就已损坏的 PNG**，见本节末尾）
  - `{hash}_W.dat` 空占位文件（1311 个，无内容）
- 文件目录结构按 `{md5(wxid)}/{YYYY-MM}/Img/` 组织。
- **图片 .dat 有三种加密格式**：

| 格式 | Magic | 加密 | Key 来源 |
|---|---|---|---|
| 旧 XOR | 无 | 单字节 XOR | 自动检测 |
| V1 | `07 08 V1 08 07` | AES-ECB + XOR | 固定 key `cfcd208495d565ef` |
| V2 | `07 08 V2 08 07` | **AES-128-ECB + XOR** | **从磁盘 kvcomm 推导（或扫进程内存）** |

- 本机 .dat 文件头 `07 08 56 32 08 07` = `07 08 'V''2' 08 07`，是 **V2 格式**。
- V2 文件结构：
  ```
  [6B signature]  = 07 08 56 32 08 07
  [4B aes_size LE]   头部字段（实测不可靠，见下）
  [4B xor_size LE]   XOR 加密部分长度
  [1B padding]
  [AES-128-ECB 加密的数据]   ← 长度 = bodyLen - xor_size
  [XOR 加密的数据]           ← 末尾 xor_size 字节
  ```
  本机样本（`_h.dat`）：`aes_size=0x400=1024, xor_size=103416, bodyLen=104456`
  → AES 区域 = `104456 - 103416 = 1040` 字节。`_t.dat`/主图样本的 AES 区域同样是 1040 字节。

- **⚠️ 关键修正（2026-10-09 实测）**：头部 `aes_size` 字段**不是** AES 区域长度。
  真实 AES 区域 = `len(body) - xor_size`（body = data[15:]）。
  - 若按 `aes_size`（=1024）解密，只有前 1024 字节被解密，其余大段数据保持密文，
    导致 PNG/JPEG 校验失败、图片损坏。
  - 正确算法：AES-128-ECB 解密 `body[0 : len(body)-xor_size]`（长度向下对齐 16），
    末尾 `xor_size` 字节做单字节 XOR，中间无明文区。
  - 本机 `_h.dat` 样本：`bodyLen=104456, xor_size=103416` → AES 区域 = 1040 字节。
- **AES 必须逐块循环**：`cipher.Block.Decrypt` 一次只解密一个 16 字节块，
  必须 `for off := 0; off+16 <= n; off += 16` 循环，否则只有第一块被解密。
- **image key 是 16 个 ASCII 字符**（直接作为 AES-128 key，不是 hex decode）。
- **扫描方法**：遍历 Weixin.exe 内存所有可读区域，找连续 ASCII 字母数字串，每个 16 字节窗口都当候选，用 AES-128-ECB 解密 .dat offset 15 处 16 字节，看解密结果是不是 JPEG(`FF D8 FF`) 或 PNG(`89 50 4E 47`)。
- **image key 可以完全从磁盘推导，不需要扫内存**：
  1. kvcomm 目录：`C:\Users\<user>\AppData\Roaming\Tencent\xwechat\net\kvcomm\`（或 `radium\ilink\<hash>\kvcomm\`）
  2. 找 `key_<code>_*.statistic` 文件，code = 文件名开头的数字（如 `3483575526`）
  3. `image_key = md5(code + wxid)[:16]`（wxid 不带 `_<hash>` 后缀）
  4. 本机验证：`md5("3483575526" + "wxid_mpvlokr4ycnu22") = b30e5d7754486daf` ✓
- **xor_key 从 .dat 文件尾部算**：`tail[-2]^0xff == tail[-1]^0xd9`，本机 = `0xE6`。
- **不需要用户点大图，不需要扫内存**。微信重启后 code 可能变，但 kvcomm 目录里会有新的 key_*.statistic 文件，重新读即可。
- **图片与消息的匹配键（已解）**：图片消息（`local_type=3`）的 XML 里**没有** `md5` 字段，
  真正的 32 位 hex 标识在 `Msg_<md5(wxid)>` 表的 **`packed_info_data`** BLOB 列里
  （protobuf 编码，形如 `\b\x02\x10\x14\x1a"\" c6294347b57c653a31d6c24afc91524c`）。
  用正则 `[0-9a-f]{32}` 提取即可，与 `msg/attach/<md5(wxid)>/<YYYY-MM>/Img/<hash>.dat`
  的文件名 **100% 对应**（本机 347/347 命中）。
- **主图 `.dat` 解密后常是 `wxgf` 私有容器**（内含 HEVC/AVIF，标准解码器打不开）；
  `_t.dat` 缩略图通常是完整可解码的标准 JPEG。导出时对每个标识的所有变体
  （main/`_t`/`_h`/`_W`）解密并**完整 `image.Decode` 验证**，只导出能解码的那一个，
  避免写出无效图片。本机 347 个图片标识中 281 个有可解码图片，导出后 281/281 全部有效。
- **⚠️ 导出结果以缩略图为主（2026-10-09 实测）**：本机 2171 个 .dat 中，
  可解码 304 个（main=7、`_h`=22、`_t`=275），不可解码 556 个，`_W` 占位 1311 个。
  即 **212 个 `_h.dat` 高清图里只有 22 个能解码**，其余 190 个是**微信自身存储就已损坏**的
  PNG（IHDR/pHYs 的 CRC 正常，但 IDAT 的 zlib 压缩流本身损坏：解压后行 filter 字节
  从某一行起出现 >4 的非法值，且长度比期望多出数千字节，Adler-32 也不匹配）。
  这种损坏**无法修复**（不是简单的 chunk 长度字段错误），因此这些图片只能退回 `_t` 缩略图。
  最终导出 281 张中：≥1000×1000 仅 17 张、500-999 5 张、200-499 5 张、<200px 255 张，
  **绝大多数是缩略图**。要拿到原图，只能靠微信客户端重新下载（点开大图触发落盘）。
- **⚠️ 与第 2/6 节设计的差异（待补）**：设计里承诺"拿不到原图时在 json 里标 `low_res=true`"，
  但当前实现**尚未写入该字段** —— 导出时只是静默选用了能解码的最大变体（多为缩略图）。
  后续需要在导出逻辑里判断"选中的变体是否为 `_t`"，是则写 `low_res=true`。
- **文件类消息（PDF/txt/apk）不加密**，在 `msg\file\{YYYY-MM}\` 下，文件名就是原始名。

### 9.8 视频与语音（M2 已解，2026-10-09 实测）

**语音（type 34）—— 可完整导出**
- 消息表里 type 34 的 `message_content`（zstd 解压后）是
  `<msg><voicemsg voiceformat="4" voicelength="11554" length="18320" aeskey=... voiceurl=... fromusername=... /></msg>`，
  **只有元数据，没有音频本体**。
- 音频本体在 `db_storage\message\media_0.db`（113MB）的 **`VoiceInfo` 表**：
  `VoiceInfo(chat_name_id INTEGER, create_time INTEGER, local_id INTEGER, svr_id INTEGER, voice_data BLOB, data_index TEXT)`。
  - `chat_name_id` 是 `Name2Id` 表的 rowid，`Name2Id(rowid, user_name)` 映射到 wxid。
  - `voice_data` 以 `02 23 21 53 49 4C 4B 5F 56 33` 开头 = `\x02#!SILK_V3`，即 **SILK v3 音频**。
  - 本机实测 **5039 条**语音。
- **导出**：`聊天记录/<好友>/media/<年-月>/voices/<create_time>_<local_id>.silk`，
  同目录写 `voices.jsonl` 索引（时间/时长/文件名）。
- **转码为 MP3（已实现）**：SILK v3 普通播放器不认，导出后自动转码：
  - 链路：`pilk`（纯 Python SILK 解码器，`pip install pilk`）解出 PCM → `ffmpeg` 编码 mp3。
  - 脚本：`tools/silk2mp3.py`（递归处理归档目录，转码成功后删除 `.silk`，只留 `.mp3`）。
  - Go 侧：`archive.TranscodeVoices()` 在语音导出后调用；若缺 python/pilk/ffmpeg 则**自动跳过**（保留 `.silk`，不报错）。
  - 本机实测：**5039 条全部转码成功**（0 失败），mp3 可正常播放（如样本 37.4 秒 / 64kbps）。
  - 依赖：`pip install pilk` + ffmpeg（本机 `D:\down\ffmpeg\bin\ffmpeg.exe`）。

**聊天视频（type 43）—— 本地无原文件**
- 消息 XML：`<videomsg aeskey=... cdnvideourl="3057020100044b..." length="226513" playlength="2" cdnthumburl=... cdnthumbaeskey=... />`。
- **本地不落原视频**：`msg\video\<年-月>\` 24 个月份目录**全部为空**（视频未下载到本地）。
  全盘只有 6 个真实 mp4：3 个朋友圈缓存 + 1 个 `msg\file\` + 2 个 `msg\migrate\Video\`。
- **导出**：在 `messages.jsonl` 里记 `type=video`、`dur`（playlength 秒）、`cdn`（CDN 地址），
  缩略图随图片链路导出。要拿原视频只能靠微信客户端点开触发下载。

**朋友圈视频 —— 本地有真实 mp4**
- 路径：`cache\<年-月>\Sns\Video\<2hex>\<30hex>.mp4`（同目录还有 `.jpg` 封面）。
- 与朋友圈图片一样，文件名是**不透明 30-hex id**，无法与 XML 关联，因此**按 年/月 统一导出**到
  `朋友圈/_媒体缓存/_视频/<年>/<月>/`（本机实测 3 个）。

### 9.9 文件消息与好友筛选（2026-10-09 实测修复）

**文件消息判别（type 49 = appmsg）**
- type 49 是「应用消息」大类，包含**文件 / 链接 / 红包 / 小程序 / 引用 / 视频号**等。
  其 XML 形如 `<appmsg><title>...</title><type>N</type>...<appattach>...</appattach></appmsg>`。
- **不能用「是否含 `<appattach>`」判断是不是文件**：链接、引用、小程序也常带一个空的
  `appattach`。本机实测 364 条含 `appattach` 的消息里，只有 **41 条**是真正的文件。
- **正确判别字段是 `<appmsg>` 里的 `<type>` 子类型**：
  | type | 含义 | 本机条数 |
  |---|---|---|
  | **6** | **文件**（唯一需要拷贝本地文件的类型） | 41 |
  | 5 | 链接 | 84 |
  | 57 | 引用（回复某条消息） | 206 |
  | 33 / 36 | 小程序 | 8 / 5 |
  | 8 | 表情 | 6 |
  | 19 | 聊天记录 | 3 |
  | 51 | 视频号 | 7 |
  | 62 / 1 | 其他 | 2 / 2 |
- **导出**：仅 `type=6` 记为 `type=file`；其余记为 `type=link`（只存标题）。
  文件本体在 `msg\file\<年-月>\`（**不加密**，文件名即原始名），用 XML 的 `<title>`
  去 `msg\file` + `msg\migrate\File` 索引里匹配并拷贝到
  `聊天记录/<好友>/media/<年-月>/files/<原始名>`。
  本机实测：35 条文件记录，其中 24 条有本地副本（其余 11 条未下载到本地，只记名字）。
- 辅助函数：`parseAppMsgType(xml)`（`internal/archive/archive.go`）。

**好友筛选（`-all`）**
- **⚠️ 不能按 `contact.local_type` 过滤**。本机 `contact.local_type` 分布为
  `0=56 / 1=113 / 2=7 / 3=379 / 5=2 / 6=5`，而**真正有聊天记录**的好友
  （存在 `Msg_<md5(username)>` 表）大多是 `local_type=1`。
  早期用 `WHERE local_type=0` 只选到 56 个「没有聊天表」的联系人，导出结果几乎为空。
- **正确做法**：直接以「`message_0.db` 里存在对应 `Msg_` 表」为准，
  与 `ExportContact` 的表存在性检查完全一致。本机实测从 23 个好友修正为 **27 个**。
- 表名映射：`Msg_<md5(username) 小写 hex>`（本机 31 张表全部命中，见 9.5）。

### 9.6 已验证做不通的
- wechat-dump-rs：对 4.1.9.57 自动扫不到 key，手动给 enc_key 也报 "Hash verification failed"（它的 raw key 格式和我们的 enc_key 不兼容，不用浪费时间试）。
- PyWxDump / wxutil DbkeyHookCMD：不支持 4.x。

---

文档到此锁定。下一步：开 M1，先探测你本机微信环境（版本号、数据路径、能否拿到 key），再打通"解密 db → 列出某好友聊天文字"这条最短链路。
