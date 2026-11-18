package gfx

import "testing"

func TestCircleFillsCentre(t *testing.T) {
	c := New(24, 24)
	red := RGB(0xFF, 0x00, 0x00)
	c.Circle(12, 12, 7, red)
	if got := c.Pix[12*24+12]; got != uint32(red) {
		t.Fatalf("centre not filled: %08x want %08x", got, uint32(red))
	}
	if got := c.Pix[12*24+2]; got != 0 {
		t.Fatalf("outside disc painted: %08x", got)
	}
	c2 := New(24, 24)
	c2.StrokeCircle(12, 12, 7, 1, red)
	if got := c2.Pix[12*24+12]; got != 0 {
		t.Fatalf("stroke circle filled the centre: %08x", got)
	}
	if got := c2.Pix[12*24+19]; got == 0 {
		t.Fatalf("stroke circle did not paint the rim")
	}
}
