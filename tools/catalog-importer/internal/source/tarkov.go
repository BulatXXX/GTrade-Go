package source

type TarkovSource struct{}

func (s *TarkovSource) Fetch() ([]RawItem, error) {
	return []RawItem{}, nil
}
