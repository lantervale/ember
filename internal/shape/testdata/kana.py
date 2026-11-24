"""Adds to a subset of Go Regular stand-ins for the characters
line-break's values break around, an em wide: ideographs, small kana, the
prolonged sound mark, iteration marks, the wave dash, the CJK hyphen, the
ellipsis, centred punctuation, a postfix and a prefix, each a Latin
character stretched over the em, or centred on it when too narrow to
stretch. Run by make.sh."""
import sys

from fontTools.ttLib import TTFont
from fontTools.ttLib.tables._g_l_y_f import Glyph, GlyphComponent

font = TTFont(sys.argv[1], recalcTimestamp=False)
glyf, hmtx = font["glyf"], font["hmtx"]
cmap = font.getBestCmap()
order = font.getGlyphOrder()
em = font["head"].unitsPerEm
unicodes = {}

for code, ch in [(0x6F22, "M"), (0x5B57, "W"), (0x30AB, "H"), (0x3041, "a"), (0x30A1, "e"), (0x3083, "y"),
                 (0x30C3, "u"), (0x30FC, "-"), (0x3005, "&"), (0x309D, "s"), (0x301C, "~"), (0x2010, "_"),
                 (0x2026, ":"), (0xFF01, "!"), (0x30FB, "."), (0xFF05, "%"), (0xFF04, "$"), (0x3002, "o")]:
    part = cmap[ord(ch)]
    glyf[part].recalcBounds(glyf)
    a = hmtx[part][0]
    new = "u%04X" % code
    g = Glyph()
    g.numberOfContours = -1
    c = GlyphComponent()
    c.glyphName, c.x, c.y, c.flags = part, 80, 0, 0
    if (em - 160) / a < 2:
        c.transform = [[(em - 160) / a, 0], [0, 1]]
    else:
        c.x = (em - a) // 2
    g.components = [c]
    glyf.glyphs[new] = g
    g.recalcBounds(glyf)
    hmtx[new] = (em, g.xMin)
    order.append(new)
    unicodes[code] = new
font.setGlyphOrder(order)
for table in font["cmap"].tables:
    if table.isUnicode():
        table.cmap.update(unicodes)
font.save(sys.argv[2])
