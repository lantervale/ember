//go:build chrome

package chrome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMajorOfReadsAChromeUserAgent(t *testing.T) {
	assert.Equal(t, 158, majorOf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "+
		"(KHTML, like Gecko) HeadlessChrome/158.0.8127.37 Safari/537.36"))
	assert.Equal(t, 156, majorOf("Mozilla/5.0 (X11; Linux x86_64) Chrome/156.0.8078.80 Safari/537.36"))
	assert.Equal(t, 0, majorOf("Mozilla/5.0 Firefox/140.0"))
}
