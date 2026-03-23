package service

import "gtrade/services/notification-service/internal/service/provider"

type EmailService struct {
	provider provider.EmailProvider
}

func NewEmailService(emailProvider provider.EmailProvider) *EmailService {
	return &EmailService{provider: emailProvider}
}

func (s *EmailService) SendEmail(to, subject, htmlBody, textBody string) error {
	return s.provider.SendEmail(to, subject, htmlBody, textBody)
}
