package gfx

import (
	"bytes"
	"embed"
	"encoding/json"
	"image"
	_ "image/png"
	"sync"
)

//go:embed font_*.png font_*.json
var fontFS embed.FS

// Glyph describes one rasterised rune inside the atlas.
type Glyph struct {
	X, Y, W, H int
	OX, OY     int
	ADV        int
}

// Font is a bitmap font atlas.
type Font struct {
	Size   int
	Line   int
	Ascent int
	Glyphs map[rune]Glyph
	Img    *image.Gray
}

type fontMeta struct {
	Size   int              `json:"size"`
	Line   int              `json:"line"`
	Ascent int              `json:"ascent"`
	Glyphs map[string]Glyph `json:"glyphs"`
}

var (
	fontMu    sync.Mutex
	fontCache = map[string]*Font{}
)

// LoadFont loads one of the embedded sizes: "sm" (20px), "md" (26px) or
// "lg" (32px).
func LoadFont(name string) *Font {
	fontMu.Lock()
	defer fontMu.Unlock()
	if f, ok := fontCache[name]; ok {
		return f
	}
	metaBytes, err := fontFS.ReadFile("font_" + name + ".json")
	if err != nil {
		panic(err)
	}
	var meta fontMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		panic(err)
	}
	pngBytes, err := fontFS.ReadFile("font_" + name + ".png")
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		panic(err)
	}
	gray, ok := img.(*image.Gray)
	if !ok {
		gray = image.NewGray(img.Bounds())
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				gray.Set(x, y, img.At(x, y))
			}
		}
	}
	f := &Font{Size: meta.Size, Line: meta.Line, Ascent: meta.Ascent, Img: gray, Glyphs: make(map[rune]Glyph, len(meta.Glyphs))}
	for k, v := range meta.Glyphs {
		rs := []rune(k)
		if len(rs) == 1 {
			f.Glyphs[rs[0]] = v
		}
	}
	fontCache[name] = f
	return f
}

// Measure returns the advance width of s in pixels.
func (f *Font) Measure(s string) int {
	w := 0
	for _, r := range s {
		if g, ok := f.Glyphs[r]; ok {
			w += g.ADV
		} else if r == ' ' {
			w += f.Size / 3
		} else {
			w += f.Size
		}
	}
	return w
}

func (f *Font) glyph(r rune) (Glyph, bool) {
	if g, ok := f.Glyphs[r]; ok {
		return g, true
	}
	switch r {
	case ' ':
		return Glyph{ADV: f.Size / 3}, true
	case '　':
		return Glyph{ADV: f.Size}, true
	}
	return Glyph{}, false
}

// Text draws s with its top left corner at (x,y) and returns the advance.
func (c *Canvas) Text(f *Font, x, y int, s string, col Color) int {
	pen := x
	for _, r := range s {
		g, ok := f.glyph(r)
		if !ok {
			pen += f.Size
			continue
		}
		if g.W > 0 {
			c.blitGlyph(f, g, pen+g.OX, y+g.OY, col)
		}
		pen += g.ADV
	}
	return pen - x
}

// TextShadow draws a one pixel dark shadow then the text.
func (c *Canvas) TextShadow(f *Font, x, y int, s string, col Color) int {
	c.Text(f, x+1, y+2, s, RGBA(0, 0, 0, 140))
	return c.Text(f, x, y, s, col)
}

func (c *Canvas) TextCenter(f *Font, cx, y int, s string, col Color) {
	c.Text(f, cx-f.Measure(s)/2, y, s, col)
}

func (c *Canvas) TextRight(f *Font, rx, y int, s string, col Color) {
	c.Text(f, rx-f.Measure(s), y, s, col)
}

func (c *Canvas) blitGlyph(f *Font, g Glyph, x, y int, col Color) {
	x0, y0 := clamp(x, c.clip[0], c.clip[2]), clamp(y, c.clip[1], c.clip[3])
	x1, y1 := clamp(x+g.W, c.clip[0], c.clip[2]), clamp(y+g.H, c.clip[1], c.clip[3])
	if x0 >= x1 || y0 >= y1 {
		return
	}
	cr, cg, cb := uint32(col.R()), uint32(col.G()), uint32(col.B())
	baseA := uint32(col.A())
	for yy := y0; yy < y1; yy++ {
		sy := yy - y + g.Y
		for xx := x0; xx < x1; xx++ {
			sx := xx - x + g.X
			a := uint32(f.Img.GrayAt(sx, sy).Y) * baseA / 255
			if a == 0 {
				continue
			}
			i := yy*c.W + xx
			dst := c.Pix[i]
			ia := 256 - a
			r := (cr*a + uint32((dst>>16)&0xFF)*ia) >> 8
			gg := (cg*a + uint32((dst>>8)&0xFF)*ia) >> 8
			b := (cb*a + uint32(dst&0xFF)*ia) >> 8
			c.Pix[i] = 0xFF000000 | r<<16 | gg<<8 | b
		}
	}
}

