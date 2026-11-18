package ui

import (
	"tractor/internal/game"
	"tractor/internal/gfx"
)

// Palette.
var (
	ColFeltTop    = gfx.RGB(0x1b, 0x5a, 0x44)
	ColFeltBottom = gfx.RGB(0x08, 0x24, 0x1b)
	ColInk        = gfx.RGB(0x07, 0x14, 0x10)
	ColPanel      = gfx.RGBA(0x06, 0x1a, 0x14, 0xE6)
	ColPanelSoft  = gfx.RGBA(0x0a, 0x24, 0x1c, 0xB8)
	ColGold       = gfx.RGB(0xE9, 0xC4, 0x6A)
	ColGoldDim    = gfx.RGB(0x8C, 0x71, 0x39)
	ColCream      = gfx.RGB(0xF7, 0xF1, 0xE0)
	ColMuted      = gfx.RGB(0x9F, 0xB6, 0xA8)
	ColRed        = gfx.RGB(0xC8, 0x3B, 0x2E)
	ColCard       = gfx.RGB(0xFB, 0xF7, 0xEB)
	ColCardEdge   = gfx.RGB(0xD8, 0xCE, 0xB4)
	ColBlack      = gfx.RGB(0x1C, 0x1C, 0x1C)
	ColGreen      = gfx.RGB(0x2E, 0x8B, 0x63)
	ColWin        = gfx.RGB(0xFF, 0xD9, 0x7A)
)

// CardStyle controls how a card is drawn.
type CardStyle struct {
	Selected  bool
	Cursor    bool
	Dim       bool
	Highlight bool
	Compact   bool
	FaceDown  bool
}

func suitColor(s game.Suit) gfx.Color {
	switch s {
	case game.Heart, game.Diamond:
		return ColRed
	default:
		return ColBlack
	}
}

// suitColorOnDark keeps black suits (spade/club) readable on the dark table and
// header; on cream cards the normal suitColor is used instead.
func suitColorOnDark(s game.Suit) gfx.Color {
	if s == game.Spade || s == game.Club {
		return ColCream
	}
	return ColRed
}

const spritePad = 4

type spriteKey struct {
	suit    game.Suit
	rank    game.Rank
	trump   game.Suit
	level   game.Rank
	w, h    int
	compact bool
}

var cardSprites = map[spriteKey]*gfx.Canvas{}

// ResetCardCache drops cached card faces; call it when the trump changes.
func ResetCardCache() { cardSprites = map[spriteKey]*gfx.Canvas{} }

func cardSprite(card game.Card, trump game.Suit, level game.Rank, w, h int, compact bool) *gfx.Canvas {
	k := spriteKey{card.Suit, card.Rank, trump, level, w, h, compact}
	if s, ok := cardSprites[k]; ok {
		return s
	}
	s := gfx.New(w+spritePad*2, h+spritePad*2)
	drawCardBase(s, spritePad, spritePad, w, h, card, trump, level, compact)
	cardSprites[k] = s
	return s
}

// DrawCard draws one face up card at the given rectangle.
func DrawCard(c *gfx.Canvas, x, y, w, h int, card game.Card, trump game.Suit, level game.Rank, st CardStyle) {
	r := 6
	if st.Compact {
		r = 5
	}
	if st.Selected || st.Cursor || st.Highlight {
		glow := gfx.RGBA(0xE9, 0xC4, 0x6A, 0x66)
		if st.Highlight {
			glow = gfx.RGBA(0xFF, 0xD9, 0x7A, 0xAA)
		}
		c.RoundRect(x-3, y-3, w+6, h+6, r+3, glow)
	}
	s := cardSprite(card, trump, level, w, h, st.Compact)
	c.Blit(s, x-spritePad, y-spritePad)
	if st.Cursor {
		c.StrokeRoundRect(x, y, w, h, r, 2, ColGold)
	} else if st.Selected {
		c.StrokeRoundRect(x, y, w, h, r, 1, gfx.RGBA(0xE9, 0xC4, 0x6A, 200))
	}
}

func drawCardBase(c *gfx.Canvas, x, y, w, h int, card game.Card, trump game.Suit, level game.Rank, compact bool) {
	r := 6
	if compact {
		r = 5
	}
	c.Shadow(x, y, w, h, r)
	c.RoundRect(x, y, w, h, r, ColCard)
	c.StrokeRoundRect(x, y, w, h, r, 1, ColCardEdge)

	col := suitColor(card.Suit)
	if card.IsJoker() {
		if card.Rank == game.RankBigJoker {
			col = ColRed
		} else {
			col = ColBlack
		}
	}

	if game.IsTrump(card, trump, level) {
		c.RoundRect(x+4, y+3, w-8, 3, 1, gfx.RGBA(0xE9, 0xC4, 0x6A, 0xCC))
	}

	rankFont := fonts.sm
	sym := 24
	cxSym, cySym := 12, 46
	center := 42
	if compact {
		sym = 15
		cxSym, cySym = 10, 30
		center = 26
	}

	if card.IsJoker() {
		mark := "小"
		if card.Rank == game.RankBigJoker {
			mark = "大"
		}
		switch {
		case compact:
			// 48x66 table card: two short lines read better than squeezing the
			// whole word onto one.
			c.TextCenter(fonts.xs, x+w/2, y+2, "JOK", col)
			c.TextCenter(fonts.xs, x+w/2, y+20, "ER", col)
			c.TextCenter(fonts.xs, x+w/2, y+38, mark, col)
		case w >= 90:
			// Large stand-alone card (title art): same vertical word, with the
			// 大/小 mark sitting in the middle of the card.
			cx := x + 18
			ly := y + 8
			for _, r := range "JOKER" {
				c.TextCenter(fonts.md, cx, ly, string(r), col)
				ly += 23
			}
			c.TextCenter(fonts.lg, x+w/2, y+h/2-26, mark, col)
		default:
			// Fanned hand: only the left 23 px of the card stay visible, so
			// stack the letters inside that strip. The big 大/小 sits in the
			// middle of the card and is covered by the next card in the fan.
			cx := x + 12
			ly := y + 2
			for _, r := range "JOKER" {
				c.TextCenter(fonts.xs, cx, ly, string(r), col)
				ly += 14
			}
			c.TextCenter(fonts.lg, x+w/2, y+h/2-26, mark, col)
		}
		return
	}

	drawRank(c, rankFont, x+1, y+2, card.Rank.String(), col, !compact)
	drawSuit(c, card.Suit, x+cxSym, y+cySym, sym, col)
	if compact {
		// Trick cards sit apart from each other, so a big centre pip is the
		// fastest way to read the suit at a glance.
		drawSuit(c, card.Suit, x+w/2, y+h-17, center, col)
	} else {
		// Fanned hand: only the left edge stays visible, so keep a single pip in
		// that strip; the soft watermark only shows on the last, fully drawn card.
		drawSuit(c, card.Suit, x+w/2+10, y+h/2+16, center, gfx.RGBA(col.R(), col.G(), col.B(), 34))
	}
}

// drawRank prints a rank in the top-left index. "10" is the only two-glyph
// rank, and its normal advance is wider than the visible strip of a fanned
// card, so the two digits are set slightly closer together.
func drawRank(c *gfx.Canvas, f *gfx.Font, x, y int, s string, col gfx.Color, tight bool) {
	if tight && s == "10" {
		c.Text(f, x, y, "1", col)
		c.Text(f, x+10, y, "0", col)
		return
	}
	c.Text(f, x, y, s, col)
}

// DrawCardBack draws the back of a card.
func DrawCardBack(c *gfx.Canvas, x, y, w, h int, accent bool) {
	r := 6
	c.Shadow(x, y, w, h, r)
	top := gfx.RGB(0x6E, 0x22, 0x2C)
	bot := gfx.RGB(0x3A, 0x10, 0x18)
	c.RoundRect(x, y, w, h, r, bot)
	for i := 1; i < h-1; i++ {
		t := float64(i) / float64(h)
		col := gfx.RGBA(
			uint8(float64(top.R())*(1-t)+float64(bot.R())*t),
			uint8(float64(top.G())*(1-t)+float64(bot.G())*t),
			uint8(float64(top.B())*(1-t)+float64(bot.B())*t), 255)
		c.FillRect(x+1, y+i, w-2, 1, col)
	}
	step := 9
	for d := -h; d < w; d += step {
		for yy := 4; yy < h-4; yy++ {
			xx := d + yy
			if xx >= 4 && xx < w-4 {
				c.SetPixel(x+xx, y+yy, gfx.RGBA(0xE9, 0xC4, 0x6A, 22))
			}
		}
	}
	border := gfx.RGB(0xE9, 0xC4, 0x6A)
	if !accent {
		border = gfx.RGB(0xB9, 0x9A, 0x5E)
	}
	c.StrokeRoundRect(x, y, w, h, r, 2, border)
	c.Circle(x+w/2, y+h/2, min(w, h)/7, gfx.RGBA(0xE9, 0xC4, 0x6A, 60))
	c.StrokeCircle(x+w/2, y+h/2, min(w, h)/7, 1, gfx.RGBA(0xE9, 0xC4, 0x6A, 150))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// drawSuit paints a suit symbol centred at (cx, cy) with the given height.
func drawSuit(c *gfx.Canvas, s game.Suit, cx, cy, size int, col gfx.Color) {
	f := float64(size)
	switch s {
	case game.Diamond:
		pts := []gfx.Point{
			{X: float64(cx), Y: float64(cy) - f*0.50},
			{X: float64(cx) + f*0.38, Y: float64(cy)},
			{X: float64(cx), Y: float64(cy) + f*0.50},
			{X: float64(cx) - f*0.38, Y: float64(cy)},
		}
		c.Polyg(pts, col)
	case game.Heart:
		r := int(f * 0.27)
		c.Circle(cx-int(f*0.23), cy-int(f*0.13), r, col)
		c.Circle(cx+int(f*0.23), cy-int(f*0.13), r, col)
		pts := []gfx.Point{
			{X: float64(cx) - f*0.48, Y: float64(cy) - f*0.06},
			{X: float64(cx) + f*0.48, Y: float64(cy) - f*0.06},
			{X: float64(cx), Y: float64(cy) + f*0.5},
		}
		c.Polyg(pts, col)
	case game.Spade:
		r := int(f * 0.27)
		c.Circle(cx-int(f*0.23), cy+int(f*0.13), r, col)
		c.Circle(cx+int(f*0.23), cy+int(f*0.13), r, col)
		pts := []gfx.Point{
			{X: float64(cx) - f*0.48, Y: float64(cy) + f*0.06},
			{X: float64(cx) + f*0.48, Y: float64(cy) + f*0.06},
			{X: float64(cx), Y: float64(cy) - f*0.5},
		}
		c.Polyg(pts, col)
		stem := []gfx.Point{
			{X: float64(cx) - f*0.07, Y: float64(cy) + f*0.28},
			{X: float64(cx) + f*0.07, Y: float64(cy) + f*0.28},
			{X: float64(cx) + f*0.18, Y: float64(cy) + f*0.5},
			{X: float64(cx) - f*0.18, Y: float64(cy) + f*0.5},
		}
		c.Polyg(stem, col)
	case game.Club:
		r := int(f * 0.24)
		c.Circle(cx, cy-int(f*0.25), r, col)
		c.Circle(cx-int(f*0.24), cy+int(f*0.08), r, col)
		c.Circle(cx+int(f*0.24), cy+int(f*0.08), r, col)
		stem := []gfx.Point{
			{X: float64(cx) - f*0.07, Y: float64(cy) + f*0.24},
			{X: float64(cx) + f*0.07, Y: float64(cy) + f*0.24},
			{X: float64(cx) + f*0.18, Y: float64(cy) + f*0.5},
			{X: float64(cx) - f*0.18, Y: float64(cy) + f*0.5},
		}
		c.Polyg(stem, col)
	}
}
