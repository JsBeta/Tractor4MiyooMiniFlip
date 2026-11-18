package ui

import (
	"testing"
	"time"
)

func TestDemoMode(t *testing.T) {
	LoadFonts()
	a := New(752, 560)
	a.Demo = true
	a.StartNewGame(9)
	now := time.Unix(1700000000, 0)
	for i := 0; i < 4000; i++ {
		now = now.Add(50 * time.Millisecond)
		a.SetNow(now)
		a.Tick(now)
	}
	if a.G.RoundNo < 2 {
		t.Fatalf("demo did not finish a round: round=%d screen=%v", a.G.RoundNo, a.Screen)
	}
}
