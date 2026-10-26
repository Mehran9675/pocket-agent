package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

func HDivider() El { return giode.Box(St{Height: 1, Width: -1, Background: theme.T.Border}) }

func VDivider() El { return giode.Box(St{Width: 1, Height: -1, Background: theme.T.Border}) }
