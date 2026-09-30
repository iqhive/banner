package banner

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestIQCapsPaletteAndGlyphs(t *testing.T) {
	theme := IQCapsTheme()
	if !reflect.DeepEqual(theme.Styles, BlueGreenTheme().Styles) || theme.Controls != BlueGreenTheme().Controls {
		t.Fatal("scene palette changed")
	}
	palette := theme.palette(19)
	warm := DefaultTheme().WordmarkStops
	if palette[Mark].Foreground != warm[0] || palette[Mark+15].Foreground != warm[len(warm)-1] {
		t.Fatal("prefix endpoints differ from logo")
	}
	if !reflect.DeepEqual(palette[Mark+16:], BlueGreenTheme().palette(3)[Mark:]) {
		t.Fatal("suffix gradient differs")
	}
	for _, word := range []string{"iqkvs", "iqhash", "prefixlookup", "mcpserver", "ABCDEFGHIJKLMNOPQRSTUVWXYZ"[:12]} {
		layout, err := LayoutWordmark(strings.ToUpper(word), WordmarkOptions{SansSerifI: true})
		if err != nil || layout.Gap < 1 {
			t.Fatalf("%s layout: %+v %v", word, layout, err)
		}
		_, card, err := Opening(Card{Word: word, Theme: theme})
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, c := range layout.Cells {
			cell := card.Cells[c.Y][c.X]
			if cell.Rune != c.Digit || cell.Style != wordmarkStyle(theme, strings.ToUpper(word), layout, c) {
				t.Fatal("uppercase glyph or split color missing")
			}
			count++
		}
		actual := 0
		for _, row := range card.Cells {
			for _, c := range row {
				if c.Style >= Mark {
					actual++
				}
			}
		}
		if actual != count {
			t.Fatal("extra wordmark cells")
		}
	}
	other := IQCapsTheme()
	theme.WordmarkPrefixStops[0] = 0
	if other.WordmarkPrefixStops[0] != warm[0] {
		t.Fatal("theme shares mutable prefix stops")
	}
	invalid := other
	invalid.WordmarkPrefix = ""
	if invalid.validate() == nil {
		t.Fatal("accepted incomplete prefix configuration")
	}
	invalid = IQCapsTheme()
	invalid.WordmarkPrefixStops[0] = 0x1000000
	if invalid.validate() == nil {
		t.Fatal("accepted non-RGB prefix color")
	}
}

func TestIQCapsOverridesPreserveFrames(t *testing.T) {
	frames, card, err := Opening(Card{Word: "iqkvs", Install: "go get example.com/iqkvs"})
	if err != nil {
		t.Fatal(err)
	}
	card.Hold = time.Second
	card.Text(1, 19, "app overlay", Bright)
	frames = append(frames, card)
	a := Animation{Frames: frames, StaticFrame: len(frames) - 1, Title: "test", Description: "test", Command: "test"}
	converted, err := a.WithTheme(IQCapsTheme())
	if err != nil {
		t.Fatal(err)
	}
	if len(converted.Frames) != len(frames) || Duration(converted.Frames) != Duration(frames) {
		t.Fatal("theme changed timing")
	}
	if converted.Frames[a.StaticFrame].Cells[19] != card.Cells[19] {
		t.Fatal("lost app overlay")
	}
	back, err := converted.WithTheme(DefaultTheme())
	if err != nil {
		t.Fatal(err)
	}
	for i := range frames {
		if back.Frames[i].Cells != frames[i].Cells || back.Frames[i].Hold != frames[i].Hold {
			t.Fatalf("roundtrip changed frame %d", i)
		}
	}
	if a.Frames[a.StaticFrame].Cells != card.Cells {
		t.Fatal("mutated input")
	}
	var svg bytes.Buffer
	a.Theme = IQCapsTheme()
	if err := WriteSVG(&svg, a); err != nil {
		t.Fatal(err)
	}
	var expected bytes.Buffer
	if err := WriteSVG(&expected, converted); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(svg.Bytes(), expected.Bytes()) {
		t.Fatal("renderer did not resolve theme")
	}
	var output, errs bytes.Buffer
	if code := RunCLI([]string{"-theme", "iqcaps"}, &output, &errs, func(uint64) (Animation, error) { a.Theme = Theme{}; return a, nil }); code != 0 {
		t.Fatal(code, errs.String())
	}
	if output.String() != PlainText(&converted.Frames[a.StaticFrame]) {
		t.Fatal("CLI theme override differs")
	}
}

func TestIQCapsSansSerifI(t *testing.T) {
	for _, font := range []map[rune]Glyph{nil, CompactFont(), DefaultFont()} {
		before := font['I']
		layout, err := LayoutWordmark("IQKVS", WordmarkOptions{Font: font, SansSerifI: true})
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, c := range layout.Cells {
			if c.Letter == 0 {
				if c.X < layout.Left || c.X > layout.Left+1 || c.Y < markTop || c.Y >= markTop+7 {
					t.Fatal("I has a serif or invalid height")
				}
				count++
			}
		}
		if count != 14 || layout.Gap < 1 {
			t.Fatal("I must be seven pixels tall with two character cells per pixel")
		}
		if font != nil && font['I'] != before {
			t.Fatal("modified caller font")
		}
	}
	// A typography-only override must rebuild the shared opening.
	theme := IQCapsTheme()
	theme.SansSerifI = false
	frames, card, err := Opening(Card{Word: "iqkvs", Theme: theme})
	if err != nil {
		t.Fatal(err)
	}
	card.Hold = time.Second
	frames = append(frames, card)
	a := Animation{Frames: frames, Theme: theme, StaticFrame: len(frames) - 1, Title: "test", Description: "test", Command: "test"}
	changed, err := a.WithTheme(IQCapsTheme())
	if err != nil {
		t.Fatal(err)
	}
	_, expected, err := Opening(Card{Word: "iqkvs", Theme: IQCapsTheme()})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Frames[a.StaticFrame].Cells != expected.Cells {
		t.Fatal("theme override did not replace serif I")
	}
	if a.Frames[a.StaticFrame].Cells != card.Cells || Duration(changed.Frames) != Duration(frames) {
		t.Fatal("override mutated input or timing")
	}
}
