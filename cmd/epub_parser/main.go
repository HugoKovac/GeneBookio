package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hkorpo/book/internal/book"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/epub_parser <book.epub> [output_dir]")
		os.Exit(1)
	}
	epubPath := os.Args[1]
	filename := strings.TrimSuffix(filepath.Base(epubPath), filepath.Ext(epubPath))
	outDir := filepath.Join(".", "output", filename)
	if len(os.Args) >= 3 {
		outDir = os.Args[2]
	}

	if err := book.ExtractEPUB(epubPath, outDir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
