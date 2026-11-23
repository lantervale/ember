package style_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/style"
)

func capsRuns(c style.FontCaps, size float32, text string) []string {
	var out []string
	for _, r := range (style.Font{Size: size, Caps: c}).SmallCaps(text) {
		out = append(out, r.Text+"@"+map[bool]string{true: "small", false: "full"}[r.Font.Size < size])
	}
	return out
}

func TestSmallCapsMakeLowercaseLettersSmallCapitals(t *testing.T) {
	assert.Equal(t, []string{"H@full", "ELLO@small", " W@full", "ORLD@small", " @full", "STRASSE@small"},
		capsRuns(style.SmallCaps, 16, "Hello World straße"), "spaces as they are, ß as SS")
	assert.Equal(t, []string{"HELLO, 42@small"}, capsRuns(style.AllSmallCaps, 16, "Hello, 42"))
	assert.Equal(t, []string{"T@small", "he@full", " Q@small", "uick@full"}, capsRuns(style.Unicase, 16, "The Quick"))
	assert.Nil(t, (style.Font{Size: 16, Caps: style.TitlingCaps}).SmallCaps("Titling"), "not synthesised")
	assert.Nil(t, (style.Font{Size: 16}).SmallCaps("Plain"))
}

func TestASmallCapitalIsSeventyPercentRoundedToAWholeSize(t *testing.T) {
	for size, want := range map[float32]float32{16: 11, 20: 14, 18: 13, 15: 11} {
		runs := (style.Font{Size: size, Caps: style.AllSmallCaps}).SmallCaps("a")
		assert.Equal(t, want, runs[0].Font.Size, "%v", size)
		assert.Equal(t, style.NormalCaps, runs[0].Font.Caps)
	}
}
