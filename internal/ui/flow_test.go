package ui

import (
	"testing"
	"time"

	"tractor/internal/game"
	"tractor/internal/gfx"
)

func TestAutoRoundCompletes(t *testing.T) {
	LoadFonts()
	for seed := int64(0); seed < 8; seed++ {
		a := New(752, 560)
		now := time.Unix(1700000000, 0)
		a.SetNow(now)
		a.StartNewGame(seed)
		for i := 0; i < 6000 && a.Screen != ScreenRoundEnd; i++ {
			now = now.Add(50 * time.Millisecond)
			a.SetNow(now)
			a.Tick(now)
			a.AutoAct()
		}
		if a.Screen != ScreenRoundEnd {
			t.Fatalf("seed %d stuck in phase %v turn %d", seed, a.G.Current.Phase, a.G.Current.Turn)
		}
		if n := len(a.G.Current.Hands[0]); n != 0 {
			t.Fatalf("seed %d hand still has %d cards", seed, n)
		}
		c := gfx.New(752, 560)
		a.Render(c)
	}
}

func TestHumanBid(t *testing.T) {
	LoadFonts()
	var a *App
	var seed int64
	for seed = 0; seed < 200; seed++ {
		a = New(752, 560)
		a.SetNow(time.Unix(1700000000, 0))
		a.StartNewGame(seed)
		if a.G.Current.BidSeat() == 0 && hasLevelCard(a.G.Current.Hands[0], a.G.Current.Level) {
			break
		}
	}
	if seed == 200 {
		t.Fatal("no seed with a human level card")
	}
	for i, c := range a.Hand() {
		if c.Rank == a.G.Current.Level && !c.IsJoker() {
			a.cursor = i
			break
		}
	}
	a.Key(KeyA)
	a.Key(KeyX)
	r := a.G.Current
	if r.Declarer != 0 {
		t.Fatalf("human did not declare, declarer=%d", r.Declarer)
	}
	if r.Trump == game.NoTrump {
		t.Fatal("trump not set")
	}
}

func TestHumanBury(t *testing.T) {
	LoadFonts()
	a := New(752, 560)
	now := time.Unix(1700000000, 0)
	a.SetNow(now)
	a.StartNewGame(0) // dealer is seat 0
	if a.G.Dealer != 0 {
		t.Fatalf("dealer = %d", a.G.Dealer)
	}
	for i := 0; i < 200 && a.G.Current.Phase != game.PhaseBury; i++ {
		now = now.Add(200 * time.Millisecond)
		a.SetNow(now)
		a.Tick(now)
		if a.G.Current.Phase != game.PhaseBury {
			a.AutoAct()
		}
	}
	if a.G.Current.Phase != game.PhaseBury {
		t.Fatalf("phase = %v", a.G.Current.Phase)
	}
	if n := len(a.G.Current.Hands[0]); n != 33 {
		t.Fatalf("dealer hand = %d", n)
	}
	if len(a.handSlots()) != 33 {
		t.Fatalf("bury layout has %d slots", len(a.handSlots()))
	}
	a.selected = map[int]bool{}
	for i, c := range a.Hand()[:8] {
		a.selected[c.ID] = true
		a.cursor = i
	}
	a.confirm()
	if a.G.Current.Phase != game.PhasePlay {
		t.Fatalf("phase after bury = %v (%s)", a.G.Current.Phase, a.errMsg)
	}
	if n := len(a.G.Current.Hands[0]); n != 25 {
		t.Fatalf("hand after bury = %d", n)
	}
}

func TestRenderScreens(t *testing.T) {
	LoadFonts()
	c := gfx.New(752, 560)
	a := New(752, 560)
	a.SetNow(time.Unix(1700000000, 0))
	a.Render(c)
	a.StartNewGame(5)
	a.Render(c)
	a.Screen = ScreenHelp
	a.Render(c)
	a.Screen = ScreenPause
	a.Render(c)
	a.Screen = ScreenTable
	a.finishRound()
	a.Screen = ScreenRoundEnd
	a.Render(c)
	a.G.Winner = 0
	a.Screen = ScreenGameOver
	a.Render(c)
}

func TestHandLayoutFits(t *testing.T) {
	LoadFonts()
	for seed := int64(0); seed < 20; seed++ {
		a := New(752, 560)
		a.SetNow(time.Unix(1700000000, 0))
		a.StartNewGame(seed)
		slots := a.handSlots()
		if len(slots) == 0 {
			t.Fatal("no hand slots")
		}
		for _, s := range slots {
			if s.X < 0 || s.X+s.W > a.W || s.Y < 0 || s.Y+s.H > 532 {
				t.Fatalf("seed %d card out of bounds: %+v", seed, s)
			}
		}
	}
}
