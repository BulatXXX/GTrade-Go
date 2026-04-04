package source

import "context"

type TarkovSource struct{}

func NewTarkovSource() *TarkovSource {
	return &TarkovSource{}
}

func (s *TarkovSource) Fetch(ctx context.Context) ([]RawItem, error) {
	_ = ctx
	return nil, ErrNotImplemented
}
