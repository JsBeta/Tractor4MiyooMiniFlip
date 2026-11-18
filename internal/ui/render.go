package ui

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"tractor/internal/game"
	"tractor/internal/gfx"
)

func panel(c *gfx.Canvas, x, y, w, h, r int, fill gfx.Color) {
	c.RoundRect(x, y, w, h, r, fill)
	c.StrokeRoundRect(x, y, w, h, r, 1, gfx.RGBA(0xE9, 0xC4, 0x6A, 95))
}

func (a *App) pulse() float64 {
	return 0.5 + 0.5*math.Sin(float64(a.now.UnixNano())/1e9*4)
}

func (a *App) ensureBG() *gfx.Canvas {
	if a.bg != nil {
		return a.bg
	}
	bg := gfx.New(a.W, a.H)
	bg.VGradient(0, 0, a.W, a.H, ColFeltTop, ColFeltBottom)
	cx, cy := float64(a.W)/2, float64(a.H)/2
	for y := 0; y < a.H; y++ {
		for x := 0; x < a.W; x++ {
			dx := (float64(x) - cx) / cx
			dy := (float64(y) - cy) / cy
			d := math.Hypot(dx*0.72, dy)
			if d > 0.60 {
				al := uint8(math.Min(110, (d-0.60)*150))
				bg.SetPixel(x, y, gfx.RGBA(0, 0, 0, al))
			}
			if d < 0.55 {
				al := uint8((0.55 - d) * 22)
				bg.SetPixel(x, y, gfx.RGBA(0xFF, 0xF4, 0xD0, al))
			}
		}
	}
	a.bg = bg
	return bg
}

func (a *App) drawBackground(c *gfx.Canvas) {
	bg := a.ensureBG()
	copy(c.Pix, bg.Pix)
}

func (a *App) drawHeader(c *gfx.Canvas) {
	r := a.G.Current
	c.FillRect(0, 0, a.W, 46, gfx.RGBA(0x04, 0x16, 0x11, 0xEC))
	c.HLine(0, 45, a.W, gfx.RGBA(0xE9, 0xC4, 0x6A, 90))
	c.Text(fonts.md, 16, 9, fmt.Sprintf("第 %d 局", a.G.RoundNo), ColCream)
	c.VLine(104, 10, 26, gfx.RGBA(0xE9, 0xC4, 0x6A, 45))

	c.Text(fonts.xs, 120, 6, "我方", ColMuted)
	c.Text(fonts.lg, 154, 3, game.LevelName(a.G.Levels[0]), teamColor(a.G, 0))
	c.Text(fonts.xs, 196, 6, "对方", ColMuted)
	c.Text(fonts.lg, 230, 3, game.LevelName(a.G.Levels[1]), teamColor(a.G, 1))

	c.Text(fonts.sm, 316, 11, "主牌", ColMuted)
	if r.Trump == game.NoTrump {
		c.Text(fonts.lg, 360, 3, "无主", ColGold)
	} else {
		drawSuit(c, r.Trump, 370, 23, 20, suitColorOnDark(r.Trump))
		c.Text(fonts.lg, 384, 3, game.LevelName(r.Level), ColGold)
	}

	score := r.DefenderPoints
	c.Text(fonts.sm, 588, 10, "闲家", ColMuted)
	c.Text(fonts.lg, 624, 2, strconv.Itoa(score), ColGold)
	c.Text(fonts.sm, 700, 12, "/80", ColMuted)
	c.FillRect(590, 38, 146, 4, gfx.RGBA(0, 0, 0, 120))
	pct := float64(score) / 80
	if pct > 1 {
		pct = 1
	}
	c.FillRect(590, 38, int(float64(146)*pct), 4, ColGold)
}

func teamColor(g *game.Game, team int) gfx.Color {
	if game.TeamOf(g.Dealer) == team {
		return ColGold
	}
	return ColCream
}

func (a *App) drawSeat(c *gfx.Canvas, seat, x, y, w, h int) {
	r := a.G.Current
	active := false
	if r.Phase == game.PhasePlay && r.Turn == seat {
		active = true
	}
	if r.Phase == game.PhaseBidding && r.BidSeat() == seat {
		active = true
	}
	if r.Phase == game.PhaseBury && r.Dealer == seat {
		active = true
	}
	panel(c, x, y, w, h, 10, gfx.RGB(0x08, 0x22, 0x1A))
	if active {
		glow := uint8(90 + 90*a.pulse())
		c.StrokeRoundRect(x, y, w, h, 10, 2, gfx.RGBA(0xE9, 0xC4, 0x6A, glow))
	}
	name := game.SeatName(seat)
	c.Text(fonts.md, x+9, y+3, name, ColCream)
	role := "对手"
	if game.TeamOf(seat) == 0 {
		role = "队友"
	}
	c.Text(fonts.xs, x+10, y+31, role, ColMuted)

	count := strconv.Itoa(len(r.Hands[seat]))
	c.TextRight(fonts.lg, x+w-9, y+h-fonts.lg.Line-2, count, ColCream)
	for i := 0; i < 3; i++ {
		if i*3+1 > len(r.Hands[seat]) {
			break
		}
		DrawCardBack(c, x+9+i*11, y+h-30, 14, 21, false)
	}
	if r.Dealer == seat {
		c.Circle(x+w-16, y+15, 11, ColGold)
		centerGlyph(c, fonts.xs, x+w-16, y+15, '庄', gfx.RGB(0x2A, 0x1C, 0x08))
	}
}

// centerGlyph draws one rune with its ink box (not its advance box) centred on
// cx,cy. Chinese glyphs have asymmetric bearings, so Text/TextCenter alone
// leaves badges like 庄 visibly off centre.
func centerGlyph(c *gfx.Canvas, f *gfx.Font, cx, cy int, r rune, col gfx.Color) {
	if g, ok := f.Glyphs[r]; ok {
		c.Text(f, cx-g.W/2-g.OX, cy-g.H/2-g.OY, string(r), col)
		return
	}
	c.TextCenter(f, cx, cy-f.Line/2, string(r), col)
}

// textInk returns the ink bounding box of s relative to the text origin. The
// bitmap glyphs carry asymmetric bearings, so centring the advance box leaves
// Chinese menu labels visibly high or low.
func textInk(f *gfx.Font, s string) (x0, y0, x1, y1 int) {
	pen := 0
	x0, y0 = 1<<30, 1<<30
	x1, y1 = -1<<30, -1<<30
	for _, r := range s {
		g, ok := f.Glyphs[r]
		if !ok {
			if r == ' ' {
				pen += f.Size / 3
			} else {
				pen += f.Size
			}
			continue
		}
		if g.W > 0 {
			left, top := pen+g.OX, g.OY
			if left < x0 {
				x0 = left
			}
			if top < y0 {
				y0 = top
			}
			if left+g.W > x1 {
				x1 = left + g.W
			}
			if top+g.H > y1 {
				y1 = top + g.H
			}
		}
		pen += g.ADV
	}
	if x1 < x0 {
		return 0, 0, 0, 0
	}
	return x0, y0, x1, y1
}

// drawTextInBox draws s vertically centred by its ink box. When centered is
// true the ink box is also centred horizontally inside the box; otherwise x is
// taken as the desired left edge of the ink.
func drawTextInBox(c *gfx.Canvas, f *gfx.Font, x, y, w, h int, s string, col gfx.Color, centered bool) {
	ix0, iy0, ix1, iy1 := textInk(f, s)
	iw, ih := ix1-ix0, iy1-iy0
	if iw <= 0 || ih <= 0 {
		c.Text(f, x, y+(h-f.Line)/2, s, col)
		return
	}
	tx := x - ix0
	if centered {
		tx = x + (w-iw)/2 - ix0
	}
	ty := y + (h-ih)/2 - iy0
	c.Text(f, tx, ty, s, col)
}

func (a *App) drawTable(c *gfx.Canvas) {
	a.drawBackground(c)
	if a.G == nil || a.G.Current == nil {
		return
	}
	r := a.G.Current
	a.drawHeader(c)
	a.drawSeat(c, 2, 276, 50, 200, 62)
	a.drawSeat(c, 3, 10, 148, 104, 92)
	a.drawSeat(c, 1, 638, 148, 104, 92)
	a.drawTrick(c)
	a.drawPrompt(c)
	a.drawHand(c)
	a.drawHintBar(c)
	a.drawToast(c)
	_ = r
}

func (a *App) drawTrick(c *gfx.Canvas) {
	r := a.G.Current
	var plays [4][]game.Card
	winner := -1
	holding := a.shown != nil
	if holding {
		plays = a.shown.Plays
		winner = a.shown.Winner
	} else {
		plays = r.Plays
		if r.Best >= 0 {
			winner = r.Best
		}
	}
	anchors := [4][2]int{{376, 252}, {584, 182}, {376, 110}, {168, 182}}
	for seat := 0; seat < 4; seat++ {
		cs := plays[seat]
		if len(cs) == 0 {
			continue
		}
		cw, ch, step := 48, 66, 27
		total := (len(cs)-1)*step + cw
		x := anchors[seat][0] - total/2
		y := anchors[seat][1]
		if !holding && !a.playAt[seat].IsZero() {
			el := a.now.Sub(a.playAt[seat])
			if el < 180*time.Millisecond {
				y += int(12 * (1 - float64(el)/(180*float64(time.Millisecond))))
			}
		}
		for i, card := range cs {
			st := CardStyle{Compact: true}
			if winner == seat {
				st.Highlight = true
			}
			DrawCard(c, x+i*step, y, cw, ch, card, r.Trump, r.Level, st)
		}
	}
	// Centre plate.
	panel(c, 296, 184, 160, 60, 10, gfx.RGB(0x06, 0x1B, 0x14))
	pts := 0
	for i := 0; i < 4; i++ {
		pts += game.PointSum(plays[i])
	}
	title := fmt.Sprintf("第 %d 墩", r.TrickNo)
	body := "—"
	bodyCol := ColMuted
	if holding {
		title = fmt.Sprintf("%s 赢墩", game.SeatName(winner))
		body = fmt.Sprintf("+%d 分", pts)
		bodyCol = ColGold
	} else if pts > 0 {
		body = fmt.Sprintf("%d 分", pts)
		bodyCol = ColGold
	}
	c.TextCenter(fonts.xs, 376, 190, title, ColMuted)
	c.TextCenter(fonts.md, 376, 207, body, bodyCol)
}

func (a *App) drawPrompt(c *gfx.Canvas) {
	r := a.G.Current
	msg := ""
	col := ColCream
	switch r.Phase {
	case game.PhaseBidding:
		if r.BidSeat() == 0 {
			msg = "轮到你亮主"
			col = ColGold
		} else {
			msg = fmt.Sprintf("等待 %s 亮主…", game.SeatName(r.BidSeat()))
		}
	case game.PhaseBury:
		if r.Dealer == 0 {
			msg = fmt.Sprintf("扣底：已选 %d / 8 张", len(a.selected))
			col = ColGold
		} else {
			msg = fmt.Sprintf("%s 正在扣底…", game.SeatName(r.Dealer))
		}
	case game.PhasePlay:
		if r.Turn == 0 {
			msg = "轮到你出牌"
			col = ColGold
		} else {
			msg = fmt.Sprintf("等待 %s 出牌…", game.SeatName(r.Turn))
		}
		if a.shown != nil {
			msg = fmt.Sprintf("%s 赢得本墩", game.SeatName(a.shown.Winner))
		}
	case game.PhaseDone:
		msg = "本局结束"
	}
	if a.now.Before(a.errUntil) && a.errMsg != "" {
		msg = a.errMsg
		col = gfx.RGB(0xFF, 0x8A, 0x7A)
	}
	panel(c, 136, 338, 480, 40, 10, gfx.RGB(0x05, 0x18, 0x12))
	c.TextCenter(fonts.md, 376, 337, msg, col)
}

func (a *App) handSlots() []struct {
	Card game.Card
	X, Y int
	W, H int
} {
	hand := a.Hand()
	type slot = struct {
		Card game.Card
		X, Y int
		W, H int
	}
	var out []slot
	if len(hand) == 0 {
		return nil
	}
	r := a.G.Current
	if r.Phase == game.PhaseBury && len(hand) > 25 {
		cw, ch, step := 50, 70, 42
		perRow := 17
		for i, card := range hand {
			row := i / perRow
			col := i % perRow
			n := perRow
			if row == 1 {
				n = len(hand) - perRow
			}
			total := (n-1)*step + cw
			x := (a.W-total)/2 + col*step
			y := 382 + row*76
			out = append(out, slot{card, x, y, cw, ch})
		}
		return out
	}
	cw, ch, step := 74, 102, 27
	gap := 4
	total := (len(hand)-1)*step + cw
	prev := game.Group(hand[0], r.Trump, r.Level)
	for i := 1; i < len(hand); i++ {
		g := game.Group(hand[i], r.Trump, r.Level)
		if g != prev {
			total += gap
		}
		prev = g
	}
	x := (a.W - total) / 2
	prev = game.Group(hand[0], r.Trump, r.Level)
	mid := float64(len(hand)-1) / 2
	for i, card := range hand {
		g := game.Group(card, r.Trump, r.Level)
		if i > 0 && g != prev {
			x += gap
		}
		prev = g
		d := math.Abs(float64(i)-mid) / math.Max(mid, 1)
		y := 410 + int(7*d*d)
		if a.selected[card.ID] {
			y -= 14
		}
		out = append(out, slot{card, x, y, cw, ch})
		x += step
	}
	return out
}

func (a *App) drawHand(c *gfx.Canvas) {
	slots := a.handSlots()
	if len(slots) == 0 {
		return
	}
	r := a.G.Current
	bury := r.Phase == game.PhaseBury && len(slots) > 25
	if !bury {
		c.RoundRect(6, 390, a.W-12, 140, 14, gfx.RGB(0x0A, 0x26, 0x1C))
		c.HLine(20, 390, a.W-40, gfx.RGBA(0xE9, 0xC4, 0x6A, 36))
		// Player label first, so the cards overlap it rather than the other way
		// round when the fan reaches the top of the tray.
		lx := 14
		c.Text(fonts.xs, lx, 393, "你 · 南", ColMuted)
		if r.Dealer == 0 {
			c.Circle(lx+74, 401, 9, ColGold)
			centerGlyph(c, fonts.xs, lx+74, 401, '庄', gfx.RGB(0x2A, 0x1C, 0x08))
		}
	}
	showLevel := r.Phase == game.PhaseBidding && r.BidSeat() == 0
	// Draw unselected first so selected cards sit on top.
	for pass := 0; pass < 2; pass++ {
		for i, s := range slots {
			sel := a.selected[s.Card.ID]
			if (pass == 0) == sel {
				continue
			}
			st := CardStyle{Selected: sel, Compact: bury}
			if i == a.cursor {
				st.Cursor = true
			}
			if showLevel && s.Card.Rank == r.Level && !s.Card.IsJoker() {
				st.Highlight = true
			}
			DrawCard(c, s.X, s.Y, s.W, s.H, s.Card, r.Trump, r.Level, st)
			if i == a.cursor {
				cx := s.X + s.W/2
				c.Polyg([]gfx.Point{
					{X: float64(cx) - 6, Y: float64(s.Y) - 4},
					{X: float64(cx) + 6, Y: float64(s.Y) - 4},
					{X: float64(cx), Y: float64(s.Y) + 4},
				}, ColGold)
			}
		}
	}
}

func (a *App) drawHintBar(c *gfx.Canvas) {
	r := a.G.Current
	c.FillRect(0, 532, a.W, 28, gfx.RGBA(0x03, 0x10, 0x0C, 0xE6))
	c.HLine(0, 532, a.W, gfx.RGBA(0xE9, 0xC4, 0x6A, 50))
	var s string
	switch r.Phase {
	case game.PhaseBidding:
		s = "← → 选牌    A 选择    X 亮主    B 不亮    Y 自动选级牌    L 日志"
	case game.PhaseBury:
		s = "方向键 选牌    A 选择    X 确认扣底    B 清空选择    L 日志"
	case game.PhasePlay:
		s = "← → 选牌    A 选择    X 出牌    Y 提示    B 取消    START 菜单    L 日志"
	case game.PhaseDone:
		s = "本局结束    L 日志"
	}
	c.TextCenter(fonts.xs, a.W/2, 537, s, gfx.RGBA(0xF7, 0xF1, 0xE0, 200))
}

func (a *App) drawToast(c *gfx.Canvas) {
	if a.toast == "" || a.now.After(a.toastUntil) {
		return
	}
	w := fonts.md.Measure(a.toast) + 36
	x := (a.W - w) / 2
	panel(c, x, 64, w, 36, 8, gfx.RGB(0x06, 0x19, 0x13))
	c.TextCenter(fonts.md, a.W/2, 70, a.toast, ColGold)
}
