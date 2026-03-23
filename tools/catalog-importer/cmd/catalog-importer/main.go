package main

import (
	"flag"
	"fmt"
	"os"

	"gtrade/tools/catalog-importer/internal/importer"
	"gtrade/tools/catalog-importer/internal/repository"
	"gtrade/tools/catalog-importer/internal/source"
	"gtrade/tools/catalog-importer/internal/transform"
)

func main() {
	sourceName := flag.String("source", "", "source name: warframe|eve|tarkov")
	flag.Parse()

	if *sourceName == "" {
		fmt.Fprintln(os.Stderr, "-source is required")
		os.Exit(1)
	}

	src, err := source.New(*sourceName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	repo := repository.NewNoopRepository()
	tr := transform.NewNoopTransformer()
	imp := importer.New(src, tr, repo)

	if err := imp.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("import placeholder completed for source=%s\n", *sourceName)
}
