//go:build linux

package main

import (
	"fmt"
	"time"

	"tractor/internal/gfx"
	"tractor/internal/platform"
	"tractor/internal/ui"
)

func diagnose(fbPath string, rotate bool) error {
	fb, err := platform.OpenFB(fbPath, rotate)
	if err != nil {
		return err
	}
	defer fb.Close()
	fmt.Println(fb.Info())
	in := platform.OpenInput("")
	c := gfx.New(fb.W(), fb.H())
	bars := []gfx.Color{
		gfx.RGB(255, 0, 0), gfx.RGB(0, 255, 0), gfx.RGB(0, 0, 255),
		gfx.RGB(0, 255, 255), gfx.RGB(255, 0, 255), gfx.RGB(255, 255, 0),
		gfx.RGB(255, 255, 255), gfx.RGB(10, 10, 10),
	}
	last := "—"
	quit := make(chan struct{})
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		for _, k := range in.Drain(time.Now()) {
			last = ui.KeyName(k)
			if k == ui.KeySelect {
				close(quit)
			}
		}
		select {
		case <-quit:
			return nil
		default:
		}
		c.Clear(gfx.RGB(0x08, 0x14, 0x10))
		bw := c.W / len(bars)
		for i, col := range bars {
			c.FillRect(i*bw, 44, bw, 110, col)
		}
		c.TextCenter(ui.Fonts().MD, c.W/2, 8, "拖拉机 屏幕诊断", gfx.RGB(0xE9, 0xC4, 0x6A))
		c.TextCenter(ui.Fonts().SM, c.W/2, 170, "上", gfx.RGB(0xF7, 0xF1, 0xE0))
		c.TextCenter(ui.Fonts().MD, c.W/2, c.H-60, "下", gfx.RGB(0xF7, 0xF1, 0xE0))
		c.Text(ui.Fonts().MD, 16, c.H/2-14, "左", gfx.RGB(0xF7, 0xF1, 0xE0))
		c.TextRight(ui.Fonts().MD, c.W-16, c.H/2-14, "右", gfx.RGB(0xF7, 0xF1, 0xE0))
		c.TextCenter(ui.Fonts().SM, c.W/2, 196, "色条应为 红 绿 蓝 青 品 黄 白 黑", gfx.RGB(0x9F, 0xB6, 0xA8))
		c.TextCenter(ui.Fonts().SM, c.W/2, c.H/2+30, "最后按键："+last, gfx.RGB(0xE9, 0xC4, 0x6A))
		c.TextCenter(ui.Fonts().XS, c.W/2, c.H-32, "按 SELECT 退出", gfx.RGB(0x9F, 0xB6, 0xA8))
		fb.Present(c)
		time.Sleep(40 * time.Millisecond)
	}
	return nil
}
