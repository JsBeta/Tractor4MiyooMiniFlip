package ui

import "testing"

func TestHelpFits(t *testing.T) {
	LoadFonts()
	for i, page := range helpPages {
		y := 92
		for _, line := range page {
			if line == "" {
				y += 12
				continue
			}
			n := len(wrapText(fonts.md, line, 752-120))
			y += n*32 + 4
		}
		t.Logf("page %d ends at y=%d", i, y)
		if y > 512 {
			t.Errorf("page %d overflows: %d", i, y)
		}
	}
}
