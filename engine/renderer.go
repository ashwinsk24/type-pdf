package main

import (
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/signintech/gopdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	_ "image/jpeg"
	_ "image/png"
	"net/http"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

type drawToken struct {
	text string
	r, g, b uint8
}

func RenderMarkdown(markdown []byte, config Config) ([]byte, error) {
	return renderWithBaseDir(markdown, config, "")
}

func renderWithBaseDir(markdown []byte, config Config, baseDir string) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Strikethrough,
			extension.Table,
			extension.Linkify,
			extension.TaskList,
			extension.Typographer,
		),
	)
	reader := text.NewReader(markdown)
	doc := md.Parser().Parse(reader)

	pdf := &gopdf.GoPdf{}
	pageSize := a4Size()
	if config.PageSize == PageSizeLetter {
		pageSize = letterSize()
	}
	pdf.Start(gopdf.Config{PageSize: *pageSize})

	loadFonts(pdf)

	pdf.AddPage()

	if config.Theme == ThemeDraft {
		pdf.SetFillColor(250, 245, 235)
		pdf.RectFromUpperLeftWithStyle(0, 0, pageSize.W, pageSize.H, "F")
		pdf.SetFillColor(0, 0, 0)
	}

	r := &Renderer{
		pdf:      pdf,
		config:   config,
		marginPt: float64(config.MarginMm) * 72.0 / 25.4,
		baseDir:  baseDir,
	}

	r.pageW = pageSize.W
	r.pageH = pageSize.H
	r.contentW = r.pageW - 2*r.marginPt
	r.x = r.marginPt
	r.y = r.marginPt

	walkNode(r, doc, markdown)

	return pdf.GetBytesPdf(), nil
}

type Renderer struct {
	pdf      *gopdf.GoPdf
	config   Config
	x, y     float64
	marginPt float64
	pageW    float64
	pageH    float64
	contentW float64
	baseDir  string
}

func (r *Renderer) checkPageBreak(nextHeight float64) {
	if r.y+nextHeight > r.pageH-r.marginPt {
		r.pdf.AddPage()
		r.y = r.marginPt
	}
}

func (r *Renderer) lineHeight(pt float64) float64 {
	return pt * 1.55
}

func a4Size() *gopdf.Rect {
	return &gopdf.Rect{W: 595.28, H: 841.89}
}

func letterSize() *gopdf.Rect {
	return &gopdf.Rect{W: 612.0, H: 792.0}
}

func walkNode(r *Renderer, node ast.Node, source []byte) {
	switch n := node.(type) {
	case *ast.Document:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkNode(r, child, source)
		}

	case *ast.Heading:
		level := n.Level
		var fontSize float64
		var marginTop, marginBottom float64
		switch level {
		case 1:
			fontSize = 24
			marginTop = 18
			marginBottom = 10
		case 2:
			fontSize = 18
			marginTop = 14
			marginBottom = 8
		case 3:
			fontSize = 14
			marginTop = 10
			marginBottom = 6
		default:
			fontSize = 12
			marginTop = 10
			marginBottom = 6
		}
		r.y += marginTop
		r.pdf.SetFont("Inter-B", "", int(fontSize))
		r.renderInlineContent(n, source, fontSize, "Inter-B")
		r.y += marginBottom

	case *ast.Paragraph:
		lh := r.lineHeight(float64(r.config.BaseFontPt))
		r.checkPageBreak(lh + 4)
		r.y += 4

		r.pdf.SetX(r.x)
		r.pdf.SetFont("Inter", "", r.config.BaseFontPt)
		r.renderInlineContent(n, source, float64(r.config.BaseFontPt), "Inter")
		r.y += 4

	case *ast.FencedCodeBlock:
		lang := string(n.Language(source))
		r.renderCodeBlockLines(n.Lines(), source, lang)
	case *ast.CodeBlock:
		r.renderCodeBlockLines(n.Lines(), source, "")

	case *ast.Blockquote:
		r.renderBlockquote(n, source)

	case *ast.List:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkNode(r, child, source)
		}

	case *ast.ListItem:
		r.renderListItem(n, source, 0)

	case *ast.ThematicBreak:
		r.checkPageBreak(24)
		r.y += 12
		r.pdf.SetStrokeColor(180, 180, 180)
		r.pdf.SetLineWidth(0.5)
		r.pdf.Line(r.x, r.y, r.x+r.contentW, r.y)
		r.pdf.SetStrokeColor(0, 0, 0)
		r.y += 12

	case *east.Table:
		r.renderTable(n, source)

	case *ast.TextBlock:
		lh := r.lineHeight(float64(r.config.BaseFontPt))
		r.checkPageBreak(lh + 4)
		r.y += 4
		r.pdf.SetX(r.x)
		r.pdf.SetFont("Inter", "", r.config.BaseFontPt)
		r.renderInlineContent(n, source, float64(r.config.BaseFontPt), "Inter")
		r.y += 4

	default:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkNode(r, child, source)
		}
	}
}

type textBatch struct {
	segs []inlineSegment
	full string
}

func (r *Renderer) renderInlineContent(node ast.Node, source []byte, fontPt float64, baseFamily string) {
	var segs []inlineSegment
	collectInlineSegments(node, source, &segs)
	if len(segs) == 0 {
		return
	}

	var batches []interface{}
	var currentText []inlineSegment
	var currentSB strings.Builder

	flushText := func() {
		if len(currentText) > 0 {
			batches = append(batches, textBatch{
				segs: currentText,
				full: currentSB.String(),
			})
			currentText = nil
			currentSB.Reset()
		}
	}

	for _, s := range segs {
		if s.imgSrc != "" {
			flushText()
			batches = append(batches, s)
		} else {
			currentText = append(currentText, s)
			currentSB.WriteString(s.text)
		}
	}
	flushText()

	lh := r.lineHeight(fontPt)

	for _, b := range batches {
		switch v := b.(type) {
		case inlineSegment:
			r.renderImage(v.imgSrc)
		case textBatch:
			r.renderTextBatch(v, fontPt, baseFamily, lh)
		}
	}
}

func (r *Renderer) renderTextBatch(batch textBatch, fontPt float64, baseFamily string, lh float64) {
	fullText := batch.full
	availW := r.pageW - r.marginPt - r.x
	if availW < 20 {
		availW = 20
	}
	lines, err := r.pdf.SplitTextWithWordWrap(fullText, availW)
	if err != nil {
		lines = []string{fullText}
	}

	type styleSpan struct {
		start  int
		end    int
		bold   bool
		italic bool
		mono   bool
		strike bool
		url    string
	}
	var spans []styleSpan
	pos := 0
	for _, s := range batch.segs {
		spans = append(spans, styleSpan{
			start:  pos,
			end:    pos + len(s.text),
			bold:   s.bold,
			italic: s.italic,
			mono:   s.mono,
			strike: s.strike,
			url:    s.url,
		})
		pos += len(s.text)
	}

	charOffset := 0
	for _, line := range lines {
		idx := strings.Index(fullText[charOffset:], line)
		if idx != -1 {
			charOffset += idx
		}

		r.checkPageBreak(lh)
		r.pdf.SetX(r.x)
		r.pdf.SetY(r.y)

		cursor := r.x
		lineStartY := r.y

		for _, sp := range spans {
			if sp.start >= charOffset+len(line) || sp.end <= charOffset {
				continue
			}
			clampStart := max(sp.start, charOffset)
			clampEnd := min(sp.end, charOffset+len(line))
			if clampStart >= clampEnd {
				continue
			}
			part := fullText[clampStart:clampEnd]

			family := baseFamily
			if sp.mono {
				family = "JetBrainsMono"
			} else {
				if sp.bold {
					if strings.Contains(baseFamily, "-I") {
						family = strings.ReplaceAll(baseFamily, "-I", "-BI")
					} else {
						family = "Inter-B"
					}
				} else if sp.italic {
					family = "Inter-I"
				}
			}

			pt := fontPt
			if sp.mono {
				pt = fontPt * 0.9 // Code spans should be slightly smaller
			}

			r.pdf.SetFont(family, "", int(pt))
			r.pdf.SetX(cursor)

			if sp.url != "" {
				r.pdf.SetTextColor(40, 80, 180)
			}
			r.pdf.Cell(nil, part)

			w, _ := r.pdf.MeasureTextWidth(part)
			spanEndX := cursor + w

			if sp.url != "" {
				underlineY := lineStartY + r.lineHeight(pt)*0.9
				r.pdf.SetLineWidth(0.5)
				r.pdf.SetStrokeColor(40, 80, 180)
				r.pdf.Line(cursor, underlineY, spanEndX, underlineY)
				r.pdf.SetStrokeColor(0, 0, 0)
				r.pdf.AddExternalLink(sp.url, cursor, lineStartY-2, w, lh+2)
				r.pdf.SetTextColor(0, 0, 0)
			}

			if sp.strike {
				strikeY := lineStartY + r.lineHeight(pt)*0.45
				r.pdf.SetLineWidth(0.5)
				r.pdf.Line(cursor, strikeY, spanEndX, strikeY)
			}

			cursor = spanEndX
		}

		r.y += lh
		charOffset += len(line)
	}
}

func (r *Renderer) renderImage(src string) {
	var imgW, imgH float64
	var isExternal bool
	var extImg image.Image
	var imgPath string

	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		isExternal = true
		resp, err := http.Get(src)
		if err != nil {
			fmt.Println("TypePDF WASM HTTP Error (Get):", err)
			r.printImagePlaceholder(src)
			return
		}
		defer resp.Body.Close()

		img, _, err := image.Decode(resp.Body)
		if err != nil {
			fmt.Println("TypePDF WASM HTTP Error (Decode):", err)
			r.printImagePlaceholder(src)
			return
		}
		extImg = img
		bounds := img.Bounds()
		imgW = float64(bounds.Dx())
		imgH = float64(bounds.Dy())
	} else {
		imgPath = src
		if r.baseDir != "" && !strings.Contains(src, ":") && !strings.HasPrefix(src, "/") {
			imgPath = r.baseDir + string(os.PathSeparator) + src
		}
		if _, err := os.Stat(imgPath); os.IsNotExist(err) {
			r.printImagePlaceholder(src)
			return
		}
		f, err := os.Open(imgPath)
		if err != nil {
			r.printImagePlaceholder(src)
			return
		}
		cfg, _, err := image.DecodeConfig(f)
		f.Close()
		if err != nil {
			r.printImagePlaceholder(src)
			return
		}
		imgW = float64(cfg.Width)
		imgH = float64(cfg.Height)
	}

	maxW := r.contentW * 0.8
	scale := 1.0
	if imgW > maxW {
		scale = maxW / imgW
	}
	finalW := imgW * scale
	finalH := imgH * scale

	r.checkPageBreak(finalH + 6)
	r.y += 3

	centerX := r.x + (r.contentW-finalW)/2

	var err error
	if isExternal {
		err = r.pdf.ImageFrom(extImg, centerX, r.y, &gopdf.Rect{W: finalW, H: finalH})
	} else {
		err = r.pdf.Image(imgPath, centerX, r.y, &gopdf.Rect{W: finalW, H: finalH})
	}

	if err != nil {
		r.printImagePlaceholder(src)
		return
	}

	r.y += finalH + 3
}

func (r *Renderer) printImagePlaceholder(src string) {
	r.pdf.SetX(r.x)
	r.pdf.SetY(r.y)
	r.pdf.SetFont("Inter-I", "", r.config.BaseFontPt)
	r.pdf.SetTextColor(150, 150, 150)
	r.pdf.Cell(nil, "[image: "+src+"]")
	r.pdf.SetTextColor(0, 0, 0)
	r.y += r.lineHeight(float64(r.config.BaseFontPt))
}

type inlineSegment struct {
	text   string
	bold   bool
	italic bool
	mono   bool
	strike bool
	url    string
	imgSrc string
}

func collectInlineSegments(node ast.Node, source []byte, segs *[]inlineSegment) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			text := string(n.Segment.Value(source))
			if n.SoftLineBreak() || n.HardLineBreak() {
				text += " "
			}
			*segs = append(*segs, inlineSegment{text: text})
		case *ast.Emphasis:
			level := n.Level
			var inner []inlineSegment
			collectInlineSegments(n, source, &inner)
			for i := range inner {
				if level == 1 {
					inner[i].italic = true
				} else {
					inner[i].bold = true
				}
			}
			*segs = append(*segs, inner...)
		case *ast.CodeSpan:
			var codeText strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					codeText.Write(t.Segment.Value(source))
				}
			}
			*segs = append(*segs, inlineSegment{text: codeText.String(), mono: true})
		case *ast.Link:
			url := string(n.Destination)
			linkText := collectText(n, source)
			*segs = append(*segs, inlineSegment{text: linkText, url: url})
		case *east.Strikethrough:
			var inner []inlineSegment
			collectInlineSegments(n, source, &inner)
			for i := range inner {
				inner[i].strike = true
			}
			*segs = append(*segs, inner...)
		case *ast.AutoLink:
			url := collectText(n, source)
			*segs = append(*segs, inlineSegment{text: url, url: url})
		case *ast.Image:
			src := string(n.Destination)
			*segs = append(*segs, inlineSegment{imgSrc: src})
		default:
			collectInlineSegments(n, source, segs)
		}
	}
}

func (r *Renderer) renderCodeBlockLines(lines *text.Segments, source []byte, lang string) {
	var sb strings.Builder
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		sb.Write(line.Value(source))
	}
	code := strings.TrimRight(sb.String(), "\n")

	var coloredLines [][]drawToken

	if lang != "" {
		lexer := lexers.Get(lang)
		if lexer == nil {
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)
		style := styles.Get("github")
		if style == nil {
			style = styles.Fallback
		}

		iterator, err := lexer.Tokenise(nil, code)
		if err == nil {
			var currentLine []drawToken
			for _, token := range iterator.Tokens() {
				styleEntry := style.Get(token.Type)
				color := styleEntry.Colour
				var rC, gC, bC uint8
				if color.IsSet() {
					rC = color.Red()
					gC = color.Green()
					bC = color.Blue()
				} else {
					rC, gC, bC = 36, 41, 46 // default text color
				}

				parts := strings.Split(token.Value, "\n")
				for i, part := range parts {
					if i > 0 {
						coloredLines = append(coloredLines, currentLine)
						currentLine = nil
					}
					if len(part) > 0 {
						currentLine = append(currentLine, drawToken{text: part, r: rC, g: gC, b: bC})
					}
				}
			}
			coloredLines = append(coloredLines, currentLine)
		}
	}

	// Fallback if no lang or chroma fails
	if len(coloredLines) == 0 {
		codeLines := strings.Split(code, "\n")
		for _, line := range codeLines {
			coloredLines = append(coloredLines, []drawToken{{text: line, r: 36, g: 41, b: 46}})
		}
	}

	fontSize := float64(r.config.BaseFontPt) * 0.85
	if fontSize < 7.0 {
		fontSize = 7.0
	}
	padding := 8.0 // Reduced from 12.0 to reduce space

	r.pdf.SetFont("JetBrainsMono", "", int(fontSize))
	maxCodeW := r.contentW - padding*2
	if maxCodeW < 20 {
		maxCodeW = 20
	}
	maxLineW := 0.0
	for _, lineTokens := range coloredLines {
		w := 0.0
		for _, t := range lineTokens {
			tw, _ := r.pdf.MeasureTextWidth(t.text)
			w += tw
		}
		if w > maxLineW {
			maxLineW = w
		}
	}
	if maxLineW > maxCodeW && fontSize > 5 {
		scale := maxCodeW / maxLineW
		newSize := fontSize * scale
		if newSize < 5 {
			newSize = 5
		}
		fontSize = newSize
		r.pdf.SetFont("JetBrainsMono", "", int(fontSize))
	}

	lineH := fontSize * 1.4

	r.checkPageBreak(lineH + padding*2 + 4)
	r.y += 4

	linesDrawn := 0

	for linesDrawn < len(coloredLines) {
		maxCodeY := r.pageH - r.marginPt
		availHeight := maxCodeY - r.y - padding*2

		if availHeight < lineH && linesDrawn == 0 {
			// If not even one line fits on the first page, break page first
			r.pdf.AddPage()
			r.y = r.marginPt
			availHeight = maxCodeY - r.y - padding*2
		}

		linesPerPage := int(availHeight / lineH)
		if linesPerPage <= 0 {
			linesPerPage = 1
		}

		end := linesDrawn + linesPerPage
		if end > len(coloredLines) {
			end = len(coloredLines)
		}

		chunkLines := coloredLines[linesDrawn:end]
		chunkH := float64(len(chunkLines))*lineH + padding*2

		bgX := r.x
		bgY := r.y
		bgW := r.contentW
		bgH := chunkH
		r.pdf.SetFillColor(240, 240, 242)
		r.pdf.RectFromUpperLeftWithStyle(bgX, bgY, bgW, bgH, "F")
		r.pdf.SetFillColor(0, 0, 0)

		r.y += padding
		for _, lineTokens := range chunkLines {
			cursor := r.x + padding
			r.pdf.SetY(r.y)
			for _, t := range lineTokens {
				r.pdf.SetX(cursor)
				r.pdf.SetTextColor(t.r, t.g, t.b)
				r.pdf.Cell(nil, t.text)
				tw, _ := r.pdf.MeasureTextWidth(t.text)
				cursor += tw
			}
			r.pdf.SetTextColor(0, 0, 0)
			r.y += lineH
		}
		r.y += padding

		linesDrawn = end
		if linesDrawn < len(coloredLines) {
			r.pdf.AddPage()
			r.y = r.marginPt
		}
	}
}

func (r *Renderer) renderBlockquote(node *ast.Blockquote, source []byte) {
	r.y += 6

	// Estimate height to keep blockquote on one page if possible
	text := collectText(node, source)
	fontSize := float64(r.config.BaseFontPt)
	lineH := r.lineHeight(fontSize)
	availW := r.pageW - r.marginPt - r.x - 14
	if availW < 20 {
		availW = 20
	}
	lines, _ := r.pdf.SplitTextWithWordWrap(text, availW)
	estH := float64(len(lines))*lineH + 12
	r.checkPageBreak(estH)

	barY := r.y

	savedX := r.x
	r.x += 14
	r.contentW -= 14

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		walkNode(r, child, source)
	}

	r.x = savedX
	r.contentW += 14

	barH := r.y - barY
	if barH > 0 {
		r.pdf.SetFillColor(100, 140, 220)
		r.pdf.RectFromUpperLeftWithStyle(r.x, barY, 3, barH, "F")
		r.pdf.SetFillColor(0, 0, 0)
	}

	r.y += 6
}

func (r *Renderer) renderListItem(node *ast.ListItem, source []byte, depth int) {
	indent := float64(depth) * 14
	fontSize := float64(r.config.BaseFontPt)
	lineH := r.lineHeight(fontSize)

	r.checkPageBreak(lineH)

	var bullet string
	isOrdered := false
	parent := node.Parent()
	if parent != nil && parent.Kind() == ast.KindList {
		list := parent.(*ast.List)
		if list.IsOrdered() {
			isOrdered = true
			idx := 0
			for child := list.FirstChild(); child != nil; child = child.NextSibling() {
				if child == node {
					break
				}
				idx++
			}
			bullet = fmt.Sprintf("%d.", idx+1)
		}
	}
	if bullet == "" {
		bullet = "•"
	}
	hasTaskCheck := false
	if firstChild := node.FirstChild(); firstChild != nil {
		if cb, ok := firstChild.(*east.TaskCheckBox); ok {
			hasTaskCheck = true
			if cb.IsChecked {
				bullet += " [x]"
			} else {
				bullet += " [ ]"
			}
		}
	}

	r.pdf.SetX(r.x + indent)
	r.pdf.SetFont("Inter", "", r.config.BaseFontPt)
	// Bullet "•" glyph is centered at x-height, which sits above the text
	// baseline. Nudge it down ~15% of font size to align visually with
	// regular text characters.
	bulletYOff := float64(r.config.BaseFontPt) * 0.15
	r.pdf.SetY(r.y + bulletYOff)
	r.pdf.Cell(nil, bullet)
	bulletLineY := r.y

	// Use a fixed content offset for all items at the same depth
	// so bullet and content are always consistently aligned
	var contentOffset float64
	if isOrdered {
		list := parent.(*ast.List)
		itemCount := 0
		for child := list.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindListItem {
				itemCount++
			}
		}
		maxW, _ := r.pdf.MeasureTextWidth(fmt.Sprintf("%d.", itemCount))
		contentOffset = indent + maxW + 6
	} else if hasTaskCheck {
		contentOffset = indent + 32
	} else {
		bulW, _ := r.pdf.MeasureTextWidth("•")
		contentOffset = indent + bulW + 6
	}

	savedX := r.x
	r.x = r.x + contentOffset

	sawContent := false
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*east.TaskCheckBox); ok {
			continue
		}
		if child.Kind() == ast.KindList {
			continue
		}
		sawContent = true
		walkNode(r, child, source)
	}

	if !sawContent || r.y <= bulletLineY {
		r.y = bulletLineY + lineH
	}

	r.x = savedX

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindList {
			r.renderList(child.(*ast.List), source, depth+1)
		}
	}
}

func (r *Renderer) renderList(node *ast.List, source []byte, depth int) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindListItem {
			r.renderListItem(child.(*ast.ListItem), source, depth)
		}
	}
}

func (r *Renderer) renderTable(node *east.Table, source []byte) {
	type rowData struct {
		isHeader bool
		cells    []*east.TableCell
	}
	var tableRows []rowData
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *east.TableHeader:
			var cells []*east.TableCell
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if tc, ok := c.(*east.TableCell); ok {
					cells = append(cells, tc)
				}
			}
			if len(cells) > 0 {
				tableRows = append(tableRows, rowData{isHeader: true, cells: cells})
			}
		case *east.TableRow:
			var cells []*east.TableCell
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if tc, ok := c.(*east.TableCell); ok {
					cells = append(cells, tc)
				}
			}
			if len(cells) > 0 {
				tableRows = append(tableRows, rowData{isHeader: false, cells: cells})
			}
		}
	}
	if len(tableRows) == 0 {
		return
	}

	numCols := len(tableRows[0].cells)
	if numCols == 0 {
		return
	}

	availW := r.pageW - r.marginPt - r.x
	if availW < 40 {
		availW = 40
	}

	fontSize := float64(r.config.BaseFontPt) - 1
	lineH := fontSize * 1.4

	padL := 6.0
	padR := 6.0
	padT := 4.0
	padB := 4.0

	type cellContent struct {
		text  string
		lines []string
	}

	rows := make([][]cellContent, len(tableRows))
	for ri, rd := range tableRows {
		for _, cellNode := range rd.cells {
			cellText := collectTableCellText(cellNode, source)
			rows[ri] = append(rows[ri], cellContent{text: cellText})
		}
	}

	colWidths := make([]float64, numCols)
	for ci := 0; ci < numCols; ci++ {
		maxW := 40.0
		for ri := range rows {
			if ci < len(rows[ri]) {
				text := rows[ri][ci].text
				face := "Inter"
				if tableRows[ri].isHeader {
					face = "Inter-B"
				}
				r.pdf.SetFont(face, "", int(fontSize))
				w, _ := r.pdf.MeasureTextWidth(text)
				if w > maxW {
					maxW = w
				}
			}
		}
		colWidths[ci] = maxW + padL + padR
	}

	totalW := 0.0
	for _, w := range colWidths {
		totalW += w
	}
	if totalW > availW {
		scale := availW / totalW
		for ci := range colWidths {
			colWidths[ci] *= scale
		}
	} else if totalW < availW {
		extra := (availW - totalW) / float64(numCols)
		for ci := range colWidths {
			colWidths[ci] += extra
		}
	}

	for ri := range rows {
		for ci := range rows[ri] {
			cellAvail := colWidths[ci] - padL - padR
			if cellAvail < 20 {
				cellAvail = 20
			}
			face := "Inter"
			if tableRows[ri].isHeader {
				face = "Inter-B"
			}
			r.pdf.SetFont(face, "", int(fontSize))
			lines, _ := r.pdf.SplitTextWithWordWrap(rows[ri][ci].text, cellAvail)
			if len(lines) == 0 {
				lines = []string{""}
			}
			rows[ri][ci].lines = lines
		}
	}

	r.y += 4

	for rowIdx, row := range rows {
		isHeader := tableRows[rowIdx].isHeader

		numLines := 1
		for _, cc := range row {
			if len(cc.lines) > numLines {
				numLines = len(cc.lines)
			}
		}

		rowHt := lineH*float64(numLines) + padT + padB
		r.checkPageBreak(rowHt)

		face := "Inter"
		if isHeader {
			face = "Inter-B"
		}
		r.pdf.SetFont(face, "", int(fontSize))

		cx := r.x
		for ci, cc := range row {
			colW := colWidths[ci]

			if isHeader {
				r.pdf.SetFillColor(50, 80, 150)
			} else if rowIdx%2 == 0 {
				r.pdf.SetFillColor(242, 244, 248)
			} else {
				r.pdf.SetFillColor(255, 255, 255)
			}
			r.pdf.RectFromUpperLeftWithStyle(cx, r.y, colW, rowHt, "F")

			if isHeader {
				r.pdf.SetTextColor(255, 255, 255)
			} else {
				r.pdf.SetTextColor(30, 30, 30)
			}
			for li, line := range cc.lines {
				ty := r.y + padT + float64(li)*lineH
				r.pdf.SetX(cx + padL)
				r.pdf.SetY(ty)
				r.pdf.Cell(nil, line)
			}

			r.pdf.SetStrokeColor(190, 195, 205)
			r.pdf.SetLineWidth(0.5)
			r.pdf.RectFromUpperLeftWithStyle(cx, r.y, colW, rowHt, "D")

			cx += colW
		}
		r.y += rowHt
	}
	r.pdf.SetTextColor(0, 0, 0)
	r.pdf.SetFillColor(0, 0, 0)
	r.pdf.SetStrokeColor(0, 0, 0)
	r.y += 4
}

func collectTableCellText(cell *east.TableCell, source []byte) string {
	var sb strings.Builder
	ast.Walk(cell, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if text, ok := n.(*ast.Text); ok {
				sb.Write(text.Segment.Value(source))
			}
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(sb.String())
}

func collectText(node ast.Node, source []byte) string {
	var sb strings.Builder
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if text, ok := n.(*ast.Text); ok {
				sb.Write(text.Segment.Value(source))
			}
		}
		return ast.WalkContinue, nil
	})
	return sb.String()
}
