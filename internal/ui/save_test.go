package ui

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tractor-save-test")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("TRACTOR_SAVE_DIR", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestSaveGlyphs(t *testing.T) {
	LoadFonts()
	for _, text := range []string{"继续上次牌局", "保存进度", "进度已保存", "已读取进度", "读取存档失败", "没有可保存的牌局"} {
		for _, r := range text {
			if _, ok := fonts.sm.Glyphs[r]; !ok {
				t.Errorf("missing glyph %q in %q", r, text)
			}
		}
	}
}

func TestSaveAndContinue(t *testing.T) {
	t.Setenv("TRACTOR_SAVE_DIR", t.TempDir())
	LoadFonts()
	a := New(752, 560)
	a.SetNow(time.Now())
	a.StartNewGame(21)
	if !hasSavedGame() {
		t.Fatal("StartNewGame did not auto-save")
	}

	b := New(752, 560)
	b.SetNow(time.Now())
	if !b.continueGame() {
		t.Fatal("continueGame failed")
	}
	if b.Screen != ScreenTable || b.G == nil || b.G.Current == nil {
		t.Fatalf("screen=%v game=%v", b.Screen, b.G)
	}
	if len(b.G.Current.Hands[0]) != len(a.G.Current.Hands[0]) {
		t.Fatalf("hand not restored")
	}
	if b.G.Current.Phase != a.G.Current.Phase || b.G.Current.Trump != a.G.Current.Trump {
		t.Fatalf("round not restored")
	}
}

func TestPauseManualSave(t *testing.T) {
	t.Setenv("TRACTOR_SAVE_DIR", t.TempDir())
	LoadFonts()
	a := New(752, 560)
	a.SetNow(time.Now())
	a.StartNewGame(22)
	_ = os.Remove(savePath())
	if hasSavedGame() {
		t.Fatal("save not removed")
	}
	a.Screen = ScreenPause
	a.pauseMenu = 1
	a.Key(KeyA)
	if !hasSavedGame() {
		t.Fatal("manual save failed")
	}
	if a.Screen != ScreenPause {
		t.Fatalf("screen changed to %v", a.Screen)
	}
}

func TestTitleResumeEntry(t *testing.T) {
	t.Setenv("TRACTOR_SAVE_DIR", t.TempDir())
	LoadFonts()
	a := New(752, 560)
	a.SetNow(time.Now())
	a.StartNewGame(23)

	b := New(752, 560)
	b.SetNow(time.Now())
	if items := b.titleItems(); len(items) != 5 || items[0] != "继续上次牌局" {
		t.Fatalf("title items %v", items)
	}
	b.menu = 0
	b.Key(KeyA)
	if b.Screen != ScreenTable {
		t.Fatalf("resume did not open table: %v", b.Screen)
	}
}

func TestTitleWithoutSave(t *testing.T) {
	t.Setenv("TRACTOR_SAVE_DIR", t.TempDir())
	LoadFonts()
	a := New(752, 560)
	if items := a.titleItems(); len(items) != 4 {
		t.Fatalf("title items %v", items)
	}
	a.SetNow(time.Now())
	a.menu = 0
	a.Key(KeyA)
	if a.Screen != ScreenTable || a.G == nil {
		t.Fatal("new game did not start")
	}
}

func TestCorruptSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TRACTOR_SAVE_DIR", dir)
	LoadFonts()
	if err := os.WriteFile(savePath(), []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}
	a := New(752, 560)
	a.SetNow(time.Now())
	if a.continueGame() {
		t.Fatal("loaded corrupt save")
	}
	if a.Screen != ScreenTitle {
		t.Fatalf("screen %v", a.Screen)
	}
}
