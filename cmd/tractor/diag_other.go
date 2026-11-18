//go:build !linux

package main

import "fmt"

func diagnose(fbPath string, rotate bool) error {
	return fmt.Errorf("diagnostics require the device build")
}
