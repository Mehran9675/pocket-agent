// Package docs turns files a user attaches - or a page a tool fetches -
// into something a model can read: plain text for documents (PDF,
// Word, Excel, PowerPoint, HTML, Markdown, code, CSV, JSON) and
// resized JPEG/PNG bytes for images.
package docs

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Kind classifies an attachment.
type Kind string

const (
	KindImage    Kind = "image"
	KindDocument Kind = "document"
)

// Result is an extracted attachment.
type Result struct {
	Kind Kind
	// MIME of Data for images; the source type for documents.
	MIME string
	// Text is the document's text.
	Text string
	// Data holds the image prepared for a vision model.
	Data []byte
	// Pages counts PDF pages or slides, when known.
	Pages int
	// Thumb is a small preview of an image.
	Thumb image.Image
}

// ErrUnsupported is returned for binary formats with no extractor.
var ErrUnsupported = errors.New("this file type can't be read")

// MaxImageSide is the longest side images are scaled down to; vision
// models downscale beyond this anyway, and it keeps requests small.
const MaxImageSide = 1568

// Extract reads the attachment named name.
func Extract(name string, data []byte) (*Result, error) {
	ext := strings.ToLower(path.Ext(name))
	mime := http.DetectContentType(data)
	switch {
	case ext == ".pdf" || strings.HasPrefix(mime, "application/pdf"):
		text, pages, err := PDFText(data)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: KindDocument, MIME: "application/pdf", Text: text, Pages: pages}, nil
	case ext == ".docx":
		text, err := docxText(data)
		return docResult("application/vnd.openxmlformats-officedocument.wordprocessingml.document", text, 0, err)
	case ext == ".xlsx":
		text, err := xlsxText(data)
		return docResult("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", text, 0, err)
	case ext == ".pptx":
		text, n, err := pptxText(data)
		return docResult("application/vnd.openxmlformats-officedocument.presentationml.presentation", text, n, err)
	case ext == ".heic" || ext == ".heif":
		return nil, fmt.Errorf("HEIC photos can't be decoded yet - export the photo as JPEG")
	case strings.HasPrefix(mime, "image/") || isImageExt(ext):
		img, data, mt, err := PrepareImage(data)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: KindImage, MIME: mt, Data: data, Thumb: Thumbnail(img, 160)}, nil
	case ext == ".html" || ext == ".htm" || strings.HasPrefix(mime, "text/html"):
		title, text := HTMLText(data)
		if title != "" {
			text = "# " + title + "\n\n" + text
		}
		return &Result{Kind: KindDocument, MIME: "text/html", Text: text}, nil
	}
	if isText(data) {
		return &Result{Kind: KindDocument, MIME: "text/plain", Text: string(data)}, nil
	}
	return nil, ErrUnsupported
}

func docResult(mime, text string, pages int, err error) (*Result, error) {
	if err != nil {
		return nil, err
	}
	return &Result{Kind: KindDocument, MIME: mime, Text: text, Pages: pages}, nil
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	}
	return false
}

// isText reports whether data looks like UTF-8 text.
func isText(data []byte) bool {
	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
		// Don't fail on a rune cut at the boundary.
		for i := 0; i < 3 && !utf8.Valid(sample); i++ {
			sample = sample[:len(sample)-1]
		}
	}
	if !utf8.Valid(sample) {
		return false
	}
	return !bytes.ContainsRune(sample, 0)
}

// PrepareImage decodes an image and re-encodes it at most MaxImageSide
// on its longest side: PNG when it has transparency or is small,
// JPEG otherwise.
func PrepareImage(data []byte) (image.Image, []byte, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, "", fmt.Errorf("can't decode the image: %w", err)
	}
	b := img.Bounds()
	scaled := img
	if b.Dx() > MaxImageSide || b.Dy() > MaxImageSide {
		scaled = resize(img, MaxImageSide)
	} else if format == "jpeg" || format == "png" {
		mt := "image/" + format
		return img, data, mt, nil
	}
	var buf bytes.Buffer
	if format == "png" && len(data) < 1<<20 {
		if err := png.Encode(&buf, scaled); err != nil {
			return nil, nil, "", err
		}
		return scaled, buf.Bytes(), "image/png", nil
	}
	if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: 85}); err != nil {
		return nil, nil, "", err
	}
	return scaled, buf.Bytes(), "image/jpeg", nil
}

// Thumbnail scales img to fit side x side.
func Thumbnail(img image.Image, side int) image.Image {
	if img == nil {
		return nil
	}
	return resize(img, side)
}

func resize(img image.Image, side int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= side && h <= side {
		return img
	}
	if w >= h {
		h = max(1, h*side/w)
		w = side
	} else {
		w = max(1, w*side/h)
		h = side
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// PDFText extracts the text of a PDF, page by page.
func PDFText(data []byte) (string, int, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", 0, fmt.Errorf("can't read the PDF: %w", err)
	}
	var b strings.Builder
	n := r.NumPage()
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := pageText(p)
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		fmt.Fprintf(&b, "--- page %d ---\n%s\n", i, strings.TrimSpace(text))
	}
	out := b.String()
	if strings.TrimSpace(out) == "" {
		return "", n, errors.New("the PDF has no extractable text (it may be a scan)")
	}
	return out, n, nil
}

// pageText rebuilds lines from positioned glyphs, which reads better
// than the library's raw content stream order.
func pageText(p pdf.Page) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("malformed page: %v", r)
		}
	}()
	rows, err := p.GetTextByRow()
	if err != nil || len(rows) == 0 {
		return p.GetPlainText(nil)
	}
	var b strings.Builder
	for _, row := range rows {
		sort.Slice(row.Content, func(i, j int) bool { return row.Content[i].X < row.Content[j].X })
		var lastX, lastW float64
		for i, t := range row.Content {
			if i > 0 && t.X-(lastX+lastW) > t.FontSize*0.2 {
				b.WriteByte(' ')
			}
			b.WriteString(t.S)
			lastX, lastW = t.X, t.W
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func openZip(data []byte) (*zip.Reader, error) {
	return zip.NewReader(bytes.NewReader(data), int64(len(data)))
}

func readZipFile(z *zip.Reader, name string) ([]byte, error) {
	for _, f := range z.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(io.LimitReader(rc, 64<<20))
		}
	}
	return nil, fmt.Errorf("%s not found", name)
}

// ooxmlText walks an Office XML part, writing text elements (w:t, a:t)
// and breaking lines at paragraphs.
func ooxmlText(data []byte, textTag, paraTag string, b *strings.Builder) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	inText := false
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case textTag:
				inText = true
			case "tab":
				b.WriteByte('\t')
			case "br":
				b.WriteByte('\n')
			}
		case xml.EndElement:
			switch t.Name.Local {
			case textTag:
				inText = false
			case paraTag:
				b.WriteByte('\n')
			case "tc":
				b.WriteString(" | ")
			}
		case xml.CharData:
			if inText {
				b.Write(t)
			}
		}
	}
}

func docxText(data []byte) (string, error) {
	z, err := openZip(data)
	if err != nil {
		return "", fmt.Errorf("can't read the document: %w", err)
	}
	part, err := readZipFile(z, "word/document.xml")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	ooxmlText(part, "t", "p", &b)
	return collapseBlankLines(b.String()), nil
}

var slideNum = regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)

func pptxText(data []byte) (string, int, error) {
	z, err := openZip(data)
	if err != nil {
		return "", 0, fmt.Errorf("can't read the presentation: %w", err)
	}
	type slide struct {
		n    int
		name string
	}
	var slides []slide
	for _, f := range z.File {
		if m := slideNum.FindStringSubmatch(f.Name); m != nil {
			n, _ := strconv.Atoi(m[1])
			slides = append(slides, slide{n, f.Name})
		}
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].n < slides[j].n })
	var b strings.Builder
	for _, s := range slides {
		part, err := readZipFile(z, s.name)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "--- slide %d ---\n", s.n)
		ooxmlText(part, "t", "p", &b)
	}
	return collapseBlankLines(b.String()), len(slides), nil
}

func xlsxText(data []byte) (string, error) {
	z, err := openZip(data)
	if err != nil {
		return "", fmt.Errorf("can't read the spreadsheet: %w", err)
	}
	var shared []string
	if part, err := readZipFile(z, "xl/sharedStrings.xml"); err == nil {
		var sst struct {
			SI []struct {
				T string `xml:"t"`
				R []struct {
					T string `xml:"t"`
				} `xml:"r"`
			} `xml:"si"`
		}
		if xml.Unmarshal(part, &sst) == nil {
			for _, si := range sst.SI {
				s := si.T
				for _, r := range si.R {
					s += r.T
				}
				shared = append(shared, s)
			}
		}
	}
	var names []string
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			names = append(names, f.Name)
		}
	}
	sort.Strings(names)
	var b strings.Builder
	for i, name := range names {
		part, err := readZipFile(z, name)
		if err != nil {
			continue
		}
		var ws struct {
			Rows []struct {
				C []struct {
					T  string `xml:"t,attr"`
					V  string `xml:"v"`
					IS struct {
						T string `xml:"t"`
					} `xml:"is"`
				} `xml:"c"`
			} `xml:"sheetData>row"`
		}
		if xml.Unmarshal(part, &ws) != nil {
			continue
		}
		fmt.Fprintf(&b, "--- sheet %d ---\n", i+1)
		for _, row := range ws.Rows {
			cells := make([]string, 0, len(row.C))
			for _, c := range row.C {
				v := c.V
				switch c.T {
				case "s":
					if idx, err := strconv.Atoi(c.V); err == nil && idx < len(shared) {
						v = shared[idx]
					}
				case "inlineStr":
					v = c.IS.T
				}
				cells = append(cells, v)
			}
			b.WriteString(strings.Join(cells, ","))
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := lines[:0]
	blank := 0
	for _, l := range lines {
		l = strings.TrimRight(l, " \t|")
		if strings.TrimSpace(l) == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
