package banner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// PlaybackOptions selects terminal coloring, looping and an optional stop channel.
// A received signal stops playback after drawing the current frame.
type PlaybackOptions struct {
	Colors    Colors
	Loop      bool
	Interrupt <-chan os.Signal
}

// IsTerminal reports whether w is a terminal that can take cursor movement.
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok || os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// Colors defines how the terminal's output is coloured.
type Colors int

const (
	NoColor Colors = iota // NO_COLOR is set
	Color256
	TrueColor
)

// TerminalColors reads the environment: NO_COLOR (https://no-color.org) turns
// colour off, and COLORTERM says whether the terminal takes 24-bit colour.
func TerminalColors() Colors {
	if _, present := os.LookupEnv("NO_COLOR"); present {
		return NoColor
	}
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		return TrueColor
	}
	return Color256
}

// sgr returns the escape sequence that selects Style s, starting from a reset
// so that nothing of the previous Style carries over.
func sgr(s Style, c Colors, palette []Paint) string {
	p := palette[s]
	var b strings.Builder
	b.WriteString("\x1b[0")
	if p.Bold {
		b.WriteString(";1")
	}
	colour := func(layer int, rgb uint32) {
		if c == TrueColor {
			fmt.Fprintf(&b, ";%d;2;%d;%d;%d", layer, rgb>>16, rgb>>8&0xff, rgb&0xff)
		} else {
			fmt.Fprintf(&b, ";%d;5;%d", layer, nearest256(rgb))
		}
	}
	colour(38, p.Foreground)
	if p.Background != 0 {
		colour(48, p.Background)
	}
	b.WriteString("m")
	return b.String()
}

// nearest256 returns the xterm-256 colour nearest to rgb: one of the 6x6x6
// cube or the 24 greys.
func nearest256(rgb uint32) int {
	r, g, b := int(rgb>>16), int(rgb>>8&0xff), int(rgb&0xff)
	level := func(i int) int {
		if i == 0 {
			return 0
		}
		return 55 + 40*i
	}
	dist := func(x, y, z int) int { return (x-r)*(x-r) + (y-g)*(y-g) + (z-b)*(z-b) }
	best, bestDist := 0, 1<<30
	for i := range 216 {
		x, y, z := level(i/36), level(i/6%6), level(i%6)
		if d := dist(x, y, z); d < bestDist {
			best, bestDist = 16+i, d
		}
	}
	for i := range 24 {
		v := 8 + 10*i
		if d := dist(v, v, v); d < bestDist {
			best, bestDist = 232+i, d
		}
	}
	return best
}

// ErrInterrupted indicates that playback stopped and restored the cursor.
var ErrInterrupted = errors.New("interrupted")

// Terminal control sequences.
const (
	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
	reset      = "\x1b[0m"
	clearLine  = "\x1b[K" // to the end of the line
)

// Play draws the frames in place: the first Frame from the cursor down, and
// each later one over it, by moving the cursor back up. The cursor stays on
// the last row while it plays, so a Frame needs no more Rows than it has.
// The last Frame stays on the screen. A signal on sig stops it between
// frames; it always restores the cursor and the Colors, and moves to the
// line below the Frame, before it returns.
func Play(w io.Writer, a Animation, opts PlaybackOptions) (err error) {
	a, err = a.WithTheme(a.theme())
	if err != nil {
		return err
	}
	if err = a.Validate(); err != nil {
		return err
	}
	frames := a.Frames
	palette := a.theme().palette(a.letters())
	c, loop, sig := opts.Colors, opts.Loop, opts.Interrupt
	if c < NoColor || c > TrueColor {
		return errors.New("banner: invalid color mode")
	}
	if _, err := io.WriteString(w, hideCursor); err != nil {
		return err
	}
	defer func() {
		if _, e := io.WriteString(w, reset+showCursor+"\n"); err == nil {
			err = e
		}
	}()
	var b bytes.Buffer
	first := true
	for {
		for i := range frames {
			b.Reset()
			if !first {
				fmt.Fprintf(&b, "\r\x1b[%dA", Rows-1)
			}
			first = false
			writeFrame(&b, &frames[i], c, palette)
			if _, err := w.Write(b.Bytes()); err != nil {
				return err
			}
			t := time.NewTimer(frames[i].Hold)
			select {
			case <-t.C:
			case <-sig:
				t.Stop()
				return ErrInterrupted
			}
		}
		if !loop {
			return nil
		}
	}
}

// writeFrame writes every row of f, each cleared to the end of the line, and
// leaves the cursor at the end of the last row.
func writeFrame(b *bytes.Buffer, f *Frame, c Colors, palette []Paint) {
	for y := range Rows {
		if y > 0 {
			b.WriteByte('\n')
		}
		writeRow(b, &f.Cells[y], c, palette)
		if c != NoColor {
			b.WriteString(reset)
		}
		b.WriteString(clearLine)
	}
}

// writeRow writes one row without its trailing blanks, switching colour only
// where the style changes, and never leaving a background colour on under a
// blank.
func writeRow(b *bytes.Buffer, row *[Cols]Cell, c Colors, palette []Paint) {
	end := Cols
	for end > 0 && isBlank(row[end-1].Rune) {
		end--
	}
	const none = Style(-1)
	cur := none
	for _, cl := range row[:end] {
		if isBlank(cl.Rune) {
			if cur != none && palette[cur].Background != 0 {
				b.WriteString(reset)
				cur = none
			}
			b.WriteByte(' ')
			continue
		}
		if c != NoColor && cl.Style != cur {
			b.WriteString(sgr(cl.Style, c, palette))
			cur = cl.Style
		}
		b.WriteRune(cl.Rune)
	}
}

// PlainText returns f as text without escape sequences or trailing blanks,
// and without the blank rows above and below what it shows.
func PlainText(f *Frame) string {
	var b bytes.Buffer
	for y := range Rows {
		writeRow(&b, &f.Cells[y], NoColor, nil)
		b.WriteByte('\n')
	}
	return strings.Trim(b.String(), "\n") + "\n"
}

// Duration returns how long the frames take to play once.
func Duration(frames []Frame) (d time.Duration) {
	for _, f := range frames {
		d += f.Hold
	}
	return d
}
