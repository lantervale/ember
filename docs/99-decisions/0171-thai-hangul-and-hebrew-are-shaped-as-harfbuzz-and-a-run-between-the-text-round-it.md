# 0171. Thai, Hangul and Hebrew are shaped as HarfBuzz, and a run between the text round it

- Status: Accepted
- Date: 2026-11-23
- Area: rendering
- Owner: engine

## Context

[0164](0164-indic-khmer-myanmar-and-use-scripts-are-shaped-by-harfbuzzs-syllabic-shapers.md) left Thai, Lao and Hangul to the default shaper, Hebrew's presentation forms and Arabic's modifier marks out, and Mongolian, N'Ko, Adlam and the other joining scripts HarfBuzz 14 gives the Universal Shaping Engine on the Arabic shaper. A run was shaped alone: an Arabic word an element cuts drew each side in its isolated and final forms, where Chrome hands HarfBuzz the paragraph round the run as context. A base and a variation selector were two glyphs whatever the face's cmap said.

## Decision

- `internal/shape` ports HarfBuzz 14.2.1's Thai shaper (sara am decomposed, its nikhahit moved before the marks above; for Thai in a face whose GSUB lacks the script, the Windows and Mac Private Use Area glyphs old faces place marks with), its Hangul shaper (jamo composed into a syllable the face has, syllables it lacks decomposed, `ljmo`, `vjmo` and `tjmo` on the jamo left, tone marks before their syllable, `calt` off, no normalisation), its Hebrew shaper (presentation forms composed when no `mark` feature places points, its mark reordering, GPOS only where it has the Hebrew script) and Arabic's modifier-combining-mark reordering. Each shaper's mark zeroing, fallback positioning and old `kern` table follow HarfBuzz's table of them; HarfBuzz's list of broken GDEF tables (the Mac's Tahoma, Courier New, Times New Roman Italic among them) is left out as it leaves them out.
- The joining scripts are routed as HarfBuzz 14.2.1 routes them: Arabic and Syriac to the Arabic shaper (Syriac only where GSUB has more than `DFLT`), the others to the USE with Arabic joining, Mongolian's free variation selectors taking their letter's form.
- `Plan.ShapeIn` shapes a run between the text before and after it, as HarfBuzz's buffer context: the joining reads it. draw gives each run of a text the rest of the text round it, and `draw.Style`'s `Before`/`After` (`twod.Between`) the text round the text; ui measures the words at the edges of a run, and draws its fragments, between the paragraph's text round them where a letter there joins. A run starting with a mark no longer takes a dotted circle: Chrome sets no beginning-of-text flag.
- A cluster holding a variation selector is left undecomposed; a base and the selector after it make the glyph cmap's format 14 maps them to, else the selector is shaped and hidden. draw and ui keep the selectors the face lacks in their runs; VS15 keeps a character's glyph where an emoji picture would draw it, VS16 asks for the picture.

## Alternatives

- Shaping the whole paragraph once and cutting it by element, as Chrome does for runs of one font: right for ligatures across elements too, but ui measures and paints per run and element; context gives the joined forms, which are what readers see, at the cost of one more measure of the edge words.
- A shape cache keyed by the context always: the context only changes joining scripts' runs, so it keys only plans that join.

## Consequences

Thai, Lao, Hangul, Hebrew, Mongolian, N'Ko, Adlam, Syriac, Mandaic, Manichaean, Psalter Pahlavi, Hanifi Rohingya and Phags-pa in the Mac's faces and in committed subsets shape as `hb-shape` does, context and variation sequences included, and ui lays them out and draws them as Chrome; an Arabic word cut by an element joins. A ligature across elements of one font is still shaped apart (Chrome shapes the run whole).
