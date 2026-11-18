package game

import (
	"fmt"
	"math/rand"
	"sort"
)

// Phase is the stage of a round.
type Phase int

const (
	PhaseBidding Phase = iota
	PhaseBury
	PhasePlay
	PhaseDone
)

// TrickRecord is kept for the trick history / review panel.
type TrickRecord struct {
	Leader int
	Plays  [4][]Card
	Winner int
	Points int
}

// Round is one deal: bidding, burying the bottom and 25 tricks.
type Round struct {
	Dealer      int
	Level       Rank
	Trump       Suit
	Declarer    int
	BidStrength int
	Hands       [4][]Card
	Bottom      []Card
	Phase       Phase

	bidStep int
	Turn    int
	Leader  int
	TrickNo int
	Plays   [4][]Card
	Played  [4]bool
	Best    int

	DefenderPoints  int
	DealerPoints    int
	BottomPoints    int
	BottomMult      int
	LastWinner      int
	LastPlay        []Card
	LastTrickPoints int

	History []TrickRecord

	// BuryLeft is how many cards the dealer still has to bury.
	BuryLeft int

	RNG *rand.Rand
}

// TeamOf returns the partnership index of a seat: 0 = South/North, 1 = East/West.
func TeamOf(seat int) int { return seat % 2 }

// Partner returns the seat of the partner.
func Partner(seat int) int { return (seat + 2) % 4 }

// SeatName returns a short Chinese seat name.
func SeatName(seat int) string {
	return [...]string{"南", "东", "北", "西"}[seat]
}

// DealerTeam returns the team that is currently dealing.
func (r *Round) DealerTeam() int { return TeamOf(r.Dealer) }

// DefenderTeam returns the team trying to collect points.
func (r *Round) DefenderTeam() int { return 1 - r.DealerTeam() }

// StartRound deals for the given dealer and level.
func StartRound(dealer int, level Rank, rng *rand.Rand) *Round {
	hands, bottom := Deal(rng)
	for i := range hands {
		SortHand(hands[i], NoTrump, level)
	}
	return &Round{
		Dealer:   dealer,
		Level:    level,
		Trump:    NoTrump,
		Declarer: -1,
		Hands:    hands,
		Bottom:   bottom,
		Phase:    PhaseBidding,
		Best:     -1,
		RNG:      rng,
	}
}

// BidSeat is the seat whose turn it is during bidding.
func (r *Round) BidSeat() int { return (r.Dealer + 1 + r.bidStep) % 4 }

// Bid declares trump with one level card or a pair of level cards.
func (r *Round) Bid(seat int, cards []Card) error {
	if r.Phase != PhaseBidding {
		return fmt.Errorf("现在不是亮主阶段")
	}
	if seat != r.BidSeat() {
		return fmt.Errorf("还没轮到你亮主")
	}
	if len(cards) == 0 || len(cards) > 2 {
		return fmt.Errorf("亮主必须是一张或一对级牌")
	}
	c := cards[0]
	if c.Rank != r.Level || c.IsJoker() {
		return fmt.Errorf("只能亮级牌")
	}
	for _, x := range cards {
		if x.Suit != c.Suit || x.Rank != c.Rank {
			return fmt.Errorf("亮主必须是一对相同的级牌")
		}
	}
	strength := len(cards)
	if strength <= r.BidStrength {
		return fmt.Errorf("亮主力度不够")
	}
	if !handHas(r.Hands[seat], cards) {
		return fmt.Errorf("手里没有这些牌")
	}
	r.Trump = c.Suit
	r.Declarer = seat
	r.BidStrength = strength
	r.nextBid()
	return nil
}

// PassBid declines to declare.
func (r *Round) PassBid(seat int) error {
	if r.Phase != PhaseBidding {
		return fmt.Errorf("现在不是亮主阶段")
	}
	if seat != r.BidSeat() {
		return fmt.Errorf("还没轮到你")
	}
	r.nextBid()
	return nil
}

func (r *Round) nextBid() {
	r.bidStep++
	if r.bidStep < 4 {
		return
	}
	if r.Declarer < 0 {
		r.Trump = NoTrump
	}
	// The dealer picks up the bottom.
	r.Hands[r.Dealer] = append(r.Hands[r.Dealer], r.Bottom...)
	r.Bottom = nil
	r.SortAll()
	r.Phase = PhaseBury
	r.BuryLeft = 8
	r.Turn = r.Dealer
}

// Bury places eight cards face down under the deck.
func (r *Round) Bury(seat int, cards []Card) error {
	if r.Phase != PhaseBury {
		return fmt.Errorf("现在不是扣底阶段")
	}
	if seat != r.Dealer {
		return fmt.Errorf("只有庄家能扣底")
	}
	if len(cards) != 8 {
		return fmt.Errorf("必须扣 8 张牌")
	}
	if !handHas(r.Hands[seat], cards) {
		return fmt.Errorf("手里没有这些牌")
	}
	hand, err := removeCards(r.Hands[seat], cards)
	if err != nil {
		return err
	}
	r.Hands[seat] = hand
	r.Bottom = append([]Card(nil), cards...)
	r.Phase = PhasePlay
	r.Leader = r.Dealer
	r.Turn = r.Dealer
	r.TrickNo = 1
	r.Plays = [4][]Card{}
	r.Played = [4]bool{}
	r.Best = -1
	return nil
}

// CurrentLeaderCombo returns the combo that was led this trick.
func (r *Round) CurrentLeaderCombo() (Combo, error) {
	return Classify(r.Plays[r.Leader], r.Trump, r.Level)
}

// Play plays cards for the seat whose turn it is.
func (r *Round) Play(seat int, cards []Card) error {
	if r.Phase != PhasePlay {
		return fmt.Errorf("现在不能出牌")
	}
	if seat != r.Turn {
		return fmt.Errorf("还没轮到你出牌")
	}
	if len(cards) == 0 {
		return fmt.Errorf("请选择要出的牌")
	}
	if !handHas(r.Hands[seat], cards) {
		return fmt.Errorf("手里没有这些牌")
	}
	if len(r.Plays[r.Leader]) == 0 || seat == r.Leader {
		if err := r.validateLead(seat, cards); err != nil {
			return err
		}
	} else {
		if err := r.validateFollow(seat, cards); err != nil {
			return err
		}
	}
	hand, err := removeCards(r.Hands[seat], cards)
	if err != nil {
		return err
	}
	r.Hands[seat] = hand
	r.Plays[seat] = append([]Card(nil), cards...)
	r.Played[seat] = true

	if seat == r.Leader {
		r.Best = seat
	} else if beats(cards, r.Plays[r.Best], r.Plays[r.Leader], r.Trump, r.Level) {
		r.Best = seat
	}

	if r.Played[0] && r.Played[1] && r.Played[2] && r.Played[3] {
		r.settleTrick()
	} else {
		r.Turn = (seat + 1) % 4
	}
	return nil
}

func (r *Round) settleTrick() {
	points := 0
	for i := 0; i < 4; i++ {
		points += PointSum(r.Plays[i])
	}
	if TeamOf(r.Best) == r.DefenderTeam() {
		r.DefenderPoints += points
	} else {
		r.DealerPoints += points
	}
	rec := TrickRecord{Leader: r.Leader, Winner: r.Best, Points: points}
	for i := 0; i < 4; i++ {
		rec.Plays[i] = append([]Card(nil), r.Plays[i]...)
	}
	r.History = append(r.History, rec)
	r.LastWinner = r.Best
	r.LastPlay = append([]Card(nil), r.Plays[r.Best]...)
	r.LastTrickPoints = points

	r.Plays = [4][]Card{}
	r.Played = [4]bool{}
	r.Best = -1
	r.Leader = r.LastWinner
	r.Turn = r.LastWinner
	r.TrickNo++

	done := true
	for i := 0; i < 4; i++ {
		if len(r.Hands[i]) > 0 {
			done = false
			break
		}
	}
	if done {
		r.finish()
	}
}

func (r *Round) finish() {
	if TeamOf(r.LastWinner) == r.DefenderTeam() {
		mult := bottomMultiplier(r.LastPlay, r.Trump, r.Level)
		bottom := PointSum(r.Bottom) * mult
		r.BottomPoints = bottom
		r.BottomMult = mult
		r.DefenderPoints += bottom
	}
	r.Phase = PhaseDone
}

// bottomMultiplier scores the buried bottom when the defenders win the last
// trick. A throw counts by its best internal structure: a tractor inside the
// throw doubles like a tractor (×8), a pair inside doubles like a pair (×4).
func bottomMultiplier(cards []Card, trump Suit, level Rank) int {
	c, err := Classify(cards, trump, level)
	if err != nil {
		return 2
	}
	switch c.Type {
	case ComboTractor:
		return 8
	case ComboPair:
		return 4
	case ComboThrow:
		if tractorLength(cards, level) >= 2 {
			return 8
		}
		if c.Pairs > 0 {
			return 4
		}
	}
	return 2
}

// BottomMultiplier reports the multiplier applied to the bottom when the
// defenders win the last trick.
func (r *Round) BottomMultiplier() int {
	if r.Phase != PhaseDone {
		return 0
	}
	return bottomMultiplier(r.LastPlay, r.Trump, r.Level)
}

func (r *Round) validateLead(seat int, cards []Card) error {
	combo, err := Classify(cards, r.Trump, r.Level)
	if err != nil {
		return err
	}
	if len(cards) == 1 {
		return nil
	}
	if combo.Type == ComboThrow {
		var opp [][]Card
		for i := 0; i < 4; i++ {
			if i != seat {
				opp = append(opp, r.Hands[i])
			}
		}
		if bad, ok := throwBeatable(cards, opp, r.Trump, r.Level); ok {
			return &ThrowError{Cards: bad}
		}
	}
	return nil
}

func (r *Round) validateFollow(seat int, cards []Card) error {
	lead := r.Plays[r.Leader]
	lc, err := Classify(lead, r.Trump, r.Level)
	if err != nil {
		return err
	}
	n := len(lead)
	if len(cards) != n {
		return fmt.Errorf("必须出 %d 张牌", n)
	}
	hand := r.Hands[seat]
	mine := GroupCards(hand, lc.Group, r.Trump, r.Level)
	if len(mine) >= n {
		if sameGroupCount(cards, lc.Group, r.Trump, r.Level) != n {
			return fmt.Errorf("必须先跟%s", groupName(lc.Group))
		}
		need := lc.Pairs
		if need > 0 {
			avail := pairUnits(mine)
			if got := pairUnits(cards); got < need && got < avail {
				return fmt.Errorf("对方出了对子，有对必须出对")
			}
		}
		if lc.Type == ComboTractor {
			k := lc.Pairs
			if tractorLength(mine, r.Level) >= k && !hasTractorOfLength(cards, k, r.Level) {
				return fmt.Errorf("对方出了拖拉机，有拖拉机必须出")
			}
		}
		return nil
	}
	if !containsAll(cards, mine) {
		return fmt.Errorf("必须先把手里的%s出完", groupName(lc.Group))
	}
	return nil
}

func groupName(group int) string {
	if group == GroupTrump {
		return "主牌"
	}
	return SuitName(Suit(group))
}

// ThrowError is returned when a throw is beaten by an opponent's hand.
type ThrowError struct {
	Cards []Card
}

func (e *ThrowError) Error() string { return "甩牌失败：有牌不是最大的" }

// unitBeatsCard reports whether o (in group go) beats the unit whose leading
// card is c. Trumps always beat non-trumps; inside one group the order decides.
func unitBeatsCard(o, c Card, trump Suit, level Rank) bool {
	go_, gc := Group(o, trump, level), Group(c, trump, level)
	if go_ == gc {
		return Order(o, trump, level) > Order(c, trump, level)
	}
	return go_ == GroupTrump && gc != GroupTrump
}

func unitTop(unit []Card, trump Suit, level Rank) int {
	top := 0
	for _, c := range unit {
		if o := Order(c, trump, level); o > top {
			top = o
		}
	}
	return top
}

// throwBeatable checks whether any unit of a throw can be beaten by an
// opponent's hand. It returns the offending cards.
func throwBeatable(lead []Card, opponents [][]Card, trump Suit, level Rank) ([]Card, bool) {
	for _, unit := range decomposeLead(lead, level) {
		switch len(unit) {
		case 1:
			c := unit[0]
			for _, opp := range opponents {
				for _, o := range opp {
					if unitBeatsCard(o, c, trump, level) {
						return []Card{c}, true
					}
				}
			}
		case 2:
			for _, opp := range opponents {
				for _, o := range opp {
					if o.Rank == unit[0].Rank && o.Suit == unit[0].Suit {
						continue
					}
					if unitBeatsCard(o, unit[0], trump, level) && HandPair(opp, o) {
						return unit, true
					}
				}
			}
		default:
			k := len(unit) / 2
			for _, opp := range opponents {
				for _, t := range tractorsOf(opp, k, level) {
					gt, gu := Group(t[0], trump, level), Group(unit[0], trump, level)
					if gt == gu && unitTop(t, trump, level) > unitTop(unit, trump, level) {
						return unit, true
					}
					if gt == GroupTrump && gu != GroupTrump {
						return unit, true
					}
				}
			}
		}
	}
	return nil, false
}

// HandPair reports whether hand holds a pair of c.
func HandPair(hand []Card, c Card) bool { return HasPair(hand, c) }

// decomposeLead splits a throw into beatable units: singles, pairs and
// tractors. The units together contain exactly the lead cards.
func decomposeLead(lead []Card, level Rank) [][]Card {
	var units [][]Card
	bySuit := make(map[Suit][]Card)
	for _, c := range lead {
		bySuit[c.Suit] = append(bySuit[c.Suit], c)
	}
	for _, cards := range bySuit {
		counts := make(map[Rank][]Card)
		for _, c := range cards {
			counts[c.Rank] = append(counts[c.Rank], c)
		}
		pairRanks := make(map[int]bool)
		for r := Rank2; r <= RankA; r++ {
			cs := counts[r]
			if len(cs) >= 2 {
				pairRanks[int(r)] = true
			} else if len(cs) == 1 {
				units = append(units, cs)
			}
		}
		// The level card is out of sequence even inside the trump suit.
		if pairRanks[int(level)] {
			units = append(units, counts[level])
			delete(pairRanks, int(level))
		}
		r := 0
		for r <= 12 {
			if !pairRanks[r] {
				r++
				continue
			}
			start := r
			for r <= 12 && pairRanks[r] {
				r++
			}
			run := r - start
			if run >= 2 {
				var u []Card
				for i := start; i < r; i++ {
					u = append(u, counts[Rank(i)]...)
				}
				units = append(units, u)
			} else {
				units = append(units, counts[Rank(start)])
			}
		}
	}
	return units
}

func handHas(hand, cards []Card) bool {
	have := make(map[int]bool, len(hand))
	for _, c := range hand {
		have[c.ID] = true
	}
	for _, c := range cards {
		if !have[c.ID] {
			return false
		}
	}
	return true
}

func removeCards(hand, cards []Card) ([]Card, error) {
	if !handHas(hand, cards) {
		return nil, fmt.Errorf("手里没有这些牌")
	}
	drop := make(map[int]bool, len(cards))
	for _, c := range cards {
		drop[c.ID] = true
	}
	out := hand[:0]
	for _, c := range hand {
		if !drop[c.ID] {
			out = append(out, c)
		}
	}
	return out, nil
}

// SortAll sorts every hand for display.
func (r *Round) SortAll() {
	for i := range r.Hands {
		SortHand(r.Hands[i], r.Trump, r.Level)
	}
}

// SortedCopy returns a sorted copy of the hand.
func (r *Round) SortedCopy(seat int) []Card {
	out := append([]Card(nil), r.Hands[seat]...)
	SortHand(out, r.Trump, r.Level)
	return out
}

// sortCards is used by tests.
func sortCards(cards []Card) {
	sort.Slice(cards, func(i, j int) bool { return cards[i].ID < cards[j].ID })
}
