package source

import (
	"context"
	"errors"
)

var ErrNotImplemented = errors.New("source is not implemented yet")

type EVESource struct{}

func NewEVESource() *EVESource {
	return &EVESource{}
}

func (s *EVESource) Fetch(ctx context.Context) ([]RawItem, error) {
	_ = ctx
	return nil, ErrNotImplemented
}
