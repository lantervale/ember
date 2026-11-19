package paint

import (
	"runtime"
	"sync"
	"weak"

	"github.com/lantervale/ember/twod"
)

// derived keeps what paint makes of a texture, once, for as long as the
// texture lives: keyed weakly, so a page's picture goes when its page
// does, and what was made of it with it.
type derived[V any] struct {
	mu sync.Mutex
	of map[weak.Pointer[twod.Texture]]V
}

func (d *derived[V]) len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.of)
}

// get is what was made of tex; d.mu is held.
func (d *derived[V]) get(tex *twod.Texture) (V, bool) {
	v, ok := d.of[weak.Make(tex)]
	return v, ok
}

// put keeps v as what is made of tex until tex is collected; d.mu is
// held.
func (d *derived[V]) put(tex *twod.Texture, v V) {
	key := weak.Make(tex)
	if d.of == nil {
		d.of = map[weak.Pointer[twod.Texture]]V{}
	}
	if _, ok := d.of[key]; !ok {
		runtime.AddCleanup(tex, d.forget, key)
	}
	d.of[key] = v
}

func (d *derived[V]) forget(key weak.Pointer[twod.Texture]) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.of, key)
}
