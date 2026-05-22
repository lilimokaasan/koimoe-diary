package admin

import (
	"context"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math/big"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	"sakurairo-go/internal/auth"
	"sakurairo-go/internal/config"
	"sakurairo-go/internal/mailer"
	"sakurairo-go/internal/models"
	"sakurairo-go/internal/store"
	"sakurairo-go/internal/view"
)

const (
	adminPasswordHashKey       = "admin_password_hash"
	adminPasswordCodeHashKey   = "admin_password_code_hash"
	adminPasswordCodeExpiryKey = "admin_password_code_expires_at"
)

type Controller struct {
	cfg      *config.Config
	posts    *store.PostStore
	settings *store.SettingsStore
	links    *store.LinkStore
	moments  *store.MomentStore
	media    *store.MediaStore
	mailer   mailer.Sender
	renderer *view.Renderer
}

type PageData struct {
	Site            config.Site
	Title           string
	Description     string
	CanonicalURL    string
	MetaImage       string
	MetaType        string
	Error           string
	Message         string
	MediaQuery      string
	Posts           []models.Post
	Pages           []models.Page
	Comments        []models.Comment
	Post            models.Post
	PageContent     models.Page
	FriendLinks     []models.FriendLink
	FriendLink      models.FriendLink
	Moments         []models.Moment
	Moment          models.Moment
	MediaAssets     []models.MediaAsset
	Categories      []models.Category
	Category        models.Category
	Tags            []models.Tag
	Tag             models.Tag
	PreviousPost    models.Post
	NextPost        models.Post
	RecentPosts     []models.Post
	PostTotal       int
	CommentTotal    int
	Settings        config.Site
	Mail            config.Mail
	MailReady       bool
	MailPasswordSet bool
	Navigation      string
	FocusCards      string
	SocialLinks     string
	ContentHTML     string
	PostTags        string
	IsNew           bool
	Now             time.Time
	AdminLoggedIn   bool
	ShowAdminNav    bool
}

func New(cfg *config.Config, posts *store.PostStore, settings *store.SettingsStore, links *store.LinkStore, moments *store.MomentStore, media *store.MediaStore, mailer mailer.Sender, renderer *view.Renderer) *Controller {
	return &Controller{cfg: cfg, posts: posts, settings: settings, links: links, moments: moments, media: media, mailer: mailer, renderer: renderer}
}

func (c *Controller) Register(server *ghttp.Server) {
	server.BindHandler("GET:/admin", c.Dashboard)
	server.BindHandler("GET:/admin/login", c.Login)
	server.BindHandler("POST:/admin/login", c.LoginPost)
	server.BindHandler("POST:/admin/logout", c.Logout)
	server.BindHandler("GET:/admin/comments", c.Comments)
	server.BindHandler("POST:/admin/comments/bulk", c.BulkUpdateComments)
	server.BindHandler("POST:/admin/comments/{id}/status", c.UpdateCommentStatus)
	server.BindHandler("POST:/admin/comments/{id}/private", c.UpdateCommentPrivacy)
	server.BindHandler("POST:/admin/comments/{id}/delete", c.DeleteComment)
	server.BindHandler("GET:/admin/settings", c.Settings)
	server.BindHandler("POST:/admin/settings", c.SaveSettings)
	server.BindHandler("POST:/admin/mail/test", c.TestMail)
	server.BindHandler("POST:/admin/password/code", c.RequestPasswordCode)
	server.BindHandler("POST:/admin/password", c.ChangePassword)
	server.BindHandler("GET:/admin/media", c.Media)
	server.BindHandler("POST:/admin/media", c.UploadMedia)
	server.BindHandler("POST:/admin/media/{id}/update", c.UpdateMedia)
	server.BindHandler("POST:/admin/media/{id}/delete", c.DeleteMedia)
	server.BindHandler("GET:/admin/links", c.Links)
	server.BindHandler("GET:/admin/links/new", c.NewLink)
	server.BindHandler("POST:/admin/links", c.SaveLink)
	server.BindHandler("GET:/admin/links/{id}/edit", c.EditLink)
	server.BindHandler("POST:/admin/links/{id}", c.SaveLink)
	server.BindHandler("POST:/admin/links/{id}/delete", c.DeleteLink)
	server.BindHandler("GET:/admin/moments", c.Moments)
	server.BindHandler("GET:/admin/moments/new", c.NewMoment)
	server.BindHandler("POST:/admin/moments", c.SaveMoment)
	server.BindHandler("GET:/admin/moments/{id}/edit", c.EditMoment)
	server.BindHandler("POST:/admin/moments/{id}", c.SaveMoment)
	server.BindHandler("POST:/admin/moments/{id}/delete", c.DeleteMoment)
	server.BindHandler("GET:/admin/taxonomy", c.Taxonomy)
	server.BindHandler("GET:/admin/categories/new", c.NewCategory)
	server.BindHandler("POST:/admin/categories", c.SaveCategory)
	server.BindHandler("GET:/admin/categories/{id}/edit", c.EditCategory)
	server.BindHandler("POST:/admin/categories/{id}", c.SaveCategory)
	server.BindHandler("POST:/admin/categories/{id}/delete", c.DeleteCategory)
	server.BindHandler("GET:/admin/tags/new", c.NewTag)
	server.BindHandler("POST:/admin/tags", c.SaveTag)
	server.BindHandler("GET:/admin/tags/{id}/edit", c.EditTag)
	server.BindHandler("POST:/admin/tags/{id}", c.SaveTag)
	server.BindHandler("POST:/admin/tags/{id}/delete", c.DeleteTag)
	server.BindHandler("GET:/admin/posts/new", c.NewPost)
	server.BindHandler("POST:/admin/posts", c.SavePost)
	server.BindHandler("GET:/admin/posts/{id}/edit", c.EditPost)
	server.BindHandler("GET:/admin/posts/{id}/preview", c.PreviewPost)
	server.BindHandler("POST:/admin/posts/{id}", c.SavePost)
	server.BindHandler("GET:/admin/pages", c.Pages)
	server.BindHandler("GET:/admin/pages/new", c.NewPage)
	server.BindHandler("POST:/admin/pages", c.SavePage)
	server.BindHandler("GET:/admin/pages/{id}/edit", c.EditPage)
	server.BindHandler("POST:/admin/pages/{id}", c.SavePage)
}

func (c *Controller) Settings(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	mailCfg := c.cfg.GetMail()
	c.render(r, "admin_settings.tmpl", PageData{
		Site:     c.cfg.GetSite(),
		Title:    "Settings - " + c.cfg.GetSite().Name,
		Message:  r.GetQuery("saved").String(),
		Settings: c.cfg.GetSite(),
		Navigation: formatNavigation(
			c.cfg.GetSite().Navigation,
		),
		FocusCards: formatFocusCards(
			c.cfg.GetSite().FocusCards,
		),
		SocialLinks: formatSocialLinks(
			c.cfg.GetSite().SocialLinks,
		),
		Mail:            mailCfg,
		MailReady:       mailReady(mailCfg),
		MailPasswordSet: mailCfg.Password != "",
		Now:             time.Now(),
	})
}

func (c *Controller) SaveSettings(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	site := config.Site{
		Name:               strings.TrimSpace(r.GetForm("site_name").String()),
		Description:        strings.TrimSpace(r.GetForm("site_description").String()),
		Author:             strings.TrimSpace(r.GetForm("site_author").String()),
		Notice:             strings.TrimSpace(r.GetForm("site_notice").String()),
		ThemeColor:         strings.TrimSpace(r.GetForm("theme_color").String()),
		HeroImage:          strings.TrimSpace(r.GetForm("hero_image").String()),
		HeroOverlayOpacity: strings.TrimSpace(r.GetForm("hero_overlay_opacity").String()),
		Avatar:             strings.TrimSpace(r.GetForm("site_avatar").String()),
		DefaultPostCover:   strings.TrimSpace(r.GetForm("default_post_cover").String()),
		PostLicenseText:    strings.TrimSpace(r.GetForm("post_license_text").String()),
		PostLicenseURL:     strings.TrimSpace(r.GetForm("post_license_url").String()),
		PostShare:          checkboxValue(r, "post_share"),
		PostCopyNotice:     checkboxValue(r, "post_copy_notice"),
		PostReward:         checkboxValue(r, "post_reward"),
		PostRewardText:     strings.TrimSpace(r.GetForm("post_reward_text").String()),
		PostRewardAlipay:   strings.TrimSpace(r.GetForm("post_reward_alipay").String()),
		PostRewardWechat:   strings.TrimSpace(r.GetForm("post_reward_wechat").String()),
		SakuraEffects:      strings.TrimSpace(r.GetForm("sakura_effects").String()),
		FooterText:         strings.TrimSpace(r.GetForm("footer_text").String()),
		FooterCredit:       strings.TrimSpace(r.GetForm("footer_credit").String()),
		Navigation:         parseNavigation(r.GetForm("navigation").String()),
		FocusCards:         parseFocusCards(r.GetForm("focus_cards").String()),
		SocialLinks:        parseSocialLinks(r.GetForm("social_links").String()),
	}
	site = normalizeSiteSettings(site, c.cfg.GetSite())
	mailCfg := normalizeMailSettings(config.Mail{
		Enabled:    r.GetForm("mail_enabled").String() == "1",
		Host:       strings.TrimSpace(r.GetForm("smtp_host").String()),
		Port:       r.GetForm("smtp_port").Int(),
		Username:   strings.TrimSpace(r.GetForm("smtp_username").String()),
		Password:   strings.TrimSpace(r.GetForm("smtp_password").String()),
		From:       strings.TrimSpace(r.GetForm("smtp_from").String()),
		FromName:   strings.TrimSpace(r.GetForm("smtp_from_name").String()),
		AdminEmail: strings.TrimSpace(r.GetForm("mail_admin_email").String()),
		TLSMode:    strings.TrimSpace(r.GetForm("smtp_tls_mode").String()),
	}, c.cfg.GetMail())
	if errText := validateMailSettings(mailCfg); errText != "" {
		c.render(r, "admin_settings.tmpl", PageData{
			Site:            c.cfg.GetSite(),
			Title:           "Settings - " + c.cfg.GetSite().Name,
			Error:           errText,
			Settings:        site,
			Navigation:      formatNavigation(site.Navigation),
			FocusCards:      formatFocusCards(site.FocusCards),
			SocialLinks:     formatSocialLinks(site.SocialLinks),
			Mail:            mailCfg,
			MailReady:       mailReady(mailCfg),
			MailPasswordSet: mailCfg.Password != "",
			Now:             time.Now(),
		})
		return
	}
	if uploadedAvatar, uploadErr := c.saveImageUpload(r, "avatar_upload", "Avatar"); uploadErr != "" {
		c.render(r, "admin_settings.tmpl", PageData{
			Site:            c.cfg.GetSite(),
			Title:           "Settings - " + c.cfg.GetSite().Name,
			Error:           uploadErr,
			Settings:        site,
			Navigation:      formatNavigation(site.Navigation),
			FocusCards:      formatFocusCards(site.FocusCards),
			SocialLinks:     formatSocialLinks(site.SocialLinks),
			Mail:            mailCfg,
			MailReady:       mailReady(mailCfg),
			MailPasswordSet: mailCfg.Password != "",
			Now:             time.Now(),
		})
		return
	} else if uploadedAvatar != "" {
		site.Avatar = uploadedAvatar
	}
	if err := c.settings.SaveSite(r.Context(), site); err != nil {
		c.error(r, err)
		return
	}
	if err := c.settings.SaveMail(r.Context(), mailCfg); err != nil {
		c.error(r, err)
		return
	}
	c.cfg.SetSite(site)
	c.cfg.SetMail(mailCfg)
	r.Response.RedirectTo("/admin/settings?saved=1", http.StatusSeeOther)
}

func (c *Controller) TestMail(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	mailCfg := c.cfg.GetMail()
	if c.mailer == nil || !mailReady(mailCfg) {
		c.render(r, "admin_settings.tmpl", c.settingsPageData("Mail is not configured yet.", ""))
		return
	}
	if err := c.mailer.Send(mailer.Message{
		To:      mailCfg.AdminEmail,
		Subject: "[" + c.cfg.GetSite().Name + "] Test mail",
		Text:    "KoiMoe Diary mail is working.",
		HTML:    `<p style="color:#4b4350">KoiMoe Diary mail is working.</p>`,
	}); err != nil {
		c.render(r, "admin_settings.tmpl", c.settingsPageData("Could not send test mail: "+err.Error(), ""))
		return
	}
	c.render(r, "admin_settings.tmpl", c.settingsPageData("", "Test mail sent."))
}

func (c *Controller) RequestPasswordCode(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	mailCfg := c.cfg.GetMail()
	if c.mailer == nil || !mailReady(mailCfg) {
		c.render(r, "admin_settings.tmpl", c.settingsPageData("Mail is not configured yet.", ""))
		return
	}
	code, err := generateVerificationCode()
	if err != nil {
		c.error(r, err)
		return
	}
	expiresAt := time.Now().Add(10 * time.Minute)
	if err := c.settings.SaveSetting(r.Context(), adminPasswordCodeHashKey, c.verificationCodeHash(code)); err != nil {
		c.error(r, err)
		return
	}
	if err := c.settings.SaveSetting(r.Context(), adminPasswordCodeExpiryKey, strconv.FormatInt(expiresAt.Unix(), 10)); err != nil {
		c.error(r, err)
		return
	}
	if err := c.mailer.Send(mailer.Message{
		To:      mailCfg.AdminEmail,
		Subject: "[" + c.cfg.GetSite().Name + "] Password change verification",
		Text:    fmt.Sprintf("Your password change verification code is %s. It expires in 10 minutes.", code),
		HTML:    passwordVerificationHTML(c.cfg.GetSite().Name, code, 10),
	}); err != nil {
		c.render(r, "admin_settings.tmpl", c.settingsPageData("Could not send verification code: "+err.Error(), ""))
		return
	}
	c.render(r, "admin_settings.tmpl", c.settingsPageData("", "Verification code sent."))
}

func (c *Controller) ChangePassword(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	currentPassword := r.GetForm("current_password").String()
	newPassword := strings.TrimSpace(r.GetForm("new_password").String())
	confirmPassword := strings.TrimSpace(r.GetForm("confirm_password").String())
	code := strings.TrimSpace(r.GetForm("verification_code").String())
	if newPassword == "" || len(newPassword) < 8 {
		c.render(r, "admin_settings.tmpl", c.settingsPageData("New password must be at least 8 characters.", ""))
		return
	}
	if newPassword != confirmPassword {
		c.render(r, "admin_settings.tmpl", c.settingsPageData("New password confirmation does not match.", ""))
		return
	}
	ok, err := c.verifyAdminPassword(r.Context(), currentPassword)
	if err != nil {
		c.error(r, err)
		return
	}
	if !ok {
		c.render(r, "admin_settings.tmpl", c.settingsPageData("Current password is incorrect.", ""))
		return
	}
	if ok, err := c.verifyPasswordCode(r.Context(), code); err != nil {
		c.error(r, err)
		return
	} else if !ok {
		c.render(r, "admin_settings.tmpl", c.settingsPageData("Verification code is invalid or expired.", ""))
		return
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		c.error(r, err)
		return
	}
	if err := c.settings.SaveSetting(r.Context(), adminPasswordHashKey, hash); err != nil {
		c.error(r, err)
		return
	}
	if err := c.settings.DeleteSetting(r.Context(), adminPasswordCodeHashKey); err != nil {
		c.error(r, err)
		return
	}
	if err := c.settings.DeleteSetting(r.Context(), adminPasswordCodeExpiryKey); err != nil {
		c.error(r, err)
		return
	}
	c.clearAdminCookies(r)
	r.Response.RedirectTo("/admin/login?error=Password updated. Please sign in again.", http.StatusSeeOther)
}

func (c *Controller) settingsPageData(errText string, message string) PageData {
	mailCfg := c.cfg.GetMail()
	return PageData{
		Site:            c.cfg.GetSite(),
		Title:           "Settings - " + c.cfg.GetSite().Name,
		Error:           errText,
		Message:         message,
		Settings:        c.cfg.GetSite(),
		Navigation:      formatNavigation(c.cfg.GetSite().Navigation),
		FocusCards:      formatFocusCards(c.cfg.GetSite().FocusCards),
		SocialLinks:     formatSocialLinks(c.cfg.GetSite().SocialLinks),
		Mail:            mailCfg,
		MailReady:       mailReady(mailCfg),
		MailPasswordSet: mailCfg.Password != "",
		Now:             time.Now(),
	}
}

func mailReady(mail config.Mail) bool {
	return mail.Enabled && mail.Host != "" && mail.Port > 0 && mail.From != "" && mail.AdminEmail != ""
}

func normalizeMailSettings(mail config.Mail, fallback config.Mail) config.Mail {
	mail.Host = strings.TrimSpace(mail.Host)
	mail.Username = strings.TrimSpace(mail.Username)
	mail.From = strings.TrimSpace(mail.From)
	mail.FromName = strings.TrimSpace(mail.FromName)
	mail.AdminEmail = strings.TrimSpace(mail.AdminEmail)
	mail.TLSMode = strings.ToLower(strings.TrimSpace(mail.TLSMode))
	if mail.Port <= 0 {
		mail.Port = fallback.Port
	}
	if mail.Port <= 0 {
		mail.Port = 465
	}
	if mail.Password == "" {
		mail.Password = fallback.Password
	}
	if mail.FromName == "" {
		mail.FromName = fallback.FromName
	}
	if mail.FromName == "" {
		mail.FromName = "KoiMoe Diary"
	}
	if mail.AdminEmail == "" && mail.From != "" {
		mail.AdminEmail = fallback.AdminEmail
	}
	if mail.TLSMode != "implicit" && mail.TLSMode != "starttls" && mail.TLSMode != "none" {
		mail.TLSMode = fallback.TLSMode
	}
	if mail.TLSMode != "implicit" && mail.TLSMode != "starttls" && mail.TLSMode != "none" {
		mail.TLSMode = "implicit"
	}
	return mail
}

func validateMailSettings(mail config.Mail) string {
	if !mail.Enabled {
		return ""
	}
	switch {
	case mail.Host == "":
		return "SMTP host is required when mail is enabled."
	case mail.Port <= 0:
		return "SMTP port is required when mail is enabled."
	case mail.Username == "":
		return "SMTP username is required when mail is enabled."
	case mail.Password == "":
		return "SMTP password is required when mail is enabled."
	case mail.From == "":
		return "Sender address is required when mail is enabled."
	case !strings.Contains(mail.From, "@"):
		return "Sender address looks invalid."
	case mail.AdminEmail == "":
		return "Admin email is required when mail is enabled."
	case !strings.Contains(mail.AdminEmail, "@"):
		return "Admin email looks invalid."
	case mail.TLSMode != "implicit" && mail.TLSMode != "starttls" && mail.TLSMode != "none":
		return "SMTP TLS mode must be implicit, starttls, or none."
	default:
		return ""
	}
}

func passwordVerificationHTML(siteName string, code string, expireMinutes int) string {
	siteName = template.HTMLEscapeString(strings.TrimSpace(siteName))
	if siteName == "" {
		siteName = "KoiMoe Diary"
	}
	code = template.HTMLEscapeString(strings.TrimSpace(code))
	expireLabel := template.HTMLEscapeString(strconv.Itoa(expireMinutes))
	body := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{site_name}} Password Verification</title>
</head>
<body style="margin:0;padding:0;background:#fff6fa;color:#4b4350;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','Microsoft YaHei',Arial,sans-serif;">
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#fff6fa;margin:0;padding:32px 12px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:560px;background:#ffffff;border:1px solid #ffe2ee;border-radius:14px;box-shadow:0 18px 50px rgba(251,152,192,0.16);overflow:hidden;">
          <tr>
            <td style="background:linear-gradient(135deg,#fff9fc,#ffeef6);padding:28px 30px 22px;text-align:center;border-bottom:1px solid #ffe3ee;">
              <div style="font-size:13px;letter-spacing:2px;text-transform:uppercase;color:#fb7fb4;font-weight:700;">{{site_name}}</div>
              <h1 style="margin:12px 0 0;font-family:Georgia,'Times New Roman','Microsoft YaHei',serif;font-size:28px;line-height:1.35;font-weight:400;color:#3f3b43;">Password Verification</h1>
              <p style="margin:10px 0 0;font-size:14px;line-height:1.8;color:#8f8791;">You requested an admin password change for {{site_name}}.</p>
            </td>
          </tr>
          <tr>
            <td style="padding:30px;">
              <p style="margin:0 0 18px;font-size:15px;line-height:1.8;color:#5b535f;">Enter the verification code below on the password change page:</p>
              <div style="margin:0 auto 22px;padding:18px 16px;background:#fff1f7;border:1px solid #ffcfe3;border-radius:12px;text-align:center;">
                <div style="font-size:34px;line-height:1.2;letter-spacing:8px;color:#fb78ad;font-weight:800;">{{code}}</div>
              </div>
              <p style="margin:0 0 14px;font-size:14px;line-height:1.8;color:#6f6670;">This code expires in <strong style="color:#e674a0;">{{expire_minutes}} minutes</strong>. To keep your account safe, do not share it with anyone.</p>
              <p style="margin:0 0 22px;font-size:14px;line-height:1.8;color:#6f6670;">If you did not request this change, you can ignore this email. Your current password will remain unchanged.</p>
              <div style="height:1px;background:#ffe2ee;margin:24px 0;"></div>
              <p style="margin:0;font-size:12px;line-height:1.7;color:#a49aa5;">This security email was sent automatically by {{site_name}}. Please do not reply directly.</p>
            </td>
          </tr>
          <tr>
            <td style="padding:18px 30px;background:#fffafd;text-align:center;border-top:1px solid #ffeaf2;">
              <p style="margin:0;font-size:12px;line-height:1.7;color:#b09cab;">{{site_name}} - A soft diary for tiny heartbeats.</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
	body = strings.ReplaceAll(body, "{{site_name}}", siteName)
	body = strings.ReplaceAll(body, "{{code}}", code)
	body = strings.ReplaceAll(body, "{{expire_minutes}}", expireLabel)
	return body
}

func (c *Controller) Media(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	if err := c.backfillLocalMediaAssets(r.Context()); err != nil {
		log.Printf("backfill media assets: %v", err)
	}
	query := strings.TrimSpace(r.GetQuery("q").String())
	assets, err := c.listMediaAssets(r.Context(), query, 240)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_media.tmpl", PageData{
		Site:        c.cfg.GetSite(),
		Title:       "Media - " + c.cfg.GetSite().Name,
		Message:     mediaMessage(r),
		Error:       r.GetQuery("error").String(),
		MediaQuery:  query,
		MediaAssets: assets,
		Now:         time.Now(),
	})
}

func (c *Controller) UploadMedia(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	uploaded, uploadErr := c.saveImageUpload(r, "media_upload", "Media image")
	if uploadErr != "" {
		c.render(r, "admin_media.tmpl", PageData{
			Site:        c.cfg.GetSite(),
			Title:       "Media - " + c.cfg.GetSite().Name,
			Error:       uploadErr,
			MediaAssets: c.mustListMediaAssets(),
			Now:         time.Now(),
		})
		return
	}
	if uploaded == "" {
		c.render(r, "admin_media.tmpl", PageData{
			Site:        c.cfg.GetSite(),
			Title:       "Media - " + c.cfg.GetSite().Name,
			Error:       "Choose an image to upload.",
			MediaAssets: c.mustListMediaAssets(),
			Now:         time.Now(),
		})
		return
	}
	r.Response.RedirectTo("/admin/media?uploaded=1", http.StatusSeeOther)
}

func (c *Controller) UpdateMedia(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	if c.media == nil {
		c.error(r, errors.New("media store is not available"))
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.media.UpdateDetails(r.Context(), store.MediaAssetInput{
		ID:          id,
		Title:       r.GetForm("title").String(),
		AltText:     r.GetForm("alt_text").String(),
		Description: r.GetForm("description").String(),
	}); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/media?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteMedia(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	if c.media == nil {
		c.error(r, errors.New("media store is not available"))
		return
	}
	id := r.GetRouter("id").Int64()
	asset, err := c.media.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.RedirectTo("/admin/media?error=Asset+not+found", http.StatusSeeOther)
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	if err := c.media.Delete(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	if err := c.deleteLocalMediaFile(asset); err != nil {
		log.Printf("delete media file: %v", err)
	}
	r.Response.RedirectTo("/admin/media?deleted=1", http.StatusSeeOther)
}

func mediaMessage(r *ghttp.Request) string {
	if r.GetQuery("deleted").String() != "" {
		return "deleted"
	}
	if r.GetQuery("saved").String() != "" {
		return "saved"
	}
	return r.GetQuery("uploaded").String()
}

func (c *Controller) Login(r *ghttp.Request) {
	if c.isLoggedIn(r) {
		r.Response.RedirectTo("/admin", http.StatusSeeOther)
		return
	}
	c.render(r, "admin_login.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "Admin Login - " + c.cfg.GetSite().Name,
		Error: r.GetQuery("error").String(),
		Now:   time.Now(),
	})
}

func (c *Controller) LoginPost(r *ghttp.Request) {
	if c.cfg.AdminPassword == "" {
		hash, err := c.settings.Setting(r.Context(), adminPasswordHashKey)
		if err != nil {
			c.error(r, err)
			return
		}
		if hash == "" {
			c.render(r, "admin_login.tmpl", PageData{
				Site:  c.cfg.GetSite(),
				Title: "Admin Login - " + c.cfg.GetSite().Name,
				Error: "Admin password is not configured.",
				Now:   time.Now(),
			})
			return
		}
	}
	username := strings.TrimSpace(r.GetForm("username").String())
	password := r.GetForm("password").String()
	ok, err := c.verifyAdminPassword(r.Context(), password)
	if err != nil {
		c.error(r, err)
		return
	}
	if username != c.cfg.AdminUsername || !ok {
		c.render(r, "admin_login.tmpl", PageData{
			Site:  c.cfg.GetSite(),
			Title: "Admin Login - " + c.cfg.GetSite().Name,
			Error: "Invalid username or password.",
			Now:   time.Now(),
		})
		return
	}
	r.Cookie.SetCookie("sakurairo_admin", c.signToken(username, time.Now().Add(24*time.Hour)), "", "/", 24*time.Hour, ghttp.CookieOptions{
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	r.Response.RedirectTo("/admin", http.StatusSeeOther)
}

func (c *Controller) Logout(r *ghttp.Request) {
	c.clearAdminCookies(r)
	r.Response.RedirectTo("/admin/login", http.StatusSeeOther)
}

func (c *Controller) clearAdminCookies(r *ghttp.Request) {
	expiredAt := time.Unix(0, 0).UTC()
	for _, domain := range []string{"", "blog.koimoe.com", ".koimoe.com"} {
		for _, path := range []string{"/", "/admin"} {
			http.SetCookie(r.Response.Writer, &http.Cookie{
				Name:     "sakurairo_admin",
				Value:    "",
				Path:     path,
				Domain:   domain,
				Expires:  expiredAt,
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}
}

func (c *Controller) Dashboard(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	posts, err := c.posts.ListAll(r.Context(), 100)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_posts.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Posts - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Posts:   posts,
		Now:     time.Now(),
	})
}

func (c *Controller) Pages(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	pages, err := c.posts.ListAllPages(r.Context(), 100)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_pages.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Pages - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Pages:   pages,
		Now:     time.Now(),
	})
}

func (c *Controller) Comments(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	comments, err := c.posts.ListAllComments(r.Context(), 200)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_comments.tmpl", PageData{
		Site:     c.cfg.GetSite(),
		Title:    "Comments - " + c.cfg.GetSite().Name,
		Message:  r.GetQuery("saved").String(),
		Comments: comments,
		Now:      time.Now(),
	})
}

func (c *Controller) UpdateCommentStatus(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	status := r.GetForm("status").String()
	if err := c.posts.UpdateCommentStatus(r.Context(), id, status); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/comments?saved=1", http.StatusSeeOther)
}

func (c *Controller) BulkUpdateComments(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	ids := commentIDsFromRequest(r)
	if len(ids) == 0 {
		r.Response.RedirectTo("/admin/comments?saved=none", http.StatusSeeOther)
		return
	}
	action := strings.TrimSpace(r.GetForm("bulk_action").String())
	var err error
	switch action {
	case "approve":
		_, err = c.posts.UpdateCommentsStatus(r.Context(), ids, "approved")
	case "hide":
		_, err = c.posts.UpdateCommentsStatus(r.Context(), ids, "hidden")
	case "private":
		_, err = c.posts.UpdateCommentsPrivacy(r.Context(), ids, true)
	case "public":
		_, err = c.posts.UpdateCommentsPrivacy(r.Context(), ids, false)
	case "delete":
		_, err = c.posts.DeleteComments(r.Context(), ids)
	default:
		r.Response.RedirectTo("/admin/comments?saved=none", http.StatusSeeOther)
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/comments?saved=bulk", http.StatusSeeOther)
}

func (c *Controller) UpdateCommentPrivacy(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	isPrivate := r.GetForm("is_private").Bool()
	if err := c.posts.UpdateCommentPrivacy(r.Context(), id, isPrivate); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/comments?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteComment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.posts.DeleteComment(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/comments?saved=1", http.StatusSeeOther)
}

func (c *Controller) Links(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	links, err := c.links.ListAdmin(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_links.tmpl", PageData{
		Site:        c.cfg.GetSite(),
		Title:       "Links - " + c.cfg.GetSite().Name,
		Message:     r.GetQuery("saved").String(),
		FriendLinks: links,
		Now:         time.Now(),
	})
}

func (c *Controller) NewLink(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_link_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Link - " + c.cfg.GetSite().Name,
		FriendLink: models.FriendLink{
			Category:  models.FriendLinkCategory{Name: "Friends"},
			Visible:   true,
			SortOrder: 0,
		},
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditLink(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	link, err := c.links.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_link_form.tmpl", PageData{
		Site:       c.cfg.GetSite(),
		Title:      "Edit Link - " + c.cfg.GetSite().Name,
		Message:    r.GetQuery("saved").String(),
		FriendLink: link,
		Now:        time.Now(),
	})
}

func (c *Controller) SaveLink(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.FriendLinkInput{
		ID:           id,
		CategoryName: r.GetForm("category").String(),
		Name:         r.GetForm("name").String(),
		URL:          r.GetForm("url").String(),
		Description:  r.GetForm("description").String(),
		ImageURL:     r.GetForm("image_url").String(),
		SortOrder:    r.GetForm("sort_order").Int(),
		Visible:      r.GetForm("visible").String() == "1",
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.URL) == "" {
		c.render(r, "admin_link_form.tmpl", PageData{
			Site:       c.cfg.GetSite(),
			Title:      "Link Form - " + c.cfg.GetSite().Name,
			Error:      "Name and URL are required.",
			FriendLink: friendLinkFromInput(input),
			IsNew:      id == 0,
			Now:        time.Now(),
		})
		return
	}
	linkID, err := c.links.SaveLink(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/links/"+strconv.FormatInt(linkID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteLink(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.links.DeleteLink(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/links?saved=1", http.StatusSeeOther)
}

func (c *Controller) Moments(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	moments, err := c.moments.ListAdmin(r.Context(), 200)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_moments.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Moments - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Moments: moments,
		Now:     time.Now(),
	})
}

func (c *Controller) NewMoment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_moment_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Moment - " + c.cfg.GetSite().Name,
		Moment: models.Moment{
			Author: c.cfg.GetSite().Author,
			Status: "published",
		},
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditMoment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	moment, err := c.moments.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_moment_form.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Edit Moment - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Moment:  moment,
		Now:     time.Now(),
	})
}

func (c *Controller) SaveMoment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.MomentInput{
		ID:      id,
		Content: r.GetForm("content").String(),
		Author:  r.GetForm("author").String(),
		Status:  r.GetForm("status").String(),
	}
	if strings.TrimSpace(input.Author) == "" {
		input.Author = c.cfg.GetSite().Author
	}
	if strings.TrimSpace(input.Content) == "" {
		c.render(r, "admin_moment_form.tmpl", PageData{
			Site:   c.cfg.GetSite(),
			Title:  "Moment Form - " + c.cfg.GetSite().Name,
			Error:  "Content is required.",
			Moment: momentFromInput(input),
			IsNew:  id == 0,
			Now:    time.Now(),
		})
		return
	}
	momentID, err := c.moments.Save(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/moments/"+strconv.FormatInt(momentID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteMoment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.moments.Delete(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/moments?saved=1", http.StatusSeeOther)
}

func (c *Controller) Taxonomy(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	categories, err := c.posts.ListCategoriesAdmin(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	tags, err := c.posts.ListTagsAdmin(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_taxonomy.tmpl", PageData{
		Site:       c.cfg.GetSite(),
		Title:      "Taxonomy - " + c.cfg.GetSite().Name,
		Message:    r.GetQuery("saved").String(),
		Categories: categories,
		Tags:       tags,
		Now:        time.Now(),
	})
}

func (c *Controller) NewCategory(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_category_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Category - " + c.cfg.GetSite().Name,
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditCategory(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	category, err := c.posts.CategoryByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_category_form.tmpl", PageData{
		Site:     c.cfg.GetSite(),
		Title:    "Edit Category - " + c.cfg.GetSite().Name,
		Message:  r.GetQuery("saved").String(),
		Category: category,
		Now:      time.Now(),
	})
}

func (c *Controller) SaveCategory(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.CategoryInput{
		ID:          id,
		Name:        r.GetForm("name").String(),
		Slug:        r.GetForm("slug").String(),
		Description: r.GetForm("description").String(),
		CoverImage:  r.GetForm("cover_image").String(),
	}
	if strings.TrimSpace(input.Name) == "" {
		c.render(r, "admin_category_form.tmpl", PageData{
			Site:     c.cfg.GetSite(),
			Title:    "Category Form - " + c.cfg.GetSite().Name,
			Error:    "Name is required.",
			Category: categoryFromInput(input),
			IsNew:    id == 0,
			Now:      time.Now(),
		})
		return
	}
	categoryID, err := c.posts.SaveCategory(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/categories/"+strconv.FormatInt(categoryID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteCategory(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.posts.DeleteCategory(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/taxonomy?saved=1", http.StatusSeeOther)
}

func (c *Controller) NewTag(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_tag_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Tag - " + c.cfg.GetSite().Name,
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditTag(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	tag, err := c.posts.TagByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_tag_form.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Edit Tag - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Tag:     tag,
		Now:     time.Now(),
	})
}

func (c *Controller) SaveTag(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.TagInput{
		ID:   id,
		Name: r.GetForm("name").String(),
		Slug: r.GetForm("slug").String(),
	}
	if strings.TrimSpace(input.Name) == "" {
		c.render(r, "admin_tag_form.tmpl", PageData{
			Site:  c.cfg.GetSite(),
			Title: "Tag Form - " + c.cfg.GetSite().Name,
			Error: "Name is required.",
			Tag:   tagFromInput(input),
			IsNew: id == 0,
			Now:   time.Now(),
		})
		return
	}
	tagID, err := c.posts.SaveTag(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/tags/"+strconv.FormatInt(tagID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteTag(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.posts.DeleteTag(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/taxonomy?saved=1", http.StatusSeeOther)
}

func (c *Controller) NewPost(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_post_form.tmpl", PageData{
		Site:        c.cfg.GetSite(),
		Title:       "New Post - " + c.cfg.GetSite().Name,
		MediaAssets: c.mustListMediaAssets(),
		Post: models.Post{
			Status:     "published",
			CoverImage: "/static/theme/content-image/d-1.jpg",
			Category:   models.Category{Name: "Blog"},
		},
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditPost(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	post, err := c.posts.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	data := c.formData(post, "Edit Post - "+c.cfg.GetSite().Name, "")
	data.Message = r.GetQuery("saved").String()
	c.render(r, "admin_post_form.tmpl", data)
}

func (c *Controller) PreviewPost(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	post, err := c.posts.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	data, err := c.previewData(r, post)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_post_preview.tmpl", data)
}

func (c *Controller) SavePost(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.PostInput{
		ID:           id,
		Slug:         r.GetForm("slug").String(),
		Title:        r.GetForm("title").String(),
		Excerpt:      r.GetForm("excerpt").String(),
		ContentHTML:  r.GetForm("content_html").String(),
		CoverImage:   r.GetForm("cover_image").String(),
		Status:       r.GetForm("status").String(),
		CategoryName: r.GetForm("category").String(),
		Tags:         splitTags(r.GetForm("tags").String()),
	}
	if uploadedCover, uploadErr := c.saveCoverUpload(r); uploadErr != "" {
		post := postFromInput(input)
		c.render(r, "admin_post_form.tmpl", c.formData(post, "Post Form - "+c.cfg.GetSite().Name, uploadErr))
		return
	} else if uploadedCover != "" {
		input.CoverImage = uploadedCover
	}
	if strings.TrimSpace(input.CoverImage) == "" {
		input.CoverImage = c.cfg.GetSite().DefaultPostCover
	}
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.ContentHTML) == "" {
		post := postFromInput(input)
		c.render(r, "admin_post_form.tmpl", c.formData(post, "Post Form - "+c.cfg.GetSite().Name, "Title and content are required."))
		return
	}
	postID, err := c.posts.SavePost(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/posts/"+strconv.FormatInt(postID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) NewPage(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_page_form.tmpl", c.pageFormData(models.Page{
		Status:     "published",
		CoverImage: c.cfg.GetSite().DefaultPostCover,
	}, "New Page - "+c.cfg.GetSite().Name, "", true))
}

func (c *Controller) EditPage(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	page, err := c.posts.PageByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	data := c.pageFormData(page, "Edit Page - "+c.cfg.GetSite().Name, "", false)
	data.Message = r.GetQuery("saved").String()
	c.render(r, "admin_page_form.tmpl", data)
}

func (c *Controller) SavePage(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.PageInput{
		ID:          id,
		Slug:        r.GetForm("slug").String(),
		Title:       r.GetForm("title").String(),
		Excerpt:     r.GetForm("excerpt").String(),
		ContentHTML: r.GetForm("content_html").String(),
		CoverImage:  r.GetForm("cover_image").String(),
		Status:      r.GetForm("status").String(),
	}
	if uploadedCover, uploadErr := c.saveCoverUpload(r); uploadErr != "" {
		c.render(r, "admin_page_form.tmpl", c.pageFormData(pageFromInput(input), "Page Form - "+c.cfg.GetSite().Name, uploadErr, id == 0))
		return
	} else if uploadedCover != "" {
		input.CoverImage = uploadedCover
	}
	if input.Title == "" || input.ContentHTML == "" {
		c.render(r, "admin_page_form.tmpl", c.pageFormData(pageFromInput(input), "Page Form - "+c.cfg.GetSite().Name, "Title and content are required.", id == 0))
		return
	}
	pageID, err := c.posts.SavePage(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/pages/"+strconv.FormatInt(pageID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) requireLogin(r *ghttp.Request) bool {
	if c.isLoggedIn(r) {
		return true
	}
	r.Response.RedirectTo("/admin/login", http.StatusSeeOther)
	return false
}

func (c *Controller) isLoggedIn(r *ghttp.Request) bool {
	cookie := r.Cookie.Get("sakurairo_admin")
	if cookie == nil {
		return false
	}
	username, ok := c.verifyToken(cookie.String())
	return ok && username == c.cfg.AdminUsername
}

func (c *Controller) signToken(username string, expires time.Time) string {
	payload := username + ":" + strconv.FormatInt(expires.Unix(), 10)
	signature := c.signature(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + ":" + signature))
}

func (c *Controller) verifyToken(token string) (string, bool) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(data), ":")
	if len(parts) != 3 {
		return "", false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", false
	}
	payload := parts[0] + ":" + parts[1]
	expected := c.signature(payload)
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expected)) != 1 {
		return "", false
	}
	return parts[0], true
}

func (c *Controller) verifyAdminPassword(ctx context.Context, password string) (bool, error) {
	hash, err := c.settings.Setting(ctx, adminPasswordHashKey)
	if err != nil {
		return false, err
	}
	if hash != "" {
		return auth.VerifyPassword(password, hash), nil
	}
	if c.cfg.AdminPassword == "" {
		return false, nil
	}
	return subtle.ConstantTimeCompare([]byte(password), []byte(c.cfg.AdminPassword)) == 1, nil
}

func (c *Controller) verifyPasswordCode(ctx context.Context, code string) (bool, error) {
	if code == "" {
		return false, nil
	}
	storedHash, err := c.settings.Setting(ctx, adminPasswordCodeHashKey)
	if err != nil {
		return false, err
	}
	if storedHash == "" {
		return false, nil
	}
	expiryValue, err := c.settings.Setting(ctx, adminPasswordCodeExpiryKey)
	if err != nil {
		return false, err
	}
	expiry, err := strconv.ParseInt(expiryValue, 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return false, nil
	}
	actual := c.verificationCodeHash(code)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(storedHash)) == 1, nil
}

func (c *Controller) verificationCodeHash(code string) string {
	secret := c.cfg.AdminSecret
	if secret == "" {
		secret = c.cfg.AdminPassword
	}
	if secret == "" {
		secret = c.cfg.AdminUsername
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strings.TrimSpace(code)))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func generateVerificationCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := crand.Int(crand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (c *Controller) signature(payload string) string {
	secret := c.cfg.AdminSecret
	if secret == "" {
		secret = c.cfg.AdminPassword
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (c *Controller) formData(post models.Post, title string, errText string) PageData {
	tags := make([]string, 0, len(post.Tags))
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	return PageData{
		Site:        c.cfg.GetSite(),
		Title:       title,
		Error:       errText,
		Message:     "",
		Post:        post,
		ContentHTML: string(post.ContentHTML),
		PostTags:    strings.Join(tags, ", "),
		MediaAssets: c.mustListMediaAssets(),
		Now:         time.Now(),
	}
}

func (c *Controller) pageFormData(page models.Page, title string, errText string, isNew bool) PageData {
	return PageData{
		Site:        c.cfg.GetSite(),
		Title:       title,
		Error:       errText,
		Message:     "",
		PageContent: page,
		ContentHTML: string(page.ContentHTML),
		MediaAssets: c.mustListMediaAssets(),
		IsNew:       isNew,
		Now:         time.Now(),
	}
}

func (c *Controller) previewData(r *ghttp.Request, post models.Post) (PageData, error) {
	recentPosts, err := c.posts.ListRecent(r.Context(), 5)
	if err != nil {
		return PageData{}, err
	}
	categories, err := c.posts.ListCategories(r.Context())
	if err != nil {
		return PageData{}, err
	}
	tags, err := c.posts.ListTags(r.Context())
	if err != nil {
		return PageData{}, err
	}
	postTotal, err := c.posts.CountPublished(r.Context())
	if err != nil {
		return PageData{}, err
	}
	commentTotal, err := c.posts.CountComments(r.Context())
	if err != nil {
		return PageData{}, err
	}
	return PageData{
		Site:          c.cfg.GetSite(),
		Title:         "Preview: " + post.Title + " - " + c.cfg.GetSite().Name,
		Description:   post.Excerpt,
		CanonicalURL:  "/admin/posts/" + strconv.FormatInt(post.ID, 10) + "/preview",
		MetaImage:     post.CoverImage,
		MetaType:      "article",
		Post:          post,
		RecentPosts:   recentPosts,
		Categories:    categories,
		Tags:          tags,
		PostTotal:     postTotal,
		CommentTotal:  commentTotal,
		Now:           time.Now(),
		AdminLoggedIn: true,
		ShowAdminNav:  false,
	}, nil
}

func (c *Controller) saveCoverUpload(r *ghttp.Request) (string, string) {
	return c.saveImageUpload(r, "cover_upload", "Cover image")
}

func (c *Controller) saveImageUpload(r *ghttp.Request, field string, label string) (string, string) {
	file := r.GetUploadFile(field)
	if file == nil || file.Filename == "" {
		return "", ""
	}
	if file.Size > 5*1024*1024 {
		return "", label + " must be smaller than 5 MB."
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		return "", label + " must be jpg, png, gif, or webp."
	}
	month := time.Now().Format("2006/01")
	targetDir := filepath.Join(c.cfg.StaticDir, "uploads", month)
	filename, err := file.Save(targetDir, true)
	if err != nil {
		log.Printf("save image upload: %v", err)
		return "", "Could not save " + strings.ToLower(label) + "."
	}
	url := "/static/uploads/" + month + "/" + filename
	if err := c.indexLocalMediaURL(r.Context(), url, file.Filename); err != nil {
		log.Printf("index media upload: %v", err)
	}
	return url, ""
}

func (c *Controller) mustListMediaAssets() []models.MediaAsset {
	assets, err := c.listMediaAssets(context.Background(), "", 120)
	if err != nil {
		log.Printf("list media assets: %v", err)
		return nil
	}
	return assets
}

func (c *Controller) listMediaAssets(ctx context.Context, query string, limit int) ([]models.MediaAsset, error) {
	if c.media != nil {
		assets, err := c.media.ListWithOptions(ctx, store.MediaListOptions{Query: query, Limit: limit})
		if err == nil && len(assets) > 0 {
			return assets, nil
		}
		if err != nil {
			return nil, err
		}
	}
	if err := c.backfillLocalMediaAssets(ctx); err != nil {
		return nil, err
	}
	if c.media != nil {
		return c.media.ListWithOptions(ctx, store.MediaListOptions{Query: query, Limit: limit})
	}
	return c.scanLocalMediaAssets()
}

func (c *Controller) backfillLocalMediaAssets(ctx context.Context) error {
	if c.media == nil {
		return nil
	}
	count, err := c.media.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	assets, err := c.scanLocalMediaAssets()
	if err != nil {
		return err
	}
	for _, asset := range assets {
		if _, err := c.media.UpsertLocal(ctx, store.MediaAssetInput{
			Filename:     asset.Filename,
			OriginalName: asset.OriginalName,
			MimeType:     asset.MimeType,
			SizeBytes:    asset.SizeBytes,
			Width:        asset.Width,
			Height:       asset.Height,
			URL:          asset.URL,
			Storage:      asset.Storage,
			CreatedAt:    asset.CreatedAt,
			UpdatedAt:    asset.UpdatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) scanLocalMediaAssets() ([]models.MediaAsset, error) {
	root := filepath.Join(c.cfg.StaticDir, "uploads")
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var assets []models.MediaAsset
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !isImageFile(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(c.cfg.StaticDir, path)
		if err != nil {
			return err
		}
		url := "/static/" + strings.ReplaceAll(filepath.ToSlash(rel), "//", "/")
		width, height := imageDimensions(path)
		assets = append(assets, models.MediaAsset{
			Filename:     entry.Name(),
			OriginalName: entry.Name(),
			MimeType:     mediaTypeForPath(path),
			SizeBytes:    info.Size(),
			Width:        width,
			Height:       height,
			URL:          url,
			Storage:      "local",
			CreatedAt:    info.ModTime(),
			UpdatedAt:    info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(assets, func(i, j int) bool {
		return assets[i].UpdatedAt.After(assets[j].UpdatedAt)
	})
	if len(assets) > 120 {
		assets = assets[:120]
	}
	return assets, nil
}

func (c *Controller) indexLocalMediaURL(ctx context.Context, url string, originalName string) error {
	if c.media == nil || !strings.HasPrefix(url, "/static/") {
		return nil
	}
	rel := strings.TrimPrefix(url, "/static/")
	path := filepath.Join(c.cfg.StaticDir, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	width, height := imageDimensions(path)
	_, err = c.media.UpsertLocal(ctx, store.MediaAssetInput{
		Filename:     filepath.Base(path),
		OriginalName: originalName,
		MimeType:     mediaTypeForPath(path),
		SizeBytes:    info.Size(),
		Width:        width,
		Height:       height,
		URL:          url,
		Storage:      "local",
		CreatedAt:    info.ModTime(),
		UpdatedAt:    info.ModTime(),
	})
	return err
}

func (c *Controller) deleteLocalMediaFile(asset models.MediaAsset) error {
	if asset.Storage != "local" || !strings.HasPrefix(asset.URL, "/static/uploads/") {
		return nil
	}
	rel := strings.TrimPrefix(asset.URL, "/static/")
	path := filepath.Join(c.cfg.StaticDir, filepath.FromSlash(rel))
	uploadsRoot := filepath.Join(c.cfg.StaticDir, "uploads")
	resolvedPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	resolvedRoot, err := filepath.Abs(uploadsRoot)
	if err != nil {
		return err
	}
	if resolvedPath != resolvedRoot && !strings.HasPrefix(resolvedPath, resolvedRoot+string(os.PathSeparator)) {
		return fmt.Errorf("refusing to delete media path outside uploads: %s", resolvedPath)
	}
	if err := os.Remove(resolvedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func imageDimensions(path string) (int, int) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func mediaTypeForPath(path string) string {
	if mediaType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); mediaType != "" {
		return mediaType
	}
	return "application/octet-stream"
}

func isImageFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func postFromInput(input store.PostInput) models.Post {
	tags := make([]models.Tag, 0, len(input.Tags))
	for _, tag := range input.Tags {
		tags = append(tags, models.Tag{Name: tag})
	}
	return models.Post{
		ID:          input.ID,
		Slug:        input.Slug,
		Title:       input.Title,
		Excerpt:     input.Excerpt,
		CoverImage:  input.CoverImage,
		Status:      input.Status,
		Category:    models.Category{Name: input.CategoryName},
		Tags:        tags,
		ContentHTML: template.HTML(input.ContentHTML),
	}
}

func pageFromInput(input store.PageInput) models.Page {
	return models.Page{
		ID:          input.ID,
		Slug:        input.Slug,
		Title:       input.Title,
		Excerpt:     input.Excerpt,
		CoverImage:  input.CoverImage,
		Status:      input.Status,
		ContentHTML: template.HTML(input.ContentHTML),
	}
}

func friendLinkFromInput(input store.FriendLinkInput) models.FriendLink {
	return models.FriendLink{
		ID:          input.ID,
		Name:        strings.TrimSpace(input.Name),
		URL:         strings.TrimSpace(input.URL),
		Description: strings.TrimSpace(input.Description),
		ImageURL:    strings.TrimSpace(input.ImageURL),
		SortOrder:   input.SortOrder,
		Visible:     input.Visible,
		Category: models.FriendLinkCategory{
			Name: strings.TrimSpace(input.CategoryName),
		},
	}
}

func momentFromInput(input store.MomentInput) models.Moment {
	status := strings.TrimSpace(input.Status)
	if status != "draft" {
		status = "published"
	}
	return models.Moment{
		ID:      input.ID,
		Content: strings.TrimSpace(input.Content),
		Author:  strings.TrimSpace(input.Author),
		Status:  status,
	}
}

func categoryFromInput(input store.CategoryInput) models.Category {
	return models.Category{
		ID:          input.ID,
		Slug:        strings.TrimSpace(input.Slug),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		CoverImage:  strings.TrimSpace(input.CoverImage),
	}
}

func tagFromInput(input store.TagInput) models.Tag {
	return models.Tag{
		ID:   input.ID,
		Slug: strings.TrimSpace(input.Slug),
		Name: strings.TrimSpace(input.Name),
	}
}

func (c *Controller) render(r *ghttp.Request, name string, data PageData) {
	c.renderer.HTML(r, name, data)
}

func (c *Controller) error(r *ghttp.Request, err error) {
	log.Printf(
		"admin error method=%s path=%s ip=%s err=%v",
		r.Method,
		r.URL.RequestURI(),
		r.GetClientIp(),
		err,
	)
	r.Response.WriteStatus(500, "Internal Server Error")
}

func splitTags(value string) []string {
	value = strings.ReplaceAll(value, "\uFF0C", ",")
	parts := strings.Split(value, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}

func checkboxValue(r *ghttp.Request, name string) string {
	if r.GetForm(name).String() == "1" {
		return "1"
	}
	return "0"
}

func commentIDsFromRequest(r *ghttp.Request) []int64 {
	if err := r.Request.ParseForm(); err != nil {
		return nil
	}
	return normalizeCommentIDs(r.Request.PostForm["comment_ids"])
}

func normalizeCommentIDs(values []string) []int64 {
	seen := make(map[int64]bool, len(values))
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		if len(ids) == 100 {
			break
		}
	}
	return ids
}

func normalizeSiteSettings(site config.Site, fallback config.Site) config.Site {
	if site.Name == "" {
		site.Name = fallback.Name
	}
	if site.Description == "" {
		site.Description = fallback.Description
	}
	if site.Author == "" {
		site.Author = fallback.Author
	}
	if site.ThemeColor == "" {
		site.ThemeColor = fallback.ThemeColor
	}
	if site.HeroImage == "" {
		site.HeroImage = fallback.HeroImage
	}
	site.HeroOverlayOpacity = normalizeHeroOverlayOpacity(site.HeroOverlayOpacity, fallback.HeroOverlayOpacity)
	if site.Avatar == "" {
		site.Avatar = fallback.Avatar
	}
	if site.DefaultPostCover == "" {
		site.DefaultPostCover = fallback.DefaultPostCover
	}
	if site.PostLicenseText == "" {
		site.PostLicenseText = fallback.PostLicenseText
	}
	site.PostLicenseURL = strings.TrimSpace(site.PostLicenseURL)
	if site.PostShare != "0" {
		site.PostShare = "1"
	}
	if site.PostCopyNotice != "0" {
		site.PostCopyNotice = "1"
	}
	if site.PostReward != "1" || site.PostRewardAlipay == "" && site.PostRewardWechat == "" {
		site.PostReward = "0"
	}
	if site.PostRewardText == "" {
		site.PostRewardText = fallback.PostRewardText
	}
	if site.SakuraEffects != "1" {
		site.SakuraEffects = "0"
	}
	if site.FooterText == "" {
		site.FooterText = fallback.FooterText
	}
	if site.FooterCredit == "" {
		site.FooterCredit = fallback.FooterCredit
	}
	if len(site.Navigation) == 0 {
		site.Navigation = fallback.Navigation
	}
	if len(site.FocusCards) == 0 {
		site.FocusCards = fallback.FocusCards
	}
	site.SocialLinks = normalizeSocialLinks(site.SocialLinks)
	return site
}

func normalizeHeroOverlayOpacity(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	opacity, err := strconv.ParseFloat(value, 64)
	if err != nil {
		opacity, err = strconv.ParseFloat(fallback, 64)
		if err != nil {
			return "1"
		}
	}
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	return strconv.FormatFloat(opacity, 'f', -1, 64)
}

func parseNavigation(value string) []config.NavItem {
	var items []config.NavItem
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		label, url, ok := strings.Cut(line, "|")
		if !ok {
			continue
		}
		label = strings.TrimSpace(label)
		url = strings.TrimSpace(url)
		if label == "" || !isAllowedNavURL(url) {
			continue
		}
		items = append(items, config.NavItem{Label: label, URL: url})
	}
	return items
}

func formatNavigation(items []config.NavItem) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		if item.Label == "" || item.URL == "" {
			continue
		}
		lines = append(lines, item.Label+" | "+item.URL)
	}
	return strings.Join(lines, "\n")
}

func parseFocusCards(value string) []config.FocusCard {
	var cards []config.FocusCard
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		title := strings.TrimSpace(parts[0])
		url := strings.TrimSpace(parts[1])
		image := strings.TrimSpace(strings.Join(parts[2:], "|"))
		if title == "" || !isAllowedNavURL(url) || image == "" {
			continue
		}
		cards = append(cards, config.FocusCard{Title: title, URL: url, Image: image})
		if len(cards) == 3 {
			break
		}
	}
	return cards
}

func formatFocusCards(cards []config.FocusCard) string {
	lines := make([]string, 0, len(cards))
	for _, card := range cards {
		if card.Title == "" || card.URL == "" || card.Image == "" {
			continue
		}
		lines = append(lines, card.Title+" | "+card.URL+" | "+card.Image)
	}
	return strings.Join(lines, "\n")
}

func parseSocialLinks(value string) []config.SocialLink {
	var links []config.SocialLink
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		label := strings.TrimSpace(parts[0])
		url := strings.TrimSpace(parts[1])
		icon := "fa-link"
		if len(parts) >= 3 {
			icon = strings.TrimSpace(strings.Join(parts[2:], "|"))
		}
		if label == "" || !isAllowedSocialURL(url) {
			continue
		}
		links = append(links, config.SocialLink{Label: label, URL: url, Icon: normalizeSocialIcon(icon)})
		if len(links) == 12 {
			break
		}
	}
	return links
}

func formatSocialLinks(links []config.SocialLink) string {
	lines := make([]string, 0, len(links))
	for _, link := range normalizeSocialLinks(links) {
		lines = append(lines, link.Label+" | "+link.URL+" | "+link.Icon)
	}
	return strings.Join(lines, "\n")
}

func normalizeSocialLinks(links []config.SocialLink) []config.SocialLink {
	normalized := make([]config.SocialLink, 0, len(links))
	for _, link := range links {
		link.Label = strings.TrimSpace(link.Label)
		link.URL = strings.TrimSpace(link.URL)
		link.Icon = normalizeSocialIcon(link.Icon)
		if link.Label == "" || !isAllowedSocialURL(link.URL) {
			continue
		}
		normalized = append(normalized, link)
		if len(normalized) == 12 {
			break
		}
	}
	return normalized
}

func normalizeSocialIcon(icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return "fa-link"
	}
	icon = strings.TrimPrefix(icon, "fa ")
	if !strings.HasPrefix(icon, "fa-") {
		icon = "fa-" + icon
	}
	icon = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return -1
	}, strings.ToLower(icon))
	if icon == "" || icon == "fa-" {
		return "fa-link"
	}
	return icon
}

func isAllowedSocialURL(url string) bool {
	return isAllowedNavURL(url) || strings.HasPrefix(url, "mailto:")
}

func isAllowedNavURL(url string) bool {
	return strings.HasPrefix(url, "/") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")
}
