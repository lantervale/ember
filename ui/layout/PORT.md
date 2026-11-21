# Porting Taffy

Taffy c6421c1c, `src/` (about 30k lines of Rust). `go test -run TestTaffy -v` reports each suite's pass count; `TAFFY=<name>`, `TAFFY_VERBOSE=1` and `TAFFY_DUMP=1` narrow and explain it.

| Taffy source | ui | Status |
|---|---|---|
| `geometry.rs`, `util/{math,resolve}.rs`, `style/available_space.rs` | `geometry.go` (NaN is none), `style.go` | done |
| `tree/cache.rs`, `compute/{mod,leaf}.rs` | `cache.go`, `compute.go`, `leaf.go` | done; leaf 30/30 |
| `compute/block.rs`, `compute/float.rs` | `block.go`, `floats.go` | done; block 490/490, float 80/80 |
| `compute/flexbox.rs` | `flex.go` | done; flex 1232/1232 |
| `compute/grid/*` | `grid*.go` | done; grid 1082/1098 |
| `compute/oof.rs`, `compute/common/*` | `oof.go`, `common.go` | done |

Counts are of the supported fixtures. Left out until style can say them: alignment safety (`safe`/`unsafe`), `self-start`/`self-end`, `stretch` sizes. blockflex is 24/26: `flex-basis: 0%` needs a zero percentage kept apart from `0px`.

Differences from Taffy, each one where Chrome disagrees with it:

- Text is a block container's inline content, not a measured leaf, so a text element in a flex or grid container has an anonymous block item. A single-line flex row measured in definite space shrinks to fit (`minContentMainSize`), as CSS's fit-content does.
- A min or max size transferred through `aspect-ratio` never goes into an axis whose size is already known.
- `Layout` keeps fractional positions, matching Chrome to 0.03 pt; rounding is only for the fixtures that ask for it.

Beyond Taffy (`inline.go`), with Chrome cases in `chrome_test.go`:

- line boxes, wrapping at spaces and U+200B, `white-space`, `text-overflow: ellipsis`, `line-height`, `text-align` (start, end, left, right, center), spans, and atomic inlines (inline-block, inline-flex, inline-grid) on the baseline.

Still to do: `position: sticky`, `z-index` ordering data, clamping `Box.Scroll`, `order`, `overflow-wrap`, padding, borders and margins of inline elements, `text-align: justify`, and relayout of only what's dirty (each `Layout` builds the tree anew).

Floats differ from Taffy in three places, each checked against Chrome (`float_chrome_test.go`):

- A formatting context's root grows to its floats' bottom plus its own bottom padding and border.
- A child's auto margins count in the insets its content sees, so lines in a centred column find the floats where they are.
- Text goes round floats: a leaf laid out among floats breaks each line in the room `lineSlot` leaves at its height (`flowLines`), and its layout is never cached, being the context's as much as its own. A float among inline content, or inside an inline element, is built as a block-level child of the block container, before the anonymous block of the text it was written in.

Multi-column layout is not Taffy's (`columns.go`, decision 0088), checked against Chrome (`columns_chrome_test.go`):

- `dispatch` hands a block container with a `column-count` or a `column-width` to `columns`, which lays the node out once with a style of its own (`flowStyle`: no sizes, no columns) at the used column width, so block and leaf layout and their caches are untouched.
- `flowUnits` lists what can't be cut, in flow order: each line, each whole box (`breaksInside` says which boxes are cut), and empty room, which cuts anywhere. `fillColumns` fills columns of a height from that list with Chrome's rules: margins dropped at a cut, forced breaks, and the best earlier break when a cut would leave too few lines (`orphans`, `widows`) or break an `avoid`. `balancedHeight` starts from the flow's height over the count, or the tallest unit, and grows by the least that lets one more unit in until the columns are enough.
- Placement is at write time: `keepColumns` gives each node of the flow its `flowFrag`, `keepFlow` moves it to its column, `writeColumns` sets `Broken`, `Fragments` and the rectangle round them on a cut box, and `writeLines` shifts each line to its column.
- Positioned boxes are laid out by the container as elsewhere; one with no inset starts in the column of its line, or of the block before it.

Not ported: `flex-wrap: balance`, `contain`, right-to-left, vertical writing, Taffy's `TaffyTree` storage and its `calc()` pointer (calc arrives resolved from css).
