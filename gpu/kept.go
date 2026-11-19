package gpu

import (
	"runtime"
	"sync"
	"weak"
)

// kept is what the GPU holds of pictures, each for as long as the picture
// itself lives: it is keyed weakly, so a picture the game or a Document
// let go is collected, and its texture released at the next sweep.
type kept[K any] struct {
	live map[weak.Pointer[K]]gpuTexture
	mu   sync.Mutex
	// gone are the pictures collected since the last sweep.
	gone []weak.Pointer[K]
}

func newKept[K any]() *kept[K] { return &kept[K]{live: map[weak.Pointer[K]]gpuTexture{}} }

func (k *kept[K]) len() int { return len(k.live) }

func (k *kept[K]) get(of *K) (gpuTexture, bool) {
	g, ok := k.live[weak.Make(of)]
	return g, ok
}

// set makes g what the GPU holds of a picture, in place of what it held.
func (k *kept[K]) set(of *K, g gpuTexture) {
	key := weak.Make(of)
	if _, ok := k.live[key]; !ok {
		runtime.AddCleanup(of, k.collected, key)
	}
	k.live[key] = g
}

// collected runs once a picture is, off the frame.
func (k *kept[K]) collected(key weak.Pointer[K]) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.gone = append(k.gone, key)
}

// sweep drops what is held of the pictures collected since the last one,
// and reports whether there was any.
func (k *kept[K]) sweep(drop func(gpuTexture)) bool {
	k.mu.Lock()
	gone := k.gone
	k.gone = nil
	k.mu.Unlock()
	for _, key := range gone {
		if g, ok := k.live[key]; ok {
			drop(g)
			delete(k.live, key)
		}
	}
	return len(gone) > 0
}
