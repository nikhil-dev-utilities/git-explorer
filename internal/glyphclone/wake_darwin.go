//go:build darwin

package glyphclone

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// wakeRead pushes one NUL into the terminal's input queue. On macOS, closing an
// os.File on /dev/tty does not interrupt a read already blocked on it (kqueue cannot
// poll it), so without this Glyph would only notice Stop after the user's next
// keypress, and swallow that key. The NUL is read by the blocked Glyph reader, which
// then fails on its closed handle and returns; it matches no binding.
func wakeRead(tty *os.File) {
	b := byte(0)
	unix.Syscall(unix.SYS_IOCTL, tty.Fd(), unix.TIOCSTI, uintptr(unsafe.Pointer(&b)))
}
