#!/bin/sh
# Rebuilds go-features.ttf, Go Regular's ASCII with ligatures, small
# capitals, figures, fractions and class kerning added (features.py):
#   brew install fonttools
#   sh make.sh <path to Go-Regular.ttf> <fonttools' python>
# Go-Regular.ttf is in golang.org/x/image/font/gofont/ttfs; fonttools'
# python is $(brew --prefix fonttools)/libexec/bin/python.
set -e
pyftsubset "$1" --unicodes="U+0020-007E" --output-file=ascii.ttf
"$2" "$(dirname "$0")/features.py" ascii.ttf go-features.ttf
rm ascii.ttf
