"""Adds to a subset of Go Regular stand-ins for ideographs and the
full-width punctuation of Chinese and Japanese, an em wide, and a halt
feature setting the punctuation half wide, as CJK faces have it: each
ideograph a Latin capital stretched over the em, each bracket Go's own
in the half of the em by the text it holds, each dot on the left, the
colon and the middle dot centred. Run by make.sh."""
import sys

from fontTools.feaLib.builder import addOpenTypeFeaturesFromString
from fontTools.ttLib import TTFont
from fontTools.ttLib.tables._g_l_y_f import Glyph, GlyphComponent

font = TTFont(sys.argv[1], recalcTimestamp=False)
glyf, hmtx = font["glyf"], font["hmtx"]
cmap = font.getBestCmap()
order = font.getGlyphOrder()
em = font["head"].unitsPerEm
half = em // 2
unicodes = {}


def name(ch):
    return cmap[ord(ch)]


def composite(code, parts):
    new = "u%04X" % code
    g = Glyph()
    g.numberOfContours = -1 if parts else 0
    g.components = []
    for part, dx, dy, sx, sy in parts:
        c = GlyphComponent()
        c.glyphName, c.x, c.y, c.flags = name(part), dx, dy, 0
        if (sx, sy) != (1, 1):
            c.transform = [[sx, 0], [0, sy]]
        g.components.append(c)
    glyf.glyphs[new] = g
    if parts:
        g.recalcBounds(glyf)
    hmtx[new] = (em, g.xMin if parts else 0)
    order.append(new)
    unicodes[code] = new
    return new


def ideograph(code, ch):
    a = hmtx[name(ch)][0]
    composite(code, [(ch, 80, 0, (em - 160) / a, 1)])


def placed(code, ch, x, s=1):
    """ch's glyph scaled by s, its ink's left edge at x."""
    g = glyf[name(ch)]
    g.recalcBounds(glyf)
    return composite(code, [(ch, round(x - g.xMin * s), 0, s, s)])


def width(ch, s=1):
    g = glyf[name(ch)]
    g.recalcBounds(glyf)
    return round((g.xMax - g.xMin) * s)


for code, ch in [(0x6F22, "M"), (0x5B57, "W"), (0x304B, "K"), (0x30AB, "H")]:
    ideograph(code, ch)
opens = [placed(code, ch, em - 120 - width(ch)) for code, ch in
         [(0x300C, "["), (0x300E, "{"), (0xFF08, "("), (0x3010, "<")]]
closes = [placed(code, ch, 120) for code, ch in [(0x300D, "]"), (0x300F, "}"), (0xFF09, ")"), (0x3011, ">")]]
dots = [placed(0x3002, "o", 120, 0.6)] + [placed(code, ch, 120) for code, ch in
                                       [(0x3001, ","), (0xFF0C, ","), (0xFF0E, ".")]]
middles = [placed(code, ch, half - width(ch) // 2) for code, ch in [(0xFF1A, ":"), (0xFF1B, ";"), (0x30FB, ".")]]
composite(0x3000, [])
font.setGlyphOrder(order)
for table in font["cmap"].tables:
    if table.isUnicode():
        table.cmap.update(unicodes)

fea = f"""
languagesystem DFLT dflt;
languagesystem hani dflt;
languagesystem kana dflt;
languagesystem latn dflt;
feature halt {{
  pos [{" ".join(opens)}] <-{half} 0 -{half} 0>;
  pos [{" ".join(closes + dots)}] <0 0 -{half} 0>;
  pos [{" ".join(middles)}] <-{half // 2} 0 -{half} 0>;
}} halt;
"""
addOpenTypeFeaturesFromString(font, fea)
font.save(sys.argv[2])
