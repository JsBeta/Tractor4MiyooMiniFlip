package ui

import (
	"strings"
	"testing"
	"time"

	"tractor/internal/game"
)

func TestCardCN(t *testing.T) {
	if got := cardCN(game.Card{Suit: game.Spade, Rank: game.RankA}); got != "黑桃A" {
		t.Fatalf("spade ace = %q", got)
	}
	if got := cardCN(game.Card{Suit: game.SuitJoker, Rank: game.RankBigJoker}); got != "大王" {
		t.Fatalf("big joker = %q", got)
	}
	if got := cardsCN([]game.Card{{Suit: game.Heart, Rank: game.Rank10}, {Suit: game.Heart, Rank: game.RankK}}); got != "红桃10、红桃K" {
		t.Fatalf("cards = %q", got)
	}
}

func TestLogOpensAndCloses(t *testing.T) {
	a := New(752, 560)
	a.SetNow(time.Now())
	a.StartNewGame(1)
	a.Key(KeyL)
	if a.Screen != ScreenLog {
		t.Fatalf("L did not open log: screen=%v", a.Screen)
	}
	a.Key(KeyDown)
	a.Key(KeyB)
	if a.Screen != ScreenTable {
		t.Fatalf("B did not close log: screen=%v", a.Screen)
	}
}

func TestLogLines(t *testing.T) {
	LoadFonts()
	a := New(752, 560)
	a.SetNow(time.Now())
	a.StartNewGame(1)
	r := a.G.Current
	r.Trump = game.Spade
	r.History = []game.TrickRecord{{
		Leader: 0, Winner: 0, Points: 5,
		Plays: [4][]game.Card{
			0: {{Suit: game.Spade, Rank: game.RankA}},
			1: {{Suit: game.Spade, Rank: game.RankK}},
		},
	}}
	var sb strings.Builder
	for _, ln := range a.logLines() {
		sb.WriteString(ln.text)
		sb.WriteString("\n")
	}
	s := sb.String()
	for _, want := range []string{"第 1 墩", "南：黑桃A（首出·单张）", "东：黑桃K（跟牌）", "南 首出即最大（单张），+5 分"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestTrickActions(t *testing.T) {
	plays := [4][]game.Card{
		0: {{Suit: game.Spade, Rank: game.RankA, ID: 1}},
		1: {{Suit: game.Spade, Rank: game.RankK, ID: 2}},
		2: {{Suit: game.Heart, Rank: game.Rank10, ID: 3}},
		3: {{Suit: game.SuitJoker, Rank: game.RankBigJoker, ID: 4}},
	}
	steps := trickActions(0, plays, game.Heart, game.Rank2)
	got := make([]string, len(steps))
	for i, s := range steps {
		got[i] = s.action
	}
	want := []string{"首出·单张", "跟牌", "毙", "盖毙"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("actions = %v, want %v", got, want)
	}
}

func TestTrickActionsPairRuff(t *testing.T) {
	plays := [4][]game.Card{
		0: {{Suit: game.Spade, Rank: game.RankA, ID: 1}, {Suit: game.Spade, Rank: game.RankA, ID: 2}},
		1: {{Suit: game.Club, Rank: game.RankK, ID: 3}, {Suit: game.Club, Rank: game.RankK, ID: 4}},
		2: {{Suit: game.Heart, Rank: game.Rank10, ID: 5}, {Suit: game.Heart, Rank: game.Rank10, ID: 6}},
		3: {{Suit: game.Heart, Rank: game.RankQ, ID: 7}, {Suit: game.Heart, Rank: game.RankQ, ID: 8}},
	}
	steps := trickActions(0, plays, game.Heart, game.Rank2)
	got := make([]string, len(steps))
	for i, s := range steps {
		got[i] = s.action
	}
	want := []string{"首出·对子", "垫牌", "对毙", "盖毙"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("actions = %v, want %v", got, want)
	}
}

func TestLogRuffText(t *testing.T) {
	LoadFonts()
	a := New(752, 560)
	a.SetNow(time.Now())
	a.StartNewGame(1)
	r := a.G.Current
	r.Trump = game.Heart
	r.Level = game.Rank2
	r.History = []game.TrickRecord{{
		Leader: 0, Winner: 3, Points: 10,
		Plays: [4][]game.Card{
			0: {{Suit: game.Spade, Rank: game.RankA, ID: 1}},
			1: {{Suit: game.Spade, Rank: game.RankK, ID: 2}},
			2: {{Suit: game.Heart, Rank: game.Rank10, ID: 3}},
			3: {{Suit: game.SuitJoker, Rank: game.RankBigJoker, ID: 4}},
		},
	}}
	var sb strings.Builder
	for _, ln := range a.logLines() {
		sb.WriteString(ln.text)
		sb.WriteString("\n")
	}
	s := sb.String()
	for _, want := range []string{"北：红桃10（毙）", "西：大王（盖毙）", "西 盖毙（单张），+10 分"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestLogGlyphs(t *testing.T) {
	LoadFonts()
	texts := []string{
		"出牌日志", "第 1 局 · 庄家 南", "主牌 黑桃2 · 闲家 10 / 80 分",
		"第 1 墩（进行中）", "南：黑桃A（首出·单张）", "东：黑桃K（跟牌）",
		"西：红桃2（对毙）", "北：大王（盖毙）", "东：方块5（垫牌）",
		"南 首出即最大（单张），+5 分", "→ 东 盖毙（单张），+10 分",
		"本局结束", "闲家 120 分 · 底牌 ×4（20 分）", "还没有出牌记录",
		"↑↓ 滚动    ←→ 翻页    L / B 返回",
	}
	for _, s := range texts {
		for _, r := range s {
			if r == ' ' {
				continue
			}
			if _, ok := fonts.sm.Glyphs[r]; !ok {
				t.Errorf("sm missing %q in %q", r, s)
			}
			if _, ok := fonts.xs.Glyphs[r]; !ok {
				t.Errorf("xs missing %q in %q", r, s)
			}
		}
	}
}
