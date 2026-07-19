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

//go:embed email_templates/verify_code_ja.html
var verifyCodeJA string

//go:embed email_templates/verify_code_fr.html
var verifyCodeFR string

//go:embed email_templates/verify_code_de.html
var verifyCodeDE string

//go:embed email_templates/verify_code_es.html
var verifyCodeES string

//go:embed email_templates/verify_code_ko.html
var verifyCodeKO string

//go:embed email_templates/verify_code_ru.html
var verifyCodeRU string

//go:embed email_templates/reset_password_zh.html
var resetPasswordZH string

//go:embed email_templates/reset_password_en.html
var resetPasswordEN string

//go:embed email_templates/reset_password_ja.html
var resetPasswordJA string

//go:embed email_templates/reset_password_fr.html
var resetPasswordFR string

//go:embed email_templates/reset_password_de.html
var resetPasswordDE string

//go:embed email_templates/reset_password_es.html
var resetPasswordES string

//go:embed email_templates/reset_password_ko.html
var resetPasswordKO string

//go:embed email_templates/reset_password_ru.html
var resetPasswordRU string

//go:embed email_templates/data_export_zh.html
var dataExportZH string

//go:embed email_templates/data_export_en.html
var dataExportEN string

//go:embed email_templates/data_export_ja.html
var dataExportJA string

//go:embed email_templates/data_export_fr.html
var dataExportFR string

//go:embed email_templates/data_export_de.html
var dataExportDE string

//go:embed email_templates/data_export_es.html
var dataExportES string

//go:embed email_templates/data_export_ko.html
var dataExportKO string

//go:embed email_templates/data_export_ru.html
var dataExportRU string

//go:embed email_templates/welcome_zh.html
var welcomeZH string

//go:embed email_templates/welcome_en.html
var welcomeEN string

//go:embed email_templates/welcome_ja.html
var welcomeJA string

//go:embed email_templates/welcome_fr.html
var welcomeFR string

//go:embed email_templates/welcome_de.html
var welcomeDE string

//go:embed email_templates/welcome_es.html
var welcomeES string

//go:embed email_templates/welcome_ko.html
var welcomeKO string

//go:embed email_templates/welcome_ru.html
var welcomeRU string

var emailTemplates = map[string]map[string]string{
	"verify_code": {
		"zh": verifyCodeZH,
		"en": verifyCodeEN,
		"ja": verifyCodeJA,
		"fr": verifyCodeFR,
		"de": verifyCodeDE,
		"es": verifyCodeES,
		"ko": verifyCodeKO,
		"ru": verifyCodeRU,
	},
	"reset_password": {
		"zh": resetPasswordZH,
		"en": resetPasswordEN,
		"ja": resetPasswordJA,
		"fr": resetPasswordFR,
		"de": resetPasswordDE,
		"es": resetPasswordES,
		"ko": resetPasswordKO,
		"ru": resetPasswordRU,
	},
	"data_export": {
		"zh": dataExportZH,
		"en": dataExportEN,
		"ja": dataExportJA,
		"fr": dataExportFR,
		"de": dataExportDE,
		"es": dataExportES,
		"ko": dataExportKO,
		"ru": dataExportRU,
	},
	"welcome": {
		"zh": welcomeZH,
		"en": welcomeEN,
		"ja": welcomeJA,
		"fr": welcomeFR,
		"de": welcomeDE,
		"es": welcomeES,
		"ko": welcomeKO,
		"ru": welcomeRU,
	},
}

// DataExportTemplateData holds the render variables for data export emails.
type DataExportTemplateData struct {
	AppName  string
	Title    string
	Body     string
	DataJSON string
	Footer   string
}

// WelcomeTemplateData holds the render variables for welcome emails.
type WelcomeTemplateData struct {
	AppName string
	Title   string
	Body    string
	Footer  string
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
		"ja": appName + " - パスワードリセットコード",
		"fr": appName + " - Code de réinitialisation du mot de passe",
		"de": appName + " - Passwort zurücksetzen-Code",
		"es": appName + " - Código de restablecimiento de contraseña",
		"ko": appName + " - 비밀번호 재설정 코드",
		"ru": appName + " - Код сброса пароля",
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
		"ja": appName + " - 認証コード",
		"fr": appName + " - Code de vérification",
		"de": appName + " - Bestätigungscode",
		"es": appName + " - Código de verificación",
		"ko": appName + " - 인증 코드",
		"ru": appName + " - Код подтверждения",
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

// SendWelcomeEmailLang sends a welcome email to a newly registered user
// in their preferred language.
func (m *Mailer) SendWelcomeEmailLang(to string, lang string) error {
	if !m.Enabled() {
		return fmt.Errorf("mailer disabled")
	}

	appName := m.appNameVal()

	titles := map[string]string{
		"zh": "欢迎加入 " + appName,
		"en": "Welcome to " + appName,
		"ja": appName + " へようこそ",
		"fr": "Bienvenue sur " + appName,
		"de": "Willkommen bei " + appName,
		"es": "Bienvenido a " + appName,
		"ko": appName + "에 오신 것을 환영합니다",
		"ru": "Добро пожаловать в " + appName,
	}
	bodies := map[string]string{
		"zh": "你已成功注册 " + appName + " 账号。现在你可以开始与朋友聊天、创建群组、分享文件等。",
		"en": "You have successfully registered for " + appName + ". Now you can start chatting with friends, create groups, share files, and more.",
		"ja": appName + " への登録が完了しました。友達とチャットを始めたり、グループを作成したり、ファイルを共有したりできます。",
		"fr": "Vous vous êtes inscrit avec succès à " + appName + ". Vous pouvez maintenant discuter avec des amis, créer des groupes, partager des fichiers, etc.",
		"de": "Sie haben sich erfolgreich bei " + appName + " registriert. Sie können jetzt mit Freunden chatten, Gruppen erstellen, Dateien teilen und mehr.",
		"es": "Te has registrado exitosamente en " + appName + ". Ahora puedes chatear con amigos, crear grupos, compartir archivos y más.",
		"ko": appName + "에 성공적으로 가입하셨습니다. 이제 친구들과 채팅하고, 그룹을 만들고, 파일을 공유하는 등의 활동을 할 수 있습니다.",
		"ru": "Вы успешно зарегистрировались в " + appName + ". Теперь вы можете общаться с друзьями, создавать группы, обмениваться файлами и многое другое.",
	}
	footers := map[string]string{
		"zh": "如果你没有注册此账号，请忽略此邮件。",
		"en": "If you did not register for this account, please ignore this email.",
		"ja": "このアカウントに登録していない場合は、このメールを無視してください。",
		"fr": "Si vous n'avez pas créé ce compte, veuillez ignorer cet email.",
		"de": "Wenn Sie sich nicht für dieses Konto registriert haben, ignorieren Sie bitte diese E-Mail.",
		"es": "Si no te registraste en esta cuenta, ignora este correo.",
		"ko": "이 계정에 가입하지 않은 경우 이 이메일을 무시하십시오.",
		"ru": "Если вы не регистрировали эту учетную запись, проигнорируйте это письмо.",
	}

	title := "Welcome to " + appName
	if t, ok := titles[lang]; ok {
		title = t
	}
	bodyText := ""
	if b, ok := bodies[lang]; ok {
		bodyText = b
	}
	footerText := ""
	if f, ok := footers[lang]; ok {
		footerText = f
	}

	tmplBody := welcomeZH
	if templates, ok := emailTemplates["welcome"]; ok {
		if t, ok := templates[lang]; ok {
			tmplBody = t
		}
	}

	tmpl, err := template.New("email").Parse(tmplBody)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	subject := title
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, WelcomeTemplateData{
		AppName: appName,
		Title:   title,
		Body:    bodyText,
		Footer:  footerText,
	}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return m.send(to, subject, buf.String())
}

// SendDataExportLang sends a data-export email with the exported JSON embedded
// in the email body, rendered in the user's preferred language.
func (m *Mailer) SendDataExportLang(to string, dataJSON string, lang string) error {
	if !m.Enabled() {
		return fmt.Errorf("mailer disabled")
	}

	appName := m.appNameVal()

	subjects := map[string]string{
		"zh": appName + " - 数据导出",
		"en": appName + " - Data Export",
		"ja": appName + " - データエクスポート",
		"fr": appName + " - Export de données",
		"de": appName + " - Datenexport",
		"es": appName + " - Exportación de datos",
		"ko": appName + " - 데이터 내보내기",
		"ru": appName + " - Экспорт данных",
	}
	subject := appName + " - Data Export"
	if s, ok := subjects[lang]; ok {
		subject = s
	}

	titles := map[string]string{
		"zh": "数据导出完成",
		"en": "Data Export Complete",
		"ja": "データエクスポート完了",
		"fr": "Export de données terminé",
		"de": "Datenexport abgeschlossen",
		"es": "Exportación de datos completada",
		"ko": "데이터 내보내기 완료",
		"ru": "Экспорт данных завершен",
	}
	bodies := map[string]string{
		"zh": "你请求的数据导出已完成。以下是你的数据文件（JSON 格式），包含个人信息、消息记录和会话记录。请妥善保管，不要分享给他人。",
		"en": "Your requested data export is ready. Below is your data file (JSON format) containing profile info, message history, and session records. Please keep this file secure.",
		"ja": "リクエストされたデータのエクスポートが完了しました。以下がデータファイル（JSON形式）です。プロフィール情報、メッセージ履歴、セッション記録が含まれます。",
		"fr": "L'export de vos données est terminé. Voici votre fichier de données (format JSON) contenant profil, messages et sessions. Veuillez garder ce fichier en sécurité.",
		"de": "Ihr angeforderter Datenexport ist fertig. Unten finden Sie Ihre Datendatei (JSON-Format) mit Profil, Nachrichten und Sitzungen. Bitte bewahren Sie diese Datei sicher auf.",
		"es": "Su exportación de datos está lista. A continuación su archivo de datos (formato JSON) con perfil, mensajes y sesiones. Guarde este archivo de forma segura.",
		"ko": "요청하신 데이터 내보내기가 완료되었습니다. 아래는 프로필 정보, 메시지 기록, 세션 기록이 포함된 데이터 파일(JSON 형식)입니다. 이 파일을 안전하게 보관하십시오.",
		"ru": "Запрошенный экспорт данных готов. Ниже ваш файл данных (формат JSON) с профилем, сообщениями и сессиями. Пожалуйста, храните этот файл в безопасности.",
	}
	footers := map[string]string{
		"zh": "如果你没有请求数据导出，请忽略此邮件。",
		"en": "If you did not request this export, please ignore this email.",
		"ja": "このエクスポートをリクエストしていない場合は、このメールを無視してください。",
		"fr": "Si vous n'avez pas demandé cet export, veuillez ignorer cet email.",
		"de": "Wenn Sie diesen Export nicht angefordert haben, ignorieren Sie bitte diese E-Mail.",
		"es": "Si no solicitó esta exportación, ignore este correo.",
		"ko": "이 내보내기를 요청하지 않은 경우 이 이메일을 무시하십시오.",
		"ru": "Если вы не запрашивали этот экспорт, проигнорируйте это письмо.",
	}

	title := "Data Export Complete"
	if t, ok := titles[lang]; ok {
		title = t
	}
	bodyText := ""
	if b, ok := bodies[lang]; ok {
		bodyText = b
	}
	footerText := ""
	if f, ok := footers[lang]; ok {
		footerText = f
	}

	tmplBody := dataExportZH
	if templates, ok := emailTemplates["data_export"]; ok {
		if t, ok := templates[lang]; ok {
			tmplBody = t
		}
	}

	tmpl, err := template.New("email").Parse(tmplBody)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, DataExportTemplateData{
		AppName:  appName,
		Title:    title,
		Body:     bodyText,
		DataJSON: dataJSON,
		Footer:   footerText,
	}); err != nil {
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
