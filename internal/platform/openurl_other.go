//go:build js || wasip1 || plan9

package platform

import "errors"

func openURL(string) error { return errors.New("can't open links on this platform") }
