package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ResendProvider struct {
	apiKey string
	client *http.Client
}

type resendSendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

type resendSendEmailResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func NewResendProvider(apiKey string) *ResendProvider {
	return &ResendProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *ResendProvider) Name() string {
	return "resend"
}

func (p *ResendProvider) SendEmail(ctx context.Context, input SendEmailInput) (*SendEmailResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("resend api key is empty")
	}
	if strings.TrimSpace(input.From) == "" {
		return nil, fmt.Errorf("from is required")
	}

	payload, err := json.Marshal(resendSendEmailRequest{
		From:    input.From,
		To:      []string{input.To},
		Subject: input.Subject,
		HTML:    input.HTMLBody,
		Text:    input.TextBody,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal resend request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send resend request: %w", err)
	}
	defer resp.Body.Close()

	var body resendSendEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode resend response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if body.Message != "" {
			return nil, fmt.Errorf("resend send email failed: status=%d message=%s", resp.StatusCode, body.Message)
		}
		return nil, fmt.Errorf("resend send email failed: status=%d", resp.StatusCode)
	}
	if body.ID == "" {
		return nil, fmt.Errorf("resend response missing id")
	}

	return &SendEmailResult{ProviderMessageID: body.ID}, nil
}
