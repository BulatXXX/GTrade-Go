package provider

import "fmt"

type ResendProvider struct {
	apiKey string
}

func NewResendProvider(apiKey string) *ResendProvider {
	return &ResendProvider{apiKey: apiKey}
}

func (p *ResendProvider) SendEmail(_, _, _, _ string) error {
	if p.apiKey == "" {
		return fmt.Errorf("resend api key is empty")
	}
	return fmt.Errorf("resend provider is not implemented")
}
