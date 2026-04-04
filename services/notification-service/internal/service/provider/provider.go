package provider

import "context"

type SendEmailInput struct {
	From     string
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}

type SendEmailResult struct {
	ProviderMessageID string
}

type EmailProvider interface {
	Name() string
	SendEmail(ctx context.Context, input SendEmailInput) (*SendEmailResult, error)
}
