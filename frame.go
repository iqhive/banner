package banner

import (
	"time"
	"unicode/utf8"
)

// Cols and Rows define the terminal grid and fixed SVG geometry.
const (
	Cols = 80
	Rows = 20
)

// Style is the colour and weight of a cell.
type Style int

const (
	Plain   Style = iota // ordinary text
	Dim                  // the hex noise the wordmark resolves out of
	Muted                // captions
	Bright               // figures the eye should land on
	Literal              // Go literals in the calls shown
	Changed              // what changed
	On                   // a set bit
	Off                  // a clear bit
	Bar                  // a diagram bar
	Prompt               // the shell prompt
	Caret                // the typing cursor
	Mark                 // Mark+i colours letter i of the wordmark
)

// Block glyphs. The terminal prints them; the SVG draws them as rectangles.
const (
	Full  = '█'
	Shade = '░'
)

func isBlock(r rune) bool { return r == Full || r == Shade }
func isBlank(r rune) bool { return r == 0 || r == ' ' }

// Cell is one character on the screen. The zero cell is blank.
type Cell struct {
	Rune  rune
	Style Style
}

// Frame is one screenful and how long it stays up.
type Frame struct {
	Cells        [Rows][Cols]Cell
	Hold         time.Duration
	opening      *openingSource
	openingIndex int
	HiveLogo     bool // render the embedded vector logo; terminal output uses Cells
}

// Text writes s at column x of row y, clipping at the screen edge.
// Each rune occupies one column. Use ASCII and single-column glyphs, not
// tabs, combining characters or emoji.
func (f *Frame) Text(x, y int, s string, st Style) {
	for _, r := range s {
		if 0 <= x && x < Cols && 0 <= y && y < Rows {
			f.Cells[y][x] = Cell{r, st}
		}
		x++
	}
}

// Center writes s centred on row y and returns the column it starts at.
func (f *Frame) Center(y int, s string, st Style) int {
	x := (Cols - utf8.RuneCountInString(s)) / 2
	f.Text(x, y, s, st)
	return x
}

// Clear blanks rows y0 up to but not including y1.
func (f *Frame) Clear(y0, y1 int) {
	for y := max(y0, 0); y < min(y1, Rows); y++ {
		f.Cells[y] = [Cols]Cell{}
	}
}
