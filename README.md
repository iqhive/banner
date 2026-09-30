# banner

`github.com/iqhive/banner` is IQ Hive's shared Go package for animated README
terminal heroes. It owns the cell model, pixel font, hexadecimal opening reveal,
themes, SVG renderer, IQ Hive vector logo, terminal playback and command flags.
Apps keep their factual taglines, real demonstration code, scene composition and
reading durations in their own repositories. The package uses only Go's standard
library. This repository is public; generated SVGs are self-contained.

## Install

```sh
go get github.com/iqhive/banner@v1.0.0
```

The public module can be fetched through the normal Go module proxy and checksum
service. No GitHub authentication or private-module configuration is required.
Pin a released version in consumers; use a temporary workspace replacement when
working on banner and a consumer together.

## Supply content rather than an opening algorithm

```go
frames, card, err := banner.Opening(banner.Card{
    Word: "myapp",
    Tagline: "A factual project description.",
    TaglineReveal: banner.TaglineWithReveal, // also the zero-value default
    Points: []string{"One useful property", "Another useful property"},
    PointInterval: 200 * time.Millisecond,
    Install: "go get example.com/myapp",
    Wordmark: banner.WordmarkOptions{Seed: 0},
    ReadHold: 2500 * time.Millisecond,
})
if err != nil {
    return banner.Animation{}, err
}

// Append your project-specific demonstration frames here.
card.Hold = 4 * time.Second
frames = append(frames, card)
static := len(frames) - 1

return banner.Animation{
    Frames: frames,
    StaticFrame: static,
    Title: "myapp",
    Description: "An accessible explanation of the demonstrations.",
    Command: "go run ./examples/hero",
}, nil
```

`Opening` supplies the whole left-to-right sweep: seeded decorative noise,
per-cell jitter, progressive letter settlement, removal of unneeded noise,
letter colors and tinted cells, followed by the tagline, supporting line and
install command. Use `Card.Points` instead of `Supporting` to reveal each
supporting point independently. `PointInterval` defaults to 200 milliseconds;
positions stay anchored to the fully centered line so previous points do not
jump as later points appear. The separator appears with the following point.
Supplying both fields, empty points or text wider than the grid is rejected.
`Card.TaglineReveal` accepts three modes:

| Option | Behavior |
| --- | --- |
| `banner.TaglineBefore` (`"before"`) | Show the complete header as the opening starts. |
| `banner.TaglineWithReveal` (`"with reveal"`, default) | Reveal header columns from left to right at the wordmark sweep's speed, without decorative jitter. |
| `banner.TaglineAfter` (`"after"`) | Show the complete header after the wordmark settles, as before. |

The empty value also selects `with reveal`. These modes change visibility
without adding frames or changing any holds.
Its returned card has no hold, so the app chooses how long to
show it again. Keep quick reveal steps and increase settled frame holds when
readers need more time. Decorative digits use seeded SHA-256 fixtures and make
no claim about an app's behavior. Compute demo outputs with the app itself.

A command's entire main function can be:

```go
func main() {
    os.Exit(banner.RunCLI(os.Args[1:], os.Stdout, os.Stderr, build))
}
```

`build(seed uint64) (banner.Animation, error)` provides the animation above.
For a complete, runnable example see [examples/demo/main.go](examples/demo/main.go).

## Draw and time scenes

A `Frame` contains `Cells [20][80]Cell`, a positive `Hold`, and an optional
`HiveLogo` flag. `Text(x, y, text, style)` clips at the grid edge;
`Center(y, text, style)` centers a line, and `Clear(y0, y1)` clears a bounded
row range. Coordinates are zero based. Use ASCII or single-column glyphs:
combining characters, emoji and wide East Asian characters are not supported
by this fixed-column model. Copy a frame before appending it so later edits do
not change previously appended frames.

Semantic styles are `Plain`, `Dim`, `Muted`, `Bright`, `Literal`, `Changed`,
`On`, `Off`, `Bar`, `Prompt` and `Caret`. A wordmark's letter styles start at
`Mark`. The `Full` and `Shade` glyphs render as vector rectangles in SVG and
block characters in a terminal. `Paint.Tile` controls separate tiles versus a
continuous bar. `TypeLine` supplies reusable typed-input steps and the cursor; pass the current
frame, line position, text and optional `TypingOptions`. It returns intermediate
frames and a completed frame whose reading hold the app chooses. For manual
cursors use a nonblank cell such as `▏` in `Caret` style. Use `HiveLogo: true` on a closing frame to add the embedded,
sanitized IQ Hive vector artwork; set ordinary cell captions for terminal
viewers. The logo is an optional element on any frame, not a special slide type.

`Animation.StaticFrame` explicitly identifies the useful complete title card.
It need not be the last frame: a logo may follow it. SVG reduced-motion and
unsupported-animation views use this index, as does redirected CLI output.
`Duration(frames)` gives the complete loop duration. `Validate` rejects empty
animations, invalid static indices, nonpositive or overflowing durations,
invalid styles, unsafe fonts and invalid text control characters.

## Change themes

The zero `Animation.Theme` uses `DefaultTheme()`, which returns a
dark terminal with the IQ Hive logo's yellow, orange and red wordmark.
The opening and returning wordmark keep these warm colors; demonstration
styles, literals, diagrams, prompt and window controls use the bluegreen palette.
`BlueGreenTheme()` (CLI name `bluegreen`) preserves the original teal, cyan,
blue and violet appearance. `IQCapsTheme()` (CLI name `iqcaps`) uses uppercase
wordmark glyphs: an initial `IQ` receives a horizontal yellow/orange/red
gradient from the logo, while subsequent letters retain the bluegreen gradient.
The uppercase `I` is a straight, serif-free stroke like the IQ Hive logo.
All demo styles, window colors and controls remain exactly bluegreen. Names
without an `IQ` prefix are uppercase and entirely bluegreen. `LightTheme()` is a working daytime alternative.
Every factory returns independent slices. Theme changes apply to both SVG and terminal playback:

```go
theme := banner.DefaultTheme()
theme.Name = "my-project"
theme.Body = 0x111827
theme.TitleBar = 0x1f2937
theme.Border = 0x374151
theme.Styles[banner.Changed].Foreground = 0xfb923c
theme.WordmarkStops = []uint32{0x2dd4bf, 0x38bdf8, 0xa78bfa}
animation.Theme = theme
```

Theme fields cover the window, controls, title text, local font stack, every
semantic style, wordmark gradient stops and background tint. RGB colors use
24 bits. `Paint.Background == 0` means transparent; `Tile == 0` draws one bar.
The IQ Hive logo retains its brand colors and a dark vector backing for
contrast in light themes. SVG fonts remain local; CSS imports, URLs and style
injection are rejected. Geometry and glyph pixel scale stay consistent across
themes. Theme values are data, not global mutable settings. The default accent stops
are exact fills from `hive.svg`: `#FFF343`, `#FFB60B`, `#FF970D`, `#FF6F0D`
and `#FF120B`. Demo styles are inherited unchanged from `BlueGreenTheme`.
Intermediate gradient colors are interpolated when the word has another letter
count. Keep the palette's source test in sync if the official artwork changes.

`Theme.UppercaseWordmark` selects uppercase glyphs in the shared opening.
`Theme.SansSerifI` selects a vertical stroke for uppercase `I` (`iqcaps` enables
it). Set `WordmarkOptions.SansSerifI` when inspecting layouts directly; it also
works with compact and custom fonts, without mutating the supplied font map.
`WordmarkPrefix` and `WordmarkPrefixStops` optionally give an initial prefix its
own horizontal gradient; provide both together. Remaining letters use
`WordmarkStops`. Prefix gradients use sixteen color slots across their lit
cells, so a short prefix still passes through the intermediate orange colors.

Set the same theme on `Card.Theme` and `Animation.Theme` when inspecting raw
frames in your builder. Renderers and the CLI also resolve shared opening
metadata automatically, so `-theme iqcaps` works without changing application
sweep code. `Animation.WithTheme(theme)` returns a copy with resolved opening
frames, preserving caller cell overlays, frame counts and holds. Commands,
headers and scene text keep their original case. Opening source metadata is
immutable and carried by frame copies; use cell/hold comparisons or
`reflect.DeepEqual` rather than pointer-sensitive `Frame == Frame` comparisons.
Color-only theme changes do not rebuild glyphs. Manually drawn wordmarks do not
carry this metadata; use `Opening` for automatic typography changes.

## Wordmark spacing and fonts

`LayoutWordmark` measures the widest row of every glyph. It reserves two
terminal columns between letters by default. If necessary to fit a longer name,
it reduces that gap to one, and removes the gap only if the name cannot fit
otherwise. It returns an error if even the glyphs without gaps exceed the
terminal width; it never stretches, squashes or clips letters.

Both fonts support uppercase and lowercase Latin letters, hyphens and underscores.
Use `WordmarkOptions.Font` with `map[rune]banner.Glyph` for another alphabet or
custom letters. Each `Glyph` has nine rows of ASCII `#`, `.` or space, and each
lit pixel occupies two columns. Unequal row widths are allowed and tested.
`WordmarkOptions.Gap` can increase preferred spacing; zero means the default,
and negative values are rejected. Inspect the returned `Left`, `Width`, `Gap`
and `Cells` when debugging a name's layout.

## Generate and play

```sh
make demo-svg                  # generate examples/demo/hero.svg
make demo                      # terminal playback; plain card when redirected
go run ./examples/demo -svg /tmp/banner-light.svg -theme light
go run ./examples/demo -svg /tmp/banner-bluegreen.svg -theme bluegreen
go run ./examples/demo -svg /tmp/banner-iqcaps.svg -theme iqcaps
go run ./examples/demo -loop -speed 0.5 -seed 7
make verify
```

Shared CLI flags are `-svg`, `-loop`, `-speed`, `-seed` and `-theme`. A theme
flag overrides the app's selected theme; built-in names are `default`, `bluegreen`,
`iqcaps` and `light`. Omitting it preserves the app's theme. Speed changes duration, not
frame count or content. Terminal playback honors `NO_COLOR` including an empty
value, `COLORTERM=truecolor` or `24bit`, and `TERM=dumb`. It restores colors and
the cursor on normal completion, interrupts and write errors. Exit codes are
zero for success, one for build or I/O errors, two for usage and 130 for an
interrupt. Signal handling belongs to `RunCLI`; library users pass an interrupt
channel to `Play` themselves.

## Source maintenance map

| Change | File |
| --- | --- |
| Add cell operations or semantic styles | `frame.go` |
| Change window colors, gradients, tiles or built-in themes | `theme.go` |
| Add or refine pixel glyphs | `font.go` |
| Resolve opening typography after a theme override | `opening_theme.go` |
| Change spacing, seeded noise, sweep or shared opening timing | `wordmark.go` |
| Change typed-input steps or visible cursor | `typing.go` |
| Change input validation or animation metadata | `animation.go` |
| Change SVG geometry, deduplication, CSS timelines or static fallback | `svg.go` |
| Change terminal colors, redraw or cursor restoration | `terminal.go` |
| Change command flags or output selection | `cli.go` |
| Refresh the official IQ Hive vector artwork | `hive.svg` |

The SVG renderer first coalesces rows into styled text or block segments,
deduplicates repeated segments across frames, merges adjacent visibility spans,
and groups elements sharing exactly the same timeline. It writes groups in
first-observed order for reproducibility and uses `step-end` CSS keyframes.
Explicit `textLength` keeps columns aligned across local fonts. A group's
static visibility comes from `StaticFrame`; reduced-motion CSS overrides
animation names with `!important`. Keep that override when changing CSS order.
The SVG contains no scripts, `foreignObject`, raster frames or external assets.

The default SVG geometry is 836 by 498 pixels: 80 columns by 20 rows, cells
10 by 22, horizontal padding 18, title bar 32, top padding 12, bottom padding
14, and a corner radius of 10. These are rendering dimensions, not benchmarks.

`banner_test.go` covers XML, reproducibility, flag validation, color handling and
cursor restoration. `theme_wordmark_test.go` covers theme isolation, alternate
colors, variable-width glyphs, spacing fallback, invalid input and writer
failures. Run `make verify` before committing. For appearance changes also
inspect direct SVG and an `<img>` in Chromium, all scenes and the loop edge,
a narrow display, and native reduced motion. Browser emulation can fail to
propagate reduced motion into SVG image resources; test with Chromium's
`--force-prefers-reduced-motion` flag as well.

## Develop with a consuming repository

Consumers should pin a released version. For local edits without publishing,
add a temporary workspace replacement in that consumer:

```sh
# From a sibling checkout such as iqkvs.
go work edit -replace github.com/iqhive/banner=../banner
make hero-svg hero-check
# Remove the local override before committing or testing the released module.
go work edit -dropreplace github.com/iqhive/banner
```

Do not commit a filesystem replacement into a consuming `go.mod`. Once shared
checks and a consuming hero pass, commit and publish banner, tag its version,
update the consumer's requirement and checksum, and regenerate its SVG. Changes
to shared code may alter every consuming hero; each consumer owns its generated
artifact and drift test. The original iqhash renderer and playback were adapted
under the MIT license retained in [LICENSE](LICENSE).

### Long project names

The default layout uses `DefaultFont`. When those glyphs cannot fit the name
with one column between letters, it selects `CompactFont`, which redraws
narrower glyphs at the same pixel size. For example, `prefixlookup` fits without
clipping or touching letters. Custom `WordmarkOptions.Font` maps are respected
and never replaced. Names wider than the terminal even in the compact font
return an error. Both font factories return independent maps for customization.

## Repository history and releases

The main branch contains a single `Initial commit` with the finished shared
generator. `v1.0.0` is the first supported release and the only repository tag;
consumers pin `github.com/iqhive/banner v1.0.0` in their tooling modules.
