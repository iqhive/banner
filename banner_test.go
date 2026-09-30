package banner

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(seed uint64) (Animation, error) {
	frames, card, err := Opening(Card{Word: "banner", Tagline: "Reusable terminal banners.", Supporting: "Deterministic frames and themes.", Install: "go get github.com/iqhive/banner", Wordmark: WordmarkOptions{Seed: seed}})
	if err != nil {
		return Animation{}, err
	}
	card.Hold = time.Second
	frames = append(frames, card)
	return Animation{Frames: frames, StaticFrame: len(frames) - 1, Title: "banner", Description: "An opening and static project card.", Command: "go run ./examples/demo"}, nil
}
func mustAnimation(t *testing.T) Animation {
	t.Helper()
	a, err := fixture(0)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestBadFlags(t *testing.T) {
	for _, args := range [][]string{{"-speed", "0"}, {"-speed", "-1"}, {"-nope"}, {"extra"}, {"-speed", "NaN"}, {"-speed", "+Inf"}, {"-speed", "1e300"}, {"-speed", "1e-300"}, {"-theme", "missing"}} {
		if code := RunCLI(args, io.Discard, io.Discard, fixture); code != 2 {
			t.Errorf("run(%q) = %d, want 2", args, code)
		}
	}
}

func TestPlay(t *testing.T) {
	a := mustAnimation(t)
	frames := a.Frames[:5]
	a.Frames = frames
	a.StaticFrame = 4
	for i := range frames {
		frames[i].Hold = time.Microsecond
	}
	var b bytes.Buffer
	if err := Play(&b, a, PlaybackOptions{Colors: Color256}); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if !strings.HasPrefix(s, hideCursor) || !strings.HasSuffix(s, reset+showCursor+"\n") {
		t.Errorf("output does not hide and then restore the cursor: %q ... %q", s[:10], s[len(s)-10:])
	}
	if n, want := strings.Count(s, fmt.Sprintf("\r\x1b[%dA", Rows-1)), len(frames)-1; n != want {
		t.Errorf("cursor moves up %d times, want %d", n, want)
	}

	// An interrupt stops it after the frame on screen, and it still
	// restores the terminal.
	for i := range frames {
		frames[i].Hold = time.Hour
	}
	sig := make(chan os.Signal, 1)
	sig <- os.Interrupt
	b.Reset()
	if err := Play(&b, a, PlaybackOptions{Colors: TrueColor, Loop: true, Interrupt: sig}); !errors.Is(err, ErrInterrupted) {
		t.Fatalf("play = %v, want ErrInterrupted", err)
	}
	if s := b.String(); strings.Contains(s, fmt.Sprintf("\x1b[%dA", Rows-1)) || !strings.HasSuffix(s, reset+showCursor+"\n") {
		t.Errorf("interrupted output drew more than one frame or did not restore the terminal")
	}
}

func TestNoColour(t *testing.T) {
	a := mustAnimation(t)
	frames := a.Frames
	for i := range frames {
		var b bytes.Buffer
		writeFrame(&b, &frames[i], NoColor, a.theme().palette(a.letters()))
		if strings.Contains(b.String(), "\x1b[0;") {
			t.Fatalf("frame %d selects a colour with colour off", i)
		}
	}
}

func TestNearest256(t *testing.T) {
	for rgb, want := range map[uint32]int{0x000000: 16, 0xffffff: 231, 0xff0000: 196, 0x5f87af: 67, 0x808080: 244} {
		if got := nearest256(rgb); got != want {
			t.Errorf("nearest256(%06x) = %d, want %d", rgb, got, want)
		}
	}
}

func TestSVG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hero.svg")
	var errs bytes.Buffer
	if code := RunCLI([]string{"-svg", path}, io.Discard, &errs, fixture); code != 0 {
		t.Fatalf("run = %d, stderr %q", code, errs.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 500<<10 {
		t.Errorf("SVG is %d bytes, want under 500 KiB", len(data))
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	var root string
	styleCount := 0
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("SVG is not well-formed XML: %v", err)
		}
		if e, ok := tok.(xml.StartElement); ok {
			switch e.Name.Local {
			case "svg", "title", "desc", "g", "rect", "circle", "path", "text":
			case "style":
				styleCount++
			default:
				t.Errorf("unexpected SVG element <%s>; SVG names are case sensitive", e.Name.Local)
			}
			if e.Name.Space != "http://www.w3.org/2000/svg" {
				t.Errorf("wrong SVG namespace on %s", e.Name.Local)
			}
			if root == "" {
				root = e.Name.Local
			}
			if e.Name.Local == "script" || e.Name.Local == "image" || e.Name.Local == "foreignObject" {
				t.Errorf("SVG has a <%s> element", e.Name.Local)
			}
			for _, a := range e.Attr {
				if a.Name.Local == "href" {
					t.Errorf("SVG has an href on <%s>", e.Name.Local)
				}
			}
		}
	}
	if styleCount != 1 {
		t.Errorf("got %d real <style> elements; animation CSS must be recognized by SVG", styleCount)
	}
	if root != "svg" {
		t.Errorf("root element is <%s>, want <svg>", root)
	}
	if bytes.Contains(data, []byte("url(")) || bytes.Contains(data, []byte("@import")) {
		t.Error("SVG refers to an external resource")
	}

	var again bytes.Buffer
	if err := WriteSVG(&again, mustAnimation(t)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again.Bytes()) {
		t.Error("two renderings of the same frames differ")
	}
}
