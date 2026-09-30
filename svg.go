package banner

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// hive.svg contains sanitized vector paths from https://iqhive.com/img/icons/logo.svg.
// Downloaded 2026-10-01; generation needs no network access.
//
//go:embed hive.svg
var hiveSVG string

// Geometry of the SVG, in pixels.
const (
	cellW     = 10 // one column
	cellH     = 22 // one row
	baseline  = 16 // from the top of a row to the text's baseline
	padX      = 18 // left and right of the text
	titleH    = 32 // the window's title bar
	padTop    = 12 // between the title bar and the first row
	padBottom = 14 // below the last row
	svgW      = 2*padX + Cols*cellW
	svgH      = titleH + padTop + Rows*cellH + padBottom
)

// fontCSS is the text's font. It names only fonts a viewer may have
// installed, since an SVG shown as an image cannot load one. Every segment of
// text carries its intended width in textLength, so a font whose advance is
// not 0.6em still keeps the columns aligned.
const fontCSS = `16.5px ui-monospace,SFMono-Regular,"SF Mono",Menlo,Consolas,"Liberation Mono","DejaVu Sans Mono",monospace`

// A segment is a stretch of one row drawn by one element: text in one Style,
// or block glyphs in one Style, drawn as rectangles.
type segment struct {
	y, x, n int
	s       Style
	logo    bool
	block   rune   // Full or Shade for blocks, 0 for text
	text    string // for text
}

// segments splits f into segments. A text segment runs on over a single blank
// between cells of its Style, so that a phrase is one element; two blanks end
// it.
func segments(f *Frame) []segment {
	var out []segment
	for y := range Rows {
		row := &f.Cells[y]
		for x := 0; x < Cols; {
			c := row[x]
			switch {
			case isBlank(c.Rune):
				x++
			case isBlock(c.Rune):
				e := x + 1
				for e < Cols && row[e] == c {
					e++
				}
				out = append(out, segment{y: y, x: x, n: e - x, s: c.Style, block: c.Rune})
				x = e
			default:
				e := textEnd(row, x)
				var t strings.Builder
				for _, d := range row[x:e] {
					if isBlank(d.Rune) {
						t.WriteByte(' ')
					} else {
						t.WriteRune(d.Rune)
					}
				}
				out = append(out, segment{y: y, x: x, n: e - x, s: c.Style, text: t.String()})
				x = e
			}
		}
	}
	if f.HiveLogo {
		out = append(out, segment{logo: true})
	}
	return out
}

// textEnd returns where the text segment that starts at column x ends.
func textEnd(row *[Cols]Cell, x int) int {
	st := row[x].Style
	same := func(i int) bool {
		return i < Cols && row[i].Style == st && !isBlank(row[i].Rune) && !isBlock(row[i].Rune)
	}
	e := x + 1
	for {
		switch {
		case same(e):
			e++
		case e < Cols && isBlank(row[e].Rune) && same(e+1):
			e += 2
		default:
			return e
		}
	}
}

// A span is a range of frames [from, to).
type span struct{ from, to int }

// A shot is a set of segments that are on screen in exactly the same spans
// of frames, and so share one animation.
type shot struct {
	spans []span
	segs  []segment
}

// WriteSVG renders frames as an animated SVG that loops forever.
//
// Each distinct segment becomes one element, however many frames show it,
// and the elements shown in the same frames share a group whose visibility a
// CSS animation switches on and off. There is no script and nothing external, so
// it plays where an SVG is shown as an image, as a README's is. Where CSS
// animation is not run, or the viewer prefers reduced motion, it shows the
// project title card.
func WriteSVG(w io.Writer, a Animation) error {
	a, err := a.WithTheme(a.theme())
	if err != nil {
		return err
	}
	if err := a.Validate(); err != nil {
		return err
	}
	theme := a.theme()
	palette := theme.palette(a.letters())
	frames := a.Frames
	var buffer bytes.Buffer
	b := &buffer
	n := len(frames)
	start := make([]time.Duration, n+1)
	for i, f := range frames {
		start[i+1] = start[i] + f.Hold
	}
	total := start[n]

	shown := map[segment][]int{}
	var order []segment
	for i := range frames {
		for _, r := range segments(&frames[i]) {
			if shown[r] == nil {
				order = append(order, r)
			}
			shown[r] = append(shown[r], i)
		}
	}
	byKey := map[string]*shot{}
	var shots []*shot
	for _, r := range order {
		var sp []span
		for _, i := range shown[r] {
			if len(sp) > 0 && sp[len(sp)-1].to == i {
				sp[len(sp)-1].to++
			} else {
				sp = append(sp, span{i, i + 1})
			}
		}
		k := fmt.Sprint(sp)
		s := byKey[k]
		if s == nil {
			s = &shot{spans: sp}
			byKey[k] = s
			shots = append(shots, s)
		}
		s.segs = append(s.segs, r)
	}

	pct := func(i int) string {
		p := strconv.FormatFloat(float64(start[i])*100/float64(total), 'f', 3, 64)
		return strings.TrimSuffix(strings.TrimRight(p, "0"), ".") + "%"
	}

	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-labelledby="hero-title hero-desc">`+"\n", svgW, svgH, svgW, svgH)
	b.WriteString(`<title id="hero-title">`)
	escape(b, a.Title)
	b.WriteString("</title>\n")
	b.WriteString(`<desc id="hero-desc">`)
	escape(b, a.Description)
	b.WriteString("</desc>\n")

	b.WriteString("<style>\n")
	fmt.Fprintf(b, ".a{visibility:hidden;animation:%s step-end infinite}.l{visibility:visible}\n", strconv.FormatFloat(total.Seconds(), 'f', -1, 64)+"s")
	b.WriteString("@media (prefers-reduced-motion:reduce){.a{animation:none!important}}\n")
	b.WriteString(".t{font:")
	escape(b, theme.Font)
	b.WriteString(";font-variant-ligatures:none}\n.title{font:")
	escape(b, theme.Font)
	b.WriteString(";font-size:13px}\n")
	used := map[Style]bool{}
	for _, s := range shots {
		for _, r := range s.segs {
			used[r.s] = true
		}
	}
	for i := range palette {
		s := Style(i)
		if !used[s] {
			continue
		}
		p := palette[s]
		fmt.Fprintf(b, ".c%d{fill:#%06x", s, p.Foreground)
		if p.Bold {
			b.WriteString(";font-weight:700")
		}
		b.WriteString("}")
		if p.Background != 0 {
			fmt.Fprintf(b, ".g%d{fill:#%06x}", s, p.Background)
		}
	}
	b.WriteString("\n")
	for id, s := range shots {
		if animated(s, n) {
			fmt.Fprintf(b, ".k%d{animation-name:k%d}", id, id)
		}
	}
	b.WriteString("\n")
	for id, s := range shots {
		if !animated(s, n) {
			continue
		}
		var hidden, visible []string
		if s.spans[0].from > 0 {
			hidden = append(hidden, "0%")
		}
		for _, sp := range s.spans {
			visible = append(visible, pct(sp.from))
			if sp.to < n {
				hidden = append(hidden, pct(sp.to))
			}
		}
		if s.spans[len(s.spans)-1].to == n {
			visible = append(visible, "100%")
		} else {
			hidden = append(hidden, "100%")
		}
		fmt.Fprintf(b, "@keyframes k%d{%s{visibility:hidden}%s{visibility:visible}}\n", id, strings.Join(hidden, ","), strings.Join(visible, ","))
	}

	b.WriteString("</style>\n")

	// The window: a dark body, a lighter title bar with three dots, and the
	// command that plays this in a terminal as its title.
	fmt.Fprintf(b, `<rect width="%d" height="%d" rx="10" fill="#%06x"/>`+"\n", svgW, svgH, theme.Body)
	fmt.Fprintf(b, `<path d="M0 10a10 10 0 0 1 10-10h%da10 10 0 0 1 10 10v%dH0z" fill="#%06x"/>`+"\n", svgW-20, titleH-10, theme.TitleBar)
	fmt.Fprintf(b, `<path d="M0 %d.5h%d" stroke="#%06x"/>`+"\n", titleH, svgW, theme.Border)
	fmt.Fprintf(b, `<rect x=".5" y=".5" width="%d" height="%d" rx="9.5" fill="none" stroke="#%06x"/>`+"\n", svgW-1, svgH-1, theme.Border)
	for i, c := range theme.Controls {
		fmt.Fprintf(b, `<circle cx="%d" cy="16" r="6" fill="#%06x"/>`, 20+20*i, c)
	}
	b.WriteString("\n")
	fmt.Fprintf(b, `<text x="%d" y="21" text-anchor="middle" fill="#%06x" class="title">`, svgW/2, theme.TitleText)
	escape(b, a.Command)
	b.WriteString("</text>\n")

	fmt.Fprintf(b, `<g class="t" transform="translate(%d %d)">`+"\n", padX, titleH+padTop)
	for id, s := range shots {
		switch {
		case animated(s, n) && shownIn(s, a.StaticFrame):
			fmt.Fprintf(b, `<g class="a l k%d">`, id)
		case animated(s, n):
			fmt.Fprintf(b, `<g class="a k%d">`, id)
		default:
			b.WriteString("<g>")
		}
		writeShot(b, s, palette)
		b.WriteString("</g>\n")
	}
	b.WriteString("</g>\n</svg>\n")
	data := b.Bytes()
	nwrite, err := w.Write(data)
	if err == nil && nwrite != len(data) {
		return io.ErrShortWrite
	}
	return err
}

// animated reports whether a shot is ever off screen.
func animated(s *shot, frames int) bool {
	return len(s.spans) != 1 || s.spans[0] != span{0, frames}
}

// writeShot writes a shot's segments. For each Style it draws the
// backgrounds of its text as one path, its blocks as another, and then its
// text in a group.
func writeShot(b *bytes.Buffer, s *shot, palette []Paint) {
	var styles []Style
	text := map[Style][]segment{}
	blocks := map[Style][]segment{}
	for _, r := range s.segs {
		if r.logo {
			fmt.Fprintf(b, `<rect x="108" y="96" width="584" height="163" rx="12" fill="#0d1117"/><svg x="120" y="108" width="560" height="139" viewBox="0 0 890 220">%s</svg>`, hiveSVG[strings.Index(hiveSVG, "<g>"):strings.LastIndex(hiveSVG, "</svg>")])
			continue
		}
		if text[r.s] == nil && blocks[r.s] == nil {
			styles = append(styles, r.s)
		}
		if r.block != 0 {
			blocks[r.s] = append(blocks[r.s], r)
		} else {
			text[r.s] = append(text[r.s], r)
		}
	}
	for _, st := range styles {
		p := palette[st]
		if rs := text[st]; rs != nil && p.Background != 0 {
			fmt.Fprintf(b, `<path class="g%d" d="`, st)
			for _, r := range rs {
				fmt.Fprintf(b, "M%d %dh%dv%dh-%dz", r.x*cellW, r.y*cellH, r.n*cellW, cellH, r.n*cellW)
			}
			b.WriteString(`"/>`)
		}
		if rs := blocks[st]; rs != nil {
			fmt.Fprintf(b, `<path class="c%d" d="`, st)
			for _, r := range rs {
				tile := p.Tile
				if tile == 0 {
					tile = r.n
				}
				for i := 0; i < r.n; i += tile {
					w := min(tile, r.n-i)*cellW - 2
					fmt.Fprintf(b, "M%d %dh%dv%dh-%dz", (r.x+i)*cellW+1, r.y*cellH+2, w, cellH-4, w)
				}
			}
			b.WriteString(`"/>`)
		}
		if rs := text[st]; rs != nil {
			fmt.Fprintf(b, `<g class="c%d">`, st)
			for _, r := range rs {
				fmt.Fprintf(b, `<text x="%d" y="%d"`, r.x*cellW, r.y*cellH+baseline)
				fmt.Fprintf(b, ` textLength="%d"`, r.n*cellW)
				b.WriteString(">")
				escape(b, r.text)
				b.WriteString("</text>")
			}
			b.WriteString("</g>")
		}
	}
}

func escape(b *bytes.Buffer, s string) {
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
}

// shownIn reports whether the shot belongs to the static project card.
func shownIn(s *shot, i int) bool {
	for _, sp := range s.spans {
		if sp.from <= i && i < sp.to {
			return true
		}
	}
	return false
}
