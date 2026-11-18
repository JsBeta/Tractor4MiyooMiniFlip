package game

import (
	"encoding/json"
	"testing"
)

func TestGameStateBiddingRoundTrip(t *testing.T) {
	g := NewGame(11)
	g.StartRound()
	r := g.Current
	r.Trump = Heart
	r.Declarer = 1
	r.BidStrength = 2
	r.bidStep = 1
	r.Turn = 2

	data, err := json.Marshal(g.State())
	if err != nil {
		t.Fatal(err)
	}
	var st GameState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	var got Game
	if err := got.Restore(st); err != nil {
		t.Fatal(err)
	}
	if got.Current == nil || got.Current.Phase != PhaseBidding {
		t.Fatalf("phase = %v", got.Current)
	}
	if got.Levels != g.Levels || got.Dealer != g.Dealer || got.RoundNo != g.RoundNo {
		t.Fatalf("match fields not restored: %+v", got)
	}
	if got.Current.Trump != Heart || got.Current.Declarer != 1 || got.Current.BidStrength != 2 || got.Current.bidStep != 1 || got.Current.Turn != 2 {
		t.Fatalf("round fields not restored: %+v", got.Current)
	}
	for i := 0; i < 4; i++ {
		if len(got.Current.Hands[i]) != len(r.Hands[i]) {
			t.Fatalf("hand %d len %d != %d", i, len(got.Current.Hands[i]), len(r.Hands[i]))
		}
	}
	if len(got.Current.Bottom) != 8 {
		t.Fatalf("bottom len %d", len(got.Current.Bottom))
	}
}

func TestGameStatePlayRoundTrip(t *testing.T) {
	g := NewGame(12)
	g.StartRound()
	r := g.Current
	// Move from bidding to burying and play one complete trick.
	r.Hands[r.Dealer] = append(r.Hands[r.Dealer], r.Bottom...)
	r.Bottom = nil
	r.Phase = PhaseBury
	r.BuryLeft = 8
	r.Trump = Spade
	r.Declarer = r.Dealer
	if err := r.Bury(r.Dealer, append([]Card(nil), r.Hands[r.Dealer][:8]...)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		cards, err := AIChoose(r, r.Turn)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Play(r.Turn, cards); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.History) != 1 {
		t.Fatalf("history %d", len(r.History))
	}

	data, err := json.Marshal(g.State())
	if err != nil {
		t.Fatal(err)
	}
	var st GameState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	var got Game
	if err := got.Restore(st); err != nil {
		t.Fatal(err)
	}
	gr := got.Current
	if gr == nil || gr.Phase != PhasePlay || gr.TrickNo != r.TrickNo || gr.Turn != r.Turn {
		t.Fatalf("round not restored: %+v", gr)
	}
	if len(gr.History) != 1 || len(gr.Bottom) != 8 {
		t.Fatalf("history %d bottom %d", len(gr.History), len(gr.Bottom))
	}
	if len(gr.Hands[0]) != len(r.Hands[0]) || gr.DefenderPoints != r.DefenderPoints {
		t.Fatalf("hand/points not restored")
	}
}

func TestGameStateDoneRoundTrip(t *testing.T) {
	g := NewGame(14)
	g.StartRound()
	r := g.Current
	r.Hands[r.Dealer] = append(r.Hands[r.Dealer], r.Bottom...)
	r.Bottom = nil
	r.Phase = PhaseBury
	r.BuryLeft = 8
	r.Trump = Club
	r.Declarer = r.Dealer
	if err := r.Bury(r.Dealer, append([]Card(nil), r.Hands[r.Dealer][:8]...)); err != nil {
		t.Fatal(err)
	}
	for r.Phase != PhaseDone {
		cards, err := AIChoose(r, r.Turn)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Play(r.Turn, cards); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.History) < 20 {
		t.Fatalf("history %d", len(r.History))
	}
	data, err := json.Marshal(g.State())
	if err != nil {
		t.Fatal(err)
	}
	var st GameState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	var got Game
	if err := got.Restore(st); err != nil {
		t.Fatal(err)
	}
	if got.Current.Phase != PhaseDone || len(got.Current.History) != len(r.History) {
		t.Fatalf("round not restored: %+v", got.Current)
	}
}

func TestRestoreRejectsBadState(t *testing.T) {
	g := NewGame(13)
	g.StartRound()
	st := g.State()
	st.Version = 99
	var got Game
	if err := got.Restore(st); err == nil {
		t.Fatal("accepted bad version")
	}
	st = g.State()
	st.Current.Hands[1] = append(st.Current.Hands[1], st.Current.Hands[0][0])
	if err := got.Restore(st); err == nil {
		t.Fatal("accepted duplicate card")
	}
}
