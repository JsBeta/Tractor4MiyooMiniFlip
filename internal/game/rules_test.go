package game

import (
	"math/rand"
	"testing"
)

var testCardID int

// newCard returns a card with a unique ID for tests.
func newCard(s Suit, r Rank) Card {
	testCardID++
	return Card{Suit: s, Rank: r, ID: testCardID}
}

func mustClassify(t *testing.T, cards []Card, trump Suit, level Rank) Combo {
	t.Helper()
	c, err := Classify(cards, trump, level)
	if err != nil {
		t.Fatalf("classify %v: %v", cards, err)
	}
	return c
}

func TestDeck(t *testing.T) {
	d := NewDeck()
	if len(d) != 108 {
		t.Fatalf("deck has %d cards", len(d))
	}
	seen := map[int]bool{}
	counts := map[[2]int]int{}
	for _, c := range d {
		if seen[c.ID] {
			t.Fatalf("duplicate id %d", c.ID)
		}
		seen[c.ID] = true
		counts[[2]int{int(c.Suit), int(c.Rank)}]++
	}
	for k, n := range counts {
		if n != 2 {
			t.Fatalf("card %v occurs %d times", k, n)
		}
	}
}

func TestClassify(t *testing.T) {
	// level 5, trump hearts.
	trump, level := Heart, Rank5
	c := newCard
	single := []Card{c(Spade, RankA)}
	if got := mustClassify(t, single, trump, level); got.Type != ComboSingle {
		t.Fatalf("single got %v", got.Type)
	}
	pair := []Card{c(Spade, Rank4), c(Spade, Rank4)}
	if got := mustClassify(t, pair, trump, level); got.Type != ComboPair {
		t.Fatalf("pair got %v", got.Type)
	}
	tractor := []Card{c(Spade, Rank6), c(Spade, Rank6), c(Spade, Rank7), c(Spade, Rank7)}
	if got := mustClassify(t, tractor, trump, level); got.Type != ComboTractor || got.Pairs != 2 {
		t.Fatalf("tractor got %v pairs %d", got.Type, got.Pairs)
	}
	// The level card breaks the sequence: 4-4 + 6-6 is not a tractor.
	gap := []Card{c(Spade, Rank4), c(Spade, Rank4), c(Spade, Rank6), c(Spade, Rank6)}
	if got := mustClassify(t, gap, trump, level); got.Type != ComboThrow {
		t.Fatalf("gap got %v", got.Type)
	}
	// Two different suits cannot be led together.
	if _, err := Classify([]Card{c(Spade, RankA), c(Club, RankA)}, trump, level); err == nil {
		t.Fatal("mixed suits accepted")
	}
}

func TestBeats(t *testing.T) {
	trump, level := Spade, Rank2
	c := newCard
	lead := []Card{c(Heart, RankK)}
	low := []Card{c(Heart, RankA)}
	trumped := []Card{c(Spade, Rank3)}
	otherSuit := []Card{c(Club, RankA)}
	if !beats(low, lead, lead, trump, level) {
		t.Fatal("A should beat K")
	}
	if !beats(trumped, low, lead, trump, level) {
		t.Fatal("trump should beat A")
	}
	if beats(otherSuit, lead, lead, trump, level) {
		t.Fatal("other suit should not beat")
	}
	// A pair lead can only be beaten by a trump pair.
	pairLead := []Card{c(Heart, RankK), c(Heart, RankK)}
	twoTrumps := []Card{c(Spade, Rank3), c(Spade, Rank4)}
	if beats(twoTrumps, pairLead, pairLead, trump, level) {
		t.Fatal("two trumps must not beat a pair")
	}
	trumpPair := []Card{c(Spade, Rank3), c(Spade, Rank3)}
	if !beats(trumpPair, pairLead, pairLead, trump, level) {
		t.Fatal("trump pair should beat pair")
	}
}

func TestThrowValidation(t *testing.T) {
	trump, level := Heart, Rank5
	c := newCard
	// Opponent holds the spade ace, so a spade K throw is bad.
	opp := [][]Card{{{Suit: Spade, Rank: RankA}}}
	if _, bad := throwBeatable([]Card{c(Spade, RankK)}, opp, trump, level); !bad {
		t.Fatal("throw should be beaten by higher single")
	}
	// Highest single is fine.
	if _, bad := throwBeatable([]Card{c(Spade, RankA)}, [][]Card{{{Suit: Spade, Rank: RankK}}}, trump, level); bad {
		t.Fatal("highest single marked beatable")
	}
	// Pair of kings is beaten by pair of aces.
	opp2 := [][]Card{{{Suit: Spade, Rank: RankA}, {Suit: Spade, Rank: RankA}}}
	if _, bad := throwBeatable([]Card{c(Spade, RankK), c(Spade, RankK)}, opp2, trump, level); !bad {
		t.Fatal("pair throw should be beatable")
	}
	// Level card pair beats a side suit ace pair (trump pair).
	opp3 := [][]Card{{{Suit: Club, Rank: Rank5}, {Suit: Club, Rank: Rank5}}}
	if _, bad := throwBeatable([]Card{c(Spade, RankA), c(Spade, RankA)}, opp3, trump, level); !bad {
		t.Fatal("level pair should beat side ace pair")
	}
}

func TestFollowRules(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	r := StartRound(0, Rank2, rng)
	r.Trump = Spade
	r.Level = Rank2
	r.Phase = PhasePlay
	r.Leader = 0
	r.Turn = 1
	r.Plays[0] = []Card{{Suit: Heart, Rank: RankK, ID: 500}, {Suit: Heart, Rank: RankK, ID: 501}}
	r.Best = 0
	// Seat 1 has a heart pair: must play it.
	r.Hands[1] = []Card{
		{Suit: Heart, Rank: Rank4, ID: 1}, {Suit: Heart, Rank: Rank4, ID: 2},
		{Suit: Club, Rank: Rank3, ID: 3}, {Suit: Club, Rank: Rank4, ID: 4},
	}
	if r.validateFollow(1, []Card{{Suit: Heart, Rank: Rank4, ID: 1}, {Suit: Club, Rank: Rank3, ID: 3}}) == nil {
		t.Fatal("must follow with a pair")
	}
	if err := r.validateFollow(1, []Card{{Suit: Heart, Rank: Rank4, ID: 1}, {Suit: Heart, Rank: Rank4, ID: 2}}); err != nil {
		t.Fatalf("valid pair rejected: %v", err)
	}
	// Seat 1 has no hearts: must play both remaining clubs? It has 4 cards,
	// hand 2 trumps and 2 clubs; all hearts gone, so anything goes.
	r.Hands[2] = []Card{{Suit: Club, Rank: RankA, ID: 10}, {Suit: Club, Rank: RankQ, ID: 11}}
	if err := r.validateFollow(2, []Card{{Suit: Club, Rank: RankA, ID: 10}, {Suit: Club, Rank: RankQ, ID: 11}}); err != nil {
		t.Fatalf("void follow rejected: %v", err)
	}
}

func TestRoundSimulation(t *testing.T) {
	for seed := int64(1); seed <= 6; seed++ {
		g := NewGame(seed)
		for round := 0; round < 3 && g.Winner < 0; round++ {
			g.StartRound()
			r := g.Current
			for r.Phase == PhaseBidding {
				seat := r.BidSeat()
				if cards, ok := AIBid(r, seat); ok {
					if err := r.Bid(seat, cards); err != nil {
						t.Fatalf("seed %d bid: %v", seed, err)
					}
				} else if err := r.PassBid(seat); err != nil {
					t.Fatalf("seed %d pass: %v", seed, err)
				}
			}
			if r.Phase == PhaseBury {
				if err := AIAutoBury(r); err != nil {
					t.Fatalf("seed %d bury: %v", seed, err)
				}
			}
			for r.Phase == PhasePlay {
				seat := r.Turn
				cards, err := AIChoose(r, seat)
				if err != nil {
					t.Fatalf("seed %d seat %d choose: %v", seed, seat, err)
				}
				if err := r.Play(seat, cards); err != nil {
					t.Fatalf("seed %d seat %d play %v: %v", seed, seat, cards, err)
				}
			}
			if r.Phase != PhaseDone {
				t.Fatalf("seed %d round not done", seed)
			}
			for i := 0; i < 4; i++ {
				if len(r.Hands[i]) != 0 {
					t.Fatalf("seed %d seat %d has %d cards left", seed, i, len(r.Hands[i]))
				}
			}
			total := r.DefenderPoints + r.DealerPoints
			if r.BottomPoints == 0 {
				total += PointSum(r.Bottom)
			}
			if total != 200 {
				t.Fatalf("seed %d points %d (def %d dealer %d bottom %d) != 200", seed, total, r.DefenderPoints, r.DealerPoints, r.BottomPoints)
			}
			res := g.FinishRound()
			if res.Success != (res.DefenderPoints < 80) {
				t.Fatalf("seed %d success mismatch", seed)
			}
			if g.Winner >= 0 {
				break
			}
		}
	}
}
