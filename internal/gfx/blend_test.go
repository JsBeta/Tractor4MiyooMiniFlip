package gfx

import "testing"

func TestBlendAlphaOnTransparent(t *testing.T) {
	got := blend(0, 0x1A000000)
	if got != 0x1A000000 {
		t.Fatalf("translucent over transparent = %08x", got)
	}
	got = blend(0x80000000, 0x80000000)
	if a := got >> 24; a <= 0x80 || a == 0xFF {
		t.Fatalf("accumulated alpha = %02x", a)
	}
}

func TestSpriteShadowKeepsAlpha(t *testing.T) {
	c := New(80, 80)
	c.Shadow(20, 20, 40, 40, 8)
	if a := c.Pix[10*80+10] >> 24; a != 0 {
		t.Fatalf("far pixel alpha = %d", a)
	}
	found := false
	for _, p := range c.Pix {
		a := p >> 24
		if a > 0 && a < 255 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("shadow produced no translucent pixels")
	}
}
