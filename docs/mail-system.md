# Mail System

Sakurairo Go has lightweight SMTP mail support for comment notifications and admin password verification.

## Current Scope

- Sends an asynchronous admin notification when a new public comment is saved.
- Sends a reply notification to the parent commenter when they choose `Email me on reply`.
- Sends an admin password verification code when changing the admin password from `/admin/settings`.
- Shows mail status on `/admin/settings`.
- Provides a `Send test mail` action on `/admin/settings`.
- Stores `comments.parent_id` for comment threads and `comments.mail_notify` for reply notices.

## Environment Variables

Mail can be configured from `/admin/settings`. The database-backed settings take effect immediately after saving and are preferred for normal operation.

The production `.env` file, usually `/opt/sakurairo-go/.env`, is now only a fallback/bootstrap source. Use it when the database has no mail settings yet or when recovering from a broken admin configuration.

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

`SMTP_TLS_MODE` supports:

- `starttls`: default for port `587`.
- `implicit`: TLS from connection start, often port `465`.
- `none`: plain SMTP. Use only for trusted local relays.

If `MAIL_ENABLED` is not `1`, or required SMTP fields are missing, mail stays disabled and comment posting still succeeds.

## Production Check

After editing the server `.env`, restart the service:

```bash
sudo systemctl restart sakurairo-go.service
```

Then open `/admin/settings`:

- The Mail System card should show `Enabled`.
- Click `Send test mail`.
- If sending fails, the settings page shows the SMTP error without exposing the password.

For Aliyun DirectMail, the recommended admin settings are:

```text
SMTP Host: smtpdm.aliyun.com
SMTP Port: 465
TLS Mode: implicit SSL/TLS
SMTP Username: the approved sender address, such as noreply@koimoe.com
SMTP Password: the SMTP password generated in Aliyun
Sender Address: same as SMTP Username
Sender Name: KoiMoe Diary
Admin Mailbox: the mailbox that should receive tests, comments, and password codes
```

Useful service checks:

```bash
systemctl is-active sakurairo-go.service
journalctl -u sakurairo-go.service -n 80 --no-pager
```

## Notes

- Do not commit SMTP credentials.
- For hosted mail providers, prefer app passwords over account passwords.
- Some cloud servers block outbound SMTP ports. If test mail times out, check provider firewall rules and try the provider-recommended submission port.
