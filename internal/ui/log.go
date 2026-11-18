package ui

import (
	"fmt"
	"strings"
	"time"

	"tractor/internal/game"
	"tractor/internal/gfx"
)

// The trick log is a read-only overlay built from Round.History: every lead,
// every follow, and who won the trick with how many points. It never reveals
// cards that have not been played.

const (
	logBodyX   = 36
	logBodyY   = 132
	logLineH   = 22
	logVisible = 16
	logScrollX = 724
	logScrollW = 5
	logFooterY = 520
)

type logLine struct {
	text string
	col  gfx.Color
}

func (a *App) openLog() {
	if a.G == nil || a.G.Current == nil {
		return
	}
	a.Screen = ScreenLog
	a.logScroll = 1 << 30 // clamped to the bottom on the first draw
	a.logMax = 0
	a.nextAI = a.now.Add(700 * time.Millisecond)
}

func (a *App) keyLog(k Key) {
	switch k {
	case KeyUp:
		a.logScroll -= 2
	case KeyDown:
		a.logScroll += 2
	case KeyLeft:
		a.logScroll -= 8
	case KeyRight:
		a.logScroll += 8
	case KeyL, KeyB, KeyStart, KeySelect:
		a.Screen = ScreenTable
		a.nextAI = a.now.Add(700 * time.Millisecond)
		a.turnAt = time.Time{}
		return
	}
	if a.logScroll < 0 {
		a.logScroll = 0
	}
	if a.logScroll > a.logMax {
		a.logScroll = a.logMax
	}
}

func cardCN(c game.Card) string {
	if c.IsJoker() {
		return c.Rank.String()
	}
	return game.SuitName(c.Suit) + c.Rank.String()
}

func cardsCN(cards []game.Card) string {
	parts := make([]string, len(cards))
	for i, c := range cards {
		parts[i] = cardCN(c)
	}
	return strings.Join(parts, "、")
}

type trickPlay struct {
	seat   int
	cards  []game.Card
	action string // 首出·单张 / 跟牌 / 压过 / 毙 / 对毙 / 盖毙 / 大过 / 垫牌
}

func ruffName(cb game.Combo, err error) string {
	if err != nil {
		return "毙"
	}
	switch cb.Type {
	case game.ComboPair:
		return "对毙"
	case game.ComboTractor:
		return "拖拉机毙"
	}
	return "毙"
}

// trickActions walks a trick in play order and labels how each play compares
// with the best play so far: follow, beat, ruff (毙), over-ruff (盖毙), discard.
func trickActions(leader int, plays [4][]game.Card, trump game.Suit, level game.Rank) []trickPlay {
	var out []trickPlay
	var lead, best []game.Card
	leadGroup := -1
	bestRuff := false
	for j := 0; j < 4; j++ {
		seat := (leader + j) % 4
		cards := plays[seat]
		if len(cards) == 0 {
			continue
		}
		cb, err := game.Classify(cards, trump, level)
		group := game.Group(cards[0], trump, level)
		if j == 0 {
			lead, best = cards, cards
			leadGroup = group
			action := "首出"
			if err == nil {
				action += "·" + cb.Type.String()
			}
			out = append(out, trickPlay{seat, cards, action})
			continue
		}
		if game.Beats(cards, best, lead, trump, level) {
			action := ""
			switch {
			case leadGroup != game.GroupTrump && group == game.GroupTrump && bestRuff:
				action = "盖毙"
			case leadGroup != game.GroupTrump && group == game.GroupTrump:
				action = ruffName(cb, err)
			case group == game.GroupTrump && leadGroup == game.GroupTrump:
				action = "大过"
			default:
				action = "压过"
			}
			best = cards
			bestRuff = leadGroup != game.GroupTrump && group == game.GroupTrump
			out = append(out, trickPlay{seat, cards, action})
			continue
		}
		action := "跟牌"
		if group != leadGroup && group != game.GroupTrump {
			action = "垫牌"
		}
		out = append(out, trickPlay{seat, cards, action})
	}
	return out
}

// logLines turns the trick history into wrapped Chinese lines.
func (a *App) logLines() []logLine {
	r := a.G.Current
	var lines []logLine
	add := func(text string, col gfx.Color) {
		if text == "" {
			lines = append(lines, logLine{"", col})
			return
		}
		for i, sub := range wrapText(fonts.sm, text, a.W-120) {
			if i > 0 {
				sub = "    " + sub
			}
			lines = append(lines, logLine{sub, col})
		}
	}
	trick := func(n int, leader int, plays [4][]game.Card, live bool) []trickPlay {
		if live {
			add(fmt.Sprintf("第 %d 墩（进行中）", n), ColMuted)
		} else {
			add(fmt.Sprintf("第 %d 墩", n), ColMuted)
		}
		steps := trickActions(leader, plays, r.Trump, r.Level)
		for _, p := range steps {
			line := fmt.Sprintf("  %s：%s", game.SeatName(p.seat), cardsCN(p.cards))
			if p.action != "" {
				line += "（" + p.action + "）"
			}
			add(line, ColCream)
		}
		return steps
	}
	for i, rec := range r.History {
		steps := trick(i+1, rec.Leader, rec.Plays, false)
		winAction := ""
		for _, p := range steps {
			if p.seat == rec.Winner {
				winAction = p.action
			}
		}
		combo := ""
		if cb, err := game.Classify(rec.Plays[rec.Winner], r.Trump, r.Level); err == nil {
			combo = "（" + cb.Type.String() + "）"
		}
		pts := "0 分"
		if rec.Points > 0 {
			pts = fmt.Sprintf("+%d 分", rec.Points)
		}
		res := ""
		if rec.Winner == rec.Leader {
			res = fmt.Sprintf("  → %s 首出即最大", game.SeatName(rec.Winner))
		} else if winAction != "" {
			res = fmt.Sprintf("  → %s %s", game.SeatName(rec.Winner), winAction)
		} else {
			res = fmt.Sprintf("  → %s 赢", game.SeatName(rec.Winner))
		}
		if combo != "" {
			res += combo
		}
		res += "，" + pts
		add(res, ColWin)
		add("", ColMuted)
	}
	if r.Phase == game.PhasePlay {
		trick(r.TrickNo, r.Leader, r.Plays, true)
		add("", ColMuted)
	}
	if r.Phase == game.PhaseDone {
		add("本局结束", ColMuted)
		pts := fmt.Sprintf("  闲家 %d 分 · 底牌 ×%d（%d 分）", r.DefenderPoints, r.BottomMult, r.BottomPoints)
		add(pts, ColWin)
	}
	if len(lines) == 0 {
		add("还没有出牌记录", ColMuted)
	}
	return lines
}

func (a *App) drawLog(c *gfx.Canvas) {
	a.drawBackground(c)
	panel(c, 14, 10, a.W-28, a.H-20, 14, gfx.RGB(0x05, 0x18, 0x12))
	c.Text(fonts.lg, 36, 18, "出牌日志", ColGold)
	if a.G == nil || a.G.Current == nil {
		return
	}
	r := a.G.Current
	trump := "无主"
	if r.Trump != game.NoTrump {
		trump = game.SuitName(r.Trump) + game.LevelName(r.Level)
	}
	c.Text(fonts.sm, 36, 66, fmt.Sprintf("第 %d 局 · 庄家 %s", a.G.RoundNo, game.SeatName(r.Dealer)), ColCream)
	c.Text(fonts.sm, 36, 94, fmt.Sprintf("主牌 %s · 闲家 %d / 80 分", trump, r.DefenderPoints), ColMuted)
	c.HLine(36, 122, a.W-72, gfx.RGBA(0xE9, 0xC4, 0x6A, 60))

	lines := a.logLines()
	a.logMax = len(lines) - logVisible
	if a.logMax < 0 {
		a.logMax = 0
	}
	if a.logScroll > a.logMax {
		a.logScroll = a.logMax
	}
	if a.logScroll < 0 {
		a.logScroll = 0
	}
	for i := 0; i < logVisible && a.logScroll+i < len(lines); i++ {
		ln := lines[a.logScroll+i]
		if ln.text == "" {
			continue
		}
		c.Text(fonts.sm, logBodyX, logBodyY+i*logLineH, ln.text, ln.col)
	}

	if len(lines) > logVisible {
		trackH := logVisible * logLineH
		c.FillRect(logScrollX, logBodyY, logScrollW, trackH, gfx.RGBA(0, 0, 0, 90))
		thumb := trackH * logVisible / len(lines)
		if thumb < 24 {
			thumb = 24
		}
		off := 0
		if a.logMax > 0 {
			off = (trackH - thumb) * a.logScroll / a.logMax
		}
		c.FillRect(logScrollX, logBodyY+off, logScrollW, thumb, ColGoldDim)
	}

	c.HLine(36, logFooterY-12, a.W-72, gfx.RGBA(0xE9, 0xC4, 0x6A, 40))
	c.TextCenter(fonts.xs, a.W/2, logFooterY, "↑↓ 滚动    ←→ 翻页    L / B 返回", ColMuted)
}
