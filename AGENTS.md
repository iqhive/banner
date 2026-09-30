# Working on banner

Read README.md and the public package documentation before changing the API.
Keep the package independent of any consuming app and free of non-standard
library dependencies. Project facts and demonstration scenes belong to their
consumers; the entire opening sweep belongs here.

Run `make verify` before every commit. Preserve deterministic SVG output,
self-contained vectors, reduced-motion fallback, terminal cursor restoration,
and the default theme. Add tests for behavioral changes, especially glyph
spacing and theme isolation. For visual changes inspect direct SVG, image
embedding, narrow width, reduced motion and the loop transition in a browser.
Regenerate `examples/demo/hero.svg` with `make demo-svg` when shared output changes.

Use `type(scope): imperative summary` commit subjects. Change repository
visibility only when the maintainer requests it. Do not publish or push without
authorization; the initial public repository creation and publication requested
in the extracting conversation are authorized.
