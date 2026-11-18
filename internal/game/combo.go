package game

import "fmt"

// ComboType classifies a single lead.
type ComboType int

const (
	ComboSingle ComboType = iota
	ComboPair
	ComboTractor
	ComboThrow
)

func (t ComboType) String() string {
	switch t {
	case ComboSingle:
		return "单张"
	case ComboPair:
		return "对子"
	case ComboTractor:
		return "拖拉机"
	}
	return "甩牌"
}

// Combo is the analysed form of a set of cards played together.
type Combo struct {
	Type  ComboType
	Group int // GroupTrump or a suit
	Cards []Card
	Pairs int // number of pair units
	Max   int // highest Order value in the set
}

// Classify analyses a lead. It returns an error when the cards do not form a
// legal same-suit lead at all.
func Classify(cards []Card, trump Suit, level Rank) (Combo, error) {
	if len(cards) == 0 {
		return Combo{}, fmt.Errorf("没有出牌")
	}
	group := Group(cards[0], trump, level)
	seen := make(map[int]bool, len(cards))
	for _, c := range cards {
		if seen[c.ID] {
			return Combo{}, fmt.Errorf("重复的牌")
		}
		seen[c.ID] = true
		if Group(c, trump, level) != group {
			return Combo{}, fmt.Errorf("必须先出同一花色")
		}
	}
	max := 0
	for _, c := range cards {
		if o := Order(c, trump, level); o > max {
			max = o
		}
	}
	combo := Combo{Group: group, Cards: cards, Max: max}
	if len(cards) == 1 {
		combo.Type = ComboSingle
		return combo, nil
	}
	if len(cards) == 2 && isPair(cards) {
		combo.Type = ComboPair
		combo.Pairs = 1
		return combo, nil
	}
	combo.Pairs = pairUnits(cards)
	if isTractor(cards, level) {
		combo.Type = ComboTractor
		return combo, nil
	}
	combo.Type = ComboThrow
	return combo, nil
}

func isPair(cards []Card) bool {
	return len(cards) == 2 && cards[0].Suit == cards[1].Suit && cards[0].Rank == cards[1].Rank
}

// pairUnits counts how many distinct ranks occur twice (or more).
func pairUnits(cards []Card) int {
	counts := make(map[[2]int]int)
	for _, c := range cards {
		counts[[2]int{int(c.Suit), int(c.Rank)}]++
	}
	n := 0
	for _, v := range counts {
		if v >= 2 {
			n++
		}
	}
	return n
}

// isTractor reports whether cards are consecutive pairs of one non-joker suit.
// The level rank is a trump card and therefore breaks a sequence.
func isTractor(cards []Card, level Rank) bool {
	if len(cards) < 4 || len(cards)%2 != 0 {
		return false
	}
	first := cards[0]
	if first.IsJoker() {
		return false
	}
	counts := make(map[[2]int]int)
	for _, c := range cards {
		if c.IsJoker() || c.Suit != first.Suit || c.Rank == level {
			return false
		}
		counts[[2]int{int(c.Suit), int(c.Rank)}]++
	}
	ranks := make([]int, 0, len(cards)/2)
	for k, v := range counts {
		if v != 2 {
			return false
		}
		ranks = append(ranks, k[1])
	}
	if len(ranks) != len(cards)/2 {
		return false
	}
	min, max := ranks[0], ranks[0]
	for _, r := range ranks {
		if r < min {
			min = r
		}
		if r > max {
			max = r
		}
	}
	if max-min != len(ranks)-1 {
		return false
	}
	return true
}

// tractorLength returns the length in pairs of the longest tractor inside the
// given cards, treating each suit separately.
func tractorLength(cards []Card, level Rank) int {
	bySuit := make(map[Suit]map[int]int)
	for _, c := range cards {
		if c.IsJoker() || c.Rank == level {
			continue
		}
		if bySuit[c.Suit] == nil {
			bySuit[c.Suit] = make(map[int]int)
		}
		bySuit[c.Suit][int(c.Rank)]++
	}
	best := 0
	for _, counts := range bySuit {
		run := 0
		for r := 0; r <= 12; r++ {
			if counts[r] >= 2 {
				run++
				if run > best {
					best = run
				}
			} else {
				run = 0
			}
		}
	}
	return best
}

// hasTractorOfLength reports whether cards contain k consecutive pairs in one
// suit.
func hasTractorOfLength(cards []Card, k int, level Rank) bool {
	if k < 2 {
		return true
	}
	bySuit := make(map[Suit]map[int]int)
	for _, c := range cards {
		if c.IsJoker() || c.Rank == level {
			continue
		}
		if bySuit[c.Suit] == nil {
			bySuit[c.Suit] = make(map[int]int)
		}
		bySuit[c.Suit][int(c.Rank)]++
	}
	for _, counts := range bySuit {
		run := 0
		for r := 0; r <= 12; r++ {
			if counts[r] >= 2 {
				run++
				if run >= k {
					return true
				}
			} else {
				run = 0
			}
		}
	}
	return false
}

// beats reports whether challenger beats best for a trick led with lead.
func beats(challenger, best, lead []Card, trump Suit, level Rank) bool {
	if len(challenger) != len(best) {
		return false
	}
	lc, err := Classify(lead, trump, level)
	if err != nil {
		return false
	}
	cc, err := Classify(challenger, trump, level)
	if err != nil {
		return false
	}
	bc, err := Classify(best, trump, level)
	if err != nil {
		return false
	}
	if lc.Type == ComboThrow {
		// A throw is checked to be unbeatable inside its own suit when it is
		// played, so only a trump beat can take the trick. Trump beats require
		// at least as much structure as the throw contains.
		if lc.Group == GroupTrump {
			if cc.Group != GroupTrump || cc.Pairs < lc.Pairs {
				return false
			}
			return cc.Max > bc.Max
		}
		if cc.Group != GroupTrump || cc.Pairs < lc.Pairs {
			return false
		}
		return true
	}
	if cc.Type != lc.Type || cc.Pairs != lc.Pairs {
		return false
	}
	if cc.Group == bc.Group {
		return cc.Max > bc.Max
	}
	if cc.Group == GroupTrump && bc.Group != GroupTrump {
		return true
	}
	return false
}

// Beats reports whether challenger beats best given the original lead. It is the
// exported form of beats, used by the UI to describe each comparison.
func Beats(challenger, best, lead []Card, trump Suit, level Rank) bool {
	return beats(challenger, best, lead, trump, level)
}

// containsAll reports whether every card in want is present in have.
func containsAll(have, want []Card) bool {
	ids := make(map[int]bool, len(have))
	for _, c := range have {
		ids[c.ID] = true
	}
	for _, c := range want {
		if !ids[c.ID] {
			return false
		}
	}
	return true
}

// sameGroupCount counts cards of a group.
func sameGroupCount(cards []Card, group int, trump Suit, level Rank) int {
	n := 0
	for _, c := range cards {
		if Group(c, trump, level) == group {
			n++
		}
	}
	return n
}
