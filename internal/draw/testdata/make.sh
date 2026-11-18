#!/bin/sh
# Rebuilds go-kerned.ttf, the Go font's capitals with one kerning pair
# in GPOS and no kern table:
#   brew install fonttools
#   sh make.sh <path to Go-Regular.ttf> <fonttools' python>
# Go-Regular.ttf is in golang.org/x/image/font/gofont/ttfs; fonttools'
# python is $(brew --prefix fonttools)/libexec/bin/python.
set -e
pyftsubset "$1" --unicodes="U+0041-005A" --output-file=capitals.ttf
"$2" -c 'import sys
from fontTools.ttLib import TTFont
from fontTools.feaLib.builder import addOpenTypeFeaturesFromString
f = TTFont(sys.argv[1], recalcTimestamp=False)
addOpenTypeFeaturesFromString(f, "feature kern { pos A V -200; } kern;")
f.save(sys.argv[2])' capitals.ttf go-kerned.ttf
rm capitals.ttf
