"""Adds to a subset of Go Regular the glyphs and the GSUB and GPOS
features shape's tests and ui's Chrome comparisons read: ligatures,
small capitals, figures, fractions and class kerning. Every new glyph
is a composite of Go's own, its advance chosen so that each feature
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
    for part, dx, dy, scale in parts:
        c = GlyphComponent()
        c.glyphName, c.x, c.y, c.flags = part, dx, dy, 0
        if scale != 1:
            c.transform = [[scale, 0], [0, scale]]
        g.components.append(c)
    glyf.glyphs[new] = g
    hmtx[new] = (advance, 0)
    order.append(new)


digits = [name(c) for c in "0123456789"]
lower = [name(c) for c in "abcdefghijklmnopqrstuvwxyz"]
upper = [name(c) for c in "ABCDEFGHIJKLMNOPQRSTUVWXYZ"]

f, i, l, c, t = (name(x) for x in "filct")
composite("f_i", [(f, 0, 0, 1), (i, adv(f) - 120, 0, 1)], adv(f) + adv(i) - 120)
composite("f_l", [(f, 0, 0, 1), (l, adv(f) - 120, 0, 1)], adv(f) + adv(l) - 120)
composite("f_f_i", [(f, 0, 0, 1), (f, adv(f) - 140, 0, 1), (i, 2 * adv(f) - 260, 0, 1)],
          2 * adv(f) + adv(i) - 260)
composite("c_t", [(c, 0, 0, 1), (t, adv(c) - 60, 0, 1)], adv(c) + adv(t) + 200)
for u in upper:
    composite(u + ".sc", [(u, 0, 0, 0.75)], round(adv(u) * 0.75) + 40)
for d in digits:
    composite(d + ".osf", [(d, 0, -180, 1)], adv(d))
    composite(d + ".tf", [(d, 100, 0, 1)], adv(d) + 200)
    composite(d + ".pf", [(d, -40, 0, 1)], adv(d) - 80)
    composite(d + ".lf", [(d, 0, 40, 1)], adv(d) + 60)
    composite(d + ".numr", [(d, 0, 600, 0.55)], round(adv(d) * 0.55))
    composite(d + ".dnom", [(d, 0, 0, 0.55)], round(adv(d) * 0.55))
slash = name("/")
composite("fraction", [(slash, 0, 0, 1)], adv(slash) - 200)
zero = name("0")
composite("zero.zero", [(zero, 0, 0, 1), (slash, 120, 0, 0.6)], adv(zero) + 100)
font.setGlyphOrder(order)


def cls(names):
    return "[" + " ".join(names) + "]"


small = " ".join(u + ".sc" for u in upper)
fea = f"""
languagesystem DFLT dflt;
languagesystem latn dflt;
@dig = {cls(digits)};
@lc = {cls(lower)};
@uc = {cls(upper)};
@sc = [{small}];
feature liga {{ sub {f} {f} {i} by f_f_i; sub {f} {i} by f_i; sub {f} {l} by f_l; }} liga;
feature dlig {{ sub {c} {t} by c_t; }} dlig;
feature smcp {{ sub @lc by @sc; }} smcp;
feature c2sc {{ sub @uc by @sc; }} c2sc;
feature onum {{ sub @dig by {cls(d + ".osf" for d in digits)}; }} onum;
feature lnum {{ sub @dig by {cls(d + ".lf" for d in digits)}; }} lnum;
feature tnum {{ sub @dig by {cls(d + ".tf" for d in digits)}; }} tnum;
feature pnum {{ sub @dig by {cls(d + ".pf" for d in digits)}; }} pnum;
feature zero {{ sub {zero} by zero.zero; }} zero;
lookup NUMR {{ sub @dig by {cls(d + ".numr" for d in digits)}; }} NUMR;
lookup DNOM {{ sub @dig by {cls(d + ".dnom" for d in digits)}; }} DNOM;
lookup FRACTION {{ sub {slash} by fraction; }} FRACTION;
feature frac {{
  sub @dig' lookup NUMR {slash};
  sub {slash}' lookup FRACTION @dig;
  sub fraction @dig' lookup DNOM;
}} frac;
@kernL = [{name("A")} {name("L")}];
@kernR = [{name("V")} {name("W")} {name("Y")} {name("T")}];
feature kern {{
  pos @kernL @kernR -160;
  pos {name("T")} <0 0 -90 0> {name("o")} <30 0 0 0>;
  pos {name("o")} {name(" ")} -70;
  pos {name(" ")} {name("A")} -110;
}} kern;
"""
addOpenTypeFeaturesFromString(font, fea)
font.save(sys.argv[2])
