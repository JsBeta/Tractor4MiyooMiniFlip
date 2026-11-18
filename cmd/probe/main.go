//go:build linux

package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

type inputEvent struct {
	Sec  int32
	Usec int32
	Type uint16
	Code uint16
	Value int32
}

func main() {
	f, err := os.OpenFile("/dev/input/event0", os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer f.Close()
	r, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), 0x40044590, 1)
	fmt.Printf("EVIOCGRAB=1 ret=%d err=%v\n", r, e)
	if e == 0 {
		syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), 0x40044590, 0)
	}
	// device name
	var name [80]byte
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), 0x80ff4506, uintptr(unsafe.Pointer(&name[0])))
	fmt.Printf("name=%s\n", string(name[:]))
	buf := make([]byte, 16)
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		n, err := f.Read(buf)
		if n > 0 {
			ev := (*inputEvent)(unsafe.Pointer(&buf[0]))
			fmt.Printf("ev type=%d code=%d value=%d\n", ev.Type, ev.Code, ev.Value)
			continue
		}
		if err != nil {
			time.Sleep(50 * time.Millisecond)
		}
	}
	fmt.Println("done")
}

