# Sakurairo Go

一个以 WordPress 主题 Sakurairo 为蓝本的轻量 GoFrame 版本。目标是在保留 Sakurairo 视觉效果、樱花氛围和博客阅读体验的前提下，用 Go + MySQL 替代 WordPress/PHP 运行时。

## 运行

需要 Go 1.22+ 和 MySQL 8。

```bash
go mod download
go run ./cmd/server
```

默认使用已经创建好的本机 MySQL 账户：

```text
sakurairo_app@localhost
database: sakurairo
```

可通过环境变量覆盖：

```bash
APP_ADDR=127.0.0.1:8080
MYSQL_DSN='user:password@tcp(127.0.0.1:3306)/sakurairo?charset=utf8mb4&parseTime=True&loc=Local'
STATIC_DIR=web/static
SEED_DEMO=1
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_MINUTES=30
SITE_NAME='Sakurairo Go'
SITE_DESCRIPTION='以 Sakurairo 为蓝本重写的轻量 Go 博客'
SITE_AUTHOR='Codex'
SITE_NOTICE='Sakurairo 的 Go 化迁移正在进行。'
THEME_COLOR='#fe9600'
HERO_IMAGE='/static/theme/screenshot.jpg'
SITE_AVATAR='/static/theme/content-image/d-1.jpg'
ADMIN_USERNAME=admin
ADMIN_PASSWORD='change-me'
ADMIN_SECRET='change-me-too'
```

## 已实现

- MySQL 表初始化和示例文章种子数据
- 首页、文章详情、归档、搜索、分类、标签页面
- 公共评论提交
- 后台登录
- 后台文章列表、新建、编辑和封面上传
- 后台评论管理
- 后台站点设置：站点名称、描述、作者、公告、主题色、顶部图、头像和导航
- 健康检查接口：`/api/health`
- 文章 JSON 接口：`/api/posts`、`/api/posts/{slug}`
- Sakurairo 原主题 CSS/JS/字体/默认图片资源映射

## 部署状态

当前服务器部署路径是 `/opt/sakurairo-go`，systemd 服务名是 `sakurairo-go.service`，监听 `127.0.0.1:8081`，由 `blog.koimoe.com` 的 Nginx 配置反向代理。

本地源码可能比服务器已部署版本更新。部署前应先运行：

```bash
go test ./...
```

然后备份服务器上的二进制和 `web` 静态/模板目录，再替换并重启服务。

## 下一步

- 对照 `D:\codex\Sakurairo-1.20.10` 做视觉一致性检查
- 添加主题设置后台：站点信息、公告、头像、顶部图、颜色、导航
- 添加分类和标签管理后台
- 增强编辑器：预览、草稿、摘要辅助和更安全的 HTML 处理
- 增加评论防垃圾、CSRF 保护和更完整的审核状态
- 支持 WordPress 导出 XML 或数据库导入
- 增加 RSS/Atom、sitemap、SEO 和 Open Graph 信息
- 增加部署脚本、备份脚本和基础测试
