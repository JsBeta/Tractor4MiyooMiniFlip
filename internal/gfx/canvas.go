// Package gfx is a small software renderer: enough shapes, text and card
// drawing to build a crisp 752x560 interface without any external dependency.
package gfx

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Color is 0xAARRGGBB.
type Color uint32

func RGB(r, g, b uint8) Color { return Color(0xFF000000 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)) }

func RGBA(r, g, b, a uint8) Color {
	return Color(uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

func (c Color) A() uint8 { return uint8(c >> 24) }
func (c Color) R() uint8 { return uint8(c >> 16) }
func (c Color) G() uint8 { return uint8(c >> 8) }
func (c Color) B() uint8 { return uint8(c) }

func (c Color) WithAlpha(a uint8) Color {
	return Color(uint32(a)<<24) | (c & 0x00FFFFFF)
}

func (c Color) Scale(f float64) Color {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	return RGBA(uint8(float64(c.R())*f), uint8(float64(c.G())*f), uint8(float64(c.B())*f), c.A())
}

// Canvas is an ARGB image.
type Canvas struct {
	W, H int
	Pix  []uint32
	clip [4]int
}

func New(w, h int) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]uint32, w*h)}
	c.clip = [4]int{0, 0, w, h}
	return c
}

func (c *Canvas) ResetClip() {
	c.clip = [4]int{0, 0, c.W, c.H}
}

func (c *Canvas) SetClip(x, y, w, h int) {
	c.clip[0] = clamp(x, 0, c.W)
	c.clip[1] = clamp(y, 0, c.H)
	c.clip[2] = clamp(x+w, 0, c.W)
	c.clip[3] = clamp(y+h, 0, c.H)
	if c.clip[2] < c.clip[0] {
		c.clip[2] = c.clip[0]
	}
	if c.clip[3] < c.clip[1] {
		c.clip[3] = c.clip[1]
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (c *Canvas) Clear(col Color) {
	for i := range c.Pix {
		c.Pix[i] = uint32(col)
	}
}

// Point is a float coordinate used by the vector helpers.
type Point struct{ X, Y float64 }

func blend(dst, src uint32) uint32 {
	a := src >> 24
	if a == 0 {
		return dst
	}
	if a == 255 {
		return src
	}
	da := dst >> 24
	if da == 0 {
		return src
	}
	if da == 255 {
		ia := 256 - a
		r := (uint32((src>>16)&0xFF)*a + uint32((dst>>16)&0xFF)*ia) >> 8
		g := (uint32((src>>8)&0xFF)*a + uint32((dst>>8)&0xFF)*ia) >> 8
		b := (uint32(src&0xFF)*a + uint32(dst&0xFF)*ia) >> 8
		return 0xFF000000 | r<<16 | g<<8 | b
	}
	// General source-over for translucent destinations (sprite building).
	outA := a + da*(255-a)/255
	if outA == 0 {
		return 0
	}
	sr, sg, sb := uint32((src>>16)&0xFF), uint32((src>>8)&0xFF), uint32(src&0xFF)
	dr, dg, db := uint32((dst>>16)&0xFF), uint32((dst>>8)&0xFF), uint32(dst&0xFF)
	w := da * (255 - a) / 255
	r := (sr*a + dr*w) / outA
	g := (sg*a + dg*w) / outA
	b := (sb*a + db*w) / outA
	return outA<<24 | r<<16 | g<<8 | b
}

func (c *Canvas) set(x, y int, col Color) {
	if x < c.clip[0] || y < c.clip[1] || x >= c.clip[2] || y >= c.clip[3] {
		return
	}
	c.Pix[y*c.W+x] = blend(c.Pix[y*c.W+x], uint32(col))
}

func (c *Canvas) FillRect(x, y, w, h int, col Color) {
	if w <= 0 || h <= 0 {
		return
	}
	x0, y0 := clamp(x, c.clip[0], c.clip[2]), clamp(y, c.clip[1], c.clip[3])
	x1, y1 := clamp(x+w, c.clip[0], c.clip[2]), clamp(y+h, c.clip[1], c.clip[3])
	if x0 >= x1 || y0 >= y1 {
		return
	}
	if col.A() == 255 {
		row := make([]uint32, x1-x0)
		for i := range row {
			row[i] = uint32(col)
		}
		for yy := y0; yy < y1; yy++ {
			copy(c.Pix[yy*c.W+x0:yy*c.W+x1], row)
		}
		return
	}
	for yy := y0; yy < y1; yy++ {
		i := yy*c.W + x0
		for xx := x0; xx < x1; xx++ {
			c.Pix[i] = blend(c.Pix[i], uint32(col))
			i++
		}
	}
}

func (c *Canvas) VGradient(x, y, w, h int, top, bottom Color) {
	if h <= 0 || w <= 0 {
		return
	}
	for yy := 0; yy < h; yy++ {
		t := float64(yy) / float64(h-1)
		col := RGBA(
			uint8(float64(top.R())*(1-t)+float64(bottom.R())*t),
			uint8(float64(top.G())*(1-t)+float64(bottom.G())*t),
			uint8(float64(top.B())*(1-t)+float64(bottom.B())*t),
			uint8(float64(top.A())*(1-t)+float64(bottom.A())*t),
		)
		c.FillRect(x, y+yy, w, 1, col)
	}
}

func (c *Canvas) HLine(x, y, w int, col Color) { c.FillRect(x, y, w, 1, col) }
func (c *Canvas) VLine(x, y, h int, col Color) { c.FillRect(x, y, 1, h, col) }

// RoundRect fills an anti aliased rounded rectangle.
func (c *Canvas) RoundRect(x, y, w, h, r int, col Color) {
	if w <= 0 || h <= 0 {
		return
	}
	if r*2 > w {
		r = w / 2
	}
	if r*2 > h {
		r = h / 2
	}
	if r <= 0 {
		c.FillRect(x, y, w, h, col)
		return
	}
	// Straight parts are solid; only the four corner squares need coverage.
	c.FillRect(x+r, y, w-2*r, h, col)
	c.FillRect(x, y+r, r, h-2*r, col)
	c.FillRect(x+w-r, y+r, r, h-2*r, col)
	a := float64(col.A()) / 255
	corner := func(cx, cy, px0, py0 int) {
		for py := py0; py < py0+r; py++ {
			dy := float64(py) + 0.5 - float64(cy)
			for px := px0; px < px0+r; px++ {
				dx := float64(px) + 0.5 - float64(cx)
				d := math.Sqrt(dx*dx+dy*dy) - float64(r)
				cov := 0.5 - d
				if cov <= 0 {
					continue
				}
				if cov > 1 {
					cov = 1
				}
				alpha := uint8(cov * a * 255)
				if alpha == 0 {
					continue
				}
				if px < c.clip[0] || py < c.clip[1] || px >= c.clip[2] || py >= c.clip[3] {
					continue
				}
				i := py*c.W + px
				c.Pix[i] = blend(c.Pix[i], uint32(col.WithAlpha(alpha)))
			}
		}
	}
	corner(x+r, y+r, x, y)
	corner(x+w-r, y+r, x+w-r, y)
	corner(x+r, y+h-r, x, y+h-r)
	corner(x+w-r, y+h-r, x+w-r, y+h-r)
}

// StrokeRoundRect draws a rounded rectangle outline of the given thickness.
func (c *Canvas) StrokeRoundRect(x, y, w, h, r, thick int, col Color) {
	if w <= 0 || h <= 0 {
		return
	}
	if r*2 > w {
		r = w / 2
	}
	if r*2 > h {
		r = h / 2
	}
	if thick < 1 {
		thick = 1
	}
	if r <= 0 {
		c.FillRect(x, y, w, thick, col)
		c.FillRect(x, y+h-thick, w, thick, col)
		c.FillRect(x, y, thick, h, col)
		c.FillRect(x+w-thick, y, thick, h, col)
		return
	}
	c.FillRect(x+r, y, w-2*r, thick, col)
	c.FillRect(x+r, y+h-thick, w-2*r, thick, col)
	c.FillRect(x, y+r, thick, h-2*r, col)
	c.FillRect(x+w-thick, y+r, thick, h-2*r, col)
	a := float64(col.A()) / 255
	corner := func(cx, cy, px0, py0 int) {
		for py := py0; py < py0+r; py++ {
			dy := float64(py) + 0.5 - float64(cy)
			for px := px0; px < px0+r; px++ {
				dx := float64(px) + 0.5 - float64(cx)
				d := math.Sqrt(dx*dx+dy*dy) - float64(r)
				cov := 0.5 - math.Abs(d) + float64(thick)/2
				if cov <= 0 {
					continue
				}
				if cov > 1 {
					cov = 1
				}
				alpha := uint8(cov * a * 255)
				if alpha == 0 {
					continue
				}
				if px < c.clip[0] || py < c.clip[1] || px >= c.clip[2] || py >= c.clip[3] {
					continue
				}
				i := py*c.W + px
				c.Pix[i] = blend(c.Pix[i], uint32(col.WithAlpha(alpha)))
			}
		}
	}
	corner(x+r, y+r, x, y)
	corner(x+w-r, y+r, x+w-r, y)
	corner(x+r, y+h-r, x, y+h-r)
	corner(x+w-r, y+h-r, x+w-r, y+h-r)
}

func (c *Canvas) Circle(cx, cy, r int, col Color) {
	if r <= 0 {
		return
	}
	c.strokeCircle(cx, cy, r, 0, col)
}

func (c *Canvas) StrokeCircle(cx, cy, r, thick int, col Color) {
	if r <= 0 {
		return
	}
	c.strokeCircle(cx, cy, r, thick, col)
}

func (c *Canvas) SetPixel(x, y int, col Color) {
	if x < c.clip[0] || y < c.clip[1] || x >= c.clip[2] || y >= c.clip[3] {
		return
	}
	c.Pix[y*c.W+x] = blend(c.Pix[y*c.W+x], uint32(col))
}

func (c *Canvas) strokeCircle(cx, cy, r, thick int, col Color) {
	a := float64(col.A()) / 255
	px0, py0 := clamp(cx-r-1, c.clip[0], c.clip[2]), clamp(cy-r-1, c.clip[1], c.clip[3])
	px1, py1 := clamp(cx+r+2, c.clip[0], c.clip[2]), clamp(cy+r+2, c.clip[1], c.clip[3])
	fr := float64(r)
	fcx, fcy := float64(cx), float64(cy)
	for py := py0; py < py1; py++ {
		dy := float64(py) + 0.5 - fcy
		for px := px0; px < px1; px++ {
			dx := float64(px) + 0.5 - fcx
			d := math.Hypot(dx, dy) - fr
			var cov float64
			if thick <= 0 {
				// Filled disc: coverage grows towards the centre.
				cov = 0.5 - d
			} else {
				// Ring of the requested thickness.
				cov = 0.5 - math.Abs(d) + float64(thick)/2
			}
			if cov <= 0 {
				continue
			}
			if cov > 1 {
				cov = 1
			}
			alpha := uint8(cov * a * 255)
			if alpha == 0 {
				continue
			}
			i := py*c.W + px
			c.Pix[i] = blend(c.Pix[i], uint32(col.WithAlpha(alpha)))
		}
	}
}

// Polyg fills a polygon with anti aliasing (2x2 supersampling).
func (c *Canvas) Polyg(pts []Point, col Color) {
	if len(pts) < 3 {
		return
	}
	minX, minY, maxX, maxY := pts[0].X, pts[0].Y, pts[0].X, pts[0].Y
	for _, p := range pts {
		minX, minY = math.Min(minX, p.X), math.Min(minY, p.Y)
		maxX, maxY = math.Max(maxX, p.X), math.Max(maxY, p.Y)
	}
	x0, y0 := clamp(int(math.Floor(minX)), c.clip[0], c.clip[2]), clamp(int(math.Floor(minY)), c.clip[1], c.clip[3])
	x1, y1 := clamp(int(math.Ceil(maxX))+1, c.clip[0], c.clip[2]), clamp(int(math.Ceil(maxY))+1, c.clip[1], c.clip[3])
	a := float64(col.A()) / 255
	const ss = 2
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			inside := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := float64(x) + (float64(sx)+0.5)/ss
					py := float64(y) + (float64(sy)+0.5)/ss
					if pointInPoly(px, py, pts) {
						inside++
					}
				}
			}
			if inside == 0 {
				continue
			}
			cov := float64(inside) / float64(ss*ss)
			alpha := uint8(cov * a * 255)
			if alpha == 0 {
				continue
			}
			i := y*c.W + x
			c.Pix[i] = blend(c.Pix[i], uint32(col.WithAlpha(alpha)))
		}
	}
}

func pointInPoly(x, y float64, pts []Point) bool {
	inside := false
	j := len(pts) - 1
	for i := 0; i < len(pts); i++ {
		yi, yj := pts[i].Y, pts[j].Y
		if (yi > y) != (yj > y) {
			xi, xj := pts[i].X, pts[j].X
			if x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

// Shadow draws a soft rectangular shadow behind a card like element.
func (c *Canvas) Shadow(x, y, w, h, r int) {
	for i := 0; i < 3; i++ {
		c.RoundRect(x-i, y-i+2, w+i*2, h+i*2, r+i, RGBA(0, 0, 0, uint8(26-i*7)))
	}
}

// Blit copies src onto c at (x,y), honouring the alpha channel.
func (c *Canvas) Blit(src *Canvas, x, y int) {
	for sy := 0; sy < src.H; sy++ {
		dy := y + sy
		if dy < c.clip[1] || dy >= c.clip[3] {
			continue
		}
		row := src.Pix[sy*src.W : (sy+1)*src.W]
		for sx := 0; sx < src.W; sx++ {
			v := row[sx]
			a := v >> 24
			if a == 0 {
				continue
			}
			dx := x + sx
			if dx < c.clip[0] || dx >= c.clip[2] {
				continue
			}
			i := dy*c.W + dx
			if a == 255 {
				c.Pix[i] = v
			} else {
				c.Pix[i] = blend(c.Pix[i], v)
			}
		}
	}
}

// PNG writes the canvas as an image.
func (c *Canvas) PNG(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			v := c.Pix[y*c.W+x]
			img.SetRGBA(x, y, color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), uint8(v >> 24)})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
