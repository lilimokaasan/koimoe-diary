# Mail System

Sakurairo Go has a lightweight SMTP mail foundation for comment notifications.

## Current Scope

- Sends an asynchronous admin notification when a new public comment is saved.
- Shows mail status on `/admin/settings`.
- Provides a `Send test mail` action on `/admin/settings`.
- Keeps `comments.mail_notify` in the database for future reply notifications.

Nested comment replies are not implemented yet, so visitor reply-notification emails are only prepared at the data-model level. The public comment form does not show a reply-notice checkbox until replies can actually send mail.

## Environment Variables

Configure these in the production `.env` file, usually `/opt/sakurairo-go/.env`.

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

Useful service checks:

```bash
systemctl is-active sakurairo-go.service
journalctl -u sakurairo-go.service -n 80 --no-pager
```

## Notes

- Do not commit SMTP credentials.
- For hosted mail providers, prefer app passwords over account passwords.
- Some cloud servers block outbound SMTP ports. If test mail times out, check provider firewall rules and try the provider-recommended submission port.
