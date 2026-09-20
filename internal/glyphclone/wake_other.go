//go:build !darwin

package glyphclone

import "os"

// wakeRead is a no-op off macOS: there Go polls /dev/tty with epoll, so closing it in
// Stop already interrupts Glyph's blocked read.
func wakeRead(*os.File) {}
