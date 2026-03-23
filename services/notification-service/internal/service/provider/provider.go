package provider

type EmailProvider interface {
	SendEmail(to, subject, htmlBody, textBody string) error
}
