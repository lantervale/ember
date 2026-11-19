#!/bin/sh
# Rebuilds the fixtures of ui/internal/woff/testdata from the Go font:
#   brew install woff2 fonttools
#   sh make.sh <path to Go-Regular.ttf> <fonttools' python>
# Go-Regular.ttf is in golang.org/x/image/font/gofont/ttfs; fonttools'
# python is $(brew --prefix fonttools)/libexec/bin/python.
set -e
go_regular=$1
python=$2
pyftsubset "$go_regular" --unicodes="U+0020-007E,U+00A0-00FF" --layout-features='*' \
	--output-file=go-regular-latin.ttf
"$python" make.py go-regular-latin.ttf go-regular-latin.woff icons-all.ttf
pyftsubset icons-all.ttf --unicodes="U+F002,U+F015,U+E900" --output-file=go-icons.ttf
rm icons-all.ttf
for name in go-regular-latin go-icons; do
	cp $name.ttf packed.ttf
	woff2_compress packed.ttf
	mv packed.woff2 $name.woff2
	cp $name.woff2 reference.woff2
	woff2_decompress reference.woff2
	mv reference.ttf $name.roundtrip.ttf
	rm packed.ttf reference.woff2
done
"$python" -c 'import sys
from fontTools.ttLib import TTFont
f = TTFont(sys.argv[1], recalcTimestamp=False)
f.flavor = "woff"
f.save(sys.argv[2])' go-icons.ttf go-icons.woff
# woff2_compress leaves hmtx as it is: fonttools packs the icon font
# again with it transformed.
"$python" -m fontTools.ttLib.woff2 compress -q --hmtx-transform -o go-icons.hmtx.woff2 go-icons.ttf
