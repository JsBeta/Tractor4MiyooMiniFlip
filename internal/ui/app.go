package ui

import (
	"fmt"
	"math/rand"
	"time"

	"tractor/internal/game"
	"tractor/internal/gfx"
)

// Key is a logical button.
type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeyA
	KeyB
	KeyX
	KeyY
	KeyL
	KeyR
	KeyStart
	KeySelect
)

// Screen is the active top level screen.
type Screen int

const (
	ScreenTitle Screen = iota
	ScreenTable
	ScreenHelp
	ScreenPause
	ScreenRoundEnd
	ScreenGameOver
	ScreenLog
)

type fontsT struct {
	xs, sm, md, lg, xl *gfx.Font
}

var fonts fontsT

// LoadFonts initialises the embedded bitmap fonts.
func LoadFonts() {
	fonts.xs = gfx.LoadFont("xs")
	fonts.sm = gfx.LoadFont("sm")
	fonts.md = gfx.LoadFont("md")
	fonts.lg = gfx.LoadFont("lg")
	fonts.xl = gfx.LoadFont("xl")
}

// FontSet is a read-only view of the loaded fonts, used by the diagnostics
// screen in the main package.
type FontSet struct {
	XS, SM, MD, LG, XL *gfx.Font
}

// Fonts returns the loaded fonts.
func Fonts() FontSet {
	return FontSet{XS: fonts.xs, SM: fonts.sm, MD: fonts.md, LG: fonts.lg, XL: fonts.xl}
}

// App holds all UI state.
type App struct {
	bg   *gfx.Canvas
	W, H int

	G      *game.Game
	Screen Screen
	Demo   bool // auto-play for attract mode / device checks

	// title menu
	menu int

	// table
	cursor   int
	selected map[int]bool
	now      time.Time
	nextAI   time.Time
	turnAt   time.Time
	lastTurn int

	toast      string
	toastUntil time.Time
	errMsg     string

	logScroll int
	logMax    int
	errUntil  time.Time

	playAt    [4]time.Time
	shown     *game.TrickRecord
	holdUntil time.Time

	// round result
	res       game.RoundResult
	oldLevels [2]game.Rank

	// help
	helpPage  int
	helpBack  Screen
	pauseMenu int

	Quit bool
}

// New creates the app.
func New(w, h int) *App {
	a := &App{W: w, H: h, Screen: ScreenTitle, selected: map[int]bool{}, helpBack: ScreenTitle}
	return a
}

// SetNow is used by the screenshot harness.
func (a *App) SetNow(t time.Time) { a.now = t }

// StartNewGame begins a fresh match.
func (a *App) StartNewGame(seed int64) {
	ResetCardCache()
	a.G = game.NewGame(seed)
	a.G.StartRound()
	a.Screen = ScreenTable
	a.cursor, a.selected = 0, map[int]bool{}
	a.shown = nil
	a.msg("", 0)
	a.nextAI = a.now.Add(700 * time.Millisecond)
	a.turnAt = time.Time{}
	a.saveGame(false)
}

// Tick advances timers and AI turns.
func (a *App) Tick(now time.Time) {
	a.now = now
	// Demo mode drives the human seat automatically and advances screens.
	if a.Demo {
		if a.Screen == ScreenRoundEnd || a.Screen == ScreenGameOver {
			a.Key(KeyX)
		}
		defer func() {
			if a.Screen == ScreenTable {
				a.AutoAct()
			}
		}()
	}
	if a.Screen != ScreenTable || a.G == nil {
		return
	}
	r := a.G.Current
	if a.shown != nil && now.After(a.holdUntil) {
		a.shown = nil
	}
	switch r.Phase {
	case game.PhaseBidding:
		seat := r.BidSeat()
		if seat == 0 {
			if !hasLevelCard(r.Hands[0], r.Level) {
				if a.turnAt.IsZero() {
					a.turnAt = now.Add(1200 * time.Millisecond)
				} else if now.After(a.turnAt) {
					r.PassBid(0)
					a.turnAt = time.Time{}
					a.afterBid()
				}
			}
			return
		}
		if now.Before(a.nextAI) {
			return
		}
		a.nextAI = now.Add(650 * time.Millisecond)
		if cards, ok := game.AIBid(r, seat); ok {
			if err := r.Bid(seat, cards); err == nil {
				a.toast = fmt.Sprintf("%s 亮主 %s%s", game.SeatName(seat), game.SuitName(r.Trump), game.LevelName(r.Level))
				a.toastUntil = now.Add(1600 * time.Millisecond)
			} else {
				r.PassBid(seat)
			}
		} else {
			r.PassBid(seat)
		}
		a.afterBid()

	case game.PhaseBury:
		if r.Dealer == 0 {
			return
		}
		if now.Before(a.nextAI) {
			return
		}
		a.nextAI = now.Add(700 * time.Millisecond)
		if err := game.AIAutoBury(r); err == nil {
			a.toast = fmt.Sprintf("%s 扣底完成", game.SeatName(r.Dealer))
			a.toastUntil = now.Add(1400 * time.Millisecond)
			a.cursor, a.selected = 0, map[int]bool{}
			a.saveGame(false)
		}

	case game.PhasePlay:
		if now.Before(a.holdUntil) {
			return
		}
		if r.Turn == 0 {
			return
		}
		if now.Before(a.nextAI) {
			return
		}
		cards, err := game.AIChoose(r, r.Turn)
		if err != nil {
			a.fail(err.Error())
			a.nextAI = now.Add(time.Second)
			return
		}
		a.nextAI = now.Add(500 * time.Millisecond)
		a.play(r.Turn, cards)

	case game.PhaseDone:
		if now.Before(a.nextAI) {
			return
		}
		a.finishRound()
	}
}

func (a *App) afterBid() {
	r := a.G.Current
	if r.Phase == game.PhaseBury {
		a.toast = fmt.Sprintf("%s 坐庄，主牌 %s%s", game.SeatName(r.Dealer), game.SuitName(r.Trump), game.LevelName(r.Level))
		if r.Trump == game.NoTrump {
			a.toast = fmt.Sprintf("%s 坐庄，无主", game.SeatName(r.Dealer))
		}
		a.toastUntil = a.now.Add(1800 * time.Millisecond)
		a.nextAI = a.now.Add(800 * time.Millisecond)
		a.cursor, a.selected = 0, map[int]bool{}
	}
	a.saveGame(false)
}

func hasLevelCard(hand []game.Card, level game.Rank) bool {
	for _, c := range hand {
		if c.Rank == level && !c.IsJoker() {
			return true
		}
	}
	return false
}

func (a *App) finishRound() {
	a.saveGame(false)
	a.oldLevels = a.G.Levels
	a.res = a.G.FinishRound()
	a.Screen = ScreenRoundEnd
	a.shown = nil
	a.cursor, a.selected = 0, map[int]bool{}
	if a.G.Winner >= 0 {
		clearSavedGame()
	}
}

func (a *App) fail(msg string) {
	a.errMsg = msg
	a.errUntil = a.now.Add(2200 * time.Millisecond)
}

func (a *App) msg(s string, d time.Duration) {
	a.toast = s
	a.toastUntil = a.now.Add(d)
}

func (a *App) play(seat int, cards []game.Card) {
	r := a.G.Current
	before := len(r.History)
	if err := r.Play(seat, cards); err != nil {
		a.fail(err.Error())
		return
	}
	a.playAt[seat] = a.now
	if seat == 0 && len(r.Hands[0]) > 0 && a.cursor >= len(r.Hands[0]) {
		a.cursor = len(r.Hands[0]) - 1
	}
	if len(r.History) > before {
		rec := r.History[len(r.History)-1]
		a.shown = &rec
		a.holdUntil = a.now.Add(1000 * time.Millisecond)
		a.nextAI = a.now.Add(1350 * time.Millisecond)
		if rec.Points > 0 {
			a.toast = fmt.Sprintf("本墩 %d 分", rec.Points)
			a.toastUntil = a.now.Add(1200 * time.Millisecond)
		}
	}
	if r.Phase == game.PhaseDone {
		a.nextAI = a.now.Add(1400 * time.Millisecond)
	}
	if len(r.History) > before || r.Phase == game.PhaseDone {
		a.saveGame(false)
	}
}

// Hand returns the human hand in display order.
func (a *App) Hand() []game.Card {
	if a.G == nil || a.G.Current == nil {
		return nil
	}
	return a.G.Current.SortedCopy(0)
}

// Key handles one button press.
func (a *App) Key(k Key) {
	switch a.Screen {
	case ScreenTitle:
		a.keyTitle(k)
	case ScreenTable:
		a.keyTable(k)
	case ScreenHelp:
		a.keyHelp(k)
	case ScreenPause:
		a.keyPause(k)
	case ScreenRoundEnd:
		a.keyRoundEnd(k)
	case ScreenGameOver:
		a.keyGameOver(k)
	case ScreenLog:
		a.keyLog(k)
	}
}

func (a *App) keyTitle(k Key) {
	items := a.titleItems()
	n := len(items)
	if a.menu >= n {
		a.menu = 0
	}
	resume := n == 5
	switch k {
	case KeyUp:
		a.menu = (a.menu + n - 1) % n
	case KeyDown:
		a.menu = (a.menu + 1) % n
	case KeyA, KeyX, KeyStart:
		switch {
		case resume && a.menu == 0:
			a.continueGame()
		case (resume && a.menu == 1) || (!resume && a.menu == 0):
			a.StartNewGame(time.Now().UnixNano())
		case (resume && a.menu == 2) || (!resume && a.menu == 1):
			a.helpBack, a.Screen, a.helpPage = ScreenTitle, ScreenHelp, 0
		case (resume && a.menu == 3) || (!resume && a.menu == 2):
			a.helpBack, a.Screen, a.helpPage = ScreenTitle, ScreenHelp, 2
		default:
			a.Quit = true
		}
	case KeyB:
		a.menu = n - 1
	}
}

func (a *App) keyTable(k Key) {
	r := a.G.Current
	if k == KeyL {
		a.openLog()
		return
	}
	if a.shown != nil && a.now.Before(a.holdUntil) {
		if k == KeyStart {
			a.Screen = ScreenPause
			a.pauseMenu = 0
		}
		return
	}
	switch k {
	case KeyStart:
		a.Screen = ScreenPause
		a.pauseMenu = 0
		return
	case KeySelect:
		a.helpBack, a.Screen, a.helpPage = ScreenTable, ScreenHelp, 0
		return
	}
	if r.Phase == game.PhaseDone {
		return
	}
	hand := a.Hand()
	switch k {
	case KeyLeft:
		if a.cursor > 0 {
			a.cursor--
		}
	case KeyRight:
		if a.cursor < len(hand)-1 {
			a.cursor++
		}
	case KeyUp:
		step := 5
		if r.Phase == game.PhaseBury && len(hand) > 25 {
			step = 17
		}
		a.cursor -= step
		if a.cursor < 0 {
			a.cursor = 0
		}
	case KeyDown:
		step := 5
		if r.Phase == game.PhaseBury && len(hand) > 25 {
			step = 17
		}
		a.cursor += step
		if a.cursor >= len(hand) {
			a.cursor = len(hand) - 1
		}
		if a.cursor < 0 {
			a.cursor = 0
		}
	case KeyA:
		if a.cursor >= 0 && a.cursor < len(hand) {
			id := hand[a.cursor].ID
			a.selected[id] = !a.selected[id]
		}
	case KeyB:
		a.selected = map[int]bool{}
		if r.Phase == game.PhaseBidding && r.BidSeat() == 0 {
			r.PassBid(0)
			a.turnAt = time.Time{}
			a.cursor, a.selected = 0, map[int]bool{}
			a.nextAI = a.now.Add(600 * time.Millisecond)
			a.afterBid()
		}
	case KeyY:
		a.hint()
	case KeyX:
		a.confirm()
	}
	if a.cursor >= len(hand) {
		a.cursor = len(hand) - 1
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
}

func (a *App) selectedCards() []game.Card {
	hand := a.Hand()
	out := make([]game.Card, 0, len(a.selected))
	for _, c := range hand {
		if a.selected[c.ID] {
			out = append(out, c)
		}
	}
	return out
}

func (a *App) confirm() {
	r := a.G.Current
	cards := a.selectedCards()
	switch r.Phase {
	case game.PhaseBidding:
		if r.BidSeat() != 0 {
			a.fail("还没轮到你亮主")
			return
		}
		if len(cards) == 0 {
			a.fail("请选择一张或一对级牌")
			return
		}
		if err := r.Bid(0, cards); err != nil {
			a.fail(err.Error())
			return
		}
		a.toast = fmt.Sprintf("你 亮主 %s%s", game.SuitName(r.Trump), game.LevelName(r.Level))
		a.toastUntil = a.now.Add(1600 * time.Millisecond)
		a.cursor, a.selected = 0, map[int]bool{}
		a.nextAI = a.now.Add(650 * time.Millisecond)
		a.afterBid()
	case game.PhaseBury:
		if r.Dealer != 0 {
			a.fail("只有庄家能扣底")
			return
		}
		if len(cards) != 8 {
			a.fail(fmt.Sprintf("请选择 8 张牌（已选 %d 张）", len(cards)))
			return
		}
		if err := r.Bury(0, cards); err != nil {
			a.fail(err.Error())
			return
		}
		a.cursor, a.selected = 0, map[int]bool{}
		a.nextAI = a.now.Add(700 * time.Millisecond)
		a.saveGame(false)
	case game.PhasePlay:
		if r.Turn != 0 {
			a.fail("还没轮到你出牌")
			return
		}
		if len(cards) == 0 {
			a.fail("请选择要出的牌")
			return
		}
		a.play(0, cards)
		a.selected = map[int]bool{}
		a.cursor = 0
	}
}

func (a *App) hint() {
	r := a.G.Current
	if r.Phase == game.PhaseBidding {
		for i, c := range a.Hand() {
			if c.Rank == r.Level && !c.IsJoker() {
				a.selected = map[int]bool{c.ID: true}
				a.cursor = i
				return
			}
		}
		a.fail("手里没有级牌")
		return
	}
	if r.Phase != game.PhasePlay || r.Turn != 0 {
		return
	}
	cards, err := game.AIChoose(r, 0)
	if err != nil {
		a.fail("没有可出的牌")
		return
	}
	a.selected = map[int]bool{}
	for _, c := range cards {
		a.selected[c.ID] = true
	}
}

func (a *App) keyHelp(k Key) {
	switch k {
	case KeyLeft, KeyUp:
		if a.helpPage > 0 {
			a.helpPage--
		}
	case KeyRight, KeyDown, KeyA, KeyX:
		if a.helpPage < 2 {
			a.helpPage++
		}
	case KeyB, KeyStart:
		a.Screen = a.helpBack
	}
}

func (a *App) keyPause(k Key) {
	switch k {
	case KeyUp:
		a.pauseMenu = (a.pauseMenu + len(pauseItems) - 1) % len(pauseItems)
	case KeyDown:
		a.pauseMenu = (a.pauseMenu + 1) % len(pauseItems)
	case KeyB, KeyStart:
		a.Screen = ScreenTable
	case KeyA, KeyX:
		switch a.pauseMenu {
		case 0:
			a.Screen = ScreenTable
		case 1:
			a.saveGame(true)
		case 2:
			a.StartNewGame(rand.Int63())
		case 3:
			a.helpBack, a.Screen, a.helpPage = ScreenPause, ScreenHelp, 0
		case 4:
			a.saveGame(false)
			a.Screen = ScreenTitle
			a.G = nil
		}
	}
}

func (a *App) keyRoundEnd(k Key) {
	if k == KeyA || k == KeyX || k == KeyStart {
		if a.G.Winner >= 0 {
			a.Screen = ScreenGameOver
			return
		}
		ResetCardCache()
		a.G.StartRound()
		a.Screen = ScreenTable
		a.cursor, a.selected = 0, map[int]bool{}
		a.shown = nil
		a.nextAI = a.now.Add(700 * time.Millisecond)
		a.turnAt = time.Time{}
		a.saveGame(false)
	}
}

func (a *App) keyGameOver(k Key) {
	switch k {
	case KeyA, KeyX, KeyStart:
		a.StartNewGame(rand.Int63())
	case KeyB:
		a.Screen = ScreenTitle
		a.G = nil
	}
}

// AutoAct makes the human seat play automatically. It is used by the demo /
// screenshot harness so the whole table can be driven without a keypad.
func (a *App) AutoAct() {
	if a.Screen != ScreenTable || a.G == nil {
		return
	}
	r := a.G.Current
	switch r.Phase {
	case game.PhaseBidding:
		if r.BidSeat() == 0 {
			r.PassBid(0)
			a.turnAt = time.Time{}
			a.cursor, a.selected = 0, map[int]bool{}
			a.afterBid()
		}
	case game.PhaseBury:
		if r.Dealer == 0 {
			if err := game.AIAutoBury(r); err == nil {
				a.cursor, a.selected = 0, map[int]bool{}
				a.nextAI = a.now.Add(700 * time.Millisecond)
			}
		}
	case game.PhasePlay:
		if r.Turn == 0 && a.now.After(a.holdUntil) {
			cards, err := game.AIChoose(r, 0)
			if err == nil && len(cards) > 0 {
				a.play(0, cards)
				a.selected = map[int]bool{}
			}
		}
	}
}

// KeyName returns a short label used by the diagnostics screen.
func KeyName(k Key) string {
	switch k {
	case KeyUp:
		return "上"
	case KeyDown:
		return "下"
	case KeyLeft:
		return "左"
	case KeyRight:
		return "右"
	case KeyA:
		return "A"
	case KeyB:
		return "B"
	case KeyX:
		return "X"
	case KeyY:
		return "Y"
	case KeyL:
		return "L"
	case KeyR:
		return "R"
	case KeyStart:
		return "START"
	case KeySelect:
		return "SELECT"
	}
	return "?"
}

// Render draws the current frame.
func (a *App) Render(c *gfx.Canvas) {
	switch a.Screen {
	case ScreenTitle:
		a.drawTitle(c)
	case ScreenTable:
		a.drawTable(c)
	case ScreenHelp:
		a.drawHelp(c)
	case ScreenPause:
		a.drawTable(c)
		a.drawPause(c)
	case ScreenRoundEnd:
		a.drawTable(c)
		a.drawRoundEnd(c)
	case ScreenGameOver:
		a.drawGameOver(c)
	case ScreenLog:
		a.drawLog(c)
	}
}
