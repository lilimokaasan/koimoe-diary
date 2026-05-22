# Sakurairo Go

A lightweight GoFrame + MySQL rebuild of the Sakurairo WordPress theme for KoiMoe Diary.

The goal is to keep the soft Sakurairo atmosphere: pale sakura colors, translucent surfaces, quiet diary-like reading, gentle comments, personal links, moments, and a small admin workspace that feels connected to the public site.

## Requirements

- Go 1.22+
- MySQL 8+
- A writable static directory for uploads, usually `web/static`

On this workspace, Go may need to be called with the full Windows path:

```powershell
& 'C:\Program Files\Go\bin\go.exe' test ./...
& 'C:\Program Files\Go\bin\go.exe' run ./cmd/server
```

## Local Run

```bash
go mod download
go run ./cmd/server
```

Core environment variables:

```env
APP_ADDR=127.0.0.1:8080
MYSQL_DSN=user:password@tcp(127.0.0.1:3306)/sakurairo?charset=utf8mb4&parseTime=True&loc=Local
STATIC_DIR=web/static
SEED_DEMO=1
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_MINUTES=30
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-me
ADMIN_SECRET=change-me-too
```

Default text direction:

```env
SITE_NAME=KoiMoe Diary
SITE_DESCRIPTION=恋と萌えの小さな場所
SITE_AUTHOR=莉莉姆
THEME_COLOR=#fb98c0
```

For this workspace, local development should use the `sakurairo` database. If MySQL is forwarded from the server, prefer `127.0.0.1:3307`.

## Implemented Features

- Public home, post detail, archives, search, category, tag, links, and moments pages.
- Sakurairo-style hero, post cards, sidebar, reading page, comments, error pages, scroll/progress effects, and soft admin styling.
- Automatic article/page table of contents for longer content, with generated heading anchors.
- Legacy Sakurairo shortcode compatibility for `[toc]`, `[begin]`, `[collapse]`, `[download]`, and notice panels such as `[warning]`.
- Configurable article share links with copy-link support.
- Optional source/license notice when visitors copy longer article text.
- Optional article reward panel with configurable support text and payment images.
- Public comments with honeypot, lightweight spam filtering, Markdown rendering with sanitized HTML, nested replies, private comment option, optional reply notification opt-in, and admin comment management.
- Admin login, sidebar navigation, post list, post editor, preview, excerpt helper, cover upload, media library with search and bulk delete, and post editor media picker.
- Site settings for title, description, profile name/avatar, notice, navigation, social links, hero image, overlay opacity, default cover, article license copy, footer copy, Focus Cards, and sakura effects.
- Category/tag management, category covers, friend links, and moments management.
- Likes, views, RSS/Atom feed, sitemap, SEO metadata, Open Graph/Twitter card metadata, random image APIs, and search-index API.
- SMTP mail notifications for new comments, reply notifications for opted-in parent commenters, and admin password verification emails.
- Git-based deployment script with local and remote locking.

## Mail

Mail is configured by environment variables and stays disabled unless `MAIL_ENABLED=1`.

```env
MAIL_ENABLED=1
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=your-account@example.com
SMTP_PASSWORD=your-password-or-app-password
SMTP_FROM=your-account@example.com
SMTP_FROM_NAME=KoiMoe Diary
MAIL_ADMIN_EMAIL=admin@example.com
SMTP_TLS_MODE=starttls
```

See [docs/mail-system.md](docs/mail-system.md) for TLS modes, the `/admin/settings` test-mail flow, and troubleshooting notes.

## Deployment

Production currently runs as:

- Path: `/opt/sakurairo-go`
- Service: `sakurairo-go.service`
- Binary: `/opt/sakurairo-go/sakurairo`
- Listen address: `127.0.0.1:8081`
- Public domain: `blog.koimoe.com`

Preferred deployment is Git push plus server-side build:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\deploy\deploy-sakurairo-go.ps1
```

The script runs local tests, pushes to the server bare repo, runs server tests/build, backs up the active binary and `web/`, restarts the service, and verifies health/public responses.

## Useful Checks

```bash
go test ./...
curl -fsS http://127.0.0.1:8081/api/health
```

On the server:

```bash
systemctl is-active sakurairo-go.service
journalctl -u sakurairo-go.service -n 80 --no-pager
```

## Project Docs

- [docs/text-design-guide.md](docs/text-design-guide.md): text, naming, author, and public copy direction.
- [docs/visual-atmosphere.md](docs/visual-atmosphere.md): visual and interaction atmosphere.
- [docs/wordpress-functional-analysis.md](docs/wordpress-functional-analysis.md): feature analysis and migration priorities.
- [docs/mail-system.md](docs/mail-system.md): SMTP mail setup and operations.
- [deploy/README.md](deploy/README.md): deployment script details.

## Next Useful Work

- Add safer comment moderation workflow with review states and spam quarantine.
- Extend the compatibility renderer for older image syntax and any remaining imported-content edge cases.
- Improve post editor ergonomics around drafts, richer image insertion, and publish-readiness checks.
- Add WordPress import path from XML or database.
- Build a visual parity punch list against `D:\codex\Sakurairo-1.20.10`, especially floating controls, font/skin tools, and mobile navigation.
