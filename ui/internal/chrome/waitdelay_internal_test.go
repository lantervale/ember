//go:build chrome && unix

package chrome

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAKilledBrowsersHelpersDoNotHoldItsOutputOpen(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	cmd := browserCommand(ctx, 200*time.Millisecond, "/bin/sh", "-c", "sleep 30 & sleep 30")
	start := time.Now()
	_, err := cmd.Output()
	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second, "the read waits for the helper's 30 s")
}
