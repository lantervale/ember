#!/bin/sh
# Rebuilds go-features.ttf, Go Regular's ASCII with ligatures, small
# capitals, figures, fractions and class kerning added (features.py),
# and go-arabic.ttf, the same ASCII with stand-ins for Arabic letters
# and marks, their joining, attachment and superiors (arabic.py):
#   brew install fonttools
#   sh make.sh <path to Go-Regular.ttf> <fonttools' python>
# Go-Regular.ttf is in golang.org/x/image/font/gofont/ttfs; fonttools'
# python is $(brew --prefix fonttools)/libexec/bin/python.
set -e
pyftsubset "$1" --unicodes="U+0020-007E" --output-file=ascii.ttf
"$2" "$(dirname "$0")/features.py" ascii.ttf go-features.ttf
"$2" "$(dirname "$0")/arabic.py" ascii.ttf go-arabic.ttf
rm ascii.ttf
# noto-devanagari.ttf is Noto Sans Devanagari (OFL, noto-devanagari-OFL.txt)
# subset to Basic Latin, Devanagari and its extensions:
#   pyftsubset NotoSansDevanagari-Regular.ttf --layout-features='*' --no-hinting \
#     --unicodes="U+0020-007E,U+0900-097F,U+200C-200D,U+25CC,U+A8E0-A8FF,U+1CD0-1CFF" \
#     --output-file=noto-devanagari.ttf
