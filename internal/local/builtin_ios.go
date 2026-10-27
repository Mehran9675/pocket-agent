//go:build ios && llama && cgo

package local

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/llama.cpp/include -I${SRCDIR}/../../third_party/llama.cpp/ggml/include -O2
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/lib/ios_arm64 -lllama -lggml -lggml-cpu -lggml-metal -lggml-blas -lggml-base -lc++ -framework Foundation -framework Metal -framework MetalKit -framework Accelerate
*/
import "C"

// builtinEngine is set when the engine is linked into the app (iOS,
// built with -tags llama) rather than downloaded.
const builtinEngine = true
