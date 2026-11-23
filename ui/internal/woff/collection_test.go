package woff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/internal/colourfont/colourtest"
)

func TestACollectionIsReadAsItsFirstFont(t *testing.T) {
	ttc := colourtest.Collection(gobold.TTF, gomono.TTF)
	out, err := Sfnt(ttc)
	require.NoError(t, err)
	f, err := sfnt.Parse(out)
	require.NoError(t, err)
	name, err := f.Name(nil, sfnt.NameIDFull)
	require.NoError(t, err)
	assert.Equal(t, "Go Bold", name)
	for cut := 16; cut < len(ttc)/2; cut += len(ttc) / 9 {
		_, err = First(ttc[:cut])
		assert.Error(t, err, "cut at %d", cut)
	}
}
