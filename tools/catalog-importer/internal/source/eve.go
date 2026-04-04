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

func (s *EVESource) Stream(ctx context.Context, consume func(RawItem) error) error {
	_ = ctx
	_ = consume
	return ErrNotImplemented
}
