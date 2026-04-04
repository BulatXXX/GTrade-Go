package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"gtrade/tools/catalog-importer/internal/importer"
	"gtrade/tools/catalog-importer/internal/repository"
	"gtrade/tools/catalog-importer/internal/source"
	"gtrade/tools/catalog-importer/internal/transform"
)

func main() {
	sourceName := flag.String("source", "", "source name: warframe|eve|tarkov")
	catalogURL := flag.String("catalog-url", "http://localhost:8084", "catalog-service base URL")
	language := flag.String("language", "en", "catalog import language")
	limit := flag.Int("limit", 0, "max number of items to import, 0 means no limit")
	dryRun := flag.Bool("dry-run", false, "fetch and transform items without writing to catalog-service")
	flag.Parse()

	if *sourceName == "" {
		fmt.Fprintln(os.Stderr, "-source is required")
		os.Exit(1)
	}

	src, err := source.New(source.Config{
		Name:       *sourceName,
		Language:   *language,
		Limit:      *limit,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	repo := repository.NewCatalogHTTPRepository(*catalogURL, *dryRun)
	tr := transform.NewNoopTransformer()
	imp := importer.New(src, tr, repo)

	if err := imp.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("catalog import completed for source=%s language=%s dry_run=%t\n", *sourceName, *language, *dryRun)
}
