// Command run runs the tests of packages in shards (package shard): each
// shard of each package one go test, -p of them at once, so a machine
// keeps its Chrome slots busy and CI splits the suite across jobs (-only).
// Each go test's output is printed whole once it ends, then the slowest
// tests; it exits 1 when any go test failed.
//
//	go run ./ui/internal/shard/run -tags 'chrome gpu' [-shards n] [-only i] [-p n] ./ui/...
package main

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lantervale/ember/ui/internal/shard"
)

func main() { os.Exit(shards()) }

func shards() int {
	tags := flag.String("tags", "", "the build tags")
	shards := flag.Int("shards", 0, "how many shards (0: the Chrome slots, EMBER_CHROME_SLOTS or half the CPUs)")
	only := flag.Int("only", -1, "run only this shard, from 0 (CI's job); every shard by default")
	jobs := flag.Int("p", 0, "how many go tests run at once (0: as many as shards)")
	timeout := flag.String("timeout", "30m", "each go test's -timeout")
	split := flag.String("split", os.Getenv(shard.SplitEnv), "the tests split by their subtests, comma-separated")
	events := flag.String("json", "", "also write every go test -json event to this file")
	slowest := flag.Int("slowest", 20, "how many of the slowest tests to list")
	flag.Parse()
	n := *shards
	if n <= 0 {
		n = slots()
	}
	if *only >= n {
		return fail(fmt.Errorf("shard %d of %d", *only, n))
	}
	p := *jobs
	if p <= 0 {
		p = n
	}
	pkgs, err := list(*tags, flag.Args())
	if err != nil {
		return fail(err)
	}
	r := &runner{tags: *tags, timeout: *timeout, split: shard.Split(*split), shards: n}
	if *events != "" {
		f, createErr := os.Create(*events)
		if createErr != nil {
			return fail(createErr)
		}
		defer f.Close()
		r.events = f
	}
	tasks, err := r.plan(pkgs, *only, p)
	if err != nil {
		return fail(err)
	}
	start := time.Now()
	failed := r.run(tasks, p)
	r.report(os.Stdout, *slowest, time.Since(start))
	if failed {
		return 1
	}
	return 0
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "shard:", err)
	return 1
}

// slots is the Chrome harness's count of Chrome slots.
func slots() int {
	if n, err := strconv.Atoi(os.Getenv("EMBER_CHROME_SLOTS")); err == nil && n > 0 {
		return n
	}
	return max(1, runtime.NumCPU()/2)
}

// list is the packages patterns name.
func list(tags string, patterns []string) ([]string, error) {
	out, err := goCmd(append([]string{"list", "-tags", tags}, patterns...)...).Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}
	return strings.Fields(string(out)), nil
}

func goCmd(args ...string) *exec.Cmd {
	cmd := exec.CommandContext(context.Background(), "go", args...)
	cmd.Stderr = os.Stderr
	return cmd
}

// task is one shard of one package.
type task struct {
	pkg   string
	shard int
	tests []string
}

type runner struct {
	tags, timeout string
	split         []string
	shards        int
	events        io.Writer

	mu      sync.Mutex
	elapsed []timed
}

type timed struct {
	name    string
	seconds float64
}

// plan lists each package's tests, p at once, and gives each shard the
// ones it runs; a shard running none of a package's has no task.
func (r *runner) plan(pkgs []string, only, p int) ([]task, error) {
	names := make([][]string, len(pkgs))
	errs := make([]error, len(pkgs))
	each(len(pkgs), p, func(k int) { names[k], errs[k] = r.tests(pkgs[k]) })
	var tasks []task
	for k, pkg := range pkgs {
		if errs[k] != nil {
			return nil, errs[k]
		}
		for i := range r.shards {
			if only >= 0 && i != only {
				continue
			}
			if picked := shard.Pick(names[k], i, r.shards, r.split); len(picked) > 0 {
				tasks = append(tasks, task{pkg: pkg, shard: i, tests: picked})
			}
		}
	}
	return tasks, nil
}

// tests is the tests, examples and fuzz targets go test runs in pkg.
func (r *runner) tests(pkg string) ([]string, error) {
	out, err := goCmd("test", "-tags", r.tags, "-list", ".", pkg).Output()
	if err != nil {
		return nil, fmt.Errorf("go test -list %s: %w", pkg, err)
	}
	var names []string
	for line := range strings.Lines(string(out)) {
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"Test", "Example", "Fuzz"} {
			if strings.HasPrefix(line, prefix) && !strings.ContainsAny(line, " \t") {
				names = append(names, line)
				break
			}
		}
	}
	return names, nil
}

// each calls do for 0 to n-1, p at once.
func each(n, p int, do func(int)) {
	work := make(chan int)
	var wg sync.WaitGroup
	for range min(p, n) {
		wg.Go(func() {
			for k := range work {
				do(k)
			}
		})
	}
	for k := range n {
		work <- k
	}
	close(work)
	wg.Wait()
}

// run runs the tasks, p at once, and reports whether any failed.
func (r *runner) run(tasks []task, p int) bool {
	failed := make([]bool, len(tasks))
	each(len(tasks), p, func(k int) { failed[k] = r.one(tasks[k]) })
	return slices.Contains(failed, true)
}

// one runs t and prints its output once it ends; it reports a failure.
func (r *runner) one(t task) bool {
	args := []string{"test", "-tags", r.tags, "-count=1", "-timeout", r.timeout, "-json"}
	if r.shards > 1 {
		args = append(args, "-run", shard.Pattern(t.tests))
	}
	//nolint:gosec // go test on the packages and tests go list named
	cmd := exec.CommandContext(context.Background(), "go", append(args, t.pkg)...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%s=%d/%d", shard.Env, t.shard, r.shards),
		shard.SplitEnv+"="+strings.Join(r.split, ","))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true
	}
	start := time.Now()
	if err = cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true
	}
	var text bytes.Buffer
	r.read(stdout, &text)
	err = cmd.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Fprintf(os.Stdout, "=== %s, shard %d of %d: %d tests in %s\n", t.pkg, t.shard, r.shards, len(t.tests),
		time.Since(start).Round(time.Second))
	_, _ = os.Stdout.Write(text.Bytes())
	_, _ = os.Stdout.Write(stderr.Bytes())
	return err != nil
}

// event is the part of a go test -json event run reads.
type event struct {
	Action  string  `json:"Action"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

// read reads go test's events from out, keeping in text what go test
// without -v prints, the output of the tests that failed and the
// package's, and how long each test took.
func (r *runner) read(out io.Reader, text *bytes.Buffer) {
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	held := map[string]*bytes.Buffer{}
	for sc.Scan() {
		line := sc.Bytes()
		var e event
		if json.Unmarshal(line, &e) != nil {
			text.Write(line)
			text.WriteByte('\n')
			continue
		}
		switch {
		case e.Test == "":
			text.WriteString(e.Output)
		case e.Action == "output":
			if held[e.Test] == nil {
				held[e.Test] = &bytes.Buffer{}
			}
			held[e.Test].WriteString(e.Output)
		case e.Action == "fail":
			if b := held[e.Test]; b != nil {
				text.Write(b.Bytes())
			}
			delete(held, e.Test)
		case e.Action == "pass" || e.Action == "skip":
			delete(held, e.Test)
		}
		r.mu.Lock()
		if r.events != nil {
			_, _ = r.events.Write(append(line, '\n'))
		}
		if e.Test != "" && (e.Action == "pass" || e.Action == "fail") {
			r.elapsed = append(r.elapsed, timed{e.Test, e.Elapsed})
		}
		r.mu.Unlock()
	}
}

// report lists the slowest tests and subtests.
func (r *runner) report(w io.Writer, slowest int, took time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	slices.SortFunc(r.elapsed, func(a, b timed) int { return cmp.Compare(b.seconds, a.seconds) })
	fmt.Fprintf(w, "=== shards of %d done in %s; the slowest tests:\n", r.shards, took.Round(time.Second))
	for _, t := range r.elapsed[:min(slowest, len(r.elapsed))] {
		fmt.Fprintf(w, "%8.1fs %s\n", t.seconds, t.name)
	}
}
