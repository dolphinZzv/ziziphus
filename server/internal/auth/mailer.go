package auth

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"sync/atomic"
	"text/template"

	"ziziphus/config"
)

//go:embed email_templates/verify_code_zh.html
var verifyCodeZH string

//go:embed email_templates/verify_code_en.html
var verifyCodeEN string

//go:embed email_templates/reset_password_zh.html
var resetPasswordZH string

//go:embed email_templates/reset_password_en.html
var resetPasswordEN string

var emailTemplates = map[string]map[string]string{
	"verify_code": {
		"zh": verifyCodeZH,
		"en": verifyCodeEN,
	},
	"reset_password": {
		"zh": resetPasswordZH,
		"en": resetPasswordEN,
	},
}

type TemplateData struct {
	AppName string
	Code    string
}

type Mailer struct {
	cfg     atomic.Pointer[config.SMTPConfig]
	appName atomic.Pointer[string]
}

func NewMailer(cfg config.SMTPConfig, appName string) *Mailer {
	m := &Mailer{}
	m.cfg.Store(&cfg)
	m.appName.Store(&appName)
	return m
}

// UpdateConfig hot-reloads the SMTP configuration at runtime.
func (m *Mailer) UpdateConfig(cfg config.SMTPConfig) {
	m.cfg.Store(&cfg)
}

// SetAppName updates the application name used in email subjects and templates.
func (m *Mailer) SetAppName(name string) {
	m.appName.Store(&name)
}

func (m *Mailer) appNameVal() string {
	p := m.appName.Load()
	if p == nil {
		return "Ziziphus"
	}
	return *p
}

func (m *Mailer) cfgVal() config.SMTPConfig {
	c := m.cfg.Load()
	if c == nil {
		return config.SMTPConfig{}
	}
	return *c
}

func (m *Mailer) Enabled() bool {
	c := m.cfgVal()
	return c.Host != "" && c.User != ""
}

// SendPasswordResetCode sends a password reset code via email (default lang: zh).
func (m *Mailer) SendPasswordResetCode(to, code string) error {
	return m.SendPasswordResetCodeLang(to, code, "zh")
}

// SendPasswordResetCodeLang sends a password reset code in the specified language.
func (m *Mailer) SendPasswordResetCodeLang(to, code, lang string) error {
	if !m.Enabled() {
		return fmt.Errorf("mailer disabled")
	}

	appName := m.appNameVal()

	subjects := map[string]string{
		"zh": appName + " - 密码重置验证码",
		"en": appName + " - Password Reset Code",
	}
	subject := appName + " - Password Reset Code"
	if s, ok := subjects[lang]; ok {
		subject = s
	}

	tmplBody := resetPasswordZH
	if templates, ok := emailTemplates["reset_password"]; ok {
		if t, ok := templates[lang]; ok {
			tmplBody = t
		}
	}

	tmpl, err := template.New("email").Parse(tmplBody)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, TemplateData{AppName: appName, Code: code}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return m.send(to, subject, buf.String())
}

// SendVerificationCode sends a 6-digit code via email (default lang: zh).
func (m *Mailer) SendVerificationCode(to, code string) error {
	return m.SendVerificationCodeLang(to, code, "zh")
}

// SendVerificationCodeLang sends a 6-digit code in the specified language.
// To add a new language:
//  1. Copy verify_code_zh.html to verify_code_{lang}.html
//  2. Translate the text
//  3. Add the embed directive and register in emailTemplates
func (m *Mailer) SendVerificationCodeLang(to, code, lang string) error {
	if !m.Enabled() {
		return fmt.Errorf("mailer disabled")
	}

	appName := m.appNameVal()

	// Build subject
	subjects := map[string]string{
		"zh": appName + " - 邮箱验证码",
		"en": appName + " - Email Verification Code",
	}
	subject := appName + " - Verification Code"
	if s, ok := subjects[lang]; ok {
		subject = s
	}

	// Render HTML template
	tmplBody := verifyCodeZH // fallback to zh
	if templates, ok := emailTemplates["verify_code"]; ok {
		if t, ok := templates[lang]; ok {
			tmplBody = t
		}
	}

	tmpl, err := template.New("email").Parse(tmplBody)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, TemplateData{AppName: appName, Code: code}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return m.send(to, subject, buf.String())
}

func (m *Mailer) send(to, subject, htmlBody string) error {
	c := m.cfgVal()
	port := c.Port
	if port == "" {
		port = "587"
	}

	// Validate email address and strip CRLF to prevent header injection.
	addr, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}
	// Also strip CRLF from the subject and from fields (defense in depth).
	safeTo := strings.ReplaceAll(strings.ReplaceAll(addr.Address, "\r", ""), "\n", "")
	safeSubject := strings.ReplaceAll(strings.ReplaceAll(subject, "\r", ""), "\n", "")
	safeFrom := strings.ReplaceAll(strings.ReplaceAll(c.From, "\r", ""), "\n", "")
	contentType := "text/html; charset=UTF-8"
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s\r\n\r\n%s",
		safeFrom, safeTo, safeSubject, contentType, htmlBody)

	smtpAddr := fmt.Sprintf("%s:%s", c.Host, port)

	conn, err := smtp.Dial(smtpAddr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", smtpAddr, err)
	}
	defer conn.Close()

	if err := conn.Hello("localhost"); err != nil {
		return err
	}

	if ok, _ := conn.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: c.Host}
		if err := conn.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	auth := smtp.PlainAuth("", c.User, c.Password, c.Host)
	if err := conn.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := conn.Mail(c.From); err != nil {
		return err
	}
	if err := conn.Rcpt(safeTo); err != nil {
		return err
	}
	wc, err := conn.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(wc, msg)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}
	return conn.Quit()
}
