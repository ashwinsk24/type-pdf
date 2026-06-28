package main

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/signintech/gopdf"
)

//go:embed fonts/Inter-Regular.ttf
var interRegular []byte

//go:embed fonts/Inter-Bold.ttf
var interBold []byte

//go:embed fonts/Inter-Italic.ttf
var interItalic []byte

//go:embed fonts/JetBrainsMono-Regular.ttf
var jetbrainsMono []byte

func loadFonts(pdf *gopdf.GoPdf) {
	loadEmbeddedFont(pdf, "Inter", interRegular)
	loadEmbeddedFont(pdf, "Inter-B", interBold)
	loadEmbeddedFont(pdf, "Inter-I", interItalic)
	loadEmbeddedFont(pdf, "JetBrainsMono", jetbrainsMono)
}

func loadEmbeddedFont(pdf *gopdf.GoPdf, family string, data []byte) {
	err := pdf.AddTTFFontByReader(family, bytes.NewReader(data))
	if err != nil {
		panic(fmt.Sprintf("failed to load embedded font %s: %v", family, err))
	}
}
