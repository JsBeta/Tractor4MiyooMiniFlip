package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"tractor/internal/game"
)

const saveFileName = "save.json"

// saveDir is where progress is kept. On the Flip this is the Saves folder so
// the save survives app updates; TRACTOR_SAVE_DIR overrides it for tests.
func saveDir() string {
	if d := os.Getenv("TRACTOR_SAVE_DIR"); d != "" {
		return d
	}
	if fi, err := os.Stat("/mnt/SDCARD/Saves"); err == nil && fi.IsDir() {
		return "/mnt/SDCARD/Saves/Tractor"
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "Tractor")
	}
	return "."
}

func savePath() string { return filepath.Join(saveDir(), saveFileName) }

func hasSavedGame() bool {
	fi, err := os.Stat(savePath())
	return err == nil && fi.Mode().IsRegular() && fi.Size() > 0
}

func clearSavedGame() {
	_ = os.Remove(savePath())
}

// saveGame writes the current match. When notify is true a toast is shown,
// which is used by the pause menu's manual save item.
func (a *App) saveGame(notify bool) bool {
	if a.Demo && !notify && os.Getenv("TRACTOR_DEMO_SAVE") != "1" {
		return false
	}
	if a.G == nil || a.G.Current == nil || a.G.Winner >= 0 {
		if notify {
			a.msg("没有可保存的牌局", 1500*time.Millisecond)
		}
		return false
	}
	data, err := json.MarshalIndent(a.G.State(), "", "  ")
	if err != nil {
		if notify {
			a.msg("保存失败", 1600*time.Millisecond)
		}
		return false
	}
	if err := writeSaveFile(savePath(), data, notify); err != nil {
		if notify {
			a.msg("保存失败", 1600*time.Millisecond)
		}
		return false
	}
	if notify {
		a.msg("进度已保存", 1600*time.Millisecond)
	}
	return true
}

func writeSaveFile(path string, data []byte, sync bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if sync {
		_ = f.Sync()
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func loadGame() (*game.Game, error) {
	data, err := os.ReadFile(savePath())
	if err != nil {
		return nil, err
	}
	var st game.GameState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	g := &game.Game{}
	if err := g.Restore(st); err != nil {
		return nil, err
	}
	return g, nil
}

// Resume loads the saved match and returns to the table. It is the exported
// entry used by tooling; the title menu calls continueGame directly.
func (a *App) Resume() bool { return a.continueGame() }

// continueGame loads the saved match and returns to the table.
func (a *App) continueGame() bool {
	g, err := loadGame()
	if err != nil {
		a.fail("读取存档失败")
		return false
	}
	ResetCardCache()
	a.G = g
	a.Screen = ScreenTable
	a.cursor, a.selected = 0, map[int]bool{}
	a.shown = nil
	a.playAt = [4]time.Time{}
	a.holdUntil = time.Time{}
	a.nextAI = a.now.Add(700 * time.Millisecond)
	a.turnAt = time.Time{}
	a.logScroll = 0
	a.errMsg = ""
	a.msg("已读取进度", 1600*time.Millisecond)
	return true
}
