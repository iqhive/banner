package banner

import (
	"errors"
	"math"
	"strings"
)

// Paint controls a semantic style in both SVG and terminal output. Colors are
// 24-bit RGB. Background zero means transparent; Tile zero joins block cells
// into one bar, while a positive Tile draws separate groups of that many cells.
type Paint struct {
	Foreground, Background uint32
	Bold                   bool
	Tile                   int
}

// Theme owns appearance, independently of project content and scene timing.
// Start with DefaultTheme or LightTheme and customize its fields. Each factory
// returns independent slices, so changes cannot affect another animation.
// Styles has exactly Mark entries; wordmark styles are generated from its stops.
// Font is a local CSS font shorthand, never a URL or external font declaration.
// The embedded IQ Hive brand artwork keeps its original colors under all themes.
type Theme struct {
	Name                              string
	Body, TitleBar, Border, TitleText uint32
	Controls                          [3]uint32
	Font                              string
	Styles                            []Paint
	WordmarkStops                     []uint32
	WordmarkTint                      float64
	// UppercaseWordmark changes shared opening glyphs, never commands or captions.
	UppercaseWordmark bool
	// SansSerifI draws uppercase I as a single vertical pixel stroke.
	SansSerifI bool
	// WordmarkPrefix receives a horizontal gradient from WordmarkPrefixStops.
	// Remaining letters use WordmarkStops. Both prefix fields must be set together.
	WordmarkPrefix      string
	WordmarkPrefixStops []uint32
}

// BlueGreenTheme preserves the original teal, cyan, blue and violet theme.
func BlueGreenTheme() Theme {
	return Theme{
		Name: "bluegreen", Body: 0x0d1117, TitleBar: 0x161b22, Border: 0x30363d, TitleText: 0x8b949e,
		Controls: [3]uint32{0xff5f57, 0xfebc2e, 0x28c840}, Font: fontCSS,
		Styles: []Paint{
			Plain: {Foreground: 0xc9d1d9}, Dim: {Foreground: 0x444c56}, Muted: {Foreground: 0x8b949e},
			Bright: {Foreground: 0xf0f6fc, Bold: true}, Literal: {Foreground: 0xa5d6ff},
			Changed: {Foreground: 0xf5a524, Bold: true, Tile: 2}, On: {Foreground: 0x58a6ff, Tile: 2},
			Off: {Foreground: 0x262c36, Tile: 2}, Bar: {Foreground: 0x58a6ff},
			Prompt: {Foreground: 0x3fb950, Bold: true}, Caret: {Foreground: 0x0d1117, Background: 0xc9d1d9},
		}, WordmarkStops: []uint32{0x2dd4bf, 0x22d3ee, 0x38bdf8, 0x60a5fa, 0x818cf8, 0xa78bfa}, WordmarkTint: 0.25,
	}
}

// DefaultTheme uses the exact IQ Hive yellow/orange/red fills for wordmarks.
// Scene styles and window controls remain bluegreen, so the warm opening gives
// way to cool diagrams, literals and a green prompt. Closing wordmarks retain
// the warm identity; the embedded logo always keeps its original brand colors.
func DefaultTheme() Theme {
	t := BlueGreenTheme()
	t.Name = "default"
	// Exact SVG fills in hive.svg; keep these synchronized with the brand artwork.
	t.WordmarkStops = []uint32{0xfff343, 0xffb60b, 0xff970d, 0xff6f0d, 0xff120b}
	return t
}

// IQCapsTheme renders uppercase names, with the IQ prefix in the exact warm
// logo colors and remaining letters in bluegreen. All scene styles, controls
// and window colors are inherited unchanged from BlueGreenTheme. The uppercase
// I is a straight sans-serif stroke, matching the IQ Hive wordmark.
func IQCapsTheme() Theme {
	t := BlueGreenTheme()
	t.Name = "iqcaps"
	t.UppercaseWordmark = true
	t.SansSerifI = true
	t.WordmarkPrefix = "IQ"
	t.WordmarkPrefixStops = DefaultTheme().WordmarkStops
	return t
}

// LightTheme demonstrates a complete alternative, including diagram and cursor
// colors. The brand card uses a dark vector backing so the white logo remains legible.
func LightTheme() Theme {
	t := BlueGreenTheme()
	t.Name = "light"
	t.Body = 0xf6f8fa
	t.TitleBar = 0xeaeef2
	t.Border = 0xd0d7de
	t.TitleText = 0x57606a
	t.Styles[Plain].Foreground = 0x24292f
	t.Styles[Dim].Foreground = 0xafb8c1
	t.Styles[Muted].Foreground = 0x57606a
	t.Styles[Bright].Foreground = 0x1f2328
	t.Styles[Literal].Foreground = 0x0550ae
	t.Styles[Changed].Foreground = 0x9a6700
	t.Styles[On].Foreground = 0x0969da
	t.Styles[Off].Foreground = 0xd0d7de
	t.Styles[Bar].Foreground = 0x0969da
	t.Styles[Prompt].Foreground = 0x1a7f37
	t.Styles[Caret] = Paint{Foreground: t.Body, Background: 0x24292f}
	t.WordmarkStops = []uint32{0x087f75, 0x09899b, 0x0969da, 0x5952b5, 0x8250df}
	t.WordmarkTint = 0.15
	return t
}

// ThemeByName returns a fresh built-in theme or an error for an unknown name.
func ThemeByName(name string) (Theme, error) {
	switch name {
	case "default":
		return DefaultTheme(), nil
	case "bluegreen":
		return BlueGreenTheme(), nil
	case "iqcaps":
		return IQCapsTheme(), nil
	case "light":
		return LightTheme(), nil
	default:
		return Theme{}, errors.New("banner: unknown theme; use default, bluegreen, iqcaps or light")
	}
}

func (t Theme) validate() error {
	if len(t.Styles) != int(Mark) || len(t.WordmarkStops) == 0 {
		return errors.New("banner: theme needs base styles and wordmark colors")
	}
	if math.IsNaN(t.WordmarkTint) || t.WordmarkTint < 0 || t.WordmarkTint > 1 {
		return errors.New("banner: wordmark tint must be between zero and one")
	}
	if (t.WordmarkPrefix == "") != (len(t.WordmarkPrefixStops) == 0) {
		return errors.New("banner: wordmark prefix and colors must be provided together")
	}
	colors := []uint32{t.Body, t.TitleBar, t.Border, t.TitleText}
	colors = append(colors, t.Controls[:]...)
	colors = append(colors, t.WordmarkStops...)
	colors = append(colors, t.WordmarkPrefixStops...)
	for _, p := range t.Styles {
		colors = append(colors, p.Foreground, p.Background)
		if p.Tile < 0 {
			return errors.New("banner: negative diagram tile width")
		}
	}
	for _, c := range colors {
		if c > 0xffffff {
			return errors.New("banner: colors must be 24-bit RGB")
		}
	}
	if t.Font == "" || strings.ContainsAny(t.Font, "{}<>;@\\\r\n") || strings.Contains(strings.ToLower(t.Font), "url(") {
		return errors.New("banner: font must name local fonts only")
	}
	return nil
}

func (t Theme) palette(letters int) []Paint {
	p := append([]Paint(nil), t.Styles...)
	add := func(stops []uint32, count int) {
		for i := range count {
			at := float64(i) * float64(len(stops)-1) / float64(max(count-1, 1))
			lo := int(at)
			c := blend(stops[min(lo+1, len(stops)-1)], stops[lo], at-float64(lo))
			p = append(p, Paint{Foreground: c, Background: blend(c, t.Body, t.WordmarkTint), Bold: true})
		}
	}
	if len(t.WordmarkPrefixStops) > 0 {
		add(t.WordmarkPrefixStops, min(letters, prefixColorSteps))
		letters = max(0, letters-prefixColorSteps)
	}
	add(t.WordmarkStops, letters)
	return p
}

func blend(a, b uint32, alpha float64) uint32 {
	var c uint32
	for shift := 16; shift >= 0; shift -= 8 {
		x := alpha*float64(a>>shift&0xff) + (1-alpha)*float64(b>>shift&0xff)
		c |= uint32(math.Round(x)) << shift
	}
	return c
}
