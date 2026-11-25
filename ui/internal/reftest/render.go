package reftest

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// What a worker replies for a page.
const (
	Drawn   = "drawn"
	Skipped = "skip"
	Failed  = "error"
)

// Render is what became of one page: drawn to Picture, skipped or failed
// for Detail.
type Render struct {
	Kind, Detail, Picture string
}

// Renderer draws pages in worker processes: Binary run with Args is an
// ember program reading "path\tpicture" lines on its standard input
// (command run with -worker). A page that takes longer than Timeout, or
// whose worker dies, fails, and a new worker takes the next.
type Renderer struct {
	Binary  string
	Args    []string
	Root    string
	Out     string
	Workers int
	Timeout time.Duration
	// Progress, when set, is told of each page done.
	Progress func(done, total int)
}

// Picture is the file p's picture is drawn to.
func (r *Renderer) Picture(p string) string {
	return filepath.Join(r.Out, strings.ReplaceAll(p, "/", "__")+".png")
}

// Render draws pages, each once.
func (r *Renderer) Render(ctx context.Context, pages []string) map[string]Render {
	queue := make(chan string, len(pages))
	for _, p := range pages {
		queue <- p
	}
	close(queue)
	var (
		mu   sync.Mutex
		got  = make(map[string]Render, len(pages))
		wg   sync.WaitGroup
		done int
	)
	record := func(p string, rd Render) {
		mu.Lock()
		got[p] = rd
		done++
		if r.Progress != nil {
			r.Progress(done, len(pages))
		}
		mu.Unlock()
	}
	for i := range max(1, r.Workers) {
		wg.Go(func() { r.drain(ctx, i, queue, record) })
	}
	wg.Wait()
	return got
}

// worker is one running worker process.
type worker struct {
	cmd     *exec.Cmd
	in      io.WriteCloser
	replies chan string
	stderr  *tail
}

func (r *Renderer) start(ctx context.Context, n int) (*worker, error) {
	cmd := exec.CommandContext(ctx, r.Binary, r.Args...) //nolint:gosec // the runner's own worker
	cmd.Env = append(os.Environ(), "EMBER_HIDDEN=1", "EMBER_SCALE=1",
		"EMBER_CRASH_DIR="+filepath.Join(r.Out, "crash", fmt.Sprint(n)))
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	w := &worker{cmd: cmd, in: in, replies: make(chan string), stderr: &tail{}}
	cmd.Stderr = w.stderr
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		s := bufio.NewScanner(out)
		s.Buffer(make([]byte, 64<<10), 1<<20)
		for s.Scan() {
			w.replies <- s.Text()
		}
		close(w.replies)
	}()
	return w, nil
}

func (w *worker) stop() {
	_ = w.in.Close()
	_ = w.cmd.Process.Kill()
	for range w.replies {
	}
	_ = w.cmd.Wait()
}

// drain draws the pages of queue one at a time in worker n, starting a
// new worker after one that died or hung.
func (r *Renderer) drain(ctx context.Context, n int, queue <-chan string, record func(string, Render)) {
	var w *worker
	defer func() {
		if w != nil {
			w.stop()
		}
	}()
	for p := range queue {
		if ctx.Err() != nil {
			record(p, Render{Kind: Failed, Detail: "cancelled"})
			continue
		}
		if w == nil {
			var err error
			if w, err = r.start(ctx, n); err != nil {
				record(p, Render{Kind: Failed, Detail: "worker: " + err.Error()})
				continue
			}
		}
		picture := r.Picture(p)
		_ = os.Remove(picture)
		if _, err := fmt.Fprintf(w.in, "%s\t%s\n", p, picture); err != nil {
			record(p, Render{Kind: Failed, Detail: "worker: " + err.Error()})
			w.stop()
			w = nil
			continue
		}
		select {
		case line, ok := <-w.replies:
			if !ok {
				w.stop()
				record(p, Render{Kind: Failed, Detail: "crashed: " + lastLine(w.stderr.String())})
				w = nil
				continue
			}
			kind, rest, _ := strings.Cut(line, "\t")
			_, detail, _ := strings.Cut(rest, "\t")
			rd := Render{Kind: kind, Detail: detail}
			if kind == Drawn {
				rd.Picture = picture
			}
			record(p, rd)
		case <-time.After(r.Timeout):
			record(p, Render{Kind: Failed, Detail: fmt.Sprintf("timed out after %s", r.Timeout)})
			w.stop()
			w = nil
		}
	}
}

// tail keeps the last few kilobytes written to it.
type tail struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf.Write(p)
	if t.buf.Len() > 8<<10 {
		b := t.buf.Bytes()
		keep := append([]byte(nil), b[len(b)-4<<10:]...)
		t.buf.Reset()
		t.buf.Write(keep)
	}
	return len(p), nil
}

// lastLine is the panic s tells of, or else its last line that says
// something, as a reason.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "panic: ") {
			return strings.TrimSpace(l)
		}
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" && !strings.HasPrefix(l, "desktop: saved") {
			return l
		}
	}
	return "the worker exited"
}
