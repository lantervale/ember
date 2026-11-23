# Decisions

The choices that shape ember, one short file each: what was decided, what else was possible, and what follows.

| # | Decision | Area |
|---|---|---|
| 0001 | [The server is authoritative](0001-the-server-is-authoritative.md) | net |
| 0002 | [Fixed-step simulation with prediction](0002-fixed-step-simulation-with-prediction.md) | net |
| 0003 | [Tests need no GPU and no real time](0003-tests-need-no-gpu-and-no-real-time.md) | core |
| 0004 | [Servers stay pure Go](0004-servers-stay-pure-go.md) | core |
| 0005 | [Transports are pluggable, QUIC is opt-in](0005-transports-are-pluggable.md) | net |
| 0006 | [The World is the centre](0006-the-world-is-the-centre.md) | core |
| 0007 | [Public APIs take functional options](0007-public-apis-take-functional-options.md) | api |
| 0008 | [Nothing is styled by default](0008-nothing-is-styled-by-default.md) | rendering |
| 0009 | [Third-party formats are read in pure Go](0009-third-party-formats-are-read-in-pure-go.md) | assets |
| 0010 | [Studio is the official tool](0010-studio-is-the-official-tool.md) | tools |
| 0011 | [Scenes are glb files](0011-scenes-are-glb-files.md) | scenes |
| 0012 | [Nodes reuse the engine](0012-nodes-reuse-the-engine.md) | scenes |
| 0013 | [Every RPC uses Connect](0013-every-rpc-uses-connect.md) | tools |
| 0014 | [The guide teaches by example](0014-the-guide-teaches-by-example.md) | docs |
| 0015 | [The engine exposes no resources](0015-the-engine-exposes-no-resources.md) | api |
| 0016 | [The editor and Studio are removed for now](0016-the-editor-and-studio-are-removed-for-now.md) | tools |
| 0017 | [UI is built on 2D](0017-ui-is-built-on-2d.md) | rendering |
| 0018 | [The 2D API follows a few rules](0018-the-2d-api-follows-a-few-rules.md) | api |
| 0019 | [Options read back and pair with setters](0019-options-read-back-and-pair-with-setters.md) | api |
| 0020 | [Sound is pure Go behind one device](0020-sound-is-pure-go-behind-one-device.md) | core |
| 0021 | [A 2D view resizes by its camera](0021-a-2d-view-resizes-by-the-camera.md) | rendering |
| 0022 | [A camera names the layers it draws](0022-a-camera-names-the-layers-it-draws.md) | rendering |
| 0023 | [Events wait for a fixed step](0023-events-wait-for-a-fixed-step.md) | core |
| 0024 | [A horde follows one flow field](0024-a-horde-follows-one-flow-field.md) | core |
| 0025 | [UI scales by subtree](0025-ui-scales-by-subtree.md) | rendering |
| 0026 | [Pads rumble through the platform's haptics](0026-pads-rumble-through-platform-haptics.md) | core |
| 0027 | [Focus captures only what ui reads](0027-focus-captures-only-what-ui-reads.md) | api |
| 0028 | [A soft crowd spreads by density](0028-a-soft-crowd-spreads-by-density.md) | core |
| 0029 | [A flat tile draws its ground](0029-a-flat-tile-draws-its-ground.md) | rendering |
| 0030 | [Tiled flips are honoured](0030-tiled-flips-are-honoured.md) | assets |
| 0031 | [A batch draws several textures](0031-a-batch-draws-several-textures.md) | rendering |
| 0032 | [Lights see exact visibility polygons](0032-lights-see-exact-visibility-polygons.md) | core |
| 0033 | [A pictured layer can draw at a lower resolution](0033-a-pictured-layer-can-draw-at-a-lower-resolution.md) | rendering |
| 0034 | [Every tile draws whole](0034-every-tile-draws-whole.md) | rendering |
| 0035 | [A tile collides by its shapes](0035-a-tile-collides-by-its-shapes.md) | core |
| 0036 | [A solid tile blocks its cell for paths](0036-a-solid-tile-blocks-its-cell-for-paths.md) | core |
| 0037 | [Lights read the light maps](0037-lights-read-the-light-maps.md) | core |
| 0038 | [A flow field may build in the background, one call late](0038-a-flow-field-may-build-in-the-background.md) | core |
| 0039 | [ui styling: boxes, classes and a cascade](0039-ui-styling-boxes-classes-and-a-cascade.md) | api |
| 0040 | [Routing follows where a body fits, not whole solid tiles](0040-routing-follows-where-a-body-fits.md) | core |
| 0041 | [One renderer on WebGPU, through ember's own binding](0041-one-renderer-on-webgpu.md) | rendering |
| 0042 | [2D lights and shadows on the GPU](0042-2d-lights-and-shadows-on-the-gpu.md) | rendering |
| 0043 | [Standard tile properties, checked by the loader](0043-standard-tile-properties.md) | assets |
| 0044 | [Names made consistent](0044-names-made-consistent.md) | api |
| 0045 | [ShadedBy draws a light picture per set of groups](0045-shaded-by-draws-a-light-picture-per-set-of-groups.md) | rendering |
| 0046 | [Silhouettes are masks subtracted per light](0046-silhouettes-are-masks-subtracted-per-light.md) | rendering |
| 0047 | [Lights past a camera's budget fade out by priority, then distance](0047-lights-past-a-camera-budget-fade-out-by-priority-then-distance.md) | rendering |
| 0048 | [A light Affects layers through a picture per set of lights left out](0048-a-light-affects-layers-through-a-picture-per-set-of-lights-left-out.md) | rendering |
| 0049 | [UI is modern HTML elements and a CSS subset, written from Go](0049-ui-is-modern-html-elements-and-a-css-subset-written-from-go.md) | api |
| 0050 | [The renderer encodes with the exact sRGB curve and blends encoded](0050-the-renderer-encodes-with-the-exact-srgb-curve-and-blends-encoded.md) | rendering |
| 0051 | [uiv2 replays its last frame when nothing changed](0051-uiv2-replays-its-last-frame-when-nothing-changed.md) | rendering |
| 0052 | [uiv2 paints live parts over a replayed frame](0052-uiv2-paints-live-parts-over-a-replayed-frame.md) | rendering |
| 0053 | [A modal owns its window's input; owners stack](0053-a-modal-owns-its-windows-input.md) | core |
| 0054 | [ember runs in the browser through WebAssembly and WebGPU](0054-ember-runs-in-the-browser-through-webassembly-and-webgpu.md) | rendering |
| 0055 | [A UI layer covers what it is drawn over, for the pointer](0055-a-ui-layer-covers-what-it-is-drawn-over.md) | core |
| 0056 | [uiv2 images filter in encoded sRGB, as CSS does](0056-uiv2-images-filter-in-encoded-srgb.md) | rendering |
| 0057 | [uiv2 elements take callbacks over the World events](0057-uiv2-elements-take-callbacks-over-the-world-events.md) | api |
| 0058 | [Rounded clips through a per-batch rounded rect](0058-rounded-clips-through-a-per-batch-rounded-rect.md) | rendering |
| 0059 | [ember/ui is uiv2: the old package removed, uiv2 renamed ui](0059-ember-ui-is-uiv2.md) | api |
| 0060 | [Back and key callbacks bubble from the focused element](0060-back-and-key-callbacks-bubble.md) | api |
| 0061 | [Picture cursors are Cursor values](0061-picture-cursors-are-cursor-values.md) | api |
| 0062 | [ui elements take a right click and drag-and-drop of their own](0062-ui-elements-take-a-right-click-and-drag-and-drop-of-their-own.md) | api |
| 0063 | [On the web, the page is the window's frame](0063-on-the-web-the-page-is-the-windows-frame.md) | platform |
| 0064 | [A server World is ember.New without presenting](0064-a-server-world-is-ember-new-without-presenting.md) | net |
| 0065 | [2D entities replicate on the ground plane](0065-2d-entities-replicate-on-the-ground-plane.md) | net |
| 0066 | [A Tiled map loads only the tilesets it uses](0066-a-tiled-map-loads-only-the-tilesets-it-uses.md) (superseded by 0067) | assets |
| 0067 | [A Tiled map reads only the tilesets it uses](0067-a-tiled-map-reads-only-the-tilesets-it-uses.md) | assets |
| 0068 | [A Tiled map hides nothing](0068-a-tiled-map-hides-nothing.md) | assets |
| 0069 | [Variants of a picture are made at load](0069-variants-of-a-picture-are-made-at-load.md) | rendering |
| 0070 | [Tiled classes are declared in Go](0070-tiled-classes-are-declared-in-go.md) | assets |
| 0071 | [Each system reads its own tile properties](0071-each-system-reads-its-own-tile-properties.md) | assets |
| 0072 | [A hidden capture draws only its frame](0072-a-hidden-capture-draws-only-its-frame.md) | rendering |
| 0073 | [A focused element under another's modal looks unfocused](0073-a-focused-element-under-anothers-modal-looks-unfocused.md) | core |
| 0074 | [A soft crowd never stacks on one point](0074-a-soft-crowd-never-stacks-on-one-point.md) | core |
| 0075 | [A wall seen from above is lit from every side](0075-a-wall-seen-from-above-is-lit-from-every-side.md) | rendering |
| 0076 | [A tile object casts as its tile](0076-a-tile-object-casts-as-its-tile.md) | rendering |
| 0077 | [What is lit around takes light past its corners](0077-what-is-lit-around-takes-light-past-its-corners.md) | rendering |
| 0078 | [A tile lit around is lit round its bodies too](0078-a-tile-lit-around-is-lit-round-its-bodies-too.md) | rendering |
| 0079 | [A soft crowd presses body against body](0079-a-soft-crowd-presses-body-against-body.md) | core |
| 0080 | [Floats are laid out, text going round them](0080-floats-are-laid-out-text-going-round-them.md) | api |
| 0081 | [System fonts are an option of a Document](0081-system-fonts-are-an-option-of-a-document.md) | api |
| 0082 | [ui has the text elements pages are written with](0082-ui-has-the-text-elements-pages-are-written-with.md) | api |
| 0083 | [Overlay scrollbars are an option of a Document](0083-overlay-scrollbars-are-an-option-of-a-document.md) | api |
| 0084 | [A scrolling viewport is an option of a Document](0084-a-scrolling-viewport-is-an-option-of-a-document.md) | api |
| 0085 | [Browser sheets are an option of a Document](0085-browser-sheets-are-an-option-of-a-document.md) | api |
| 0086 | [Media queries are answered for the Document's viewport](0086-media-queries-are-answered-for-the-documents-viewport.md) | api |
| 0087 | [SVG is read by ember's own reader, imported where it is wanted](0087-svg-is-read-by-embers-own-reader-imported-where-it-is-wanted.md) | assets |
| 0088 | [Columns cut one flow, and a cut box stays one box](0088-columns-cut-one-flow-and-a-cut-box-stays-one-box.md) | api |
| 0089 | [A generated box is a node of its element](0089-a-generated-box-is-a-node-of-its-element.md) | api |
| 0090 | [A screen's text takes no selection until its sheet says so](0090-a-screens-text-takes-no-selection-until-its-sheet-says-so.md) | api |
| 0091 | [A window keeps the system's frame under its own title bar](0091-a-window-keeps-the-systems-frame-under-its-own-title-bar.md) | api |
| 0092 | [Presentational attributes are hints under every rule](0092-presentational-attributes-are-hints-under-every-rule.md) | api |
| 0093 | [A mask image is the shape of the background colour](0093-a-mask-image-is-the-shape-of-the-background-colour.md) | rendering |
| 0094 | [Display table off a table is a block that shrinks and holds its floats](0094-display-table-off-a-table-is-a-block-that-shrinks-and-holds-its-floats.md) | rendering |
| 0095 | [A Document answers what a tool asks about an element](0095-a-document-answers-what-a-tool-asks-about-an-element.md) | api |
| 0096 | [WebP pictures are read with the pictures of the web](0096-webp-pictures-are-read-with-the-pictures-of-the-web.md) | assets |
| 0097 | [A Document's sheets are changed through handles, once a frame](0097-a-documents-sheets-are-changed-through-handles-once-a-frame.md) | api |
| 0098 | [An svg element is one picture holding its markup](0098-an-svg-element-is-one-picture-holding-its-markup.md) | api |
| 0099 | [A page's fonts are fetched, and WOFF2 is read in pure Go](0099-a-pages-fonts-are-fetched-and-woff2-is-read-in-pure-go.md) | assets |
| 0100 | [The GPU holds a picture while the picture lives](0100-the-gpu-holds-a-picture-while-the-picture-lives.md) | rendering |
| 0101 | [Paint and hit testing skip the boxes out of view](0101-paint-and-hit-testing-skip-the-boxes-out-of-view.md) | rendering |
| 0102 | [A window's close is a request its rule answers](0102-a-windows-close-is-a-request-its-rule-answers.md) | api |
| 0103 | [An iframe shows a Document the program gives it](0103-an-iframe-shows-a-document-the-program-gives-it.md) | api |
| 0104 | [vertical-align places boxes on their line as Chrome does](0104-vertical-align-places-boxes-on-their-line-as-chrome-does.md) | api |
| 0105 | [A table is what display says, with the boxes it misses](0105-a-table-is-what-display-says-with-the-boxes-it-misses.md) | rendering |
| 0106 | [css asks what a unit stands for, and leaves layout the bounds over a percentage](0106-css-asks-what-a-unit-stands-for-and-leaves-layout-the-bounds-over-a-percentage.md) | rendering |
| 0107 | [ICO and BMP pictures are read by ember's own reader](0107-ico-and-bmp-pictures-are-read-by-embers-own-reader.md) | assets |
| 0108 | [A program follows the system's appearance and languages](0108-a-program-follows-the-systems-appearance-and-languages.md) | api |
| 0109 | [A right drag selects text only where a Document asks](0109-a-right-drag-selects-text-only-where-a-document-asks.md) | api |
| 0110 | [A Document scales what the wheel scrolls](0110-a-document-scales-what-the-wheel-scrolls.md) | api |
| 0111 | [A program scrolls a Document by an element](0111-a-program-scrolls-a-document-by-an-element.md) | api |
| 0112 | [Imported sheets are fetched, and spliced where the import stands](0112-imported-sheets-are-fetched-and-spliced-where-the-import-stands.md) | api |
| 0113 | [:has() flags its anchors before the cascade](0113-has-flags-its-anchors-before-the-cascade.md) | api |
| 0114 | [Animations run on transitions' clock, over the cascade](0114-animations-run-on-transitions-clock-over-the-cascade.md) | rendering |
| 0115 | [An img chooses its picture as Chrome does](0115-an-img-chooses-its-picture-as-chrome-does.md) | api |
| 0117 | [A background keeps each longhand's list and paints its layers one by one](0117-a-background-keeps-each-longhands-list-and-paints-its-layers-one-by-one.md) | rendering |
| 0118 | [A filtered element is a picture drawn through WGSL effects](0118-a-filtered-element-is-a-picture-drawn-through-wgsl-effects.md) | rendering |
| 0119 | [A clip-path is a mask the GPU cuts by, and an outline the pointer is tested against](0119-a-clip-path-is-a-mask-the-gpu-cuts-by-and-an-outline-the-pointer-is-tested-against.md) | rendering |
| 0120 | [Container queries restyle after layout until their answers hold](0120-container-queries-restyle-after-layout-until-their-answers-hold.md) | api |
| 0121 | [A Document answers media features from the preferences it is given](0121-a-document-answers-media-features-from-the-preferences-it-is-given.md) | api |
| 0122 | [Each longhand declares how it animates, in the properties' table](0122-each-longhand-declares-how-it-animates-in-the-properties-table.md) | rendering |
| 0123 | [Math functions over percentages are a program layout runs](0123-math-functions-over-percentages-are-a-program-layout-runs.md) | api |
| 0124 | [A field's validity is a state ui keeps](0124-a-fields-validity-is-a-state-ui-keeps.md) | api |
| 0125 | [A subgrid's items size its grid's tracks, hoisted to its lines](0125-a-subgrids-items-size-its-grids-tracks-hoisted-to-its-lines.md) | rendering |
| 0126 | [An element is drawn in the scheme its color-scheme allows](0126-an-element-is-drawn-in-the-scheme-its-color-scheme-allows.md) | api |
| 0127 | [A first line is laid out in its own styles](0127-a-first-line-is-laid-out-in-its-own-styles.md) | api |
| 0128 | [Right to left is a mirror of the layout, and bidi orders each line](0128-right-to-left-is-a-mirror-of-the-layout-and-bidi-orders-each-line.md) | api |
| 0129 | [Spanners cut the flow in parts, each laid out alone](0129-spanners-cut-the-flow-in-parts-each-laid-out-alone.md) | api |
| 0130 | [A mixed element reads its group so far as a picture](0130-a-mixed-element-reads-its-group-so-far-as-a-picture.md) | rendering |
| 0131 | [Discrete values and composition move as Chrome's](0131-discrete-values-and-composition-move-as-chromes.md) | rendering |
| 0132 | [Border sides are mitered pieces smoothed by their own geometry](0132-border-sides-are-mitered-pieces-smoothed-by-their-own-geometry.md) | rendering |
| 0133 | [A masked element is a picture cut by the picture of its mask](0133-a-masked-element-is-a-picture-cut-by-the-picture-of-its-mask.md) | rendering |
| 0134 | [Counter styles are rules, the predefined ones a sheet of them](0134-counter-styles-are-rules-the-predefined-ones-a-sheet.md) | api |
| 0135 | [Small capitals and skipped ink are synthesised from the face's outlines](0135-small-capitals-and-skipped-ink-are-synthesised-from-the-faces-outlines.md) | rendering |
| 0136 | [SVG pictures draw clips, masks, patterns, markers, images and text](0136-svg-pictures-draw-clips-masks-patterns-markers-images-and-text.md) | assets |
| 0137 | [A box placed past a scroller is moved back as it scrolls](0137-a-box-placed-past-a-scroller-is-moved-back-as-it-scrolls.md) | api |
| 0138 | [Animated pictures play as one texture per address](0138-animated-pictures-play-as-one-texture-per-address.md) | assets |
| 0139 | [Colours in any space are worked out in sRGB as the cascade reads them](0139-colours-in-any-space-are-worked-out-in-srgb-as-the-cascade-reads-them.md) | api |
| 0140 | [A Document's window focus is a state the cascade answers](0140-a-documents-window-focus-is-a-state-the-cascade-answers.md) | api |
| 0141 | [Keys and the arrows cross a frame; the frame fades and moves with its iframe](0141-keys-and-the-arrows-cross-a-frame-the-frame-fades-and-moves-with-its-iframe.md) | api |
| 0142 | [The pointer and the keys reach callbacks as a page's DOM events](0142-the-pointer-and-the-keys-reach-callbacks-as-a-pages-dom-events.md) | api |
| 0143 | [A face's look, and the faces a family has, follow its sheets](0143-a-faces-look-and-the-faces-a-family-has-follow-its-sheets.md) | rendering |
| 0144 | [A background clipped to text shows through a picture of its glyphs](0144-a-background-clipped-to-text-shows-through-a-picture-of-its-glyphs.md) | rendering |
| 0145 | [Text is shaped by a pure-Go OpenType layout reader](0145-text-is-shaped-by-a-pure-go-opentype-layout-reader.md) | rendering |
| 0146 | [A right-to-left scroller scrolls from zero to below it](0146-a-right-to-left-scroller-scrolls-from-zero-to-below-it.md) | api |
| 0147 | [Lines are clamped as Chrome clamps them](0147-lines-are-clamped-as-chrome-clamps-them.md) | api |
| 0148 | [Variable fonts are read in pure Go, and font-display is timed on the Document's clock](0148-variable-fonts-are-read-in-pure-go-and-font-display-is-timed-on-the-documents-clock.md) | rendering |
| 0149 | [Rounded sides are bands of the ring cut to their side, as Chrome paints them](0149-rounded-sides-are-bands-of-the-ring-cut-to-their-side-as-chrome-paints-them.md) | rendering |
| 0150 | [SVG filters run on the CPU over the pixels they show at](0150-svg-filters-run-on-the-cpu-over-the-pixels-they-show-at.md) | assets |
| 0151 | [Stroked text is a font of its own, its glyphs' outlines stroked on the CPU](0151-stroked-text-is-a-font-of-its-own-glyphs-outlines.md) | rendering |
| 0152 | [The old attributes of tables, rules and fonts are hints as Chrome's](0152-the-old-attributes-of-tables-rules-and-fonts-are-hints-as-chromes.md) | api |
| 0153 | [Text is rasterized at its own size, and dashes are laid out in device pixels](0153-text-is-rasterized-at-its-own-size-and-dashes-are-laid-in-device-pixels.md) | rendering |
| 0154 | [Complex counter styles, inside markers and scroll state follow Chrome's algorithms](0154-complex-counter-styles-inside-markers-and-scroll-state-follow-chromes-algorithms.md) | rendering |
| 0155 | [The system says its contrast, forced colours and selection colours, read-only](0155-the-system-says-its-contrast-forced-colours-and-selection-colours-read-only.md) | api |
| 0156 | [Columns cut what they hold through one flow of units](0156-columns-cut-what-they-hold-through-one-flow-of-units.md) | rendering |
| 0157 | [Arabic is joined, and marks attached, by HarfBuzz's stages and masks](0157-arabic-is-joined-and-marks-attached-by-harfbuzzs-stages-and-masks.md) | rendering |
| 0158 | [Styled scrollbars are boxes of their parts, as Chrome's](0158-styled-scrollbars-are-boxes-of-their-parts-as-chromes.md) | api |
| 0159 | [A pattern is read as JavaScript, and a form is submitted by its button](0159-a-pattern-is-read-as-javascript-and-a-form-is-submitted-by-its-button.md) | api |
| 0161 | [SVG filters run in the element's own space, with every primitive Chrome draws](0161-svg-filters-run-in-the-elements-own-space-with-every-primitive-chrome-draws.md) | assets |
| 0162 | [A copy puts the selection's markup beside its text](0162-a-copy-puts-the-selections-markup-beside-its-text.md) | api |
| 0163 | [Scrolls snap and glide as Chrome's strategies, on the Document's clock](0163-scrolls-snap-and-glide-as-chromes-strategies-on-the-documents-clock.md) | api |
| 0169 | [The user's scrolls glide on Chrome's curves, and two-axis snaps land where both areas show](0169-the-users-scrolls-glide-on-chromes-curves-and-two-axis-snaps-land-where-both-areas-show.md) | api |

## Writing one

- Only engine decisions belong here: a real choice between alternatives that would be costly to reverse. Implementation details belong in package docs, and design notes in [design](../design).
- Record a decision once it has settled, not at every step of a change.
- Copy [TEMPLATE.md](TEMPLATE.md) to `NNNN-short-title.md` with the next number, and add a row above, in the same commit as the change.
- Keep it under a screen. If there was no alternative, it isn't a decision.
- Never rewrite an accepted decision: write a new one and mark the old one `Superseded by [NNNN](…)`.
