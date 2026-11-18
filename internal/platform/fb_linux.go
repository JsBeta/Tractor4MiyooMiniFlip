//go:build linux

// Package platform talks to the Miyoo Mini framebuffer and keypad directly.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"tractor/internal/gfx"
	"tractor/internal/ui"
)

const (
	fbioGetVScreen = 0x4600
	fbioPutVScreen = 0x4601
	fbioGetFScreen = 0x4602
	fbioPanDisplay = 0x4606
)

type fbBitfield struct {
	Offset   uint32
	Length   uint32
	MSBRight uint32
}

type fbVarScreeninfo struct {
	Xres, Yres          uint32
	XresVirtual         uint32
	YresVirtual         uint32
	Xoffset, Yoffset    uint32
	BitsPerPixel        uint32
	Grayscale           uint32
	Red, Green, Blue    fbBitfield
	Transp              fbBitfield
	Nonstd              uint32
	Activate            uint32
	Height, Width       uint32
	AccelFlags          uint32
	Pixels              [256]uint32
}

type fbFixScreeninfo struct {
	ID           [16]byte
	SmemStart    uintptr
	SmemLen      uint32
	Type         uint32
	TypeAux      uint32
	Visual       uint32
	XpanStep     uint16
	YpanStep     uint16
	YwrapStep    uint16
	LineLength   uint32
	MmioStart    uintptr
	MmioLen      uint32
	Accel        uint32
	Capabilities uint16
	Reserved     [2]uint16
}

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// Framebuffer is the Linux framebuffer presenter.
type Framebuffer struct {
	f      *os.File
	mem    []byte
	w, h   int
	stride int
	page   int
	pages  int
	fbVar  fbVarScreeninfo
	rotate bool
	row    []uint32
	direct bool
}

// OpenFB opens /dev/fb0 and maps its memory.
func OpenFB(path string, rotate bool) (*Framebuffer, error) {
	if path == "" {
		path = "/dev/fb0"
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	fb := &Framebuffer{f: f, rotate: rotate}
	if err := ioctl(f.Fd(), fbioGetVScreen, unsafe.Pointer(&fb.fbVar)); err != nil {
		f.Close()
		return nil, fmt.Errorf("FBIOGET_VSCREENINFO: %w", err)
	}
	var fix fbFixScreeninfo
	if err := ioctl(f.Fd(), fbioGetFScreen, unsafe.Pointer(&fix)); err != nil {
		f.Close()
		return nil, fmt.Errorf("FBIOGET_FSCREENINFO: %w", err)
	}
	if fb.fbVar.BitsPerPixel != 32 {
		f.Close()
		return nil, fmt.Errorf("unsupported bpp %d", fb.fbVar.BitsPerPixel)
	}
	fb.w = int(fb.fbVar.Xres)
	fb.h = int(fb.fbVar.Yres)
	fb.stride = int(fix.LineLength)
	fb.pages = int(fb.fbVar.YresVirtual) / fb.h
	if fb.pages < 1 {
		fb.pages = 1
	}
	if fb.pages > 3 {
		fb.pages = 3
	}
	size := fb.stride * fb.h * fb.pages
	mem, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, err
	}
	fb.mem = mem
	fb.page = int(fb.fbVar.Yoffset) / fb.h
	bf := fb.fbVar
	fb.direct = bf.Red.Offset == 16 && bf.Red.Length == 8 &&
		bf.Green.Offset == 8 && bf.Green.Length == 8 &&
		bf.Blue.Offset == 0 && bf.Blue.Length == 8
	return fb, nil
}

func (f *Framebuffer) W() int { return f.w }
func (f *Framebuffer) H() int { return f.h }

func (f *Framebuffer) pack(r, g, b uint8) uint32 {
	v := uint32(0)
	bf := f.fbVar
	v |= uint32(r>>(8-bf.Red.Length)) << bf.Red.Offset
	v |= uint32(g>>(8-bf.Green.Length)) << bf.Green.Offset
	v |= uint32(b>>(8-bf.Blue.Length)) << bf.Blue.Offset
	return v
}

// Present copies the canvas into the framebuffer and flips the page.
func (f *Framebuffer) Present(c *gfx.Canvas) {
	w, h := c.W, c.H
	if w != f.w || h != f.h {
		return
	}
	page := 0
	if f.pages > 1 {
		page = (f.page + 1) % f.pages
	}
	base := page * f.stride * f.h
	if f.row == nil {
		f.row = make([]uint32, w)
	}
	row := f.row
	for dy := 0; dy < h; dy++ {
		sy := dy
		if f.rotate {
			sy = h - 1 - dy
		}
		src := c.Pix[sy*w : (sy+1)*w]
		if f.direct {
			if f.rotate {
				for x := 0; x < w; x++ {
					row[x] = src[w-1-x]
				}
			} else {
				copy(row, src)
			}
		} else if f.rotate {
			for x := 0; x < w; x++ {
				row[x] = f.pack(uint8(src[w-1-x]>>16), uint8(src[w-1-x]>>8), uint8(src[w-1-x]))
			}
		} else {
			for x := 0; x < w; x++ {
				row[x] = f.pack(uint8(src[x]>>16), uint8(src[x]>>8), uint8(src[x]))
			}
		}
		off := base + dy*f.stride
		if f.stride%4 != 0 {
			dst := f.mem[off : off+w*4]
			for x := 0; x < w; x++ {
				dst[x*4] = byte(row[x])
				dst[x*4+1] = byte(row[x] >> 8)
				dst[x*4+2] = byte(row[x] >> 16)
				dst[x*4+3] = 0xFF
			}
			continue
		}
		dst := unsafe.Slice((*uint32)(unsafe.Pointer(&f.mem[off])), w)
		copy(dst, row)
	}
	if f.pages > 1 {
		v := f.fbVar
		v.Yoffset = uint32(page * f.h)
		if err := ioctl(f.f.Fd(), fbioPanDisplay, unsafe.Pointer(&v)); err == nil {
			f.page = page
		} else {
			// Panning is not available: fall back to the visible page.
			f.pages = 1
			f.page = 0
		}
	}
}

// Info describes the opened panel.
func (f *Framebuffer) Info() string {
	return fmt.Sprintf("fb=%dx%d stride=%d pages=%d bpp=%d", f.w, f.h, f.stride, f.pages, f.fbVar.BitsPerPixel)
}

func (f *Framebuffer) Close() {
	_ = syscall.Munmap(f.mem)
	f.f.Close()
}

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

var keyMap = map[uint16]ui.Key{
	1:   ui.KeyStart, // ESC / menu
	28:  ui.KeyStart, // enter
	57:  ui.KeyA,     // space
	29:  ui.KeyB,     // left ctrl
	42:  ui.KeyX,     // left shift
	56:  ui.KeyY,     // left alt
	103: ui.KeyUp,
	108: ui.KeyDown,
	105: ui.KeyLeft,
	106: ui.KeyRight,
	15:  ui.KeyL, // tab (L2)
	18:  ui.KeyL, // E (L1)
	14:  ui.KeyR, // backspace (R2)
	20:  ui.KeyR, // T (R1)
	97:  ui.KeySelect,
	304: ui.KeyA,
	305: ui.KeyB,
	307: ui.KeyX,
	308: ui.KeyY,
	544: ui.KeyUp,
	545: ui.KeyDown,
	546: ui.KeyLeft,
	547: ui.KeyRight,
	310: ui.KeyL,
	311: ui.KeyR,
	314: ui.KeySelect,
	315: ui.KeyStart,
	316: ui.KeyStart,
}

// Input reads the keypad event devices.
type Input struct {
	mu       sync.Mutex
	events   []ui.Key
	held     map[ui.Key]time.Time
	last     map[ui.Key]time.Time
	lastCode map[uint16]time.Time
	log      *os.File
	files    []*os.File
}

// OpenInput opens every /dev/input/event* device.
func OpenInput(explicit string) *Input {
	in := &Input{
		held:     map[ui.Key]time.Time{},
		last:     map[ui.Key]time.Time{},
		lastCode: map[uint16]time.Time{},
	}
	var paths []string
	if explicit != "" {
		paths = []string{explicit}
	} else {
		matches, _ := filepath.Glob("/dev/input/event*")
		paths = matches
	}
	for _, p := range paths {
		f, err := os.OpenFile(p, os.O_RDONLY, 0)
		if err != nil {
			continue
		}
		in.files = append(in.files, f)
		go in.readLoop(f)
	}
	if f, err := os.OpenFile("/mnt/SDCARD/Saves/Tractor/keys.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		in.log = f
	}
	return in
}

func (in *Input) readLoop(f *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if err != nil || n <= 0 {
			return
		}
		size := int(unsafe.Sizeof(inputEvent{}))
		for off := 0; off+size <= n; off += size {
			ev := (*inputEvent)(unsafe.Pointer(&buf[off]))
			if ev.Type != 1 { // EV_KEY
				continue
			}
			if ev.Value == 0 {
				in.release(ev.Code)
				continue
			}
			in.press(ev.Code)
		}
	}
}

func (in *Input) press(code uint16) {
	now := time.Now()
	in.mu.Lock()
	defer in.mu.Unlock()
	if t, ok := in.lastCode[code]; ok && now.Sub(t) < 80*time.Millisecond {
		return
	}
	in.lastCode[code] = now
	k, ok := keyMap[code]
	if !ok {
		if in.log != nil {
			fmt.Fprintf(in.log, "%s unknown key code %d\n", now.Format("15:04:05"), code)
		}
		return
	}
	if in.log != nil {
		fmt.Fprintf(in.log, "%s key %d -> %d\n", now.Format("15:04:05"), code, k)
	}
	in.events = append(in.events, k)
	in.held[k] = now
	in.last[k] = now
}

func (in *Input) release(code uint16) {
	k, ok := keyMap[code]
	if !ok {
		return
	}
	in.mu.Lock()
	delete(in.held, k)
	in.mu.Unlock()
}

// Drain returns pending presses, plus auto repeat for the d-pad.
func (in *Input) Drain(now time.Time) []ui.Key {
	in.mu.Lock()
	defer in.mu.Unlock()
	out := in.events
	in.events = nil
	for _, k := range []ui.Key{ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown} {
		start, ok := in.held[k]
		if !ok {
			continue
		}
		if now.Sub(start) < 320*time.Millisecond {
			continue
		}
		if now.Sub(in.last[k]) < 95*time.Millisecond {
			continue
		}
		in.last[k] = now
		out = append(out, k)
	}
	return out
}

// Run is the device main loop.
func Run(a *ui.App, fbPath string, rotate bool) error {
	fb, err := OpenFB(fbPath, rotate)
	if err != nil {
		return err
	}
	defer fb.Close()
	if fb.w != a.W || fb.h != a.H {
		return fmt.Errorf("panel is %dx%d, expected %dx%d", fb.w, fb.h, a.W, a.H)
	}
	in := OpenInput(os.Getenv("TRACTOR_INPUT"))
	c := gfx.New(a.W, a.H)
	ui.LoadFonts()

	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()
	for {
		now := time.Now()
		for _, k := range in.Drain(now) {
			a.Key(k)
			if a.Quit {
				return nil
			}
		}
		a.Tick(now)
		a.Render(c)
		fb.Present(c)
		<-ticker.C
	}
}
