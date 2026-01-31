package mailer

import (
	"bytes"
	"html/template"
	"sync"
)

type VerificationData struct {
	Username  string
	ActionURL string
	AppName   string
}

type PasswordResetData struct {
	Username  string
	ActionURL string
	AppName   string
}

var (
	verificationHTMLTmpl *template.Template
	verificationTextTmpl *template.Template
	resetHTMLTmpl        *template.Template
	resetTextTmpl        *template.Template
	tmplOnce             sync.Once
)

const baseLayout = `<!DOCTYPE html>
<html>
<head>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta charset="UTF-8">
    <style>
        body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; background-color: #f6f9fc; margin: 0; padding: 0; }
        .container { max-width: 600px; margin: 40px auto; background: #ffffff; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.05); overflow: hidden; }
        .header { background: #1a1f36; padding: 20px; text-align: center; }
        .header h1 { color: #ffffff; margin: 0; font-size: 24px; font-weight: 600; }
        .content { padding: 40px; color: #525f7f; line-height: 1.6; }
        .button { display: inline-block; background-color: #5850ec; color: #ffffff !important; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; margin-top: 20px; }
        .button-danger { background-color: #ef4444; }
        .footer { background: #f6f9fc; padding: 20px; text-align: center; color: #8898aa; font-size: 12px; }
        .link-text { color: #5850ec; word-break: break-all; font-size: 14px; }
        h2 { color: #1a1f36; margin-top: 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            {{template "body" .}}
        </div>
        <div class="footer">
            <p>&copy; {{.AppName}} Team. If you dIDn't request this, please ignore it.</p>
        </div>
    </div>
</body>
</html>`

const verificationBodyHTML = `{{define "body"}}
<h2>Verify your email address</h2>
<p>Hi <strong>{{.Username}}</strong>,</p>
<p>Thanks for joining {{.AppName}}! We're excited to have you on board.</p>
<p>Please confirm your account by clicking the button below:</p>
<div style="text-align: center; margin: 30px 0;">
    <a href="{{.ActionURL}}" class="button">Verify Email</a>
</div>
<p>Or paste this link into your browser:<br>
<a href="{{.ActionURL}}" class="link-text">{{.ActionURL}}</a></p>
<p style="color: #8898aa; font-size: 14px;">This link expires in 24 hours.</p>
{{end}}`

const resetBodyHTML = `{{define "body"}}
<h2>Reset your password</h2>
<p>Hi <strong>{{.Username}}</strong>,</p>
<p>We received a request to reset your password for your {{.AppName}} account.</p>
<div style="text-align: center; margin: 30px 0;">
    <a href="{{.ActionURL}}" class="button button-danger">Reset Password</a>
</div>
<p>Or paste this link into your browser:<br>
<a href="{{.ActionURL}}" class="link-text">{{.ActionURL}}</a></p>
<p style="color: #8898aa; font-size: 14px;">This link expires in 1 hour. If you dIDn't request a password reset, you can safely ignore this email.</p>
{{end}}`

const verificationTextTemplate = `Verify your email address

Hi {{.Username}},

Thanks for joining {{.AppName}}! We're excited to have you on board.

Please confirm your account by visiting the link below:
{{.ActionURL}}

This link expires in 24 hours.

If you dIDn't create an account, please ignore this email.

--
{{.AppName}} Team`

const resetTextTemplate = `Reset your password

Hi {{.Username}},

We received a request to reset your password for your {{.AppName}} account.

To reset your password, visit the link below:
{{.ActionURL}}

This link expires in 1 hour.

If you dIDn't request a password reset, you can safely ignore this email.

--
{{.AppName}} Team`

func initTemplates() {
	verificationHTMLTmpl = template.Must(template.New("verification_html").Parse(baseLayout + verificationBodyHTML))
	verificationTextTmpl = template.Must(template.New("verification_text").Parse(verificationTextTemplate))
	resetHTMLTmpl = template.Must(template.New("reset_html").Parse(baseLayout + resetBodyHTML))
	resetTextTmpl = template.Must(template.New("reset_text").Parse(resetTextTemplate))
}

func getTemplates() {
	tmplOnce.Do(initTemplates)
}

func RenderVerificationEmail(data VerificationData, html bool) (string, error) {
	getTemplates()

	if data.AppName == "" {
		data.AppName = "CTFBoard"
	}

	var buf bytes.Buffer
	var err error

	if html {
		err = verificationHTMLTmpl.Execute(&buf, data)
	} else {
		err = verificationTextTmpl.Execute(&buf, data)
	}

	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func RenderPasswordResetEmail(data PasswordResetData, html bool) (string, error) {
	getTemplates()

	if data.AppName == "" {
		data.AppName = "CTFBoard"
	}

	var buf bytes.Buffer
	var err error

	if html {
		err = resetHTMLTmpl.Execute(&buf, data)
	} else {
		err = resetTextTmpl.Execute(&buf, data)
	}

	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
