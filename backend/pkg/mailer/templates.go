package mailer

import "github.com/TakuyaYagam1/AstroCTFb/pkg/emailtemplate"

type VerificationData = emailtemplate.VerificationData

type PasswordResetData = emailtemplate.PasswordResetData

// RenderVerificationEmail renders the account verification email body. When html is true an HTML
// template is used; otherwise a plain-text template is used.
func RenderVerificationEmail(data VerificationData, html bool) (string, error) {
	return emailtemplate.RenderVerificationEmail(data, html)
}

// RenderPasswordResetEmail renders the password reset email body. When html is true an HTML
// template is used; otherwise a plain-text template is used.
func RenderPasswordResetEmail(data PasswordResetData, html bool) (string, error) {
	return emailtemplate.RenderPasswordResetEmail(data, html)
}
