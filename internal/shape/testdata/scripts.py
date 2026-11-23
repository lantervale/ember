"""Adds to a subset of Go Regular stand-ins for Thai and for Hebrew, as
old fonts have them, with no layout tables: go-thai.ttf has Thai's
letters and marks and the glyphs Windows and the Mac put in the Private
Use Area for marks moved down or left and letters without their
descender, and two variation sequences (cmap format 14); go-hebrew.ttf has Hebrew's letters and points and the
presentation forms of letters with points. Each letter is a Latin one,
each mark a small one above or below, each moved form the same shifted.
Run by make.sh."""
import sys

from fontTools.ttLib import TTFont
from fontTools.ttLib.tables._c_m_a_p import CmapSubtable
from fontTools.ttLib.tables._g_l_y_f import Glyph, GlyphComponent


def build(src, out, add):
    font = TTFont(src, recalcTimestamp=False)
    glyf, hmtx = font["glyf"], font["hmtx"]
    cmap = font.getBestCmap()
    order = font.getGlyphOrder()
    unicodes = {}

    def name(ch):
        return cmap[ord(ch)]

    def composite(code, parts, advance):
        new = "u%04X" % code
        g = Glyph()
        g.numberOfContours = -1
        g.components = []
        for part, dx, dy, scale in parts:
            c = GlyphComponent()
            c.glyphName, c.x, c.y, c.flags = name(part), dx, dy, 0
            if scale != 1:
                c.transform = [[scale, 0], [0, scale]]
            g.components.append(c)
        glyf.glyphs[new] = g
        g.recalcBounds(glyf)
        hmtx[new] = (advance, g.xMin)
        order.append(new)
        unicodes[code] = new

    def letter(code, ch):
        composite(code, [(ch, 0, 0, 1)], hmtx[name(ch)][0])

    def mark(code, ch, above, dx=0, dy=0):
        composite(code, [(ch, -500 + dx, (1100 if above else -500) + dy, 0.4)], 0)

    uvs = add(letter, mark, composite)
    font.setGlyphOrder(order)
    for table in font["cmap"].tables:
        if table.isUnicode():
            table.cmap.update(unicodes)
    if uvs is not None:
        font["cmap"].tables.append(uvs)
    font.save(out)


def thai(letter, mark, composite):
    latin = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUV"
    for i, code in enumerate(range(0x0E01, 0x0E2F)):
        letter(code, latin[i % len(latin)])
    for code, ch in [(0x0E30, ":"), (0x0E32, "J"), (0x0E33, "W"), (0x0E40, "("), (0x0E41, "["), (0x0E42, "{"),
                     (0x0E43, "<"), (0x0E44, ">"), (0x0E45, "|"), (0x0E46, "%")]:
        letter(code, ch)
    above = {0x0E31: "u", 0x0E34: "i", 0x0E35: "y", 0x0E36: "o", 0x0E37: "w", 0x0E47: "s", 0x0E48: "1",
             0x0E49: "2", 0x0E4A: "3", 0x0E4B: "4", 0x0E4C: "x", 0x0E4D: "0", 0x0E4E: "~"}
    below = {0x0E38: "v", 0x0E39: "n", 0x0E3A: "."}
    for code, ch in above.items():
        mark(code, ch, True)
    for code, ch in below.items():
        mark(code, ch, False)
    # Windows's Private Use Area: marks down, down and left, left; letters
    # without their descender.
    for codes, dx, dy in [((0xF70A, 0xF70B, 0xF70C, 0xF70D, 0xF70E), 0, -300),
                          ((0xF705, 0xF706, 0xF707, 0xF708, 0xF709), -200, -300),
                          ((0xF713, 0xF714, 0xF715, 0xF716, 0xF717), -200, 0)]:
        for code, base in zip(codes, (0x0E48, 0x0E49, 0x0E4A, 0x0E4B, 0x0E4C)):
            mark(code, above[base], True, dx, dy)
    for code, base in [(0xF718, 0x0E38), (0xF719, 0x0E39), (0xF71A, 0x0E3A)]:
        mark(code, below[base], False, 0, -200)
    for code, base in [(0xF710, 0x0E31), (0xF701, 0x0E34), (0xF702, 0x0E35), (0xF703, 0x0E36), (0xF704, 0x0E37),
                       (0xF712, 0x0E47), (0xF711, 0x0E4D)]:
        mark(code, above[base], True, -200, 0)
    letter(0xF70F, "Z")
    # The Mac's only, for tho than.
    letter(0xF89E, "Y")
    # Variation sequences: a with VS1 drawn as A, b with VS2 its own glyph.
    uvs = CmapSubtable.newSubtable(14)
    uvs.platformID, uvs.platEncID, uvs.language, uvs.cmap = 0, 5, 0xFF, {}
    uvs.uvsDict = {0xFE00: [(ord("a"), "A")], 0xFE01: [(ord("b"), None)]}
    return uvs


def hebrew(letter, mark, composite):
    latin = "abcdefghijklmnopqrstuvwxyzABC"
    for i, code in enumerate(range(0x05D0, 0x05EB)):
        letter(code, latin[i % len(latin)])
    letter(0x05F2, "W")
    points = "0123456789!@#$%^&*()"
    for i, code in enumerate(range(0x05B0, 0x05C3)):
        if code not in (0x05BE, 0x05C0):
            mark(code, points[i % len(points)], code in (0x05B9, 0x05BF, 0x05C1, 0x05C2))
    for code, ch in [(0xFB1D, "Y"), (0xFB1F, "X"), (0xFB2A, "S"), (0xFB2B, "T"), (0xFB2C, "U"), (0xFB2D, "V"),
                     (0xFB2E, "D"), (0xFB2F, "E"), (0xFB31, "F"), (0xFB35, "G"), (0xFB3C, "H"), (0xFB44, "I"),
                     (0xFB49, "J"), (0xFB4A, "K"), (0xFB4B, "L"), (0xFB4C, "M"), (0xFB4D, "N"), (0xFB4E, "O")]:
        composite(code, [(ch, 0, 0, 1), (".", 300, 300, 1)], 1300)


build(sys.argv[1], sys.argv[2], thai)
build(sys.argv[1], sys.argv[3], hebrew)
