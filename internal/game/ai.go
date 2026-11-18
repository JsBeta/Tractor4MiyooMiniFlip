package game

import (
	"fmt"
	"sort"
)

// ---------- bidding ----------

// suitStrength estimates how good a suit would be as trump.
func suitStrength(hand []Card, s Suit, level Rank, trump Suit) int {
	score := 0
	for _, c := range hand {
		if c.IsJoker() {
			score += 3
			continue
		}
		if c.Rank == level {
			if c.Suit == s {
				score += 4
			} else {
				score += 1
			}
			continue
		}
		if c.Suit == s {
			score++
			switch c.Rank {
			case RankA, RankK:
				score++
			}
		}
	}
	for _, p := range pairsOf(hand) {
		if p[0].Suit == s && p[0].Rank != level {
			score += 2
		}
	}
	return score
}

// AIBid decides whether the seat wants to declare trump.
func AIBid(r *Round, seat int) ([]Card, bool) {
	if r.Phase != PhaseBidding || r.BidSeat() != seat {
		return nil, false
	}
	hand := r.Hands[seat]
	bestSuit := Spade
	best := -1
	for s := Spade; s <= Diamond; s++ {
		v := suitStrength(hand, s, r.Level, r.Trump)
		if v > best {
			best, bestSuit = v, s
		}
	}
	switch r.BidStrength {
	case 0:
		if best >= 10 {
			for _, c := range hand {
				if c.Rank == r.Level && c.Suit == bestSuit {
					return []Card{c}, true
				}
			}
		}
	case 1:
		if best >= 9 {
			var pair []Card
			for _, c := range hand {
				if c.Rank == r.Level && c.Suit == bestSuit {
					pair = append(pair, c)
				}
			}
			if len(pair) >= 2 {
				return pair[:2], true
			}
		}
	}
	return nil, false
}

// AIBury picks eight low cards to put under the deck.
func AIBury(r *Round, seat int) []Card {
	type scored struct {
		c Card
		v int
	}
	var all []scored
	for _, c := range r.Hands[seat] {
		v := Order(c, r.Trump, r.Level)
		if IsTrump(c, r.Trump, r.Level) {
			v += 25
		}
		if c.PointValue() > 0 {
			v += 12
		}
		if HasPair(r.Hands[seat], c) {
			v += 3
		}
		all = append(all, scored{c, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].v != all[j].v {
			return all[i].v < all[j].v
		}
		return all[i].c.ID < all[j].c.ID
	})
	out := make([]Card, 0, 8)
	for i := 0; i < 8 && i < len(all); i++ {
		out = append(out, all[i].c)
	}
	return out
}

// ---------- helpers ----------

// pairsOf returns all identical pairs in cards.
func pairsOf(cards []Card) [][]Card {
	type key struct {
		s Suit
		r Rank
	}
	groups := make(map[key][]Card)
	for _, c := range cards {
		if c.IsJoker() {
			continue
		}
		k := key{c.Suit, c.Rank}
		groups[k] = append(groups[k], c)
	}
	// Joker pairs are pairs too.
	var out [][]Card
	for _, g := range groups {
		if len(g) >= 2 {
			out = append(out, append([]Card(nil), g[:2]...))
		}
	}
	var sj, bj []Card
	for _, c := range cards {
		if c.Suit == SuitJoker && c.Rank == RankSmallJoker {
			sj = append(sj, c)
		}
		if c.Suit == SuitJoker && c.Rank == RankBigJoker {
			bj = append(bj, c)
		}
	}
	if len(sj) >= 2 {
		out = append(out, sj[:2])
	}
	if len(bj) >= 2 {
		out = append(out, bj[:2])
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0].ID < out[j][0].ID })
	return out
}

// tractorsOf returns every window of k consecutive pairs in cards.
func tractorsOf(cards []Card, k int, level Rank) [][]Card {
	if k < 2 {
		return nil
	}
	bySuit := make(map[Suit]map[int][]Card)
	for _, c := range cards {
		if c.IsJoker() || c.Rank == level {
			continue
		}
		if bySuit[c.Suit] == nil {
			bySuit[c.Suit] = make(map[int][]Card)
		}
		bySuit[c.Suit][int(c.Rank)] = append(bySuit[c.Suit][int(c.Rank)], c)
	}
	var out [][]Card
	for _, ranks := range bySuit {
		for start := 0; start+k-1 <= 12; start++ {
			ok := true
			var t []Card
			for i := 0; i < k; i++ {
				cs := ranks[start+i]
				if len(cs) < 2 {
					ok = false
					break
				}
				t = append(t, cs[0], cs[1])
			}
			if ok {
				out = append(out, t)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0].ID < out[j][0].ID })
	return out
}

func subtract(cards, used []Card) []Card {
	drop := make(map[int]bool, len(used))
	for _, c := range used {
		drop[c.ID] = true
	}
	var out []Card
	for _, c := range cards {
		if !drop[c.ID] {
			out = append(out, c)
		}
	}
	return out
}

func sortByOrder(cards []Card, r *Round) {
	sort.SliceStable(cards, func(i, j int) bool {
		oi, oj := Order(cards[i], r.Trump, r.Level), Order(cards[j], r.Trump, r.Level)
		if oi != oj {
			return oi < oj
		}
		return cards[i].ID < cards[j].ID
	})
}

func takeLowest(r *Round, cards []Card, n int) []Card {
	if n <= 0 {
		return nil
	}
	c := append([]Card(nil), cards...)
	sortByOrder(c, r)
	if n > len(c) {
		n = len(c)
	}
	return c[:n]
}

func takeHighest(r *Round, cards []Card, n int) []Card {
	if n <= 0 {
		return nil
	}
	c := append([]Card(nil), cards...)
	sortByOrder(c, r)
	for i, j := 0, len(c)-1; i < j; i, j = i+1, j-1 {
		c[i], c[j] = c[j], c[i]
	}
	if n > len(c) {
		n = len(c)
	}
	return c[:n]
}

func candidateKey(cards []Card) string {
	ids := make([]int, len(cards))
	for i, c := range cards {
		ids[i] = c.ID
	}
	sort.Ints(ids)
	b := make([]byte, 0, len(ids)*4)
	for _, id := range ids {
		b = append(b, byte(id), byte(id>>8), byte(id>>16), 0)
	}
	return string(b)
}

// ---------- follow candidate generation ----------

func followCandidates(r *Round, seat int) [][]Card {
	lead := r.Plays[r.Leader]
	lc, err := Classify(lead, r.Trump, r.Level)
	if err != nil {
		return nil
	}
	n := len(lead)
	hand := r.Hands[seat]
	mine := GroupCards(hand, lc.Group, r.Trump, r.Level)
	seen := make(map[string]bool)
	var out [][]Card
	add := func(cards []Card) {
		if len(cards) != n {
			return
		}
		key := candidateKey(cards)
		if seen[key] {
			return
		}
		if r.validateFollow(seat, cards) != nil {
			return
		}
		seen[key] = true
		out = append(out, cards)
	}
	addFill := func(chosen []Card, pool []Card) {
		need := n - len(chosen)
		if need < 0 {
			return
		}
		for _, fill := range [][]Card{takeLowest(r, pool, need), takeHighest(r, pool, need)} {
			c := append(append([]Card(nil), chosen...), fill...)
			add(c)
		}
	}

	if len(mine) >= n {
		if lc.Type == ComboTractor {
			k := lc.Pairs
			tractors := tractorsOf(mine, k, r.Level)
			if len(tractors) > 0 {
				for i, t := range tractors {
					if i >= 6 {
						break
					}
					addFill(t, subtract(mine, t))
				}
				return out
			}
		}
		pairs := pairsOf(mine)
		need := lc.Pairs
		if need > len(pairs) {
			need = len(pairs)
		}
		var selections [][]Card
		if need == 0 {
			selections = append(selections, nil)
			if n >= 2 && len(pairs) > 0 {
				for i, p := range pairs {
					if i >= 3 {
						break
					}
					selections = append(selections, p)
				}
			}
		} else if need >= len(pairs) {
			var all []Card
			for _, p := range pairs {
				all = append(all, p...)
			}
			selections = append(selections, all)
		} else {
			selections = append(selections, combinePairs(pairs, need, 8)...)
			var all []Card
			for _, p := range pairs {
				all = append(all, p...)
			}
			low := combinePairs(pairs, need, 1)
			selections = append(selections, low...)
			_ = all
		}
		for _, sel := range selections {
			if len(sel) > n {
				continue
			}
			pool := subtract(mine, sel)
			addFill(sel, pool)
		}
		return out
	}

	// Void (or short) in the led suit: all those cards must be played.
	base := mine
	fillCount := n - len(base)
	others := subtract(hand, base)
	if fillCount > len(others) {
		fillCount = len(others)
	}
	var nonTrump, trumps []Card
	for _, c := range others {
		if IsTrump(c, r.Trump, r.Level) {
			trumps = append(trumps, c)
		} else {
			nonTrump = append(nonTrump, c)
		}
	}
	// Lowest cards, avoiding trumps and points.
	var quiet []Card
	for _, c := range nonTrump {
		if c.PointValue() == 0 {
			quiet = append(quiet, c)
		}
	}
	addFill(base, takeLowest(r, quiet, fillCount))
	addFill(base, takeLowest(r, nonTrump, fillCount))
	addFill(base, takeLowest(r, trumps, fillCount))
	addFill(base, takeHighest(r, others, fillCount))

	// Feeding points to a winning partner.
	var points []Card
	for _, c := range others {
		if c.PointValue() > 0 {
			points = append(points, c)
		}
	}
	addFill(base, takeHighest(r, points, fillCount))

	// Trumping: a matching pair/tractor of trumps when the lead has structure.
	if lc.Pairs > 0 || lc.Type == ComboTractor {
		var structure []Card
		if lc.Type == ComboTractor {
			ts := tractorsOf(trumps, lc.Pairs, r.Level)
			if len(ts) > 0 {
				structure = ts[0]
			}
		} else if lc.Pairs == 1 {
			ps := pairsOf(trumps)
			if len(ps) > 0 {
				structure = ps[0]
			}
		}
		if len(structure) > 0 && len(structure) <= n {
			addFill(structure, subtract(trumps, structure))
		}
	}
	return out
}

// combinePairs returns up to limit combinations of k pairs.
func combinePairs(pairs [][]Card, k, limit int) [][]Card {
	var out [][]Card
	var rec func(start int, cur []Card)
	rec = func(start int, cur []Card) {
		if len(out) >= limit {
			return
		}
		if len(cur) == k*2 {
			out = append(out, append([]Card(nil), cur...))
			return
		}
		for i := start; i < len(pairs); i++ {
			rec(i+1, append(cur, pairs[i]...))
		}
	}
	rec(0, nil)
	return out
}

// ---------- lead candidate generation ----------

func leadCandidates(r *Round, seat int) [][]Card {
	hand := r.Hands[seat]
	var out [][]Card
	for _, c := range hand {
		out = append(out, []Card{c})
	}
	for _, p := range pairsOf(hand) {
		out = append(out, p)
	}
	for k := 2; k <= 4; k++ {
		out = append(out, tractorsOf(hand, k, r.Level)...)
	}
	return out
}

// ---------- choice ----------

func playedCount(r *Round) int {
	n := 0
	for i := 0; i < 4; i++ {
		if r.Played[i] {
			n++
		}
	}
	return n
}

func trumpCount(cards []Card, trump Suit, level Rank) int {
	n := 0
	for _, c := range cards {
		if IsTrump(c, trump, level) {
			n++
		}
	}
	return n
}

// playCost estimates how expensive a set of cards is to spend.
func playCost(r *Round, cards []Card) float64 {
	cost := 0.0
	for _, c := range cards {
		cost += float64(Order(c, r.Trump, r.Level)) / 6
		if IsTrump(c, r.Trump, r.Level) {
			cost += 6
		}
		cost += float64(c.PointValue()) * 0.8
	}
	return cost
}

// AIChoose returns a play for the seat.
func AIChoose(r *Round, seat int) ([]Card, error) {
	if r.Phase != PhasePlay {
		return nil, fmt.Errorf("现在不能出牌")
	}
	if seat != r.Turn {
		return nil, fmt.Errorf("还没轮到你")
	}
	lead := r.Plays[r.Leader]
	if len(lead) == 0 || seat == r.Leader {
		return aiLead(r, seat), nil
	}
	cands := followCandidates(r, seat)
	if len(cands) == 0 {
		return nil, fmt.Errorf("没有合法的出牌")
	}
	best := r.Plays[r.Best]
	partnerWinning := r.Best == Partner(seat)
	tablePoints := 0
	for i := 0; i < 4; i++ {
		tablePoints += PointSum(r.Plays[i])
	}
	last := playedCount(r) == 3

	type option struct {
		cards []Card
		win   bool
		cost  float64
		value float64
	}
	opts := make([]option, 0, len(cands))
	for _, c := range cands {
		win := beats(c, best, lead, r.Trump, r.Level)
		cost := playCost(r, c)
		value := 0.0
		if win {
			value += 50
			if last {
				value += 25
			}
			value += float64(tablePoints) * 1.5
		}
		if partnerWinning {
			if win {
				value -= 40
			} else {
				value += float64(PointSum(c)) * 2
			}
		}
		opts = append(opts, option{c, win, cost, value})
	}
	sort.SliceStable(opts, func(i, j int) bool {
		si := opts[i].value - opts[i].cost
		sj := opts[j].value - opts[j].cost
		if si != sj {
			return si > sj
		}
		return opts[i].cost < opts[j].cost
	})
	// When the trick is worthless and the partner is not winning, prefer to
	// keep high cards instead of dumping them.
	if !partnerWinning && tablePoints == 0 && !opts[0].win {
		sort.SliceStable(opts, func(i, j int) bool { return opts[i].cost < opts[j].cost })
	}
	return opts[0].cards, nil
}

func aiLead(r *Round, seat int) []Card {
	hand := r.Hands[seat]
	trumps := trumpCount(hand, r.Trump, r.Level)

	// Best tractor: play a high one early.
	var bestTractor []Card
	bestTop := -1
	for k := 4; k >= 2; k-- {
		for _, t := range tractorsOf(hand, k, r.Level) {
			top := 0
			for _, c := range t {
				o := Order(c, r.Trump, r.Level)
				if IsTrump(c, r.Trump, r.Level) {
					o -= 90
				}
				if o > top {
					top = o
				}
			}
			if top > bestTop {
				bestTop, bestTractor = top, t
			}
		}
	}
	if bestTractor != nil && bestTop >= int(RankQ) {
		return bestTractor
	}

	// A pair of aces or kings.
	var bestPair []Card
	bestPairTop := -1
	for _, p := range pairsOf(hand) {
		top := Order(p[0], r.Trump, r.Level)
		if !IsTrump(p[0], r.Trump, r.Level) && top >= int(RankK) && top > bestPairTop {
			bestPairTop, bestPair = top, p
		}
	}
	if bestPair != nil {
		return bestPair
	}

	// Long trump suit: draw trumps while holding the advantage.
	if trumps >= 9 && len(hand) > 12 {
		return takeLowest(r, GroupCards(hand, GroupTrump, r.Trump, r.Level), 1)
	}

	// Otherwise lead a low card from the longest side suit, avoiding points.
	bySuit := make(map[Suit][]Card)
	for _, c := range hand {
		if IsTrump(c, r.Trump, r.Level) {
			continue
		}
		bySuit[c.Suit] = append(bySuit[c.Suit], c)
	}
	var bestSuit []Card
	for _, cs := range bySuit {
		if len(cs) > len(bestSuit) {
			bestSuit = cs
		}
	}
	if len(bestSuit) > 0 {
		var quiet []Card
		for _, c := range bestSuit {
			if c.PointValue() == 0 {
				quiet = append(quiet, c)
			}
		}
		if len(quiet) > 0 {
			return takeLowest(r, quiet, 1)
		}
		return takeLowest(r, bestSuit, 1)
	}
	return takeLowest(r, hand, 1)
}

// AIAutoBury runs the AI bury for the current dealer.
func AIAutoBury(r *Round) error {
	cards := AIBury(r, r.Dealer)
	return r.Bury(r.Dealer, cards)
}
