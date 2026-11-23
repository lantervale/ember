#!/bin/sh
# Rebuilds go-features.ttf, Go Regular's ASCII with ligatures, small
# capitals, figures, fractions and class kerning added (features.py),
# go-arabic.ttf, the same ASCII with stand-ins for Arabic letters and
# marks, their joining, attachment and superiors (arabic.py),
# go-thai.ttf and go-hebrew.ttf, stand-ins for Thai and Hebrew as old
# fonts have them, with no layout tables (scripts.py), go-spaced.ttf,
# a ligature and contextual alternates reaching across a space (spaced.py),
# and go-han.ttf, stand-ins for ideographs and full-width punctuation with
# halt (han.py):
#   brew install fonttools
#   sh make.sh <path to Go-Regular.ttf> <fonttools' python>
# Go-Regular.ttf is in golang.org/x/image/font/gofont/ttfs; fonttools'
# python is $(brew --prefix fonttools)/libexec/bin/python.
set -e
pyftsubset "$1" --unicodes="U+0020-007E" --output-file=ascii.ttf
"$2" "$(dirname "$0")/features.py" ascii.ttf go-features.ttf
"$2" "$(dirname "$0")/arabic.py" ascii.ttf go-arabic.ttf
"$2" "$(dirname "$0")/scripts.py" ascii.ttf go-thai.ttf go-hebrew.ttf
"$2" "$(dirname "$0")/spaced.py" ascii.ttf go-spaced.ttf
"$2" "$(dirname "$0")/han.py" ascii.ttf go-han.ttf
rm ascii.ttf
# noto-devanagari.ttf is Noto Sans Devanagari (OFL, noto-devanagari-OFL.txt)
# subset to Basic Latin, Devanagari and its extensions:
#   pyftsubset NotoSansDevanagari-Regular.ttf --layout-features='*' --no-hinting \
#     --unicodes="U+0020-007E,U+0900-097F,U+200C-200D,U+25CC,U+A8E0-A8FF,U+1CD0-1CFF" \
#     --output-file=noto-devanagari.ttf
# noto-thai.ttf, noto-lao.ttf, noto-hebrew.ttf and noto-hangul.ttf are Google
# Fonts' Noto Sans Thai, Lao, Hebrew and KR (OFL, noto-*-OFL.txt), their
# variable files instanced at wght=400 (and wdth=100) by
#   fonttools varLib.instancer -o NotoSansThai-400.ttf 'NotoSansThai[wdth,wght].ttf' wght=400 wdth=100
# then subset:
#   pyftsubset NotoSansThai-400.ttf --layout-features='*' --no-hinting \
#     --unicodes="U+0020-007E,U+0E01-0E5B,U+200C-200D,U+25CC" --output-file=noto-thai.ttf
#   pyftsubset NotoSansLao-400.ttf --layout-features='*' --no-hinting \
#     --unicodes="U+0020-007E,U+0E81-0EDF,U+200C-200D,U+25CC" --output-file=noto-lao.ttf
#   pyftsubset NotoSansHebrew-400.ttf --layout-features='*' --no-hinting \
#     --unicodes="U+0020-007E,U+0591-05F4,U+FB1D-FB4F,U+200C-200D,U+25CC" --output-file=noto-hebrew.ttf
#   pyftsubset NotoSansKR-400.ttf --layout-features='*' --no-hinting --output-file=noto-hangul.ttf \
#     --unicodes="U+0020-007E,U+1100-1112,U+115F-1175,U+119E,U+11A8-11C2,U+11FF,U+A960,U+D7B0,U+302E-302F,U+25CC,U+AC00,U+AC01,U+AC04,U+D55C,U+AD6D,U+C5B4,U+AE00,U+B9D0"
