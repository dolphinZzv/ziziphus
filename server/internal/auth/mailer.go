package auth

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net/smtp"
	"text/template"

	"siciv.space/agent/panda_ai/config"
)

//go:embed email_templates/verify_code_zh.html
var verifyCodeZH string

//go:embed email_templates/verify_code_en.html
var verifyCodeEN string

var emailTemplates = map[string]map[string]string{
	"verify_code": {
		"zh": verifyCodeZH,
		"en": verifyCodeEN,
	},
}

type TemplateData struct {
	Code string
}

type Mailer struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

func NewMailer(cfg config.SMTPConfig) *Mailer {
	return &Mailer{
		host:     cfg.Host,
		port:     cfg.Port,
		user:     cfg.User,
		password: cfg.Password,
		from:     cfg.From,
	}
}

func (m *Mailer) Enabled() bool {
	return m.host != "" && m.user != ""
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

	// Build subject
	subjects := map[string]string{
		"zh": "Panda AI - 邮箱验证码",
		"en": "Panda AI - Email Verification Code",
	}
	subject := "Panda AI - Verification Code"
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
	if err := tmpl.Execute(&buf, TemplateData{Code: code}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return m.send(to, subject, buf.String())
}

func (m *Mailer) send(to, subject, htmlBody string) error {
	port := m.port
	if port == "" {
		port = "587"
	}

	contentType := "text/html; charset=UTF-8"
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s\r\n\r\n%s",
		m.from, to, subject, contentType, htmlBody)

	addr := fmt.Sprintf("%s:%s", m.host, port)

	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	defer conn.Close()

	if err := conn.Hello("localhost"); err != nil {
		return err
	}

	if ok, _ := conn.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: m.host}
		if err := conn.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	auth := smtp.PlainAuth("", m.user, m.password, m.host)
	if err := conn.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := conn.Mail(m.from); err != nil {
		return err
	}
	if err := conn.Rcpt(to); err != nil {
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
