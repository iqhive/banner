package banner

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Glyph is a nine-row pixel letter. '#' lights both character columns; '<'
// lights only the left column and '>' only the right. '.' and space are blank.
// A pixel occupies two terminal columns. Rows may differ in width; layout uses
// the widest row, which prevents wide lower strokes touching the next letter.
type Glyph [9]string

// WordmarkOptions controls deterministic decoration and spacing. Gap zero
// selects the default two columns between letters. Layout reduces that gap only
// to fit; it preserves at least one column whenever one column fits. Font nil
// selects DefaultFont, falling back to CompactFont for long names. Negative Gap
// is invalid. Seed changes decorative digits.
type WordmarkOptions struct {
	Seed uint64
	Gap  int
	Font map[rune]Glyph
	// SansSerifI replaces uppercase I with a straight stroke in either font.
	SansSerifI bool
}

// MarkCell identifies one lit character and its letter's gradient index.
type MarkCell struct {
	X, Y, Letter int
	Digit        rune
}

// Wordmark records the centered layout and its actual inter-letter gap.
type Wordmark struct {
	Cells            []MarkCell
	Left, Width, Gap int
}

// LayoutWordmark lays out a supported name without clipping or resizing glyphs.
// Unsupported glyphs and names too wide even without gaps return errors.
func LayoutWordmark(word string, opts WordmarkOptions) (Wordmark, error) {
	font := opts.Font
	if font == nil {
		font = DefaultFont()
	}
	if opts.SansSerifI {
		copyFont := make(map[rune]Glyph, len(font))
		for r, g := range font {
			copyFont[r] = g
		}
		copyFont['I'] = Glyph{"#", "#", "#", "#", "#", "#", "#"}
		font = copyFont
	}
	letters := []rune(word)
	if len(letters) == 0 || opts.Gap < 0 {
		return Wordmark{}, errors.New("banner: wordmark needs letters and nonnegative spacing")
	}
	widths := make([]int, len(letters))
	base := 0
	for i, l := range letters {
		g, ok := font[l]
		if !ok {
			return Wordmark{}, fmt.Errorf("banner: unsupported wordmark glyph %q", l)
		}
		for _, row := range g {
			for _, pixel := range row {
				if pixel != '#' && pixel != '<' && pixel != '>' && pixel != '.' && pixel != ' ' {
					return Wordmark{}, errors.New("banner: glyphs must use ASCII #, <, >, . or space")
				}
			}
			widths[i] = max(widths[i], len(row))
		}
		if widths[i] == 0 {
			return Wordmark{}, errors.New("banner: empty glyph")
		}
		base += 2 * widths[i]
	}
	if opts.Font == nil && base+len(letters)-1 > Cols {
		opts.Font = CompactFont()
		return LayoutWordmark(word, opts)
	}
	if base > Cols {
		return Wordmark{}, errors.New("banner: wordmark is wider than the terminal")
	}
	gap := opts.Gap
	if gap == 0 {
		gap = 2
	}
	if len(letters) > 1 {
		gap = min(gap, (Cols-base)/(len(letters)-1))
	}
	width := base + gap*(len(letters)-1)
	out := Wordmark{Left: (Cols - width) / 2, Width: width, Gap: gap}
	x := out.Left
	for i, l := range letters {
		digits := fmt.Sprintf("%016x", noise(opts.Seed, string(l)))
		k := 0
		for y, row := range font[l] {
			for col, pixel := range row {
				if pixel == '#' || pixel == '<' || pixel == '>' {
					for d := range 2 {
						if pixel == '<' && d == 1 || pixel == '>' && d == 0 {
							continue
						}
						out.Cells = append(out.Cells, MarkCell{x + 2*col + d, markTop + y, i, rune(digits[k%16])})
						k++
					}
				}
			}
		}
		x += 2*widths[i] + gap
	}
	return out, nil
}

const markTop = 2

func noise(seed uint64, s string) uint64 {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", seed, s)))
	return binary.BigEndian.Uint64(sum[:8])
}

func hexDigit(d uint64, i int) rune { return rune("0123456789abcdef"[d>>(60-4*i)&15]) }

// TaglineReveal selects when the opening's header line becomes visible.
// Its zero value reveals the header with the title sweep.
type TaglineReveal string

const (
	// TaglineBefore shows the complete header as the opening starts.
	TaglineBefore TaglineReveal = "before"
	// TaglineWithReveal reveals header columns with the wordmark sweep (default).
	TaglineWithReveal TaglineReveal = "with reveal"
	// TaglineAfter preserves the delayed reveal after the wordmark settles.
	TaglineAfter TaglineReveal = "after"
)

// Card supplies project-specific words for the common opening sweep. ReadHold
// defaults to 900ms; it controls only the settled opening card, not sweep speed.
// Theme controls the opening typography; enclosing Animation.Theme can override
// it when rendered or played.
type Card struct {
	Word, Tagline, Supporting, Install string
	Wordmark                           WordmarkOptions
	Theme                              Theme
	ReadHold                           time.Duration
	// TaglineReveal defaults to TaglineWithReveal (including the empty string).
	TaglineReveal TaglineReveal
	// Points optionally replaces Supporting with separately revealed points.
	// Points stay at their final centered positions; the separator appears with
	// the following point. Use either Supporting or Points, never both.
	Points []string
	// PointInterval defaults to 200ms between point reveals.
	PointInterval time.Duration
}

// Opening returns the shared noise sweep and its fully settled title card.
// Noise is decorative SHA-256 data, unrelated to any demonstrated project output.
// The returned card has no hold; callers choose its final reading duration.
func Opening(spec Card) ([]Frame, Frame, error) {
	mode := spec.TaglineReveal
	if mode == "" {
		mode = TaglineWithReveal
	}
	if mode != TaglineBefore && mode != TaglineWithReveal && mode != TaglineAfter {
		return nil, Frame{}, errors.New("banner: invalid tagline reveal mode")
	}
	theme := Animation{Theme: spec.Theme}.theme()
	if err := theme.validate(); err != nil {
		return nil, Frame{}, err
	}
	word := spec.Word
	if theme.UppercaseWordmark {
		word = strings.ToUpper(word)
	}
	opts := spec.Wordmark
	opts.SansSerifI = opts.SansSerifI || theme.SansSerifI
	layout, err := LayoutWordmark(word, opts)
	if err != nil {
		return nil, Frame{}, err
	}
	read := spec.ReadHold
	if read == 0 {
		read = 900 * time.Millisecond
	}
	if read < 0 {
		return nil, Frame{}, errors.New("banner: negative card hold")
	}
	supporting := spec.Supporting
	interval := spec.PointInterval
	if interval == 0 {
		interval = 200 * time.Millisecond
	}
	if interval < 0 {
		return nil, Frame{}, errors.New("banner: negative point interval")
	}
	if len(spec.Points) > 0 {
		if supporting != "" {
			return nil, Frame{}, errors.New("banner: choose Supporting or Points")
		}
		for _, point := range spec.Points {
			if strings.TrimSpace(point) == "" {
				return nil, Frame{}, errors.New("banner: supporting points must not be empty")
			}
		}
		supporting = strings.Join(spec.Points, " · ")
	}
	for _, line := range []string{spec.Tagline, supporting, "$ " + spec.Install} {
		if utf8.RuneCountInString(line) > Cols {
			return nil, Frame{}, errors.New("banner: card text is wider than the terminal")
		}
	}
	inMark := make(map[[2]int]MarkCell, len(layout.Cells))
	for _, c := range layout.Cells {
		inMark[[2]int{c.X, c.Y}] = c
	}
	left, right := max(0, layout.Left-4), min(Cols, layout.Left+layout.Width+4)
	var frames []Frame
	var f Frame
	cut := func(hold time.Duration) { frame := f; frame.Hold = hold; frames = append(frames, frame) }
	for step := range 22 {
		f = Frame{}
		for y := markTop - 1; y <= markTop+9; y++ {
			for x := left; x < right; x++ {
				digit := hexDigit(noise(spec.Wordmark.Seed, fmt.Sprintf("%d:%d:%d", step/2, y, x/16)), x%16)
				sweep := (x - left) * 11 / (right - left)
				jitter := int(noise(spec.Wordmark.Seed, fmt.Sprintf("%d:%d", y, x)) % 6)
				if c, ok := inMark[[2]int{x, y}]; ok {
					if step >= 3+sweep+jitter {
						f.Cells[y][x] = Cell{Rune: c.Digit, Style: wordmarkStyle(theme, word, layout, c)}
						continue
					}
				} else if step >= 4+sweep+jitter {
					continue
				}
				f.Cells[y][x] = Cell{Rune: digit, Style: Dim}
			}
		}
		switch mode {
		case TaglineBefore:
			f.Center(13, spec.Tagline, Plain)
		case TaglineWithReveal:
			// Use the same column-to-sweep mapping as the wordmark, without
			// decorative jitter: the header stays readable as it is revealed.
			runes := []rune(spec.Tagline)
			start := (Cols - len(runes)) / 2
			for i, r := range runes {
				x := start + i
				sweep := (min(max(x, left), right) - left) * 11 / (right - left)
				if step >= 3+sweep {
					f.Cells[13][x] = Cell{Rune: r, Style: Plain}
				}
			}
		}
		cut(75 * time.Millisecond)
	}
	f.Center(13, spec.Tagline, Plain)
	cut(160 * time.Millisecond)
	if len(spec.Points) == 0 {
		f.Center(14, supporting, Muted)
		cut(160 * time.Millisecond)
	} else {
		x := (Cols - utf8.RuneCountInString(supporting)) / 2
		for i := range spec.Points {
			f.Text(x, 14, strings.Join(spec.Points[:i+1], " · "), Muted)
			cut(interval)
		}
	}
	x := f.Center(17, "$ "+spec.Install, Plain)
	f.Text(x, 17, "$", Prompt)
	cut(read)
	f.Hold = 0
	spec.TaglineReveal = mode
	spec.Points = append([]string(nil), spec.Points...)
	if spec.Wordmark.Font != nil {
		font := make(map[rune]Glyph, len(spec.Wordmark.Font))
		for r, g := range spec.Wordmark.Font {
			font[r] = g
		}
		spec.Wordmark.Font = font
	}
	spec.Theme = theme
	spec.Theme.Styles = append([]Paint(nil), theme.Styles...)
	spec.Theme.WordmarkStops = append([]uint32(nil), theme.WordmarkStops...)
	spec.Theme.WordmarkPrefixStops = append([]uint32(nil), theme.WordmarkPrefixStops...)
	source := &openingSource{card: spec, frames: append([]Frame(nil), frames...)}
	for i := range frames {
		frames[i].opening = source
		frames[i].openingIndex = i
	}
	f.opening = source
	f.openingIndex = len(frames) - 1
	return frames, f, nil
}

// Sixteen prefix slots allow a spatial gradient even for the two letters IQ.
// Suffix styles follow those slots and retain a separate per-letter gradient.
const prefixColorSteps = 16

func wordmarkStyle(t Theme, word string, layout Wordmark, c MarkCell) Style {
	if len(t.WordmarkPrefixStops) == 0 {
		return Mark + Style(c.Letter)
	}
	prefix := 0
	if strings.HasPrefix(word, t.WordmarkPrefix) {
		prefix = utf8.RuneCountInString(t.WordmarkPrefix)
	}
	if c.Letter >= prefix {
		return Mark + prefixColorSteps + Style(c.Letter-prefix)
	}
	left, right := Cols, 0
	for _, cell := range layout.Cells {
		if cell.Letter < prefix {
			left = min(left, cell.X)
			right = max(right, cell.X)
		}
	}
	return Mark + Style((c.X-left)*(prefixColorSteps-1)/max(1, right-left))
}
