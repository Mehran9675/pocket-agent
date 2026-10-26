package markdown

import (
	"strconv"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (d *Doc) table(key string, b mdBlock, base float32) widgets.El {
	size := base - 1
	cell := func(k string, sp []span, header bool, ci int) widgets.El {
		fg := theme.T.Text
		st := widgets.St{Padding: giode.SymmetricInset(10, 8)}
		el := d.rich(k, sp, size, fg, header)
		return giode.TableCell(st, el)
	}
	var rows []widgets.El
	if len(b.header) > 0 {
		var cells []widgets.El
		for ci, c := range b.header {
			cells = append(cells, cell(key+".h"+strconv.Itoa(ci), c, true, ci))
		}
		rows = append(rows, giode.TableHeader(widgets.St{Background: theme.T.Surface2}, cells...))
	}
	for ri, r := range b.rows {
		var cells []widgets.El
		for ci, c := range r {
			cells = append(cells, cell(key+".r"+strconv.Itoa(ri)+"."+strconv.Itoa(ci), c, false, ci))
		}
		rows = append(rows, giode.TableRow(widgets.St{BorderColor: theme.T.Border}, cells...))
	}
	return giode.Box(widgets.St{BorderRadius: 12, BorderWidth: 1, BorderColor: theme.T.Border, Overflow: giode.OverflowHidden},
		giode.Table(widgets.St{BorderColor: theme.T.Border}, rows...))
}
