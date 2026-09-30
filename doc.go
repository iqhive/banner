// Package banner creates deterministic terminal animations and self-contained
// animated SVGs from the same styled character frames.
//
// A project supplies its facts and scenes as an Animation. Opening implements
// the complete shared hexadecimal-noise reveal; callers supply Card text and
// timing, not sweep code. DefaultTheme uses warm wordmarks and bluegreen scenes; BlueGreenTheme preserves
// the original palette. IQCapsTheme combines uppercase glyphs and a warm IQ
// prefix with bluegreen remaining letters and scenes. LightTheme and custom themes reuse the same geometry.
//
// WriteSVG deduplicates row segments and groups those with identical visibility
// timelines. Discrete step-end CSS keyframes work inside README image elements
// without JavaScript, external resources or raster frames. StaticFrame selects
// the fallback for reduced motion and unsupported animation.
//
// RunCLI adds generation and terminal playback flags to a project's builder.
// Play uses the same frames and theme, restores the cursor on return, and can
// stop through a signal channel. PlainText is suitable for redirected output.
//
// This package has no non-standard-library dependencies. See README.md for the
// complete authoring and theme workflow and the source-file maintenance map.
package banner
