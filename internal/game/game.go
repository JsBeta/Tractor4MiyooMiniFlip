package game

import (
	"math/rand"
)

// RoundResult summarises a completed round.
type RoundResult struct {
	RoundNo        int
	Dealer         int
	Declarer       int
	Trump          Suit
	Level          Rank
	DefenderPoints int
	DealerPoints   int
	BottomPoints   int
	BottomMult     int
	Success        bool
	Upgrade        int
	Winner         int // 0 or 1 when the game was won, -1 otherwise
	NextDealer     int
	LastWinner     int
	BigWin         bool // 大光
	SmallWin       bool // 小光
}

// Game is a full match: a sequence of rounds with per team levels.
type Game struct {
	Levels  [2]Rank
	Dealer  int
	RoundNo int
	Current *Round
	RNG     *rand.Rand
	Winner  int
	History []RoundResult
	Last    *RoundResult
}

// NewGame creates a match. Seat 0 (South) is the human.
func NewGame(seed int64) *Game {
	return &Game{
		Levels:  [2]Rank{Rank2, Rank2},
		Dealer:  int(seed % 4),
		RoundNo: 1,
		RNG:     rand.New(rand.NewSource(seed)),
		Winner:  -1,
	}
}

// TeamName returns the name of a partnership.
func TeamName(team int) string {
	if team == 0 {
		return "南北方"
	}
	return "东西方"
}

// LevelName renders a level rank.
func LevelName(r Rank) string {
	switch r {
	case RankSmallJoker:
		return "小王"
	case RankBigJoker:
		return "大王"
	}
	return r.String()
}

// StartRound deals the next round using the current dealer's team level.
func (g *Game) StartRound() {
	level := g.Levels[TeamOf(g.Dealer)]
	g.Current = StartRound(g.Dealer, level, g.RNG)
}

// FinishRound scores the completed round and advances the match. It must be
// called exactly once per round, after Current.Phase == PhaseDone.
func (g *Game) FinishRound() RoundResult {
	r := g.Current
	dp := r.DefenderPoints
	success := dp < 80
	res := RoundResult{
		RoundNo:        g.RoundNo,
		Dealer:         r.Dealer,
		Declarer:       r.Declarer,
		Trump:          r.Trump,
		Level:          r.Level,
		DefenderPoints: dp,
		DealerPoints:   r.DealerPoints,
		BottomPoints:   r.BottomPoints,
		BottomMult:     r.BottomMult,
		Success:        success,
		Winner:         -1,
		LastWinner:     r.LastWinner,
	}
	dealerTeam := TeamOf(r.Dealer)
	if success {
		n := 1
		switch {
		case dp == 0:
			n = 3
			res.BigWin = true
		case dp < 40:
			n = 2
			res.SmallWin = true
		}
		res.Upgrade = n
		if g.Levels[dealerTeam] == RankA {
			g.Winner = dealerTeam
			res.Winner = dealerTeam
		} else {
			lv := int(g.Levels[dealerTeam]) + n
			if lv > int(RankA) {
				lv = int(RankA)
			}
			g.Levels[dealerTeam] = Rank(lv)
		}
		res.NextDealer = Partner(r.Dealer)
	} else {
		n := (dp - 80) / 40
		res.Upgrade = n
		defTeam := 1 - dealerTeam
		if g.Levels[defTeam] != RankA {
			lv := int(g.Levels[defTeam]) + n
			if lv > int(RankA) {
				lv = int(RankA)
			}
			g.Levels[defTeam] = Rank(lv)
		}
		if r.LastWinner >= 0 && TeamOf(r.LastWinner) == defTeam {
			res.NextDealer = r.LastWinner
		} else {
			res.NextDealer = (r.Dealer + 1) % 4
		}
	}
	g.Dealer = res.NextDealer
	g.History = append(g.History, res)
	g.Last = &g.History[len(g.History)-1]
	g.RoundNo++
	return res
}

// Hint returns a suggested play for the human.
func (g *Game) Hint() ([]Card, error) {
	return AIChoose(g.Current, 0)
}
