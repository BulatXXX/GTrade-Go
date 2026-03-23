package service

import "context"

type ServiceClient interface {
	Forward(ctx context.Context, service, path string, payload any) (any, error)
}

type PlaceholderClient struct{}

func NewPlaceholderClient() *PlaceholderClient {
	return &PlaceholderClient{}
}

func (c *PlaceholderClient) Forward(_ context.Context, service, path string, _ any) (any, error) {
	return map[string]any{
		"service": service,
		"path":    path,
		"status":  "not_implemented",
	}, nil
}
