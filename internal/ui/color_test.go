package ui

import (
	"testing"
	"time"

	"tractor/internal/game"
	"tractor/internal/gfx"
)

func TestSuitColorOnDark(t *testing.T) {
	for _, s := range []game.Suit{game.Spade, game.Club} {
		if got := suitColorOnDark(s); got != ColCream {
			t.Fatalf("suitColorOnDark(%v) = %v, want cream", s, got)
		}
	}
	for _, s := range []game.Suit{game.Heart, game.Diamond} {
		if got := suitColorOnDark(s); got != ColRed {
			t.Fatalf("suitColorOnDark(%v) = %v, want red", s, got)
		}
	}
}

func TestHeaderBlackTrumpVisible(t *testing.T) {
	LoadFonts()
	a := New(752, 560)
	a.SetNow(time.Now())
	a.StartNewGame(1)
	a.G.Current.Trump = game.Spade
	c := gfx.New(752, 560)
	a.drawHeader(c)
	cream := 0
	for y := 9; y < 37; y++ {
		for x := 356; x < 384; x++ {
			p := c.Pix[y*c.W+x]
			r := (p >> 16) & 0xFF
			g := (p >> 8) & 0xFF
			b := p & 0xFF
			if r > 200 && g > 190 && b > 170 {
				cream++
			}
		}
	}
	if cream < 20 {
		t.Fatalf("spade trump icon not visible on dark header: cream pixels=%d", cream)
	}
}
