package theme

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/assets"
)

// RegisterIcons loads the embedded icon set.
func RegisterIcons() error {
	return giode.RegisterIconsFS(assets.Icons, "icons")
}
