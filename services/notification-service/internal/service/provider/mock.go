package provider

import "github.com/rs/zerolog"

type MockProvider struct {
	logger zerolog.Logger
}

func NewMockProvider(logger zerolog.Logger) *MockProvider {
	return &MockProvider{logger: logger}
}

func (p *MockProvider) SendEmail(to, subject, htmlBody, textBody string) error {
	p.logger.Info().
		Str("provider", "mock").
		Str("to", to).
		Str("subject", subject).
		Int("html_len", len(htmlBody)).
		Int("text_len", len(textBody)).
		Msg("send email")
	return nil
}
