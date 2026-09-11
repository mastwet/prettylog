//go:build !windows

package prettylog

// EnableVirtualTerminal is a no-op outside Windows.
func EnableVirtualTerminal() bool { return true }
