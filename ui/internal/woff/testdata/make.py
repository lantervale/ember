"""The WOFF of the Latin subset, and an icon font: three of its glyphs
at Private Use Area code points, each left side bearing its glyph's
xMin, as icon fonts have them, so woff2_compress transforms hmtx."""
import sys

from fontTools.ttLib import TTFont

src, woff, icons = sys.argv[1:4]
f = TTFont(src, recalcTimestamp=False)
f.flavor = "woff"
f.save(woff)

f = TTFont(src, recalcTimestamp=False)
names = {0xF002: "question", 0xF015: "ampersand", 0xE900: "at"}
for t in f["cmap"].tables:
    t.cmap = dict(names)
for name in f.getGlyphOrder():
    advance, _ = f["hmtx"].metrics[name]
    f["hmtx"].metrics[name] = (advance, getattr(f["glyf"][name], "xMin", 0))
f.save(icons)
