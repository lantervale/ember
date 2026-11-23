package sysprefs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestADesktopsToolIsFoundInTheStandardPlacesOnly(t *testing.T) {
	dirs := []string{"/usr/bin", "/bin", "/run/current-system/sw/bin"}
	there := map[string]bool{"/bin/gdbus": true, "/usr/bin/gdbus": true, "/run/current-system/sw/bin/gsettings": true}
	exists := func(path string) bool { return there[path] }

	path, ok := findTool("gdbus", dirs, exists)
	assert.True(t, ok)
	assert.Equal(t, "/usr/bin/gdbus", path, "the first place holding it")
	path, ok = findTool("gsettings", dirs, exists)
	assert.True(t, ok)
	assert.Equal(t, "/run/current-system/sw/bin/gsettings", path)
	_, ok = findTool("gsettings", dirs[:2], exists)
	assert.False(t, ok, "nowhere else, whatever PATH says")
}
