package style

import "strings"

// Longhand is how a longhand East Asian counter style writes a value:
// each digit but 0 followed by the marker of its place.
type Longhand struct {
	Lang LonghandLang
	// Digits are 0 to 9, Markers those of the tens, hundreds and
	// thousands, Groups those of the groups of four digits past the
	// first (10⁴, 10⁸, 10¹²).
	Digits  [10]string
	Markers [3]string
	Groups  [3]string
	// Informal drops a 1 before a marker: every one in Japanese and
	// Korean, a group's leading ten's in Chinese (十一, but 一百一十).
	Informal bool
}

// LonghandLang is whose rules a longhand style writes a value by.
type LonghandLang uint8

const (
	// LonghandJapanese writes -9999 to 9999 as CSS Counter Styles does,
	// every zero dropped.
	LonghandJapanese LonghandLang = iota
	// LonghandChinese writes any value in groups of four digits, a run of
	// zeros inside it as one zero, as Chrome does.
	LonghandChinese
	// LonghandKorean writes any value in groups of four digits, zeros
	// dropped and a space after each group's marker, as Chrome does.
	LonghandKorean
)

func (l *Longhand) inRange(n int) bool {
	return l.Lang != LonghandJapanese || n >= -9999 && n <= 9999
}

func (l *Longhand) write(n int) string {
	if n == 0 {
		return l.Digits[0]
	}
	if l.Lang == LonghandJapanese {
		return l.japanese(n)
	}
	return l.grouped(n)
}

func (l *Longhand) japanese(n int) string {
	var b strings.Builder
	for place, div := 3, 1000; place >= 0; place, div = place-1, div/10 {
		d := n / div % 10
		if d == 0 {
			continue
		}
		if place == 0 || d != 1 || !l.Informal {
			b.WriteString(l.Digits[d])
		}
		if place > 0 {
			b.WriteString(l.Markers[place-1])
		}
	}
	return b.String()
}

// The symbols grouped writes: none, the digits 0 to 9, then the markers
// of the tens, hundreds and thousands, then the group markers.
const (
	noSymbol    = -1
	tensMarker  = 10
	groupMarker = 13
)

// grouped is Chrome's CJK ideographic algorithm: each group of four
// digits, least significant first, with its marker; a group's zeros kept
// in Chinese where digits follow them, collapsed to one, and moved past
// the group's marker where the group ends in them.
func (l *Longhand) grouped(n int) string {
	var buf [4][9]int
	for i := range buf {
		for j := range buf[i] {
			buf[i][j] = noSymbol
		}
	}
	chinese := l.Lang == LonghandChinese
	dropOne := func(d int) bool { return l.Lang == LonghandKorean && l.Informal && d == 1 }
	for i := range 4 {
		v := n % 10000
		n /= 10000
		g := &buf[3-i]
		if v != 0 && i > 0 {
			g[7] = groupMarker + i - 1
		}
		d := v % 10
		trailing := chinese && d == 0
		if d != 0 && (i == 0 || !dropOne(d)) {
			g[6] = d
		}
		for place, div := 1, 10; place <= 3; place, div = place+1, div*10 {
			if n == 0 && v < div {
				break
			}
			d = v / div % 10
			if d != 0 && !dropOne(d) || d == 0 && !trailing {
				g[6-2*place] = d
			}
			if place < 3 {
				trailing = trailing && d == 0
			}
			if d != 0 {
				g[7-2*place] = tensMarker + place - 1
			}
		}
		if trailing && i > 0 {
			g[6], g[7], g[8] = g[7], g[8], 0
		}
		if l.Informal && v < 20 {
			g[4] = noSymbol
		}
		if n == 0 {
			break
		}
	}
	var b strings.Builder
	last, wrote := noSymbol, ""
	for _, g := range buf {
		for _, a := range g {
			if a == noSymbol {
				continue
			}
			if a != 0 || chinese && last != 0 {
				wrote = l.symbol(a)
				b.WriteString(wrote)
				if l.Lang == LonghandKorean && a >= groupMarker {
					wrote = " "
					b.WriteString(wrote)
				}
			}
			last = a
		}
	}
	s := b.String()
	if chinese && last == 0 || wrote == " " {
		s = s[:len(s)-len(wrote)]
	}
	return s
}

func (l *Longhand) symbol(a int) string {
	switch {
	case a >= groupMarker:
		return l.Groups[a-groupMarker]
	case a >= tensMarker:
		return l.Markers[a-tensMarker]
	}
	return l.Digits[a]
}

// ethiopic writes n, at least 1, in Ethiopic numerals: groups of two
// digits from the right, each written in tens and ones, those of an odd
// place followed by ፻ (100) and those of an even place past the first by
// ፼ (10000), a group of 1 written by its sign alone where it leads or
// stands at an odd place.
func ethiopic(n int) string {
	if n == 1 {
		return "፩"
	}
	var groups []int
	for ; n > 0; n /= 100 {
		groups = append(groups, n%100)
	}
	var b strings.Builder
	for i := len(groups) - 1; i >= 0; i-- {
		g := groups[i]
		if g != 0 && (g != 1 || i != len(groups)-1 && i%2 == 0) {
			if g/10 > 0 {
				b.WriteRune(rune(0x1372 + g/10 - 1))
			}
			if g%10 > 0 {
				b.WriteRune(rune(0x1369 + g%10 - 1))
			}
		}
		switch {
		case i%2 == 1 && g != 0:
			b.WriteRune(0x137B)
		case i%2 == 0 && i > 0:
			b.WriteRune(0x137C)
		}
	}
	return b.String()
}
