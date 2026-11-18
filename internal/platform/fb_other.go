//go:build !linux

package platform

import (
	"fmt"

	"tractor/internal/ui"
)

// Run is only available on the device build.
func Run(a *ui.App, fbPath string, rotate bool) error {
	return fmt.Errorf("framebuffer output is only supported on linux")
}
