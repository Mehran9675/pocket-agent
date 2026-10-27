//go:build cgo

package local

// #include <stdint.h>
import "C"

import (
	"runtime/cgo"
	"unsafe"
)

// paGoPiece is the engine's piece callback; user is the generation's
// cgo.Handle.
//
//export paGoPiece
func paGoPiece(user unsafe.Pointer, piece *C.char, n C.int32_t) C.int32_t {
	g := cgo.Handle(uintptr(user)).Value().(*generation)
	if g.piece(C.GoBytes(unsafe.Pointer(piece), C.int(n))) {
		return 1
	}
	return 0
}
