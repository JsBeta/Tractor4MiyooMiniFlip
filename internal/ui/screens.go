package ui

import (
	"fmt"
	"strconv"
	"strings"

	"tractor/internal/game"
	"tractor/internal/gfx"
)

var pauseItems = []string{"继续游戏", "保存进度", "重新开始", "玩法说明", "返回主菜单"}

// titleItems includes the resume entry only when a save is present.
func (a *App) titleItems() []string {
	if hasSavedGame() {
		return []string{"继续上次牌局", "开始新牌局", "玩法说明", "按键说明", "退出游戏"}
	}
	return []string{"开始新牌局", "玩法说明", "按键说明", "退出游戏"}
}

func (a *App) drawTitle(c *gfx.Canvas) {
	a.drawBackground(c)

	// Decorative cards on the right.
	c.RoundRect(480, 152, 246, 330, 18, gfx.RGB(0x08, 0x20, 0x19))
	c.StrokeRoundRect(480, 152, 246, 330, 18, 1, gfx.RGBA(0xE9, 0xC4, 0x6A, 46))
	cards := []game.Card{
		{Suit: game.Spade, Rank: game.RankA},
		{Suit: game.Heart, Rank: game.RankK},
		{Suit: game.Club, Rank: game.RankQ},
		{Suit: game.SuitJoker, Rank: game.RankBigJoker},
	}
	for i, card := range cards {
		x := 500 + (i%2)*106
		y := 170 + (i/2)*150
		DrawCard(c, x, y, 96, 136, card, game.Spade, game.Rank2, CardStyle{})
	}

	c.TextShadow(fonts.xl, 56, 110, "拖拉机", ColGold)
	c.Text(fonts.md, 60, 178, "升级 · 双扣", ColCream)
	c.FillRect(60, 214, 250, 2, gfx.RGBA(0xE9, 0xC4, 0x6A, 110))
	c.Text(fonts.xs, 60, 226, "四人两副牌 · 抢分升级", ColMuted)

	items := a.titleItems()
	start := 280
	if len(items) > 4 {
		start = 260
	}
	for i, item := range items {
		top := start + i*46 - 4
		const rowH = 40
		if i == a.menu {
			w := fonts.md.Measure(item) + 52
			c.RoundRect(48, top, w, rowH, 8, gfx.RGBA(0xE9, 0xC4, 0x6A, 34))
			c.FillRect(48, top, 4, rowH, ColGold)
			drawTextInBox(c, fonts.md, 66, top, w-18, rowH, item, ColGold, false)
			ax, cy := 48+w-26, top+rowH/2
			c.Polyg([]gfx.Point{
				{X: float64(ax), Y: float64(cy - 8)},
				{X: float64(ax + 12), Y: float64(cy)},
				{X: float64(ax), Y: float64(cy + 8)},
			}, ColGold)
		} else {
			drawTextInBox(c, fonts.md, 66, top, 0, rowH, item, ColCream, false)
		}
	}
	if a.errMsg != "" && a.now.Before(a.errUntil) {
		c.TextCenter(fonts.sm, a.W/2, 496, a.errMsg, ColRed)
	}
	c.Text(fonts.xs, 56, 528, "Miyoo Mini Flip · 752 × 560", gfx.RGBA(0x9F, 0xB6, 0xA8, 150))
}

func (a *App) drawPause(c *gfx.Canvas) {
	c.FillRect(0, 0, a.W, a.H, gfx.RGBA(0, 0, 0, 170))
	panel(c, 196, 118, 360, 356, 14, gfx.RGB(0x06, 0x1A, 0x14))
	c.TextCenter(fonts.lg, a.W/2, 136, "暂 停", ColGold)
	for i, item := range pauseItems {
		top := 186 + i*46
		const rowH = 40
		if i == a.pauseMenu {
			w := fonts.md.Measure(item) + 56
			x := (a.W - w) / 2
			c.RoundRect(x, top, w, rowH, 8, gfx.RGBA(0xE9, 0xC4, 0x6A, 36))
			drawTextInBox(c, fonts.md, x, top, w, rowH, item, ColGold, true)
		} else {
			drawTextInBox(c, fonts.md, 0, top, a.W, rowH, item, ColCream, true)
		}
	}
	if a.toast != "" && a.now.Before(a.toastUntil) {
		c.TextCenter(fonts.sm, a.W/2, 438, a.toast, ColGold)
	}
}

var helpPages = [][]string{
	{
		"【目标】",
		"两副牌共 108 张，四人两两结对。每人 25 张，余下 8 张为底牌。",
		"庄家一方守分，闲家一方抢分：5 记 5 分，10 和 K 各记 10 分，全场共 200 分。",
		"",
		"【升级】",
		"闲家拿到 80 分即可上台；不足 80 分则庄家一方升级。",
		"0 分大光升 3 级，40 分以下小光升 2 级，80 分以下升 1 级。",
		"闲家得分每多 40 分，上台后多升 1 级。",
	},
	{
		"【主牌】",
		"大小王、所有级牌和主花色都是主牌。",
		"单张大小：大王 > 小王 > 主级牌 > 副级牌 > 主花色 A K Q J 10 … 2。",
		"",
		"【牌型】",
		"对子：两张完全相同的牌。",
		"拖拉机：同花色相邻的两对，例如 3344；级牌会截断顺子。",
		"甩牌：一次打出同花色多张牌，但每一张都必须是场上最大的，否则甩牌失败。",
	},
	{
		"【跟牌】",
		"必须先跟同花色；有对子必须出对子，有拖拉机必须出拖拉机。",
		"没有该花色时，才可以垫别的牌，或者用主牌毙牌。",
		"",
		"【亮主与扣底】",
		"拿到级牌即可亮主，一对级牌可以反主；无人亮主则打无主。",
		"庄家拿起底牌 8 张，再扣 8 张回去。",
		"闲家赢下最后一墩时底牌分翻倍：单张 ×2、对子 ×4、拖拉机 ×8；",
		"甩牌按其结构：含拖拉机 ×8，含对子 ×4，否则 ×2。",
	},
}

func (a *App) drawHelp(c *gfx.Canvas) {
	a.drawBackground(c)
	panel(c, 22, 16, a.W-44, a.H-32, 14, gfx.RGB(0x05, 0x18, 0x12))
	title := "玩法说明"
	if a.helpPage == 2 {
		title = "按键说明"
	}
	c.Text(fonts.lg, 46, 32, title, ColGold)
	c.TextRight(fonts.sm, a.W-48, 40, fmt.Sprintf("%d / 3", a.helpPage+1), ColMuted)
	c.HLine(46, 74, a.W-92, gfx.RGBA(0xE9, 0xC4, 0x6A, 70))

	y := 92
	for _, line := range helpPages[a.helpPage] {
		if line == "" {
			y += 12
			continue
		}
		for _, sub := range wrapText(fonts.md, line, a.W-120) {
			col := ColCream
			if strings.HasPrefix(sub, "【") {
				col = ColGold
			}
			c.Text(fonts.md, 46, y, sub, col)
			y += 32
		}
		y += 4
	}
	c.TextCenter(fonts.xs, a.W/2, a.H-38, "← → 翻页    B 返回", ColMuted)
}

func (a *App) drawRoundEnd(c *gfx.Canvas) {
	res := a.res
	c.FillRect(0, 0, a.W, a.H, gfx.RGBA(0, 0, 0, 150))
	panel(c, 88, 64, a.W-176, 424, 16, gfx.RGB(0x0E, 0x2E, 0x24))
	c.RoundRect(90, 66, a.W-180, 6, 3, gfx.RGBA(0xE9, 0xC4, 0x6A, 90))
	title := "本局结束"
	if res.Winner >= 0 {
		title = "通关！"
	}
	c.TextCenter(fonts.lg, a.W/2, 84, title, ColGold)

	dealerTeam := game.TeamOf(res.Dealer)
	defTeam := 1 - dealerTeam
	rows := [][2]string{
		{"闲家得分", fmt.Sprintf("%d 分", res.DefenderPoints-res.BottomPoints)},
	}
	if res.BottomPoints > 0 {
		rows = append(rows, [2]string{fmt.Sprintf("底牌 ×%d", res.BottomMult), fmt.Sprintf("%d 分", res.BottomPoints)})
	}
	rows = append(rows, [2]string{"闲家合计", fmt.Sprintf("%d 分", res.DefenderPoints)})
	y := 150
	for _, row := range rows {
		c.Text(fonts.md, 150, y, row[0], ColMuted)
		c.TextRight(fonts.md, a.W-150, y, row[1], ColCream)
		y += 38
	}
	c.HLine(150, y+2, a.W-300, gfx.RGBA(0xE9, 0xC4, 0x6A, 50))
	y += 16

	var result string
	var accent gfx.Color = ColGold
	if res.Success {
		result = fmt.Sprintf("%s 守庄成功，升 %d 级", game.TeamName(dealerTeam), res.Upgrade)
		if res.BigWin {
			result = fmt.Sprintf("%s 大光，升 3 级！", game.TeamName(dealerTeam))
		} else if res.SmallWin {
			result = fmt.Sprintf("%s 小光，升 2 级！", game.TeamName(dealerTeam))
		}
	} else {
		if res.Upgrade > 0 {
			result = fmt.Sprintf("%s 上台，升 %d 级", game.TeamName(defTeam), res.Upgrade)
		} else {
			result = fmt.Sprintf("%s 上台", game.TeamName(defTeam))
		}
	}
	c.TextCenter(fonts.md, a.W/2, y, result, accent)
	y += 46

	lv := func(team int) string {
		return fmt.Sprintf("%s  %s → %s", teamNameShort(team), game.LevelName(a.oldLevels[team]), game.LevelName(a.G.Levels[team]))
	}
	c.TextCenter(fonts.sm, a.W/2, y, lv(0), ColCream)
	c.TextCenter(fonts.sm, a.W/2, y+26, lv(1), ColCream)

	c.TextCenter(fonts.sm, a.W/2, a.H-92, "X 继续", ColGold)
}

func teamNameShort(team int) string {
	if team == 0 {
		return "我方"
	}
	return "对方"
}

func (a *App) drawGameOver(c *gfx.Canvas) {
	a.drawBackground(c)
	winner := a.G.Winner
	if winner < 0 {
		winner = 0
	}
	c.Circle(a.W/2, 214, 168, gfx.RGBA(0xE9, 0xC4, 0x6A, 14))
	c.StrokeCircle(a.W/2, 214, 168, 2, gfx.RGBA(0xE9, 0xC4, 0x6A, 52))
	c.TextCenter(fonts.md, a.W/2, 76, "比赛结束", ColMuted)
	name := game.TeamName(winner) + " 获胜"
	c.TextShadow(fonts.xl, a.W/2-fonts.xl.Measure(name)/2, 124, name, ColGold)
	c.TextCenter(fonts.md, a.W/2, 246, fmt.Sprintf("我方 %s 级      对方 %s 级", game.LevelName(a.G.Levels[0]), game.LevelName(a.G.Levels[1])), ColCream)
	c.FillRect(a.W/2-130, 314, 260, 1, gfx.RGBA(0xE9, 0xC4, 0x6A, 80))
	c.TextCenter(fonts.md, a.W/2, 338, "X 再来一局", ColGold)
	c.TextCenter(fonts.sm, a.W/2, 382, "B 返回主菜单", ColMuted)
	suits := []game.Suit{game.Spade, game.Heart, game.Club, game.Diamond}
	for i, s := range suits {
		drawSuit(c, s, a.W/2-105+i*70, 480, 30, gfx.RGBA(0xE9, 0xC4, 0x6A, 90))
	}
}

func wrapText(f *gfx.Font, s string, width int) []string {
	var out []string
	var cur []rune
	w := 0
	for _, r := range s {
		rw := f.Measure(string(r))
		if w+rw > width && len(cur) > 0 {
			out = append(out, string(cur))
			cur = cur[:0]
			w = 0
		}
		cur = append(cur, r)
		w += rw
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	return out
}

var _ = strconv.Itoa
