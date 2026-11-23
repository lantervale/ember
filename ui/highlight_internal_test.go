package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui/paint"
)

func TestTheSystemsSelectionColoursRepaintTheSelection(t *testing.T) {
	prefs := platform.NewPreferences()
	child := NewDocument()
	d := NewDocument(Preferences(prefs))
	d.Body().Append(P(Text("selected")), Iframe(child))
	win := platform.NewWindow(platform.Size(300, 200))
	win.Add(d)
	w := ember.New().AddEntities(win, prefs)
	w.RunFrames(4)
	require.True(t, d.replay.replayedLast, "still frames replay")
	assert.Equal(t, paint.Highlight{}, d.replay.frame.Highlight, "none known: Chrome's own")

	blue := platform.Highlight{Background: color.Hex("#0078d7"), Text: color.White}
	prefs.SyncHighlight(blue)
	w.RunFrames(1)
	assert.False(t, d.replay.replayedLast, "the system's colours changed: painted again")
	want := paint.Highlight{Background: blue.Background, Text: blue.Text}
	assert.Equal(t, want, d.replay.frame.Highlight)
	assert.Equal(t, want, child.systemHighlight(), "an iframe selects in its host's system colours")
	w.RunFrames(2)
	assert.True(t, d.replay.replayedLast, "and replayed once drawn")
}
