//go:build windows

package tui

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func termSize() (width, height int) {
	h := windows.Handle(os.Stdout.Fd())
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(h, &info); err != nil {
		return 0, 0
	}
	w := int(info.Window.Right - info.Window.Left + 1)
	h2 := int(info.Window.Bottom - info.Window.Top + 1)
	_ = unsafe.Sizeof(0)
	return w, h2
}
