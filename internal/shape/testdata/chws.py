"""Gives go-han.ttf a chws feature of its own besides halt: a closing
bracket before an opening one set half wide (where Chrome's rules trim the
opening one), a closing bracket or a dot before a closing bracket half
wide, an opening bracket after an opening one half wide, and nothing to a
dot before an opening bracket (which Chrome's rules trim), so a page shows
whose rules spaced a pair. Run by make.sh."""
import sys

from fontTools.feaLib.builder import addOpenTypeFeaturesFromString
from fontTools.ttLib import TTFont

font = TTFont(sys.argv[1], recalcTimestamp=False)
cmap = font.getBestCmap()
half = font["head"].unitsPerEm // 2


def names(chars):
    return " ".join(cmap[ord(c)] for c in chars)


opens, closes, dots, middles = names("「『（【"), names("」』）】"), names("。、，．"), names("：；・")
fea = f"""
languagesystem DFLT dflt;
languagesystem hani dflt;
languagesystem kana dflt;
languagesystem latn dflt;
@open = [{opens}];
@close = [{closes}];
@dot = [{dots}];
feature halt {{
  pos @open <-{half} 0 -{half} 0>;
  pos [@close @dot] <0 0 -{half} 0>;
  pos [{middles}] <-{half // 2} 0 -{half} 0>;
}} halt;
feature chws {{
  pos @close' <0 0 -{half} 0> @open;
  pos [@close @dot]' <0 0 -{half} 0> @close;
  pos @open @open' <-{half} 0 -{half} 0>;
}} chws;
"""
addOpenTypeFeaturesFromString(font, fea)
font.save(sys.argv[2])
