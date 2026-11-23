package clipboard_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/internal/clipboard"
)

func TestHTMLFormatSaysWhereItsFragmentIs(t *testing.T) {
	fragment := `<b>bold</b> and café`
	data := clipboard.CFHTML(fragment)
	assert.True(t, strings.HasPrefix(string(data), "Version:0.9\r\nStartHTML:"))
	assert.Equal(t, fragment, clipboard.Fragment(data), "the offsets count bytes, é two")

	var start, end int
	for _, line := range strings.Split(string(data), "\r\n") {
		if v, ok := strings.CutPrefix(line, "StartHTML:"); ok {
			start = atoi(t, v)
		}
		if v, ok := strings.CutPrefix(line, "EndHTML:"); ok {
			end = atoi(t, v)
		}
	}
	assert.True(t, strings.HasPrefix(string(data[start:]), "<html>"))
	assert.Equal(t, len(data), end)
}

func TestAFragmentIsReadFromAnotherAppsHTMLFormat(t *testing.T) {
	page := "Version:0.9\r\nStartHTML:-1\r\nEndHTML:-1\r\nStartFragment:0000000089\r\nEndFragment:0000000093\r\n" +
		"<p>dog</p>"
	assert.Equal(t, "<p>d", clipboard.Fragment([]byte(page)))
	assert.Empty(t, clipboard.Fragment([]byte("Version:0.9\r\n")), "no offsets")
	assert.Empty(t, clipboard.Fragment([]byte("StartFragment:10\r\nEndFragment:900\r\n")), "past its end")
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	require.NoError(t, err)
	return n
}
