package ui

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
)

func picture(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, w, h))))
	return b.Bytes()
}

// asked is a fetcher that notes what it was asked for and answers from
// pictures, an error for a url it doesn't hold.
type asked struct {
	mu       sync.Mutex
	urls     []string
	pictures map[string][]byte
}

func (a *asked) fetch(url string) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.urls = append(a.urls, url)
	if b, ok := a.pictures[url]; ok {
		return b, nil
	}
	return nil, errors.New("404 Not Found")
}

func TestAPictureNoFileHoldsIsFetchedOffTheFrameAndShownOnceItLands(t *testing.T) {
	a := &asked{pictures: map[string][]byte{"https://art.test/door.png": picture(t, 30, 20)}}
	door := Img(Attr("src", "https://art.test/door.png"))
	d := NewDocument(FetchImages(a.fetch))
	d.Body().Append(door)

	d.Layout(geom.V2(200, 100))
	assert.Empty(t, d.Warnings(), "a picture on its way is no warning")
	assert.NotEqual(t, float32(30), door.Box().W, "it has no size before it lands")

	d.fetching.Wait()
	d.Layout(geom.V2(200, 100))
	assert.Equal(t, geom.V2(30, 20), geom.V2(door.Box().W, door.Box().H), "laid out again at the picture's size")
	d.Layout(geom.V2(200, 100))
	d.fetching.Wait()
	assert.Equal(t, []string{"https://art.test/door.png"}, a.urls, "asked once, as written")
	assert.Empty(t, d.Warnings())
}

func TestABackgroundIsFetchedWhenItIsFirstPainted(t *testing.T) {
	a := &asked{pictures: map[string][]byte{"wall.png": picture(t, 8, 8)}}
	d := NewDocument(CSS(`body { background-image: url(wall.png); }`), FetchImages(a.fetch))
	d.Layout(geom.V2(200, 100))
	assert.Empty(t, a.urls, "layout needs no background")

	assert.Nil(t, d.image("wall.png"), "what paint gets before it lands")
	d.fetching.Wait()
	d.Layout(geom.V2(200, 100))
	require.NotNil(t, d.image("wall.png"))
	assert.Equal(t, geom.V2(8, 8), d.image("wall.png").Size())
	assert.True(t, d.replay.repaint, "the frame kept before it landed is painted again")
}

func TestAFileWinsOverTheFetcher(t *testing.T) {
	a := &asked{}
	d := NewDocument(Images(fstest.MapFS{"bow.png": {Data: picture(t, 16, 12)}}), FetchImages(a.fetch))
	bow := Img(Attr("src", "bow.png"))
	d.Body().Append(bow)
	d.Layout(geom.V2(200, 100))
	d.fetching.Wait()
	assert.Equal(t, float32(16), bow.Box().W)
	assert.Empty(t, a.urls)
}

func TestAPictureThatCannotBeFetchedOrReadIsAWarningAskedOnce(t *testing.T) {
	a := &asked{pictures: map[string][]byte{"notes.txt": []byte("not a picture")}}
	d := NewDocument(FetchImages(a.fetch))
	d.Body().Append(Img(Attr("src", "gone.png")), Img(Attr("src", "notes.txt")), Img(Attr("src", "gone.png")))
	for range 3 {
		d.Layout(geom.V2(200, 100))
		d.fetching.Wait()
	}
	require.Len(t, d.Warnings(), 2)
	texts := d.Warnings()[0].Error() + "\n" + d.Warnings()[1].Error()
	assert.Contains(t, texts, "gone.png: 404 Not Found")
	assert.Contains(t, texts, "notes.txt: ")
	assert.ElementsMatch(t, []string{"gone.png", "notes.txt"}, a.urls)
}

func TestFetchesRunAFewAtATime(t *testing.T) {
	var mu sync.Mutex
	running, most := 0, 0
	release := make(chan struct{})
	d := NewDocument(FetchImages(func(string) ([]byte, error) {
		mu.Lock()
		running++
		most = max(most, running)
		mu.Unlock()
		<-release
		mu.Lock()
		running--
		mu.Unlock()
		return nil, errors.New("gone")
	}))
	for i := range 3 * fetchesAtOnce {
		d.image(string(rune('a' + i)))
	}
	close(release)
	d.fetching.Wait()
	assert.LessOrEqual(t, most, fetchesAtOnce)
}
