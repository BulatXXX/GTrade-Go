package source

import "context"

type TarkovSource struct{}

func NewTarkovSource() *TarkovSource {
	return &TarkovSource{}
}

func (s *TarkovSource) Stream(ctx context.Context, consume func(RawItem) error) error {
	_ = ctx
	_ = consume
	return ErrNotImplemented
}
