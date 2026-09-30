package banner

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// RunCLI provides the shared -svg, -loop, -speed, -seed and -theme flags.
// The builder owns project facts, scenes and the static frame index. Exit codes
// are 0 for success, 1 for rendering or I/O failure, 2 for usage, and 130 for interruption.
func RunCLI(args []string, stdout, stderr io.Writer, build func(uint64) (Animation, error)) int {
	// Nothing useful can be done when stderr itself fails.
	logf := func(format string, a ...any) { _, _ = fmt.Fprintf(stderr, format, a...) }

	fs := flag.NewFlagSet("hero", flag.ContinueOnError)
	fs.SetOutput(stderr)
	loop := fs.Bool("loop", false, "play until interrupted")
	speed := fs.Float64("speed", 1, "playback speed; 2 plays twice as fast")
	seed := fs.Uint64("seed", 0, "seed for decorative hexadecimal noise")
	themeName := fs.String("theme", "", "theme override: default, bluegreen, iqcaps or light")
	svg := fs.String("svg", "", "write the animation as an animated SVG to `file` instead of playing it")
	fs.Usage = func() {
		logf("usage: hero [-loop] [-speed x] [-seed n] [-theme name] [-svg file]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *speed <= 0 || math.IsNaN(*speed) || math.IsInf(*speed, 0) || fs.NArg() > 0 {
		fs.Usage()
		return 2
	}

	animation, err := build(*seed)
	if err != nil {
		logf("hero: fixture failed: %v\n", err)
		return 1
	}
	if *themeName != "" {
		theme, err := ThemeByName(*themeName)
		if err != nil {
			logf("hero: %v\n", err)
			return 2
		}
		animation.Theme = theme
	}
	animation, err = animation.WithTheme(animation.theme())
	if err != nil {
		logf("hero: %v\n", err)
		return 1
	}
	for i := range animation.Frames {
		hold := float64(animation.Frames[i].Hold) / *speed
		if hold < 1 || hold >= float64(math.MaxInt64) {
			logf("hero: speed produces an invalid frame duration\n")
			return 2
		}
		animation.Frames[i].Hold = time.Duration(hold)
	}
	if err := animation.Validate(); err != nil {
		logf("hero: %v\n", err)
		return 1
	}

	if *svg != "" {
		var b bytes.Buffer
		if err := WriteSVG(&b, animation); err != nil {
			logf("hero: %v\n", err)
			return 1
		}
		if err := os.WriteFile(*svg, b.Bytes(), 0o644); err != nil { //nolint:gosec // generated README artwork is public and must be readable
			logf("hero: %v\n", err)
			return 1
		}
		logf("hero: wrote %s: %d frames, %.1f s, %d bytes\n", *svg, len(animation.Frames), Duration(animation.Frames).Seconds(), b.Len())
		return 0
	}

	if !IsTerminal(stdout) {
		if _, err := io.WriteString(stdout, PlainText(&animation.Frames[animation.StaticFrame])); err != nil {
			logf("hero: %v\n", err)
			return 1
		}
		return 0
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)
	if err := Play(stdout, animation, PlaybackOptions{Colors: TerminalColors(), Loop: *loop, Interrupt: sig}); err != nil {
		if errors.Is(err, ErrInterrupted) {
			return 130
		}
		logf("hero: %v\n", err)
		return 1
	}
	return 0
}
