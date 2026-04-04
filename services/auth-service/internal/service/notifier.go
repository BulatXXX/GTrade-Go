package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

type EmailNotifier interface {
	SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error
}

type NoopEmailNotifier struct{}

func (n NoopEmailNotifier) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	return nil
}

type NotificationClient struct {
	client *resty.Client
}

type sendEmailRequest struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
	TextBody string `json:"text_body"`
}

type sendEmailResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewNotificationClient(baseURL string) *NotificationClient {
	return &NotificationClient{
		client: resty.New().SetBaseURL(strings.TrimRight(baseURL, "/")),
	}
}

func (c *NotificationClient) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	var success sendEmailResponse
	var failure errorResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(sendEmailRequest{
			To:       to,
			Subject:  subject,
			HTMLBody: htmlBody,
			TextBody: textBody,
		}).
		SetResult(&success).
		SetError(&failure).
		Post("/send-email")
	if err != nil {
		return fmt.Errorf("call notification-service: %w", err)
	}

	if resp.IsError() {
		if failure.Error != "" {
			return fmt.Errorf("notification-service returned %d: %s", resp.StatusCode(), failure.Error)
		}
		return fmt.Errorf("notification-service returned %d", resp.StatusCode())
	}

	if success.Status == "" {
		return fmt.Errorf("notification-service returned empty status")
	}

	return nil
}
