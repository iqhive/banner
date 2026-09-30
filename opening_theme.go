package banner

// openingSource retains the original cells before caller overlays. Theme
// resolution rebuilds only shared opening frames, preserving user edits and
// holds. It is immutable and shared by frame copies; scenes remain app-owned.
type openingSource struct {
	card   Card
	frames []Frame
}

// WithTheme returns independent frames using theme, including theme-specific
// uppercase glyphs and prefix gradients. Shared opening metadata makes CLI
// overrides work without consumer sweep code. Caller overlays and timings are
// preserved; manually drawn wordmarks should use the theme's style convention.
func (a Animation) WithTheme(theme Theme) (Animation, error) {
	a.Theme = theme
	if err := a.Validate(); err != nil {
		return Animation{}, err
	}
	a.Frames = append([]Frame(nil), a.Frames...)
	cache := make(map[*openingSource][]Frame)
	for i, frame := range a.Frames {
		source := frame.opening
		if source == nil {
			continue
		}
		old := source.card.Theme
		if old.UppercaseWordmark == theme.UppercaseWordmark && old.SansSerifI == theme.SansSerifI && old.WordmarkPrefix == theme.WordmarkPrefix && (len(old.WordmarkPrefixStops) > 0) == (len(theme.WordmarkPrefixStops) > 0) {
			continue
		}
		rebuilt, ok := cache[source]
		if !ok {
			spec := source.card
			spec.Theme = theme
			var err error
			rebuilt, _, err = Opening(spec)
			if err != nil {
				return Animation{}, err
			}
			cache[source] = rebuilt
		}
		next := rebuilt[frame.openingIndex]
		baseline := source.frames[frame.openingIndex]
		for y, row := range frame.Cells {
			for x, cell := range row {
				if cell != baseline.Cells[y][x] {
					next.Cells[y][x] = cell
				}
			}
		}
		next.Hold = frame.Hold
		next.HiveLogo = frame.HiveLogo
		a.Frames[i] = next
	}
	return a, nil
}
