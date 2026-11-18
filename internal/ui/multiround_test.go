package ui

import (
	"testing"
	"time"
)

func TestMultiRoundFlow(t *testing.T) {
	LoadFonts()
	a := New(752, 560)
	now := time.Unix(1700000000, 0)
	a.SetNow(now)
	a.StartNewGame(7)
	for round := 0; round < 5; round++ {
		for i := 0; i < 6000 && a.Screen != ScreenRoundEnd; i++ {
			now = now.Add(50 * time.Millisecond)
			a.SetNow(now)
			a.Tick(now)
			a.AutoAct()
		}
		if a.Screen != ScreenRoundEnd {
			t.Fatalf("round %d stuck phase %v turn %d", round, a.G.Current.Phase, a.G.Current.Turn)
		}
		if a.G.RoundNo < round+2 {
			t.Fatalf("round number %d after round %d", a.G.RoundNo, round)
		}
		a.Key(KeyX)
		if a.Screen == ScreenGameOver {
			break
		}
		if a.Screen != ScreenTable {
			t.Fatalf("after continue screen=%v", a.Screen)
		}
	}
}
