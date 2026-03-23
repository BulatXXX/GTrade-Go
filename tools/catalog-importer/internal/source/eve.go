package source

type EVESource struct{}

func (s *EVESource) Fetch() ([]RawItem, error) {
	return []RawItem{}, nil
}
