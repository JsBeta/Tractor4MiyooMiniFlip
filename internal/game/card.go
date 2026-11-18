// Package game implements the rules of Tractor (拖拉机 / 升级), a four player
// partnership climbing game played with two full decks.
package game

import (
	"fmt"
	"math/rand"
	"sort"
)

// Suit is a card suit. Jokers use SuitJoker.
type Suit int8

const (
	Spade Suit = iota
	Heart
	Club
	Diamond
	SuitJoker
)

// NoTrump is the trump setting used when nobody declares a suit.
const NoTrump Suit = -1

// GroupTrump is the effective "suit" of every trump card when comparing plays.
const GroupTrump = 4

// Rank: 0..12 are 2,3,...,A. 13 is the small joker and 14 the big joker.
type Rank int8

const (
	Rank2 Rank = iota
	Rank3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
	Rank9
	Rank10
	RankJ
	RankQ
	RankK
	RankA
	RankSmallJoker
	RankBigJoker
)

var rankNames = [...]string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A", "小王", "大王"}

func (r Rank) String() string {
	if int(r) >= 0 && int(r) < len(rankNames) {
		return rankNames[r]
	}
	return "?"
}

// IsJoker reports whether the rank is one of the jokers.
func (r Rank) IsJoker() bool { return r >= RankSmallJoker }

// Card is a single physical card. ID is unique in the 108 card deck and is what
// the UI uses to keep track of selections.
type Card struct {
	Suit Suit
	Rank Rank
	ID   int
}

func (c Card) IsJoker() bool { return c.Suit == SuitJoker }

func (c Card) String() string {
	if c.IsJoker() {
		return c.Rank.String()
	}
	return fmt.Sprintf("%s%s", suitSymbols[c.Suit], c.Rank)
}

var suitSymbols = [...]string{"♠", "♥", "♣", "♦", ""}

func SuitSymbol(s Suit) string {
	if int(s) >= 0 && int(s) < len(suitSymbols) {
		return suitSymbols[s]
	}
	return ""
}

// SuitName returns the Chinese name of a suit.
func SuitName(s Suit) string {
	switch s {
	case Spade:
		return "黑桃"
	case Heart:
		return "红桃"
	case Club:
		return "梅花"
	case Diamond:
		return "方块"
	}
	return "无主"
}

// PointValue is the score value of a card: 5 -> 5, 10 -> 10, K -> 10.
func (c Card) PointValue() int {
	switch c.Rank {
	case Rank5:
		return 5
	case Rank10, RankK:
		return 10
	}
	return 0
}

// NewDeck returns the two-deck 108 card set.
func NewDeck() []Card {
	cards := make([]Card, 0, 108)
	id := 0
	for d := 0; d < 2; d++ {
		for s := Spade; s <= Diamond; s++ {
			for r := Rank2; r <= RankA; r++ {
				cards = append(cards, Card{Suit: s, Rank: r, ID: id})
				id++
			}
		}
		cards = append(cards, Card{Suit: SuitJoker, Rank: RankSmallJoker, ID: id})
		id++
		cards = append(cards, Card{Suit: SuitJoker, Rank: RankBigJoker, ID: id})
		id++
	}
	return cards
}

// Deal shuffles a fresh deck, deals 25 cards to each seat and returns the 8
// bottom cards. Seat order is 0=South(human), 1=East, 2=North, 3=West.
func Deal(rng *rand.Rand) (hands [4][]Card, bottom []Card) {
	deck := NewDeck()
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	for i := 0; i < 100; i++ {
		hands[i%4] = append(hands[i%4], deck[i])
	}
	bottom = append(bottom, deck[100:]...)
	return hands, bottom
}

// IsTrump reports whether c belongs to the trump group for this round.
func IsTrump(c Card, trump Suit, level Rank) bool {
	if c.IsJoker() {
		return true
	}
	if c.Rank == level {
		return true
	}
	return trump != NoTrump && c.Suit == trump
}

// Group returns the comparison group of a card: the trump group or its suit.
func Group(c Card, trump Suit, level Rank) int {
	if IsTrump(c, trump, level) {
		return GroupTrump
	}
	return int(c.Suit)
}

// Order returns the strength of a card inside its own group. Higher wins.
func Order(c Card, trump Suit, level Rank) int {
	switch {
	case c.Rank == RankBigJoker:
		return 1000
	case c.Rank == RankSmallJoker:
		return 999
	case c.Rank == level && c.Suit == trump:
		return 998
	case c.Rank == level:
		return 997
	case trump != NoTrump && c.Suit == trump:
		return 100 + int(c.Rank)
	default:
		return int(c.Rank)
	}
}

// SortHand sorts a hand the way the table usually shows it: trumps first, then
// black/red suits, each group descending by rank. Jokers at the front.
func SortHand(hand []Card, trump Suit, level Rank) {
	suitRank := func(s Suit) int {
		switch s {
		case Spade:
			return 0
		case Heart:
			return 1
		case Club:
			return 2
		case Diamond:
			return 3
		}
		return 4
	}
	sort.SliceStable(hand, func(i, j int) bool {
		a, b := hand[i], hand[j]
		ga, gb := Group(a, trump, level), Group(b, trump, level)
		if ga != gb {
			if ga == GroupTrump {
				return true
			}
			if gb == GroupTrump {
				return false
			}
			return suitRank(a.Suit) < suitRank(b.Suit)
		}
		oa, ob := Order(a, trump, level), Order(b, trump, level)
		if oa != ob {
			return oa > ob
		}
		return a.ID < b.ID
	})
}

// GroupCards returns the cards of a hand that belong to the given group.
func GroupCards(hand []Card, group int, trump Suit, level Rank) []Card {
	out := make([]Card, 0, len(hand))
	for _, c := range hand {
		if Group(c, trump, level) == group {
			out = append(out, c)
		}
	}
	return out
}

// HasPair reports whether the hand contains an identical pair of c.
func HasPair(hand []Card, c Card) bool {
	n := 0
	for _, h := range hand {
		if h.Suit == c.Suit && h.Rank == c.Rank {
			n++
		}
	}
	return n >= 2
}

// PointSum returns the score value of a set of cards.
func PointSum(cards []Card) int {
	s := 0
	for _, c := range cards {
		s += c.PointValue()
	}
	return s
}
