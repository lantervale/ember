//go:build chrome

package chrome

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAProfileStillWrittenIsRemovedOnceItsWritersStop has a profile
// removed while a helper Chrome left behind still writes in it, as
// Chrome's on Linux do for a moment after the browser exits.
func TestAProfileStillWrittenIsRemovedOnceItsWritersStop(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profile")
	sub := filepath.Join(dir, "Default")
	require.NoError(t, os.MkdirAll(sub, 0o700))
	stop := time.Now().Add(300 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; time.Now().Before(stop); i++ {
			_ = os.WriteFile(filepath.Join(sub, fmt.Sprint(i)), nil, 0o600)
		}
	}()
	require.NoError(t, removeProfile(dir, 5*time.Second))
	<-done
	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err), "the profile is gone")
}
