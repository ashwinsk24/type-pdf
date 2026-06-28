//go:build !js || !wasm

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	fs := flag.NewFlagSet("md2pdf", flag.ExitOnError)
	pageSize := fs.String("pageSize", "", "page size: a4 or letter")
	marginMm := fs.Int("margin", 0, "margin in mm: 15, 25, or 35")
	fontSize := fs.Int("fontSize", 0, "base font size in pt: 10, 11, or 12")
	theme := fs.String("theme", "", "theme: light or draft")
	fs.Parse(os.Args[1:])
	args := fs.Args()

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <input.md> [output.pdf]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags: --pageSize a4|letter, --margin 15|25|35, --fontSize 10|11|12, --theme light|draft\n")
		os.Exit(1)
	}

	inputPath := args[0]
	outputPath := "output.pdf"
	if len(args) >= 2 {
		outputPath = args[1]
	}

	markdown, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	baseDir := ""
	if fi, err := os.Stat(inputPath); err == nil && !fi.IsDir() {
		idx := strings.LastIndex(inputPath, string(os.PathSeparator))
		if idx >= 0 {
			baseDir = inputPath[:idx]
		}
	}

	config := DefaultConfig()
	if *pageSize != "" {
		config.PageSize = PageSize(*pageSize)
	}
	if *marginMm > 0 {
		config.MarginMm = *marginMm
	}
	if *fontSize > 0 {
		config.BaseFontPt = *fontSize
	}
	if *theme != "" {
		config.Theme = Theme(*theme)
	}

	pdfBytes, err := renderWithBaseDir(markdown, config, baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering PDF: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, pdfBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %d bytes to %s\n", len(pdfBytes), outputPath)
}
