"""Adds to a subset of Go Regular a ligature and contextual alternates
that reach across a space, which ui's Chrome comparisons read: "x x"
is one glyph, a z before " z" and a q after "q " wider ones. Every new
glyph is a composite of Go's own, its advance chosen so that each rule
shows in a width. Run by make.sh."""
import sys

from fontTools.feaLib.builder import addOpenTypeFeaturesFromString
from fontTools.ttLib import TTFont
from fontTools.ttLib.tables._g_l_y_f import Glyph, GlyphComponent

font = TTFont(sys.argv[1], recalcTimestamp=False)
glyf, hmtx = font["glyf"], font["hmtx"]
cmap = font.getBestCmap()
order = font.getGlyphOrder()


def name(ch):
    return cmap[ord(ch)]


def adv(g):
    return hmtx[g][0]


def composite(new, parts, advance):
    g = Glyph()
    g.numberOfContours = -1
    g.components = []
    for part, dx in parts:
        c = GlyphComponent()
        c.glyphName, c.x, c.y, c.flags = part, dx, 0, 0
        g.components.append(c)
    glyf.glyphs[new] = g
    order.append(new)
    font.setGlyphOrder(order)
    g.recalcBounds(glyf)
    hmtx[new] = (advance, g.xMin)


x, z, q, space = name("x"), name("z"), name("q"), name(" ")
composite("x_space_x", [(x, 0), (x, adv(x) + adv(space) - 500)], 2 * adv(x) + adv(space) - 500)
composite("z.wide", [(z, 150)], adv(z) + 300)
composite("q.wide", [(q, 150)], adv(q) + 400)
fea = f"""
languagesystem DFLT dflt;
languagesystem latn dflt;
feature liga {{ sub {x} {space} {x} by x_space_x; }} liga;
feature calt {{
  sub {z}' {space} {z} by z.wide;
  sub {q} {space} {q}' by q.wide;
}} calt;
"""
addOpenTypeFeaturesFromString(font, fea)
font.save(sys.argv[2])
