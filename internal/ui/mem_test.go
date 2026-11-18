package ui

import (
	"runtime"
	"testing"
	"time"

	"tractor/internal/gfx"
)

func TestMemoryFootprint(t *testing.T) {
	LoadFonts()
	a := New(752, 560)
	a.StartNewGame(3)
	c := gfx.New(752, 560)
	now := time.Unix(1700000000, 0)
	for i := 0; i < 2400; i++ {
		now = now.Add(33 * time.Millisecond)
		a.SetNow(now)
		a.Tick(now)
		a.AutoAct()
		a.Render(c)
	}
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	t.Logf("heap=%d heapSys=%d sys=%d numgc=%d sprites=%d", m.HeapAlloc, m.HeapSys, m.Sys, m.NumGC, len(cardSprites))
}
