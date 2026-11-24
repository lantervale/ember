//go:build darwin

package desktop //nolint:testpackage // tests the unexported pasteboard markup, never the pasteboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThePasteboardsHTMLIsMarkedUTF8AsChromesIs(t *testing.T) {
	assert.Equal(t, "<meta charset='utf-8'><b>déjà</b>", pasteboardHTML("<b>d\x00éjà</b>"))
}
