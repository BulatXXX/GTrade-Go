package source

type WarframeSource struct{}

func (s *WarframeSource) Fetch() ([]RawItem, error) {
	return []RawItem{}, nil
}
