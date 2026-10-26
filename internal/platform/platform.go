// Package platform fills gaps between Giode and the operating system:
// opening links in the browser, for now. See docs/giode-requests.md for
// what should move into Giode itself.
package platform

// OpenURL opens a web link in the system browser. It returns an error
// when the platform can't, so the caller can fall back (copy the link).
func OpenURL(url string) error { return openURL(url) }
