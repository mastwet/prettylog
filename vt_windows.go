//go:build windows

package prettylog

import (
	"syscall"
	"unsafe"
)

// EnableVirtualTerminal turns on ENABLE_VIRTUAL_TERMINAL_PROCESSING for the
// current process console so legacy conhost can render ANSI.
// Safe to call multiple times; returns true when VT is active afterwards.
func EnableVirtualTerminal() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	const enableVT = 0x0004

	h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil || h == 0 || h == syscall.InvalidHandle {
		return false
	}

	var mode uint32
	if r, _, _ := getConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode))); r == 0 {
		return false
	}
	if mode&enableVT != 0 {
		return true
	}
	r, _, _ := setConsoleMode.Call(uintptr(h), uintptr(mode|enableVT))
	return r != 0
}
