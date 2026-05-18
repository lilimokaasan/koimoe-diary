# Sakurairo WordPress 功能分析

分析日期：2026-05-18

分析对象：

- 本地运行站点：http://localhost:8881
- 后台入口：http://localhost:8881/wp-admin
- 主题源码：`D:\codex\Sakurairo-1.20.10`

本文只记录功能点、数据设计、交互流程和 Go 重写建议，不讨论审美或视觉还原。

## 1. 总体结构

旧项目本质上是一个 WordPress 主题，但主题承担了不少插件级职责：

- 注册 WordPress 主题能力：特色图、导航菜单、HTML5 表单、文章格式、自定义背景、友情链接管理。
- 扩展内容类型：新增 `shuoshuo` 自定义文章类型，用来发布短动态/说说。
- 提供主题选项后台：基于 Options Framework，把大量站点、首页、文章、社交、前台控件、后台、功能、增强、字体和主题参数存入 WordPress options。
- 扩展前台页面模板：归档页、友情链接页、说说页、前台登录页、前台注册页、B 站追番页等。
- 扩展评论系统：Ajax 评论、Markdown 评论、表情、图片上传、QQ 头像、私密评论、邮件通知、评论者等级、UA/地理位置展示。
- 提供自定义 REST API：随机图、实时搜索缓存、评论图片上传、QQ 信息、Bilibili 追番、音乐播放器数据。
- 修改后台体验：登录页、仪表盘、后台菜单、后台配色、评论列表列、编辑器按钮等。

Go 版迁移时不建议把这些能力都放进模板层。更合适的边界是：

- 内容模型和查询：`internal/store`
- 公开页面 handlers：`internal/handlers` 或现有路由层
- 主题/站点配置：数据库表 `site_settings` 或拆分后的配置表
- 评论能力：独立 comment service
- 媒体与随机图：media service
- 外部集成：单独 integration package
- 后台管理：admin handlers + templates

## 2. 内容与路由能力

### 2.1 标准内容

旧主题依赖 WordPress 原生内容：

- 文章 `post`
- 页面 `page`
- 分类 `category`
- 标签 `post_tag`
- 评论 `comment`
- 用户 `user`
- 友情链接 `link` / `link_category`
- 媒体附件 `attachment`

现有 Go 版已覆盖文章、分类、标签、评论和基础后台。后续如果要承接旧主题功能，还需要考虑页面、友情链接、媒体库和用户资料的独立模型。

### 2.2 自定义内容类型：说说

源码位置：`functions.php` 中 `shuoshuo_custom_init()`。

功能：

- 注册 `shuoshuo` post type。
- 支持 title、editor、author。
- 有独立列表页模板 `page-word.php`。
- 查询所有已发布说说并按时间线输出。

Go 版建议：

- 新增 `notes` 或 `moments` 表，而不是塞进 `posts`。
- 字段建议：`id`, `content`, `author_id`, `status`, `created_at`, `updated_at`。
- 后台提供简单发布/编辑/删除。
- 前台提供 `/moments` 页面。

### 2.3 页面模板

旧主题包含的功能模板：

- `page-archive.php`：按年月归档文章，显示文章标题、日期、评论数。
- `page-links.php`：友情链接页，读取 WordPress link manager 和 link categories。
- `page-word.php`：说说时间线。
- `user/page-login.php`：前台登录页。
- `user/page-register.php`：前台注册页，包含用户名、邮箱、密码、确认密码、滑块验证。
- `user/page-bangumi.php`：Bilibili 追番页，读取主题设置中的 Bilibili UID/cookie。
- `user/page-timeline.php`：另一套时间轴模板。

Go 版建议：

- 归档页可以作为核心能力优先实现。
- 友情链接和说说属于个人站常用功能，建议中优先级实现。
- 前台注册/多用户能力可以暂缓，因为当前产品假设是单用户维护。
- Bilibili 追番和外部集成可做成可选模块。

## 3. 主题设置与配置

旧主题的设置集中在 `options.php`，按 heading 分组。功能相关设置大致如下。

### 3.1 基础设置

- 站点标题：`site_name`
- 作者名：`author_name`
- 头像：`focus_logo`
- 文本 LOGO：`focus_logo_text`
- LOGO：`akina_logo`
- favicon：`favicon_link`
- 自定义 SEO keywords/description：`akina_meta`, `akina_meta_keywords`, `akina_meta_description`
- 导航展开：`shownav`
- 搜索按钮开关：`top_search`
- 首页文章列表模式：`post_list_style`
- 分页模式：`pagenav_style`
- 自动加载下一页：`auto_load_post`
- footer HTML：`footer_info`

Go 版建议：

- 已有 `site_settings` 可以继续承接其中的站点标题、作者、头像、footer、导航、主题色。
- SEO metadata、分页模式、自动加载下一页应作为单独设置扩展。

### 3.2 首页功能设置

- 首页主开关：`main-switch`
- 信息栏/社交卡片开关：`infor-bar`, `social-card`
- 首页随机背景图：`background-rgs`, `cover_cdn_options`, `cover_cdn`, `cover_cdn_mobile`, `cover_beta`
- 首页视频背景：`focus_amv`, `focus_mvlive`, `amv_url`, `amv_title`
- 公告栏：`head_notice`, `notice_title`
- 首页文章特色图来源：`post_cover_options`, `post_cover`
- 首页文章细节图标开关：`hpage-art-dis`
- Focus Area：`focus-area`, `focus-area-style`, `focus-area-title`, `feature1_*`, `feature2_*`, `feature3_*`

Go 版建议：

- 公告栏、默认封面图、随机图源、Focus Area 适合进入后台“主题设置”。
- 视频背景属于可选高级项。
- 首页文章列表模式和分页模式会影响 handlers 查询与模板结构，应在实现前确定是否保留可配置性。

### 3.3 文章页设置

- 文章内容样式选择：`entry_content_theme`
- 点赞开关：`post_like`
- 分享开关：`post_share`
- 上一篇/下一篇：`post_nepre`
- 作者信息：`author_profile`, `show_authorprofile`
- 评论收缩：`toggle-menu`
- 评论 textarea 装饰图：`comment-image`
- 版权声明：`post-lincenses`
- 打赏二维码：`alipay_code`, `wechat_code`

Go 版建议：

- 点赞、上一篇/下一篇、作者卡片、版权声明可以作为核心阅读体验扩展。
- 分享和打赏适合作为可选配置。
- 评论收缩属于前端交互，不需要影响后端数据。

### 3.4 社交设置

旧主题提供多个社交链接字段：

- WeChat, Weibo, QQ, Telegram, Qzone, GitHub, Lofter, Bilibili, Youku, Netease Cloud Music, Twitter, Facebook, Jianshu, CSDN, Zhihu, Email。

Go 版建议：

- 不要为每个平台固定建列。
- 使用 `social_links` JSON 或独立表：`platform`, `label`, `url`, `icon`, `sort_order`, `enabled`。

### 3.5 功能设置

重要开关：

- Bilibili UID/cookie：`bilibili_id`, `bilibili_cookie`
- 评论 UA 信息：`open_useragent`
- 评论地理位置：`open_location`
- 评论图片上传 API：`img_upload_api`
- Imgur/SM.MS/Chevereto 凭据与代理：`imgur_client_id`, `smms_client_id`, `chevereto_api_key`, `cheverto_url`, `cmt_image_proxy`, `imgur_upload_image_proxy`
- 访问统计来源：`statistics_api`
- 统计展示格式：`statistics_format`
- 私密评论：`open_private_message`
- QQ 头像加密代理：`qq_avatar_link`
- 实时搜索：`live_search`, `live_search_comment`
- 友情链接布局：`friend_center`
- 文章 lazyload：`lazyload`, `lazyload_spinner`
- 复制版权：`clipboard_copyright`
- 评论回复邮件：`mail_notify`, `admin_notify`, `mail_img`, `mail_user_name`

Go 版建议：

- 评论图片上传和外部图床应抽象成 provider，不要把 provider 逻辑写死在 handler。
- 实时搜索建议先实现本地数据库搜索，后续再做缓存 JSON 或全文索引。
- 评论 UA/地理位置属于隐私敏感增强项，应默认关闭。

## 4. 前台功能

### 4.1 首页

功能点：

- 显示首页首屏信息、头像/文本 LOGO、社交链接、公告。
- 支持随机背景图和背景切换。
- 支持视频背景。
- 支持 Focus Area 三个推荐卡片。
- 文章列表支持标准/图文两种模式。
- 分页支持普通上一页/下一页和 Ajax 加载。
- 可按分类 ID 排除首页文章：`classify_display`。

Go 版核心迁移：

- 首页查询发布文章，支持分页。
- 支持公告、社交链接、默认封面和推荐区。
- 支持按分类排除。

Go 版可选迁移：

- Ajax 无限加载。
- 视频背景。
- 多套背景切换。

### 4.2 文章详情

功能点：

- 阅读量统计：访问单篇时递增 `views` post meta。
- 阅读量可来自内置 post meta 或 WP-Statistics 插件。
- 点赞：Ajax action `specs_zan`，写入 `specs_zan` post meta，并用 cookie 防止重复显示。
- 分享区：`layouts/sharelike.php`。
- 打赏二维码：支付宝/微信二维码 URL。
- 版权声明。
- 标签输出。
- 上一篇/下一篇导航，缩略图来源顺序：特色图 -> 文章首图 -> 随机图。
- 作者信息区域。
- 内容处理：图片 lazyload、fancybox 语法、TOC 标记、首字样式、表情替换、外链 nofollow/target。

Go 版建议字段：

- `posts.views_count`
- `posts.likes_count`
- `posts.cover_url`
- `posts.license_type` 或站点级默认版权
- `posts.allow_comments`
- `posts.status`

Go 版建议接口：

- `POST /api/posts/{id}/like`
- 详情页 handler 中递增浏览量，或用异步 endpoint 减少刷新重复计数。
- 上下篇查询按发布时间或 ID。

### 4.3 搜索

旧主题搜索有两套能力：

- WordPress 普通搜索页。
- 实时搜索缓存 API：`GET /wp-json/sakura/v1/cache_search/json`

实时搜索 JSON 包含：

- posts：标题、链接、评论数、纯文本内容
- pages：标题、链接、评论数、纯文本内容
- tags：标签名和链接
- categories：分类名和链接
- comments：可选，私密评论仅返回占位文案

Go 版建议：

- 第一阶段使用 SQL `LIKE` 或已有搜索页。
- 第二阶段增加 `/api/search-index`，返回轻量 JSON，前端本地过滤。
- 数据源应包括 posts、categories、tags；comments 默认不进入公开搜索。

### 4.4 归档

旧主题提供两类归档：

- WordPress archive/category/tag/search 模板。
- 自定义归档页 `page-archive.php` 和函数 `memory_archives_list()`，按年月聚合全部文章。

Go 版建议：

- 增加 `/archives` 页面，查询所有已发布文章并按 `YEAR(created_at), MONTH(created_at)` 分组。
- 每篇显示标题、日期、浏览量、评论数。

### 4.5 友情链接

旧主题使用 WordPress Link Manager：

- `get_link_items()` 读取 link categories。
- 每个 link 有 name、url、description、image。
- 分类可有 description。

Go 版建议新增表：

```sql
friend_link_categories(id, name, description, sort_order)
friend_links(id, category_id, name, url, description, image_url, sort_order, visible, created_at, updated_at)
```

后台提供增删改查，前台提供 `/links`。

### 4.6 用户入口

旧主题前台 header 有用户菜单：

- 未登录显示登录入口。
- 已登录显示头像和用户菜单。
- 管理员显示 Dashboard/New post/Profile/Sign out。
- 普通用户显示 Profile/Sign out。

Go 版当前单用户场景：

- 保留 admin 登录入口即可。
- 当前右上角三横杠菜单不应是空交互；在导航菜单中至少提供指向 `/admin/login` 的登录入口。若管理员已登录，访问登录入口可沿用后台现有逻辑跳转到 `/admin`。
- 多用户资料、前台注册、普通用户菜单可以暂缓。

右上角三横杠菜单建议承载“轻量站内导航 + 管理入口”，不要放太多后台细项。基于当前 Go 版已有路由，建议默认菜单项如下：

| 菜单项 | 链接 | 当前状态 | 说明 |
| --- | --- | --- | --- |
| Home | `/` | 已有 | 返回首页，适合作为第一个固定项。 |
| Archives | `/archives` | 已有 | 按时间浏览全部文章，比 `/archive` 更适合放在菜单中。 |
| Links | `/links` | 已有 | 友情链接页，当前代码已有前台与后台管理能力。 |
| Search | `/search` | 已有 | 搜索页；即使已有搜索图标，菜单中保留文字入口对移动端更清楚。 |
| Admin Login | `/admin/login` | 已有 | 未登录进入登录页，已登录可跳转后台。 |

不建议现在放入菜单的项：

- `/admin/posts/new`、`/admin/comments`、`/admin/settings`、`/admin/links`：这些属于后台二级操作，登录后在后台导航里处理即可。
- `/category/{slug}` 和 `/tag/{slug}`：分类/标签数量会变化，更适合放在侧栏、文章元信息或后续“分类/标签索引页”中。
- `/feed` 或 `/feed.xml`：适合放 footer 或 `<link rel="alternate">`，不必占主菜单位置。
- `/api/*`：纯接口，不作为用户菜单项。

## 5. 评论系统

旧主题评论是功能最重的部分。

### 5.1 评论提交

功能点：

- 使用 WordPress comment form。
- 支持 Ajax 提交：`admin-ajax.php?action=ajax_comment`。
- 支持嵌套回复。
- 评论成功后返回单条评论 HTML。
- 可选“我不是机器人”复选框：`norobot`。
- 可选私密评论：`is-private`。
- 可选邮件通知：`mail-notify`。
- 字段：author、email、url、hidden QQ、comment。

Go 版建议：

- 当前已有评论提交，可以扩展字段：`parent_id`, `website`, `qq`, `is_private`, `mail_notify`, `user_agent`, `ip`, `status`。
- Ajax 与普通 form 可以共用同一个 POST handler，按 `Accept` 返回 HTML fragment 或 redirect。

### 5.2 Markdown 评论

源码逻辑：

- `preprocess_comment` 中解析 Markdown。
- 禁止非 Markdown HTML 标签。
- 动态给 `wp_comments` 增加 `comment_markdown` 字段。
- 使用 Parsedown 渲染 HTML。

Go 版建议：

- 表结构直接包含 `content_markdown` 和 `content_html`。
- Markdown 渲染使用 Go 库，例如 goldmark。
- HTML sanitize 必须在服务端完成。

### 5.3 私密评论

功能点：

- 提交时如果有 `is-private`，写入 comment meta `_private=true`。
- 管理员可以通过 Ajax action `siren_private` 把评论标为私密。
- 展示时，只有评论作者、父评论作者或管理员能看到内容，否则显示“私密评论”占位。

Go 版建议：

- `comments.is_private BOOLEAN`
- 判断可见性时基于：
  - 管理员 session
  - 当前评论者 cookie/email token
  - 父评论作者 email token
- 单用户个人站初期可以只做“管理员可见，访客不可见”的简化版。

### 5.4 评论图片

功能点：

- 评论支持 BBCode：`[img]url[/img]`。
- 支持 `{UPLOAD}` 替换为图床前缀。
- REST API 上传图片：`POST /wp-json/sakura/v1/image/upload`
- Provider：Imgur、SM.MS、Chevereto。
- 返回标准 JSON：`status`, `success`, `message`, `link`, `proxy`。

Go 版建议：

- 第一阶段可只允许 URL 插图，不开放匿名上传。
- 后续增加 `media_uploads` 表和本地上传，图床 provider 作为可选。

### 5.5 QQ 头像与资料

功能点：

- 评论表单可输入 QQ 号。
- QQ 号写入 comment meta `new_field_qq`。
- 如果有 QQ，头像优先走 QQ 头像。
- REST API:
  - `/sakura/v1/qqinfo/json`
  - `/sakura/v1/qqinfo/avatar`
- 头像链接可明文、加密代理或接口获取。

Go 版建议：

- 评论表保留 `qq` 可选字段。
- QQ 头像可以不作为首期功能；如果做，封装为 avatar resolver，不影响评论核心。

### 5.6 评论增强信息

功能点：

- 评论者等级：按同 email 历史评论数量分级。
- UA 展示：解析浏览器和操作系统。
- 地理位置：调用淘宝 IP API。
- 评论回复自动 @ 父评论作者。
- 外链自动 `target="_blank"` 与 `rel="nofollow"`。
- 表情面板：Bilibili、Tieba、颜文字，正文和评论均会替换标记。

Go 版建议：

- 评论等级可用 SQL count 动态计算或存缓存。
- UA/IP 地理位置默认关闭。
- 表情替换可以后置，先保留纯文本/Markdown。

### 5.7 邮件通知

功能点：

- 回复评论后给父评论作者发 HTML 邮件。
- 可配置邮件发件名前缀、邮件头图、是否通知管理员、用户是否订阅通知。
- 注册邮件会附带注册 IP。
- 密码重置邮件会修正尖括号导致的问题。

Go 版建议：

- 引入 `mailer` 接口，先支持 SMTP。
- 评论回复通知作为异步任务更稳。
- 用户可选 `mail_notify` 字段应写入评论表。

## 6. REST API 与 Ajax 接口

旧主题 REST namespace：`sakura/v1`。

| 方法 | 路径 | 功能 | 迁移建议 |
| --- | --- | --- | --- |
| POST | `/image/upload` | 评论图片上传到第三方图床 | 可选，后置 |
| GET | `/cache_search/json` | 返回实时搜索索引 JSON | 中优先级 |
| GET | `/image/cover` | 302 到随机首页图 | 中优先级 |
| GET | `/image/feature` | 302 到随机文章特色图 | 中优先级 |
| GET | `/database/update` | 更新随机图 manifest 缓存 | 可选 |
| GET | `/qqinfo/json` | 获取 QQ 信息 | 可选 |
| GET | `/qqinfo/avatar` | 代理/跳转 QQ 头像 | 可选 |
| POST | `/bangumi/bilibili` | 加载 Bilibili 追番分页 | 可选 |
| GET | `/meting/aplayer` | 音乐播放器数据/歌词/音频跳转 | 可选 |

旧主题 Ajax action：

- `ajax_comment`：提交评论并返回评论 HTML。
- `specs_zan`：文章点赞。
- `siren_private`：管理员将评论设为私密。

Go 版建议 API：

```text
GET  /api/search-index
GET  /api/random-cover
GET  /api/random-feature
GET  /feed
GET  /feed.xml
POST /api/comments
POST /api/posts/{id}/like
POST /admin/comments/{id}/private
```

Feed implementation note:

- `/feed` and `/feed.xml` are implemented as Atom feeds for the latest published posts.
- Public templates advertise the feed with `<link rel="alternate" type="application/atom+xml">`.
- The feed is a lightweight WordPress compatibility/convenience layer and should not require extra database tables.

Random image implementation note:

- `/api/random-cover` and `/api/random-feature` are implemented as lightweight 302 image redirects, with `?format=json` available for debugging and later frontend integrations.
- Random cover images prefer the configured hero/avatar, theme defaults, and local `web/static/curated-sakura-images` assets.
- Random feature images prefer published post cover images, then fall back to curated square images and the cover pool.

## 7. 随机图与缓存设计

旧主题会创建自定义表 `wp_sakurairo`：

```text
mate_key varchar(50) primary key
mate_value text
```

默认写入：

- `manifest_json`
- `mobile_manifest_json`，仅 `cover_beta` 开启时
- `json_time`
- `privkey`

随机图来源：

- `type_1`：读取 manifest JSON，按浏览器是否支持 WebP 返回 webp/jpeg。
- `type_2`：读取主题目录 `manifest/gallary/*.{gif,jpg,png}`。
- `type_3`：直接使用外部随机图 API。

Go 版建议：

- 不复刻 key-value 表名错误 `mate_key`，使用明确表：

```sql
random_image_sources(id, name, kind, url, enabled, device, sort_order)
random_images(id, source_id, url, webp_url, jpeg_url, device, created_at)
```

- 简化版也可以先在 `site_settings` 中保存：
  - `cover_source_type`
  - `cover_source_url`
  - `mobile_cover_source_url`
  - `default_post_cover_url`

## 8. 后台功能

旧主题后台增强：

- Options Framework 主题设置页。
- 登录页背景、logo、验证滑块。
- 前台注册开关。
- 非管理员隐藏 Dashboard 并跳转 Profile。
- 移除/简化 Dashboard widgets。
- 后台评论列表增加 QQ 列。
- TinyMCE 增加第三排按钮。
- Quicktags 增加 `[download]` 按钮。
- 自定义后台配色方案。
- 后台通知可 dismiss。
- 上传目录可用 CDN base URL 替换。
- 分类/标签支持图片字段，来源 `inc/categories-images.php`。

Go 版建议：

- 已有 admin 可以逐步增加：
  - 主题设置
  - 导航管理
  - 评论管理
  - 分类/标签管理
  - 友情链接管理
  - 媒体库
- 后台主题化/登录页属于体验层，已部分实现。
- 分类图片若要支持分类页头图，应在 `categories` 表增加 `cover_url`。

## 9. 短代码与内容语法

旧主题支持：

- `[download]url[/download]`
- `[show_ip]`
- `[task]...[/task]`
- `[warning]...[/warning]`
- `[noway]...[/noway]`
- `[buy]...[/buy]`
- `[collapse title="..."]...[/collapse]`
- `[toc]`
- `[begin]...[/begin]`
- 图片语法：`!{alt}(url)` 和 `!{alt}(url)[thumb_url]`
- 评论图片：`[img]url[/img]`
- Bilibili 表情：`{{name}}`
- Tieba 表情：`::name::`

Go 版建议：

- Markdown 处理层支持最小集合：`[toc]`, `[begin]`, `[collapse]`。
- 旧内容迁移时需要一个 compatibility renderer，把旧短代码转换成 HTML。
- 不建议新编辑器继续鼓励大量私有短代码；应提供更明确的 Markdown/HTML 插入工具。

## 10. 外部依赖

旧主题依赖或可选依赖：

- WordPress Options Framework。
- WordPress Link Manager。
- WP-Statistics 插件。
- Jetpack infinite scroll/responsive videos。
- Gravatar。
- QQ 头像和 QQ 信息接口。
- 淘宝 IP 信息接口。
- Imgur、SM.MS、Chevereto。
- Bilibili。
- Meting/APlayer/网易云音乐等音乐数据。
- jsDelivr/CDN 静态资源。
- Google Analytics、CNZZ。

Go 版建议：

- 外部依赖都应配置化，并有关闭开关。
- 首期以本地能力为主，减少第三方 API 失败对站点核心阅读的影响。

## 11. 数据模型迁移建议

现有 Go 版已有：

- `posts`
- `categories`
- `tags`
- `post_tags`
- `comments`
- `site_settings`

建议新增或扩展：

```sql
-- 页面
pages(id, title, slug, content, status, created_at, updated_at)

-- 说说/短动态
moments(id, content, author_name, status, created_at, updated_at)

-- 友情链接
friend_link_categories(id, name, description, sort_order)
friend_links(id, category_id, name, url, description, image_url, sort_order, visible, created_at, updated_at)

-- 媒体库
media_assets(id, filename, original_name, mime_type, size, width, height, url, storage, created_at)

-- 评论扩展字段，可直接加到 comments
comments.parent_id
comments.website
comments.qq
comments.is_private
comments.mail_notify
comments.user_agent
comments.ip
comments.location
comments.content_markdown
comments.content_html
comments.status

-- 文章扩展字段，可直接加到 posts
posts.views_count
posts.likes_count
posts.cover_url
posts.allow_comments
posts.license_type
```

配置建议：

- 简单键值继续放 `site_settings`。
- 复杂列表用 JSON 或独立表：
  - navigation：已实现 JSON
  - social_links：建议 JSON 或独立表
  - focus_cards：建议 JSON
  - random image sources：长期建议独立表

## 12. 迁移优先级

### P0：已经基本具备或应保持

- 文章列表、文章详情、分类、标签、搜索、归档。
- 评论提交和后台评论管理。
- 站点基础设置、导航设置。
- 文章封面、默认封面。
- 后台登录和文章编辑。

### P1：建议下一阶段迁移

- 归档页按年月聚合。
- 友情链接页和后台管理。
- 说说/短动态。
- 文章浏览量与点赞。
- 上一篇/下一篇。
- 作者卡片、版权声明、打赏配置。
- 实时搜索 JSON。
- 分类/标签管理与分类封面。
- 评论嵌套回复、私密评论、邮件通知。

### P2：体验增强，可逐步迁移

- 评论 Markdown 与表情兼容。
- 评论图片上传。
- QQ 头像。
- 首页 Focus Area。
- 首页随机图管理。
- 前台登录页。
- 复制版权、TOC、旧短代码兼容。

### P3：可选或暂缓

- 多用户注册。
- Bilibili 追番页。
- APlayer/Meting 音乐接口。
- WordPress 插件式统计兼容。
- Jetpack infinite scroll 兼容。
- 外部 CDN/图床 provider 全量复刻。

## 13. Go 版设计原则

- 不照搬 WordPress 的 hook/meta 结构；用明确表结构和 service 层承接。
- 主题设置可以保留灵活性，但核心数据不要全部塞进 key-value。
- 评论系统先保证稳定、可审核、可回复，再做图片/表情/QQ/地理位置。
- 外部 API 失败不能影响文章阅读、评论展示、后台登录。
- 旧短代码要考虑迁移兼容，但新编辑体验应尽量使用清晰 Markdown。
- 单用户个人站优先：多用户注册、权限矩阵和复杂社交登录可以后置。
