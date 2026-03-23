package source

import "fmt"

type RawItem struct {
	ID   string
	Name string
}

type Source interface {
	Fetch() ([]RawItem, error)
}

func New(name string) (Source, error) {
	switch name {
	case "warframe":
		return &WarframeSource{}, nil
	case "eve":
		return &EVESource{}, nil
	case "tarkov":
		return &TarkovSource{}, nil
	default:
		return nil, fmt.Errorf("unsupported source: %s", name)
	}
}
