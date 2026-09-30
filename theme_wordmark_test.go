package banner

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestWordmarkSpacing(t *testing.T) {
	for _, word := range []string{"iqkvs", "iqhash", "banner", "prefixlookup"} {
		m, err := LayoutWordmark(word, WordmarkOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if m.Gap < 1 {
			t.Fatalf("%s has no letter gap", word)
		}
		lo, hi := make([]int, len(word)), make([]int, len(word))
		for i := range lo {
			lo[i] = Cols
			hi[i] = -1
		}
		for _, c := range m.Cells {
			if c.X < 0 || c.X >= Cols || c.Y < 0 || c.Y >= Rows {
				t.Fatal("clipped wordmark")
			}
			lo[c.Letter] = min(lo[c.Letter], c.X)
			hi[c.Letter] = max(hi[c.Letter], c.X)
		}
		for i := 1; i < len(lo); i++ {
			if gap := lo[i] - hi[i-1] - 1; gap < 1 {
				t.Fatalf("%s letters %d and %d touch", word, i-1, i)
			}
		}
	}
	// The v deliberately has a narrow first row and wide later rows.
	m, err := LayoutWordmark("vs", WordmarkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if m.Width != 20 {
		t.Fatalf("width %d, want 10 + 2 + 8", m.Width)
	}
}

func TestWordmarkFitsBeforeRemovingSpacing(t *testing.T) {
	font := map[rune]Glyph{'a': {"##"}}
	for _, tc := range []struct{ count, gap int }{{16, 1}, {20, 0}} {
		m, err := LayoutWordmark(strings.Repeat("a", tc.count), WordmarkOptions{Font: font})
		if err != nil {
			t.Fatal(err)
		}
		if m.Gap != tc.gap {
			t.Errorf("%d letters: gap %d, want %d", tc.count, m.Gap, tc.gap)
		}
	}
	for _, word := range []string{"", "!", strings.Repeat("a", 21)} {
		if _, err := LayoutWordmark(word, WordmarkOptions{Font: font}); err == nil {
			t.Errorf("accepted %q", word)
		}
	}
}

func TestOpeningDeterminism(t *testing.T) {
	spec := Card{Word: "iqkvs", Tagline: "A tagline.", Supporting: "A supporting line.", Install: "go get example.com/project"}
	first, card, err := Opening(spec)
	if err != nil {
		t.Fatal(err)
	}
	second, again, err := Opening(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(card, again) {
		t.Fatal("opening is not reproducible")
	}
	for _, row := range card.Cells {
		for _, c := range row {
			if c.Style == Dim && !isBlank(c.Rune) {
				t.Fatal("unneeded noise remains on the title card")
			}
		}
	}
	spec.Wordmark.Seed = 7
	different, _, err := Opening(spec)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(first, different) {
		t.Fatal("seed does not change decorative noise")
	}
	font := DefaultFont()
	font['v'] = Glyph{}
	if DefaultFont()['v'] == font['v'] {
		t.Fatal("font factory shares mutable data")
	}
}

func TestThemes(t *testing.T) {
	custom := DefaultTheme()
	custom.Name = "custom"
	custom.Body = 0x123456
	custom.TitleBar = 0x234567
	custom.Border = 0x345678
	custom.TitleText = 0x456789
	custom.Styles[Plain].Foreground = 0x56789a
	custom.WordmarkStops = []uint32{0x102030, 0x708090}
	for _, theme := range []Theme{DefaultTheme(), BlueGreenTheme(), LightTheme(), custom} {
		a := mustAnimation(t)
		a.Theme = theme
		a.Frames[a.StaticFrame].HiveLogo = true
		var out bytes.Buffer
		if err := WriteSVG(&out, a); err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(out.Bytes(), []byte("prefers-reduced-motion")) {
			t.Fatal("missing reduced-motion support")
		}
		if theme.Name == "custom" {
			for _, c := range []string{"#123456", "#234567", "#345678", "#456789", "#56789a"} {
				if !bytes.Contains(out.Bytes(), []byte(c)) {
					t.Errorf("custom theme color %s missing", c)
				}
			}
		}
	}
	changed := DefaultTheme()
	changed.Styles[Plain].Foreground = 0
	if DefaultTheme().Styles[Plain].Foreground == 0 {
		t.Fatal("theme factory shares slices")
	}
	t.Setenv("NO_COLOR", "")
	if TerminalColors() != NoColor {
		t.Fatal("NO_COLOR presence must disable coloring")
	}
}

func TestInvalidAnimation(t *testing.T) {
	mutations := []func(*Animation){
		func(a *Animation) { a.Frames = nil }, func(a *Animation) { a.StaticFrame = -1 },
		func(a *Animation) { a.Frames[0].Hold = 0 }, func(a *Animation) { a.Frames[0].Cells[0][0].Style = -1 },
		func(a *Animation) { a.Frames[0].Cells[0][0].Rune = '\x01' }, func(a *Animation) { a.Title = "\x01" },
		func(a *Animation) { a.Theme = DefaultTheme(); a.Theme.Font = "url(https://example.com/font)" },
		func(a *Animation) { a.Theme = DefaultTheme(); a.Theme.Body = 0xffffffff },
	}
	for i, change := range mutations {
		a := mustAnimation(t)
		change(&a)
		if err := WriteSVG(io.Discard, a); err == nil {
			t.Errorf("mutation %d accepted", i)
		}
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
func TestWriterFailure(t *testing.T) {
	if err := WriteSVG(shortWriter{}, mustAnimation(t)); err != io.ErrShortWrite {
		t.Fatalf("got %v, want short write", err)
	}
}

func TestFrameClearClips(t *testing.T) {
	var f Frame
	f.Text(0, 0, "test", Plain)
	f.Clear(-3, Rows+3)
	if f.Cells[0][0].Rune != 0 {
		t.Fatal("clear did not erase row")
	}
	f.Hold = time.Second
}

func TestTypingUsesRuneColumns(t *testing.T) {
	var start Frame
	start.Text(0, 2, "preserved", Plain)
	frames, final, err := TypeLine(start, 4, 4, "hello · world", TypingOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) == 0 || !strings.Contains(PlainText(&frames[0]), "▏") {
		t.Fatal("no cursor")
	}
	if !strings.Contains(PlainText(&final), "hello · world") || !strings.Contains(PlainText(&final), "preserved") {
		t.Fatal("typing changed surrounding content")
	}
	if final.Hold != 0 || strings.Contains(PlainText(&final), "▏") {
		t.Fatal("completed typing retains a cursor or hold")
	}
	if _, _, err := TypeLine(start, 79, 4, "x", TypingOptions{}); err == nil {
		t.Fatal("accepted a clipped cursor")
	}
}

func TestDefaultThemeUsesLogoColors(t *testing.T) {
	theme := DefaultTheme()
	for _, c := range theme.WordmarkStops {
		want := fmt.Sprintf("#%06X", c)
		if !strings.Contains(hiveSVG, want) {
			t.Fatalf("default accent %s is absent from the IQ Hive logo", want)
		}
	}
	if theme.Name != "default" || theme.WordmarkStops[0] != 0xfff343 || theme.WordmarkStops[len(theme.WordmarkStops)-1] != 0xff120b {
		t.Fatal("default is not the warm brand theme")
	}
	if !reflect.DeepEqual(theme.Styles, BlueGreenTheme().Styles) || theme.Controls != BlueGreenTheme().Controls {
		t.Fatal("default scenes and controls must retain bluegreen colors")
	}
	blue, err := ThemeByName("bluegreen")
	if err != nil {
		t.Fatal(err)
	}
	if blue.WordmarkStops[0] != 0x2dd4bf || blue.Styles[Bar].Foreground != 0x58a6ff {
		t.Fatal("bluegreen did not preserve original colors")
	}
}

func TestSupportingPoints(t *testing.T) {
	spec := Card{Word: "banner", Tagline: "Tagline", Install: "go get example.com/banner", Points: []string{"first", "second", "third"}}
	frames, card, err := Opening(spec)
	if err != nil {
		t.Fatal(err)
	}
	x := (Cols - utf8.RuneCountInString("first · second · third")) / 2
	for i := range 3 {
		f := frames[23+i]
		if f.Hold != 200*time.Millisecond {
			t.Fatal("point interval does not default to 200ms")
		}
		if f.Cells[14][x].Rune != 'f' {
			t.Fatal("points moved as later points appeared")
		}
		text := PlainText(&f)
		if i < 1 && strings.Contains(text, "second") {
			t.Fatal("second point revealed too early")
		}
		if i < 2 && strings.Contains(text, "third") {
			t.Fatal("third point revealed too early")
		}
	}
	if !strings.Contains(PlainText(&card), "first · second · third") {
		t.Fatal("settled card lacks points")
	}
	spec.Supporting = "conflict"
	if _, _, err := Opening(spec); err == nil {
		t.Fatal("accepted conflicting supporting fields")
	}
}

func TestTaglineRevealModes(t *testing.T) {
	spec := Card{Word: "banner", Tagline: "Header with title", Supporting: "Supporting text", Install: "go get example.com/banner"}
	with, _, err := Opening(spec)
	if err != nil {
		t.Fatal(err)
	}
	start := (Cols - len([]rune(spec.Tagline))) / 2
	previous := 0
	partial := false
	for i := range 22 {
		count := 0
		for x, cell := range with[i].Cells[13] {
			if cell.Rune != 0 {
				if x != start+count {
					t.Fatalf("noncontiguous header in frame %d", i)
				}
				count++
			}
		}
		if count < previous {
			t.Fatal("header reveal goes backwards")
		}
		if count > 0 && count < len([]rune(spec.Tagline)) {
			partial = true
		}
		previous = count
	}
	if !partial || !strings.Contains(PlainText(&with[21]), spec.Tagline) {
		t.Fatal("default must sweep across header")
	}
	for _, mode := range []TaglineReveal{TaglineBefore, TaglineWithReveal, TaglineAfter} {
		spec.TaglineReveal = mode
		frames, _, err := Opening(spec)
		if err != nil {
			t.Fatal(err)
		}
		for i := range 22 {
			visible := strings.Contains(PlainText(&frames[i]), spec.Tagline)
			if mode == TaglineBefore && !visible {
				t.Fatalf("before header missing in frame %d", i)
			}
			if mode == TaglineAfter && visible {
				t.Fatalf("after header appears early in frame %d", i)
			}
		}
		if mode == TaglineWithReveal && !reflect.DeepEqual(frames, with) {
			t.Fatal("zero value differs from with reveal")
		}
		if !strings.Contains(PlainText(&frames[22]), spec.Tagline) {
			t.Fatal("header never revealed")
		}
		if len(frames) != len(with) || Duration(frames) != Duration(with) {
			t.Fatal("option changes pacing")
		}
	}
	spec.TaglineReveal = "invalid"
	if _, _, err := Opening(spec); err == nil {
		t.Fatal("invalid header option accepted")
	}
}

func TestCompactFontIsolationAndCustomBoundary(t *testing.T) {
	a, b := CompactFont(), CompactFont()
	a['p'] = Glyph{"#"}
	if reflect.DeepEqual(a['p'], b['p']) {
		t.Fatal("compact fonts share state")
	}
	if _, err := LayoutWordmark("prefixlookup", WordmarkOptions{Font: DefaultFont()}); err == nil {
		t.Fatal("explicit font must not be silently replaced")
	}
	compact, err := LayoutWordmark("prefixlookup", WordmarkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := LayoutWordmark("prefixlookup", WordmarkOptions{Font: CompactFont()})
	if err != nil || !reflect.DeepEqual(compact, explicit) {
		t.Fatal("long names must use compact glyphs")
	}
	if _, err := LayoutWordmark(strings.Repeat("w", 30), WordmarkOptions{}); err == nil {
		t.Fatal("overlong names must still fail")
	}
}
