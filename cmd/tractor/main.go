package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"tractor/internal/game"
	"tractor/internal/gfx"
	"tractor/internal/platform"
	"tractor/internal/ui"
)

func init() {
	// The Flip shares a small amount of RAM with the stock front end. Keep the
	// Go heap short so the runtime never reserves a large arena mid-game.
	debug.SetGCPercent(30)
	debug.SetMemoryLimit(20 << 20)
	runtime.GOMAXPROCS(1)
}

func main() {
	shotDir := flag.String("shot", "", "render demo screenshots into this directory and exit")
	diag := flag.Bool("diag", false, "show a panel/keypad diagnostic screen")
	demo := flag.Bool("demo", false, "auto-play an attract-mode game")
	fbPath := flag.String("fb", "/dev/fb0", "framebuffer device")
	rotate := flag.Bool("rotate180", true, "rotate the panel 180 degrees")
	flag.Parse()

	ui.LoadFonts()
	if *shotDir != "" {
		if err := shots(*shotDir); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *diag {
		if err := diagnose(*fbPath, *rotate); err != nil {
			log.Fatal(err)
		}
		return
	}
	a := ui.New(752, 560)
	if *demo || os.Getenv("TRACTOR_DEMO") == "1" {
		a.Demo = true
		a.StartNewGame(time.Now().UnixNano())
	}
	if err := platform.Run(a, *fbPath, *rotate); err != nil {
		log.Fatal(err)
	}
}

func renderPNG(a *ui.App, dir, name string) error {
	c := gfx.New(a.W, a.H)
	a.Render(c)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return c.PNG(filepath.Join(dir, name))
}

func drive(a *ui.App, from time.Time, steps int, stop func(*ui.App) bool) time.Time {
	now := from
	for i := 0; i < steps; i++ {
		now = now.Add(400 * time.Millisecond)
		a.SetNow(now)
		a.Tick(now)
		a.AutoAct()
		if stop != nil && stop(a) {
			break
		}
	}
	return now
}

func shots(dir string) error {
	base := time.Now()
	// Keep screenshots independent of a real player save.
	saveDir := filepath.Join(dir, ".save")
	_ = os.RemoveAll(saveDir)
	_ = os.Setenv("TRACTOR_SAVE_DIR", saveDir)

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		if err := renderPNG(a, dir, "01_title.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.StartNewGame(1)
		if err := renderPNG(a, dir, "02_bidding.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.StartNewGame(3)
		drive(a, base, 60, func(a *ui.App) bool {
			r := a.G.Current
			n := 0
			for i := 0; i < 4; i++ {
				if len(r.Plays[i]) > 0 {
					n++
				}
			}
			return r.Phase == game.PhasePlay && n == 2 && r.DefenderPoints > 0
		})
		if err := renderPNG(a, dir, "03_play.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.StartNewGame(4)
		drive(a, base, 40, func(a *ui.App) bool {
			return a.G.Current.Phase == game.PhaseBury
		})
		if err := renderPNG(a, dir, "04_bury.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.StartNewGame(2)
		drive(a, base, 600, func(a *ui.App) bool {
			return a.Screen == ui.ScreenRoundEnd
		})
		if err := renderPNG(a, dir, "05_roundend.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.Key(ui.KeyDown)
		a.Key(ui.KeyA)
		if err := renderPNG(a, dir, "06_help.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.StartNewGame(1)
		a.G.Winner = 0
		a.G.Levels = [2]game.Rank{game.RankA, game.RankK}
		a.Screen = ui.ScreenGameOver
		if err := renderPNG(a, dir, "07_gameover.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.StartNewGame(3)
		drive(a, base, 200, func(a *ui.App) bool {
			return len(a.G.Current.History) >= 3
		})
		a.Key(ui.KeyL)
		if err := renderPNG(a, dir, "09_log.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.StartNewGame(30)
		a.Screen = ui.ScreenTitle
		if err := renderPNG(a, dir, "10_resume.png"); err != nil {
			return err
		}
	}

	{
		a := ui.New(752, 560)
		a.SetNow(base)
		a.StartNewGame(31)
		a.Key(ui.KeyStart)
		if err := renderPNG(a, dir, "11_pause.png"); err != nil {
			return err
		}
	}

	fmt.Println("screenshots written to", dir)
	return nil
}
