package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gtrade/services/notification-service/internal/model"
	"gtrade/services/notification-service/internal/repository"
	"gtrade/services/notification-service/internal/service/provider"
)

type NotificationRepository interface {
	CreateOutboxRecord(ctx context.Context, recipient, subject, htmlBody, textBody, provider string) (*repository.OutboxRecord, error)
	MarkSent(ctx context.Context, id int64, providerMessageID string, sentAt time.Time) error
	MarkFailed(ctx context.Context, id int64, errMessage string) error
}

type EmailService struct {
	repo        NotificationRepository
	provider    provider.EmailProvider
	defaultFrom string
}

func NewEmailService(repo NotificationRepository, emailProvider provider.EmailProvider, defaultFrom string) *EmailService {
	return &EmailService{repo: repo, provider: emailProvider, defaultFrom: defaultFrom}
}

func (s *EmailService) SendEmail(ctx context.Context, req model.SendEmailRequest) (*model.SendEmailResponse, error) {
	if err := validateSendEmailRequest(req); err != nil {
		return nil, err
	}

	from := strings.TrimSpace(req.From)
	if from == "" {
		from = s.defaultFrom
	}
	if from == "" {
		return nil, fmt.Errorf("from is required")
	}

	record, err := s.repo.CreateOutboxRecord(ctx, req.To, req.Subject, req.HTMLBody, req.TextBody, s.provider.Name())
	if err != nil {
		return nil, err
	}

	result, err := s.provider.SendEmail(ctx, provider.SendEmailInput{
		From:     from,
		To:       req.To,
		Subject:  req.Subject,
		HTMLBody: req.HTMLBody,
		TextBody: req.TextBody,
	})
	if err != nil {
		if markErr := s.repo.MarkFailed(ctx, record.ID, err.Error()); markErr != nil {
			return nil, fmt.Errorf("send email: %v; mark failed: %w", err, markErr)
		}
		return nil, err
	}

	if err := s.repo.MarkSent(ctx, record.ID, result.ProviderMessageID, time.Now().UTC()); err != nil {
		return nil, err
	}

	return &model.SendEmailResponse{ID: record.ID, Status: "queued"}, nil
}

func validateSendEmailRequest(req model.SendEmailRequest) error {
	if strings.TrimSpace(req.To) == "" {
		return fmt.Errorf("to is required")
	}
	if strings.TrimSpace(req.Subject) == "" {
		return fmt.Errorf("subject is required")
	}
	if strings.TrimSpace(req.HTMLBody) == "" && strings.TrimSpace(req.TextBody) == "" {
		return fmt.Errorf("html_body or text_body is required")
	}
	return nil
}
