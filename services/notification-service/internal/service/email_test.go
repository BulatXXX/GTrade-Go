package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gtrade/services/notification-service/internal/model"
	"gtrade/services/notification-service/internal/repository"
	"gtrade/services/notification-service/internal/service/provider"
)

type fakeRepository struct {
	record      *repository.OutboxRecord
	createErr   error
	markSentErr error
	markFailErr error

	createdRecipient string
	createdSubject   string
	createdHTMLBody  string
	createdTextBody  string
	createdProvider  string

	markedSentID        int64
	markedSentMessageID string
	markedFailedID      int64
	markedFailedError   string
}

func (f *fakeRepository) CreateOutboxRecord(ctx context.Context, recipient, subject, htmlBody, textBody, provider string) (*repository.OutboxRecord, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}

	f.createdRecipient = recipient
	f.createdSubject = subject
	f.createdHTMLBody = htmlBody
	f.createdTextBody = textBody
	f.createdProvider = provider

	if f.record != nil {
		return f.record, nil
	}

	return &repository.OutboxRecord{ID: 1}, nil
}

func (f *fakeRepository) MarkSent(ctx context.Context, id int64, providerMessageID string, sentAt time.Time) error {
	f.markedSentID = id
	f.markedSentMessageID = providerMessageID
	return f.markSentErr
}

func (f *fakeRepository) MarkFailed(ctx context.Context, id int64, errMessage string) error {
	f.markedFailedID = id
	f.markedFailedError = errMessage
	return f.markFailErr
}

type fakeProvider struct {
	result *provider.SendEmailResult
	err    error
	input  provider.SendEmailInput
}

func (f *fakeProvider) Name() string {
	return "fake"
}

func (f *fakeProvider) SendEmail(ctx context.Context, input provider.SendEmailInput) (*provider.SendEmailResult, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &provider.SendEmailResult{ProviderMessageID: "provider-msg-id"}, nil
}

func TestEmailServiceSendEmailSuccess(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{record: &repository.OutboxRecord{ID: 7}}
	p := &fakeProvider{}
	svc := NewEmailService(repo, p, "GTrade <onboarding@resend.dev>")

	resp, err := svc.SendEmail(context.Background(), model.SendEmailRequest{
		To:       "user@example.com",
		Subject:  "Subject",
		HTMLBody: "<p>Hello</p>",
		TextBody: "Hello",
	})
	if err != nil {
		t.Fatalf("SendEmail: %v", err)
	}

	if resp == nil || resp.Status != "queued" || resp.ID != 7 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if repo.createdProvider != "fake" {
		t.Fatalf("provider = %q, want %q", repo.createdProvider, "fake")
	}
	if p.input.From != "GTrade <onboarding@resend.dev>" {
		t.Fatalf("from = %q, want default sender", p.input.From)
	}
	if repo.markedSentID != 7 {
		t.Fatalf("markedSentID = %d, want %d", repo.markedSentID, 7)
	}
	if repo.markedSentMessageID != "provider-msg-id" {
		t.Fatalf("markedSentMessageID = %q, want %q", repo.markedSentMessageID, "provider-msg-id")
	}
}

func TestEmailServiceSendEmailProviderFailureMarksOutboxFailed(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{record: &repository.OutboxRecord{ID: 11}}
	p := &fakeProvider{err: errors.New("provider failed")}
	svc := NewEmailService(repo, p, "GTrade <onboarding@resend.dev>")

	_, err := svc.SendEmail(context.Background(), model.SendEmailRequest{
		To:       "user@example.com",
		Subject:  "Subject",
		TextBody: "Hello",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if repo.markedFailedID != 11 {
		t.Fatalf("markedFailedID = %d, want %d", repo.markedFailedID, 11)
	}
	if repo.markedFailedError == "" {
		t.Fatal("expected failed error to be recorded")
	}
}

func TestEmailServiceSendEmailValidation(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	p := &fakeProvider{}
	svc := NewEmailService(repo, p, "GTrade <onboarding@resend.dev>")

	_, err := svc.SendEmail(context.Background(), model.SendEmailRequest{
		Subject:  "Subject",
		TextBody: "Hello",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if repo.createdRecipient != "" {
		t.Fatal("expected repository not to be called on validation error")
	}
}
