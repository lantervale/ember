# The pictures svg is tested on

`go test -tags chrome ./svg` draws each of these and compares it with headless Chrome's picture of the same file in
an `img`, at 256×256 on white and on grey, 48×48 and 200×120; `chrome.txt` holds the most each may differ by
(`EMBER_SVG_RECORD=1` rewrites it from what is measured). They are also the fuzz test's seeds.

## ember's own work, CC0

Written for ember in this repository and given to the public domain under
[CC0 1.0](https://creativecommons.org/publicdomain/zero/1.0/): every `own-*.svg`, and the pictures their `data:`
addresses hold. `own-text.svg`, `own-texts.svg` and `own-text-spacing.svg` are set in Go's faces where Chrome sets them in its own, so their
thresholds are what the other letters cost.

## From others

Icons as their projects publish them, unchanged:

| File | From | Licence |
|---|---|---|
| `bootstrap-alarm.svg`, `bootstrap-patch-check.svg` | [Bootstrap Icons](https://github.com/twbs/icons) 1.11.3, `alarm` and `patch-check` | MIT |
| `feather-activity.svg`, `feather-mail.svg`, `feather-settings.svg` | [Feather](https://github.com/feathericons/feather) 4.29.2 | MIT |
| `heroicons-bell.svg`, `heroicons-solid-cog.svg` | [Heroicons](https://github.com/tailwindlabs/heroicons) 2.1.5, `24/outline/bell` and `24/solid/cog-6-tooth` | MIT |
| `material-home.svg` | [Material Design Icons](https://github.com/marella/material-design-icons) 0.14.13, `filled/home` | Apache-2.0 |
| `octicon-mark-github.svg` | [Octicons](https://github.com/primer/octicons) 19.11.0, `mark-github-24` | MIT |
| `si-github.svg`, `si-go.svg` | [Simple Icons](https://github.com/simple-icons/simple-icons) 13.0.0 | CC0 1.0 (the marks they draw are their owners' trademarks) |
| `tabler-brand-github.svg` | [Tabler Icons](https://github.com/tabler/tabler-icons) 3.19.0, `outline/brand-github` | MIT |
| `twemoji-smile.svg` | [Twemoji](https://github.com/twitter/twemoji) 14.0.2, `1f600` | CC-BY 4.0 |
