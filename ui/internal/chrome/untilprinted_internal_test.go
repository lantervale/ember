//go:build chrome

package chrome

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAQuietChromeIsRunAgainLaterUntilItPrintsAPage(t *testing.T) {
	var slept []time.Duration
	runs := 0
	out, err := untilPrinted(6, func(d time.Duration) { slept = append(slept, d) }, func() ([]byte, error) {
		runs++
		if runs < 4 {
			return nil, &quietError{"crashed"}
		}
		return []byte("<html></html>"), nil
	})
	require.NoError(t, err)
	assert.Equal(t, "<html></html>", string(out))
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}, slept)
}

func TestAChromeThatNeverPrintsSaysWhatItLastSaid(t *testing.T) {
	runs := 0
	_, err := untilPrinted(3, func(time.Duration) {}, func() ([]byte, error) {
		runs++
		return nil, &quietError{"crashpad"}
	})
	require.ErrorIs(t, err, errNoPage)
	assert.Contains(t, err.Error(), "crashpad")
	assert.Equal(t, 3, runs)
}

func TestAHalfPrintedPageIsRunAgain(t *testing.T) {
	runs := 0
	_, err := untilPrinted(2, func(time.Duration) {}, func() ([]byte, error) {
		runs++
		return []byte("<html><body>"), nil
	})
	require.ErrorIs(t, err, errNoPage)
	assert.Equal(t, 2, runs)
}

func TestAChromeThatFailsOtherwiseIsNotRunAgain(t *testing.T) {
	runs := 0
	failed := errors.New("no browser")
	_, err := untilPrinted(6, func(time.Duration) {}, func() ([]byte, error) {
		runs++
		return nil, failed
	})
	require.ErrorIs(t, err, failed)
	assert.Equal(t, 1, runs)
}
