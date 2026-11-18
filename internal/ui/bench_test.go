package ui

import (
	"testing"
	"time"

	"tractor/internal/gfx"
)

func BenchmarkRenderTable(b *testing.B) {
	LoadFonts()
	a := New(752, 560)
	a.SetNow(time.Unix(1700000000, 0))
	a.StartNewGame(3)
	c := gfx.New(752, 560)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Render(c)
	}
}
