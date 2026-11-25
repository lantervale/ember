package fonts

import (
	"runtime"

	"github.com/lantervale/ember/internal/shape"
)

// preferredFamily is a family the platform falls back to, and the weight
// of its face for regular text and for bold, 0 for the asked one.
type preferredFamily struct {
	name          string
	regular, bold int
}

// preferred are the families the platform falls back to for a character
// of script (or, cjk, of the punctuation Chinese, Japanese and Korean
// share) in a context, in order, as Chrome finds them there.
//
//nolint:gochecknoglobals // the platform's, set apart by tests
var preferred = preferredOn(runtime.GOOS)

func preferredOn(goos string) func(script shape.Tag, cjk bool, loc locale, class fallbackClass) []preferredFamily {
	switch goos {
	case goosDarwin, "ios":
		return preferredMac
	case "windows":
		return preferredWindows
	case "js", "wasip1":
		return func(shape.Tag, bool, locale, fallbackClass) []preferredFamily { return nil }
	}
	return preferredLinux
}

//nolint:gochecknoglobals // the scripts the tables name
var (
	tagHani = shape.MakeTag("hani")
	tagKana = shape.MakeTag("kana")
	tagHang = shape.MakeTag("hang")
	tagArab = shape.MakeTag("arab")
	tagHebr = shape.MakeTag("hebr")
	tagThai = shape.MakeTag("thai")
	tagDeva = shape.MakeTag("deva")
)

func families(weights [2]int, names ...string) []preferredFamily {
	out := make([]preferredFamily, len(names))
	for i, n := range names {
		out[i] = preferredFamily{name: n, regular: weights[0], bold: weights[1]}
	}
	return out
}

//nolint:gochecknoglobals // weights of the regular and bold faces fallen back to
var (
	w300600 = [2]int{300, 600}
	w400700 = [2]int{400, 700}
)

// preferredMac is CoreText's fallback as Chrome on macOS gets it:
// Hiragino for Japanese, kana and the shared punctuation without a
// language, PingFang (Songti in serif) for Han, Apple SD Gothic Neo
// (AppleMyungjo in serif) for Korean. PingFang's outlines are Apple's own
// (hvgl), which no reader but CoreText's draws: Hiragino Sans GB stands
// for it.
func preferredMac(script shape.Tag, cjk bool, loc locale, class fallbackClass) []preferredFamily {
	serif := class == classSerif
	hiraSans := families(w300600, "Hiragino Sans")
	kaku := families(w300600, "Hiragino Kaku Gothic ProN", "Hiragino Sans")
	mincho := families(w300600, "Hiragino Mincho ProN")
	gothic := families(w400700, "Apple SD Gothic Neo")
	myungjo := families(w400700, "AppleMyungjo")
	simplified := append(families(w400700, "PingFang SC"), families(w300600, "Hiragino Sans GB")...)
	traditional := append(families(w400700, "PingFang TC"), families(w300600, "Hiragino Sans GB")...)
	songSC, songTC := families(w400700, "Songti SC"), families(w400700, "Songti TC", "Songti SC")
	switch {
	case script == tagHani || script == tagKana || cjk:
		switch loc {
		case localeJa:
			if serif {
				return mincho
			}
			return kaku
		case localeKo:
			if serif {
				return myungjo
			}
			return gothic
		case localeNone, localeZhHans, localeZhHant:
		}
		if serif {
			if loc == localeZhHant {
				return songTC
			}
			return songSC
		}
		switch {
		case script == tagKana:
			return hiraSans
		case cjk && loc == localeNone:
			return hiraSans
		case loc == localeZhHant:
			return traditional
		}
		return simplified
	case script == tagHang:
		if serif {
			return myungjo
		}
		return gothic
	case script == tagArab:
		if class == classMono {
			return families(w400700, "Courier New", "Geeza Pro")
		}
		return families(w400700, "Geeza Pro")
	case script == tagHebr:
		return families([2]int{}, "Lucida Grande", "Arial Hebrew")
	case script == tagThai:
		return families(w400700, "Thonburi")
	case script == tagDeva:
		return families(w400700, "Kohinoor Devanagari")
	}
	return nil
}

// preferredLinux is fontconfig's fallback with the fonts distributions
// ship: Noto's CJK faces by language, then the older ones.
func preferredLinux(script shape.Tag, cjk bool, loc locale, class fallbackClass) []preferredFamily {
	serif := class == classSerif
	cjkFamilies := func(tag string) []preferredFamily {
		if serif {
			return families(w400700, "Noto Serif CJK "+tag, "Source Han Serif "+tag)
		}
		return families(w400700, "Noto Sans CJK "+tag, "Source Han Sans "+tag)
	}
	switch {
	case script == tagHani || script == tagKana || cjk:
		switch loc {
		case localeJa:
			return append(cjkFamilies("JP"), families(w400700, "IPAGothic", "TakaoGothic", "VL Gothic")...)
		case localeKo:
			return append(cjkFamilies("KR"), families(w400700, "NanumGothic", "UnDotum")...)
		case localeZhHant:
			return append(cjkFamilies("TC"), families(w400700, "AR PL UMing TW", "WenQuanYi Zen Hei")...)
		case localeNone, localeZhHans:
		}
		if script == tagKana {
			return cjkFamilies("JP")
		}
		return append(cjkFamilies("SC"), families(w400700, "WenQuanYi Zen Hei", "WenQuanYi Micro Hei",
			"Droid Sans Fallback")...)
	case script == tagHang:
		return append(cjkFamilies("KR"), families(w400700, "NanumGothic", "UnDotum")...)
	case script == tagArab:
		return families(w400700, "Noto Sans Arabic", "Noto Naskh Arabic", "DejaVu Sans", "FreeSans")
	case script == tagHebr:
		return families(w400700, "Noto Sans Hebrew", "DejaVu Sans", "FreeSans")
	case script == tagThai:
		return families(w400700, "Noto Sans Thai", "Garuda", "Loma")
	case script == tagDeva:
		return families(w400700, "Noto Sans Devanagari", "Lohit Devanagari", "FreeSans")
	}
	return nil
}

// preferredWindows is DirectWrite's fallback as Chrome on Windows gets
// it: Yu Gothic, Microsoft YaHei, Microsoft JhengHei and Malgun Gothic
// by language (their Mincho, SimSun, PMingLiU and Batang in serif),
// Segoe UI for Arabic and Hebrew.
func preferredWindows(script shape.Tag, cjk bool, loc locale, class fallbackClass) []preferredFamily {
	serif := class == classSerif
	switch {
	case script == tagHani || script == tagKana || cjk:
		switch loc {
		case localeJa:
			if serif {
				return families(w400700, "Yu Mincho", "MS Mincho")
			}
			return families(w400700, "Yu Gothic UI", "Yu Gothic", "Meiryo", "MS Gothic")
		case localeKo:
			if serif {
				return families(w400700, "Batang")
			}
			return families(w400700, "Malgun Gothic")
		case localeZhHant:
			if serif {
				return families(w400700, "PMingLiU", "MingLiU")
			}
			return families(w400700, "Microsoft JhengHei UI", "Microsoft JhengHei")
		case localeNone, localeZhHans:
		}
		if script == tagKana {
			return families(w400700, "Yu Gothic UI", "Yu Gothic", "Meiryo")
		}
		if serif {
			return families(w400700, "SimSun", "NSimSun")
		}
		return families(w400700, "Microsoft YaHei UI", "Microsoft YaHei")
	case script == tagHang:
		if serif {
			return families(w400700, "Batang")
		}
		return families(w400700, "Malgun Gothic")
	case script == tagArab, script == tagHebr:
		return families(w400700, "Segoe UI", "Tahoma")
	case script == tagThai:
		return families(w400700, "Leelawadee UI", "Tahoma")
	case script == tagDeva:
		return families(w400700, "Nirmala UI", "Mangal")
	}
	return nil
}
