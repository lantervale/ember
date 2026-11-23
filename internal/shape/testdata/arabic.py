"""Adds to a subset of Go Regular stand-ins for Arabic letters and
marks with the GSUB and GPOS an Arabic face has, which shape's tests
and ui's Chrome comparisons read: joined forms (isol, init, medi,
fina), a lam-alef ligature (rlig), a contextual alternate (calt), marks
on bases, on a ligature's components and on marks (mark, mkmk),
cursive attachment (curs), and superiors and inferiors (sups, subs).
Each letter is a Latin one, each form of it the letter with a stroke on
the sides it joins, its advance growing with them so that every form
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


unicodes = {}
conn = 260
stroke = name("_")


def letter(new, code, ch, joins):
    a = adv(name(ch))
    composite(new, [(name(ch), 0, 0, 1)], a)
    unicodes[code] = new
    if joins == "D":
        composite(new + ".init", [(stroke, 0, 0, 0.25), (name(ch), conn, 0, 1)], a + conn)
        composite(new + ".medi", [(stroke, 0, 0, 0.25), (name(ch), conn, 0, 1), (stroke, conn + a, 0, 0.25)],
                  a + 2 * conn)
    composite(new + ".fina", [(name(ch), 0, 0, 1), (stroke, a, 0, 0.25)], a + conn)


dual = {"beh": (0x0628, "b"), "teh": (0x062A, "t"), "seen": (0x0633, "s"), "lam": (0x0644, "l"),
        "meem": (0x0645, "m"), "noon": (0x0646, "n"), "heh": (0x0647, "h"), "yeh": (0x064A, "y")}
right = {"alef": (0x0627, "I"), "dal": (0x062F, "d"), "reh": (0x0631, "r"), "waw": (0x0648, "w")}
for n, (code, ch) in dual.items():
    letter(n, code, ch, "D")
for n, (code, ch) in right.items():
    letter(n, code, ch, "R")
composite("tatweel", [(stroke, 0, 0, 0.5)], 2 * conn)
unicodes[0x0640] = "tatweel"
composite("beh.init.alt", [(stroke, 0, 0, 0.25), (name("p"), conn, 0, 1)], adv(name("p")) + conn + 140)
lam, alef = name("l"), name("I")
lig = adv(lam) + adv(alef) + 80
composite("lam_alef", [(lam, 0, 0, 1), (alef, adv(lam) + 80, 0, 1)], lig)
composite("lam_alef.fina", [(lam, 0, 0, 1), (alef, adv(lam) + 80, 0, 1), (stroke, lig, 0, 0.25)], lig + conn)

quote = name("'")
composite("fatha", [(quote, 0, 0, 0.7)], 0)
composite("kasra", [(name(","), 0, 0, 0.8)], 250)
composite("shadda", [(name("w"), 0, 0, 0.45)], 0)
composite("acutecomb", [(quote, 0, 0, 0.6)], 180)
unicodes.update({0x064E: "fatha", 0x0650: "kasra", 0x0651: "shadda", 0x0301: "acutecomb"})

digits = [name(c) for c in "0123456789"]
for d in digits:
    composite(d + ".sups", [(d, 0, 640, 0.6)], round(adv(d) * 0.6))
    composite(d + ".subs", [(d, 0, -160, 0.6)], round(adv(d) * 0.6))
font.setGlyphOrder(order)
for table in font["cmap"].tables:
    if table.isUnicode():
        table.cmap.update(unicodes)


def cls(names):
    return "[" + " ".join(names) + "]"


forms = list(dual) + list(right)
init = [n + ".init" for n in dual]
medi = [n + ".medi" for n in dual]
fina = [n + ".fina" for n in forms]
arabic = forms + init + medi + fina + ["tatweel", "beh.init.alt"]
marks = ["fatha", "kasra", "shadda", "acutecomb"]
ligatures = ["lam_alef", "lam_alef.fina"]
latin = [g for g in order if g not in arabic + marks + ligatures + [".notdef"]]

bases = []
for g in arabic + [name(c) for c in "aeiou"]:
    a = adv(g)
    bases.append(f"pos base {g} <anchor {a // 2} 1200> mark @TOP <anchor {a // 2} -300> mark @BOTTOM;")
cursive = []
for n, rise in (("seen", 60), ("heh", 0)):
    cursive.append(f"pos cursive {n}.init <anchor NULL> <anchor 0 {rise}>;")
    cursive.append(f"pos cursive {n}.medi <anchor {adv(n + '.medi')} 0> <anchor 0 {2 * rise + 40}>;")
    cursive.append(f"pos cursive {n}.fina <anchor {adv(n + '.fina')} -40> <anchor NULL>;")
nl = "\n  "
fea = f"""
languagesystem DFLT dflt;
languagesystem latn dflt;
languagesystem arab dflt;
table GDEF {{
  GlyphClassDef {cls(latin + arabic)}, {cls(ligatures)}, {cls(marks)}, ;
}} GDEF;
feature isol {{ script arab; sub {cls(forms)} by {cls(forms)}; }} isol;
feature fina {{ script arab; sub {cls(forms)} by {cls(fina)}; }} fina;
feature medi {{ script arab; sub {cls(dual)} by {cls(medi)}; }} medi;
feature init {{ script arab; sub {cls(dual)} by {cls(init)}; }} init;
feature rlig {{
  script arab;
  lookupflag IgnoreMarks;
  sub lam.init alef.fina by lam_alef;
  sub lam.medi alef.fina by lam_alef.fina;
}} rlig;
feature calt {{
  script arab;
  lookupflag IgnoreMarks;
  sub beh.init' [meem.medi meem.fina] by beh.init.alt;
}} calt;
feature sups {{ sub {cls(digits)} by {cls(d + ".sups" for d in digits)}; }} sups;
feature subs {{ sub {cls(digits)} by {cls(d + ".subs" for d in digits)}; }} subs;
markClass [fatha shadda acutecomb] <anchor 120 0> @TOP;
markClass [kasra] <anchor 120 0> @BOTTOM;
markClass [fatha] <anchor 100 -20> @ON_SHADDA;
feature mark {{
  {nl.join(bases)}
  pos ligature lam_alef <anchor 300 1300> mark @TOP <anchor 300 -300> mark @BOTTOM
    ligComponent <anchor 900 1300> mark @TOP <anchor 900 -300> mark @BOTTOM;
  pos ligature lam_alef.fina <anchor 300 1300> mark @TOP <anchor 300 -300> mark @BOTTOM
    ligComponent <anchor 900 1300> mark @TOP <anchor 900 -300> mark @BOTTOM;
}} mark;
feature mkmk {{
  pos mark shadda <anchor 140 420> mark @ON_SHADDA;
}} mkmk;
feature curs {{
  script arab;
  lookupflag RightToLeft IgnoreMarks;
  {nl.join(cursive)}
}} curs;
"""
addOpenTypeFeaturesFromString(font, fea)
font.save(sys.argv[2])
