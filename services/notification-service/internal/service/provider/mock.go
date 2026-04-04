package provider

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
)

type MockProvider struct {
	logger zerolog.Logger
}

func NewMockProvider(logger zerolog.Logger) *MockProvider {
	return &MockProvider{logger: logger}
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) SendEmail(_ context.Context, input SendEmailInput) (*SendEmailResult, error) {
	p.logger.Info().
		Str("provider", "mock").
		Str("to", input.To).
		Str("subject", input.Subject).
		Int("html_len", len(input.HTMLBody)).
		Int("text_len", len(input.TextBody)).
		Msg("send email")
	return &SendEmailResult{ProviderMessageID: fmt.Sprintf("mock-%s", input.To)}, nil
}
