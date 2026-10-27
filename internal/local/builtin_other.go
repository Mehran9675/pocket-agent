//go:build !(ios && llama && cgo)

package local

// builtinEngine is set when the engine is linked into the app (iOS,
// built with -tags llama) rather than downloaded.
const builtinEngine = false
