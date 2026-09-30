package banner

import (
	"errors"
	"math"
	"unicode"
	"unicode/utf8"
)

// Animation is the shared input to SVG, terminal playback and the command CLI.
// Frames are chronological. StaticFrame explicitly selects the useful card
// shown with reduced motion, unsupported CSS animation, or redirected output.
// A zero Theme selects DefaultTheme. Metadata is escaped as text in the SVG.
// Rendering does not mutate frames or themes; RunCLI scales its builder's frames.
type Animation struct {
	Frames                      []Frame
	StaticFrame                 int
	Theme                       Theme
	Title, Description, Command string
}

func (a Animation) theme() Theme {
	if a.Theme.Name == "" && a.Theme.Styles == nil {
		return DefaultTheme()
	}
	return a.Theme
}

func (a Animation) letters() int {
	n := 0
	for _, f := range a.Frames {
		for _, row := range f.Cells {
			for _, c := range row {
				if c.Style >= Mark {
					n = max(n, int(c.Style-Mark)+1)
				}
			}
		}
	}
	return n
}

// Validate checks frame bounds, durations, styles, theme and XML-safe runes.
// Text placement clips at the grid edge; callers should test their own layouts.
func (a Animation) Validate() error {
	if len(a.Frames) == 0 || a.StaticFrame < 0 || a.StaticFrame >= len(a.Frames) {
		return errors.New("banner: animation needs frames and a valid static frame")
	}
	if a.Title == "" || a.Description == "" || a.Command == "" {
		return errors.New("banner: title, description and playback command are required")
	}
	if err := a.theme().validate(); err != nil {
		return err
	}
	for _, text := range []string{a.Title, a.Description, a.Command} {
		for _, r := range text {
			if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
				return errors.New("banner: metadata contains an invalid control character")
			}
		}
	}
	var total int64
	for _, f := range a.Frames {
		if f.Hold <= 0 || int64(f.Hold) > math.MaxInt64-total {
			return errors.New("banner: frame duration must be positive and total must not overflow")
		}
		total += int64(f.Hold)
		for _, row := range f.Cells {
			for _, c := range row {
				if c.Style < 0 || c.Style > Mark+4096 {
					return errors.New("banner: invalid cell style")
				}
				if c.Rune != 0 && (!utf8.ValidRune(c.Rune) || unicode.IsControl(c.Rune)) {
					return errors.New("banner: cell contains an invalid or control rune")
				}
			}
		}
	}
	return nil
}
