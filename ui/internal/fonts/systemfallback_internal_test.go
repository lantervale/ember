package fonts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/ui/internal/style"
)

const shapeTestdata = "../../../internal/shape/testdata/"

// renamed is the font file at path with its family, Go, named as two
// other letters: the test faces are all Go's.
func renamed(t *testing.T, path, family string) []byte {
	t.Helper()
	require.Len(t, family, 2)
	ttf, err := os.ReadFile(path)
	require.NoError(t, err)
	_, tables, err := tableDirectory(bytesReader(ttf), 0)
	require.NoError(t, err)
	for _, tb := range tables {
		if tb.tag != tagName {
			continue
		}
		name := ttf[tb.offset : tb.offset+tb.length]
		for i := 0; i+4 <= len(name); i++ {
			if name[i] == 0 && name[i+1] == 'G' && name[i+2] == 0 && name[i+3] == 'o' {
				name[i+1], name[i+3] = family[0], family[1]
			}
		}
		for i := 0; i+2 <= len(name); i++ {
			if name[i] == 'G' && name[i+1] == 'o' {
				name[i], name[i+1] = family[0], family[1]
			}
		}
	}
	return ttf
}

type bytesReader []byte

func (b bytesReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b)) {
		return 0, os.ErrInvalid
	}
	return copy(p, b[off:]), nil
}

// machine is a font folder of the faces named, each a test face renamed.
func machine(t *testing.T, faces map[string]string) *Installed {
	t.Helper()
	dir := t.TempDir()
	for family, file := range faces {
		require.NoError(t, os.WriteFile(filepath.Join(dir, family+".ttf"), renamed(t, shapeTestdata+file, family), 0o644))
	}
	return Scan(dir)
}

// onPlatform has the platform fall back to Ja for Han and kana in
// Japanese, Zh for them otherwise, and Ar for Arabic, for the test.
func onPlatform(t *testing.T) {
	t.Helper()
	was := preferred
	preferred = func(script shape.Tag, cjk bool, loc locale, _ fallbackClass) []preferredFamily {
		switch {
		case script == tagArab:
			return families(w400700, "Ar")
		case script == tagHani || script == tagKana || cjk:
			if loc == localeJa {
				return families(w400700, "Ja")
			}
			return families(w400700, "Zh")
		}
		return nil
	}
	t.Cleanup(func() { preferred = was })
}

func fallbackSet(t *testing.T, on bool) *Set {
	t.Helper()
	onPlatform(t)
	s := Builtin()
	s.SetSystemFallback(on)
	s.SetInstalled(machine(t, map[string]string{"Ja": "go-han.ttf", "Zh": "go-han.ttf", "Ar": "go-arabic.ttf",
		"Ko": "noto-hangul.ttf"}))
	s.Await()
	return s
}

func goFont(lang string) style.Font {
	return style.Font{Family: []string{"Go", "sans-serif"}, Size: 20, Weight: 400, Lang: lang}
}

const han = "漢字"

func TestWithoutTheSystemFallbackACharacterNoFaceHasIsMissing(t *testing.T) {
	s := fallbackSet(t, false)
	f := goFont("")
	assert.False(t, s.Draws(f, han))
	assert.Empty(t, s.SystemFaces(f))
	_, other := s.FallbackMetrics(f, han)
	assert.False(t, other)
}

func TestACharacterNoFaceHasIsMeasuredInTheInstalledFaceThatHasIt(t *testing.T) {
	s := fallbackSet(t, true)
	f := goFont("")
	assert.True(t, s.Draws(f, han))
	assert.InDelta(t, 2*f.Size, s.Advance(f, han), 0.01, "two ideographs an em wide each")
	faces := s.SystemFaces(f)
	require.Len(t, faces, 1)
	assert.Equal(t, "Zh", faces[0].Face.Family)
	assert.Equal(t, []Range{{Lo: '字', Hi: '字'}, {Lo: '漢', Hi: '漢'}}, faces[0].Ranges)
	m, other := s.FallbackMetrics(f, "a"+han)
	assert.True(t, other)
	assert.Equal(t, s.metricsOf(faces[0].Face, f.Size), m, "its metrics stand in the line")
	assert.Equal(t, s.Advance(f, "a")+s.Advance(f, han), s.Advance(f, "a"+han))
}

func TestTheLanguagePicksTheFaceOfHan(t *testing.T) {
	s := fallbackSet(t, true)
	gen := s.SystemGen()
	ja, zh := goFont("ja-JP"), goFont("zh-Hant")
	s.Advance(ja, han)
	s.Advance(zh, han)
	assert.Equal(t, "Ja", s.SystemFaces(ja)[0].Face.Family)
	assert.Equal(t, "Zh", s.SystemFaces(zh)[0].Face.Family)
	assert.Greater(t, s.SystemGen(), gen)
	gen = s.SystemGen()
	s.Advance(ja, han)
	assert.Equal(t, gen, s.SystemGen(), "a character is chosen once")
}

func TestArabicJoinsInItsFallbackFace(t *testing.T) {
	s := fallbackSet(t, true)
	const word = "بتب"
	in := s.Advance(goFont(""), word)
	var letters float32
	for _, r := range word {
		letters += s.Advance(goFont(""), string(r))
	}
	assert.Greater(t, in, letters, "joined forms are wider in the test face")
	own := Builtin()
	require.NoError(t, own.Add("Ar", 400, false, renamed(t, shapeTestdata+"go-arabic.ttf", "Ar")))
	assert.InDelta(t, own.Advance(style.Font{Family: []string{"Ar"}, Size: 20, Weight: 400}, word), in, 0.001,
		"as the face itself shapes it")
}

func TestACharacterOfAScriptThePlatformNamesNoFaceForIsLookedForInEveryFace(t *testing.T) {
	s := fallbackSet(t, true)
	f := goFont("")
	assert.True(t, s.Draws(f, "한"))
	faces := s.SystemFaces(f)
	require.Len(t, faces, 1)
	assert.Equal(t, "Noto Sans KR Thin", faces[0].Face.Family)
}

func TestTextSpacingTrimReadsThePunctuationOfTheFallbackFace(t *testing.T) {
	assert.False(t, fallbackSet(t, false).HanGlyph(goFont(""), '「', false).Halt)
	assert.True(t, fallbackSet(t, true).HanGlyph(goFont(""), '「', false).Halt, "the face drawing it has halt")
}

func TestSpacesAndSelectorsFallToNoFace(t *testing.T) {
	s := fallbackSet(t, true)
	for _, r := range []rune{'　', '️', '​'} {
		c, _ := s.systemGlyph(goFont(""), r)
		assert.Nil(t, c, "%U", r)
	}
}

func TestABoldFontPicksTheBoldFaceAndASerifOneItsOwnList(t *testing.T) {
	s := fallbackSet(t, true)
	bold := goFont("")
	bold.Weight = 700
	serif := goFont("")
	serif.Family = []string{"serif"}
	assert.NotEqual(t, s.contextOf(goFont("")), s.contextOf(bold))
	assert.Equal(t, classSerif, s.contextOf(serif).class)
	named := goFont("")
	named.Family = []string{"nope", "Go", "serif"}
	assert.Equal(t, classSans, s.contextOf(named).class, "a family the set has comes first")
}

func TestAWebFontsChainFallsBackToTheMachine(t *testing.T) {
	s := fallbackSet(t, true)
	file := &File{}
	s.SetWeb([]*Face{s.AddWeb("Page", 400, 400, false, nil, file)})
	s.Loaded(file, s.Fallback().TTF, s.Fallback().font)
	f := style.Font{Family: []string{"Page"}, Size: 20, Weight: 400}
	require.NotNil(t, s.Chained(f))
	assert.True(t, s.Draws(f, han))
	assert.InDelta(t, 2*f.Size, s.Advance(f, han), 0.01)
	assert.Equal(t, "Zh", s.SystemFaces(f)[0].Face.Family)
}

func TestTheCoverageOfAFontIsItsCmaps(t *testing.T) {
	for _, file := range []string{"go-han.ttf", "go-arabic.ttf", "noto-hangul.ttf", "noto-devanagari.ttf"} {
		path, err := filepath.Abs(shapeTestdata + file)
		require.NoError(t, err)
		cov := (&installedFace{path: path}).readCoverage()
		assert.True(t, cov.drawable, file)
		ttf, err := os.ReadFile(path)
		require.NoError(t, err)
		f, err := sfnt.Parse(ttf)
		require.NoError(t, err)
		var buf sfnt.Buffer
		for r := rune(0); r < 0x20000; r++ {
			g, _ := f.GlyphIndex(&buf, r)
			if (g != 0) != cov.has(r) {
				t.Fatalf("%s: %U: glyph %d, covered %v", file, r, g, cov.has(r))
			}
		}
	}
}

func TestLocales(t *testing.T) {
	for lang, want := range map[string]locale{
		"": localeNone, "en": localeNone, "ja": localeJa, "JA-jp": localeJa, "ko-KR": localeKo,
		"zh": localeZhHans, "zh-CN": localeZhHans, "zh-Hans-HK": localeZhHans, "zh-TW": localeZhHant,
		"zh-HK": localeZhHant, "zh-Hant": localeZhHant, "zh_TW": localeZhHant, "yue": localeZhHant,
	} {
		assert.Equal(t, want, localeOf(lang), lang)
	}
}
