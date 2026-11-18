package ui

import (
	"testing"

	"tractor/internal/game"
	"tractor/internal/gfx"
)

func TestFontAtlas(t *testing.T) {
	for _, name := range []string{"xs", "sm", "md", "lg", "xl"} {
		f := gfx.LoadFont(name)
		if len(f.Glyphs) < 100 {
			t.Fatalf("%s only has %d glyphs", name, len(f.Glyphs))
		}
		if f.Line <= 0 || f.Size <= 0 {
			t.Fatalf("%s bad metrics %+v", name, f)
		}
		b := f.Img.Bounds()
		for r, g := range f.Glyphs {
			if g.W <= 0 || g.H <= 0 {
				continue
			}
			if g.X < 0 || g.Y < 0 || g.X+g.W > b.Dx() || g.Y+g.H > b.Dy() {
				t.Fatalf("%s glyph %q out of atlas: %+v", name, r, g)
			}
		}
		for _, r := range "拖拉机升级双扣东南西北我方闲家底牌亮主扣底" {
			if _, ok := f.Glyphs[r]; !ok {
				t.Errorf("%s missing glyph %q", name, r)
			}
		}
	}
}

func TestTextRenders(t *testing.T) {
	f := gfx.LoadFont("md")
	c := gfx.New(400, 80)
	c.Clear(gfx.RGB(0, 0, 0))
	n := c.Text(f, 8, 8, "拖拉机 10 K 主牌", gfx.RGB(255, 255, 255))
	if n <= 0 {
		t.Fatal("no advance")
	}
	bright := 0
	for _, p := range c.Pix {
		if p&0xFF > 64 {
			bright++
		}
	}
	if bright < 50 {
		t.Fatalf("text drew only %d bright pixels", bright)
	}
	if f.Measure("拖拉机") != 3*f.Measure("拖") {
		t.Fatalf("CJK advance not uniform: %d", f.Measure("拖拉机"))
	}
}

func TestCardRendering(t *testing.T) {
	c := gfx.New(100, 140)
	c.Clear(gfx.RGB(0, 0, 0))
	DrawCard(c, 5, 5, 74, 102, game.Card{Suit: game.Spade, Rank: game.RankA}, game.Spade, game.Rank2, CardStyle{})
	white := 0
	for _, p := range c.Pix {
		if p == 0xFFFBF7EB {
			white++
		}
	}
	if white < 3000 {
		t.Fatalf("card body only %d white pixels", white)
	}
}
