package game

import (
	"fmt"
	"math/rand"
	"time"
)

// SaveVersion is bumped when the on-disk save layout changes.
const SaveVersion = 1

// GameState is the serialisable form of a match and its current round.
type GameState struct {
	Version int
	SavedAt time.Time
	Levels  [2]Rank
	Dealer  int
	RoundNo int
	Winner  int
	History []RoundResult
	Last    *RoundResult
	Current *RoundState
}

// RoundState is the serialisable form of one deal.
type RoundState struct {
	Dealer      int
	Level       Rank
	Trump       Suit
	Declarer    int
	BidStrength int
	Hands       [4][]Card
	Bottom      []Card
	Phase       Phase

	BidStep int
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

	History  []TrickRecord
	BuryLeft int
}

// State snapshots the match for saving.
func (g *Game) State() GameState {
	s := GameState{
		Version: SaveVersion,
		SavedAt: time.Now(),
		Levels:  g.Levels,
		Dealer:  g.Dealer,
		RoundNo: g.RoundNo,
		Winner:  g.Winner,
		History: cloneResults(g.History),
	}
	if g.Last != nil {
		last := *g.Last
		s.Last = &last
	}
	if g.Current != nil {
		r := g.Current
		s.Current = &RoundState{
			Dealer:          r.Dealer,
			Level:           r.Level,
			Trump:           r.Trump,
			Declarer:        r.Declarer,
			BidStrength:     r.BidStrength,
			Hands:           cloneHands(r.Hands),
			Bottom:          cloneCards(r.Bottom),
			Phase:           r.Phase,
			BidStep:         r.bidStep,
			Turn:            r.Turn,
			Leader:          r.Leader,
			TrickNo:         r.TrickNo,
			Plays:           cloneHands(r.Plays),
			Played:          r.Played,
			Best:            r.Best,
			DefenderPoints:  r.DefenderPoints,
			DealerPoints:    r.DealerPoints,
			BottomPoints:    r.BottomPoints,
			BottomMult:      r.BottomMult,
			LastWinner:      r.LastWinner,
			LastPlay:        cloneCards(r.LastPlay),
			LastTrickPoints: r.LastTrickPoints,
			History:         cloneHistory(r.History),
			BuryLeft:        r.BuryLeft,
		}
	}
	return s
}

// Restore replaces g with a saved state. The RNG cannot be serialised; future
// deals get a fresh seed, while the current deal is restored exactly.
func (g *Game) Restore(s GameState) error {
	if s.Version != SaveVersion {
		return fmt.Errorf("存档版本不支持")
	}
	if err := validateState(s); err != nil {
		return err
	}
	*g = Game{
		Levels:  s.Levels,
		Dealer:  s.Dealer,
		RoundNo: s.RoundNo,
		Winner:  s.Winner,
		History: cloneResults(s.History),
		RNG:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if s.Last != nil {
		last := *s.Last
		g.Last = &last
	}
	if s.Current != nil {
		rs := s.Current
		r := &Round{
			Dealer:          rs.Dealer,
			Level:           rs.Level,
			Trump:           rs.Trump,
			Declarer:        rs.Declarer,
			BidStrength:     rs.BidStrength,
			Hands:           cloneHands(rs.Hands),
			Bottom:          cloneCards(rs.Bottom),
			Phase:           rs.Phase,
			bidStep:         rs.BidStep,
			Turn:            rs.Turn,
			Leader:          rs.Leader,
			TrickNo:         rs.TrickNo,
			Plays:           cloneHands(rs.Plays),
			Played:          rs.Played,
			Best:            rs.Best,
			DefenderPoints:  rs.DefenderPoints,
			DealerPoints:    rs.DealerPoints,
			BottomPoints:    rs.BottomPoints,
			BottomMult:      rs.BottomMult,
			LastWinner:      rs.LastWinner,
			LastPlay:        cloneCards(rs.LastPlay),
			LastTrickPoints: rs.LastTrickPoints,
			History:         cloneHistory(rs.History),
			BuryLeft:        rs.BuryLeft,
			RNG:             g.RNG,
		}
		for i := range r.Hands {
			SortHand(r.Hands[i], r.Trump, r.Level)
		}
		g.Current = r
	}
	return nil
}

func validateState(s GameState) error {
	if s.Dealer < 0 || s.Dealer > 3 {
		return fmt.Errorf("存档庄家无效")
	}
	if s.RoundNo < 1 {
		return fmt.Errorf("存档局数无效")
	}
	if s.Winner != -1 {
		return fmt.Errorf("存档牌局已结束")
	}
	if s.Current == nil {
		return fmt.Errorf("存档没有进行中的牌局")
	}
	rs := s.Current
	if rs.Dealer < 0 || rs.Dealer > 3 || rs.Turn < 0 || rs.Turn > 3 ||
		rs.Leader < 0 || rs.Leader > 3 || rs.LastWinner < -1 || rs.LastWinner > 3 {
		return fmt.Errorf("存档座位无效")
	}
	if rs.Phase < PhaseBidding || rs.Phase > PhaseDone {
		return fmt.Errorf("存档阶段无效")
	}
	if rs.Trump != NoTrump && (rs.Trump < Spade || rs.Trump > Diamond) {
		return fmt.Errorf("存档主牌无效")
	}
	if rs.Level < Rank2 || rs.Level > RankA {
		return fmt.Errorf("存档级牌无效")
	}
	if rs.TrickNo < 0 || rs.TrickNo > 25 || rs.BuryLeft < 0 || rs.BuryLeft > 8 {
		return fmt.Errorf("存档进度无效")
	}

	seen := make(map[int]bool, 108)
	add := func(cards []Card) error {
		for _, c := range cards {
			if err := validateCard(c); err != nil {
				return err
			}
			if seen[c.ID] {
				return fmt.Errorf("存档有重复牌")
			}
			seen[c.ID] = true
		}
		return nil
	}
	for i := 0; i < 4; i++ {
		if err := add(rs.Hands[i]); err != nil {
			return err
		}
		if err := add(rs.Plays[i]); err != nil {
			return err
		}
	}
	if err := add(rs.Bottom); err != nil {
		return err
	}
	for _, rec := range rs.History {
		if rec.Leader < 0 || rec.Leader > 3 || rec.Winner < 0 || rec.Winner > 3 {
			return fmt.Errorf("存档墩记录无效")
		}
		for i := 0; i < 4; i++ {
			if err := add(rec.Plays[i]); err != nil {
				return err
			}
		}
	}
	if len(seen) != 108 {
		return fmt.Errorf("存档牌数无效")
	}
	switch rs.Phase {
	case PhaseBidding:
		for i := 0; i < 4; i++ {
			if len(rs.Hands[i]) != 25 {
				return fmt.Errorf("存档手牌数量无效")
			}
		}
		if len(rs.Bottom) != 8 {
			return fmt.Errorf("存档底牌无效")
		}
	case PhaseBury:
		if len(rs.Bottom) != 0 || len(rs.Hands[rs.Dealer]) != 33 {
			return fmt.Errorf("存档扣底状态无效")
		}
		for i := 0; i < 4; i++ {
			if i != rs.Dealer && len(rs.Hands[i]) != 25 {
				return fmt.Errorf("存档手牌数量无效")
			}
		}
	case PhasePlay, PhaseDone:
		if len(rs.Bottom) != 8 {
			return fmt.Errorf("存档底牌无效")
		}
	}
	return nil
}

func validateCard(c Card) error {
	if c.ID < 0 || c.ID > 107 {
		return fmt.Errorf("存档牌编号无效")
	}
	if c.Suit == SuitJoker {
		if c.Rank != RankSmallJoker && c.Rank != RankBigJoker {
			return fmt.Errorf("存档王无效")
		}
		return nil
	}
	if c.Suit < Spade || c.Suit > Diamond || c.Rank < Rank2 || c.Rank > RankA {
		return fmt.Errorf("存档牌面无效")
	}
	return nil
}

func cloneCards(cards []Card) []Card {
	return append([]Card(nil), cards...)
}

func cloneHands(h [4][]Card) [4][]Card {
	var out [4][]Card
	for i := range h {
		out[i] = cloneCards(h[i])
	}
	return out
}

func cloneHistory(h []TrickRecord) []TrickRecord {
	out := make([]TrickRecord, len(h))
	for i, rec := range h {
		out[i] = rec
		out[i].Plays = cloneHands(rec.Plays)
	}
	return out
}

func cloneResults(rs []RoundResult) []RoundResult {
	return append([]RoundResult(nil), rs...)
}
