package game

import "testing"

func TestBottomMultiplier(t *testing.T) {
	trump, level := Heart, Rank2
	c := func(s Suit, r Rank, id int) Card { return Card{Suit: s, Rank: r, ID: id} }
	cases := []struct {
		name  string
		cards []Card
		want  int
	}{
		{"single", []Card{c(Spade, RankA, 1)}, 2},
		{"pair", []Card{c(Spade, RankA, 1), c(Spade, RankA, 2)}, 4},
		{"tractor", []Card{c(Spade, RankA, 1), c(Spade, RankA, 2), c(Spade, RankK, 3), c(Spade, RankK, 4)}, 8},
		{"throw singles", []Card{c(Spade, RankA, 1), c(Spade, RankK, 2), c(Spade, RankQ, 3)}, 2},
		{"throw with pair", []Card{c(Spade, RankA, 1), c(Spade, RankA, 2), c(Spade, RankK, 3)}, 4},
		{"throw with tractor", []Card{c(Spade, RankA, 1), c(Spade, RankA, 2), c(Spade, RankK, 3), c(Spade, RankK, 4), c(Spade, RankQ, 5)}, 8},
		{"throw two pairs apart", []Card{c(Spade, RankA, 1), c(Spade, RankA, 2), c(Spade, RankQ, 3), c(Spade, RankQ, 4)}, 4},
		{"three pair tractor", []Card{c(Spade, RankA, 1), c(Spade, RankA, 2), c(Spade, RankK, 3), c(Spade, RankK, 4), c(Spade, RankQ, 5), c(Spade, RankQ, 6)}, 8},
	}
	for _, tc := range cases {
		if got := bottomMultiplier(tc.cards, trump, level); got != tc.want {
			t.Errorf("%s: multiplier = %d, want %d", tc.name, got, tc.want)
		}
	}
}
