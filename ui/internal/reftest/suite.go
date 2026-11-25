package reftest

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"os"
	"runtime"
	"slices"
	"sort"
	"sync"
)

// Status is a test's outcome.
type Status string

const (
	Pass  Status = "pass"
	Fail  Status = "fail"
	Skip  Status = "skip"
	Error Status = "error"
)

// Result is one test's outcome against its references (ember against
// ember), and, once compared, against Chrome.
type Result struct {
	Path   string `json:"path"`
	Status Status `json:"status"`
	// Reason is why a test was skipped or failed to draw, or the
	// difference of a failing comparison.
	Reason string `json:"reason,omitempty"`
	// Properties are the CSS properties the test uses that its first
	// reference doesn't, or all it uses when that leaves none.
	Properties []string `json:"properties,omitempty"`
	// Chrome is the comparison of ember's picture of the test with
	// Chrome's, when one was made.
	Chrome *ChromeResult `json:"chrome,omitempty"`
}

// ChromeThreshold is the largest mean difference of a channel, 0 to 1,
// between ember's picture of a test and Chrome's for them to match: the
// pixel pages' rule, at a threshold that lets text antialiasing differ.
const ChromeThreshold = 0.002

// ChromeResult is ember's picture of a test against Chrome's, under the
// pixel pages' rule: the mean difference of a channel at most Threshold.
type ChromeResult struct {
	Mean      float64 `json:"mean"`
	Threshold float64 `json:"threshold"`
	Pixels    int     `json:"pixels"`
	OK        bool    `json:"ok"`
}

// Survey draws tests and their references with r and decides each
// test.
func Survey(ctx context.Context, root string, tests []Test, r *Renderer) []Result {
	pages := map[string]*Page{}
	failed := map[string]error{}
	read := func(p string) *Page {
		if page, ok := pages[p]; ok {
			return page
		}
		page, err := ReadPage(root, p)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			page = &Page{Path: p, Skip: "not in the checkout"}
		case err != nil:
			failed[p] = err
		}
		pages[p] = page
		return page
	}
	var draw []string
	var visit func(p string, refs []Ref)
	seen := map[string]bool{}
	visit = func(p string, refs []Ref) {
		if seen[p] {
			return
		}
		seen[p] = true
		page := read(p)
		if page == nil || page.Skip != "" {
			return
		}
		draw = append(draw, p)
		if refs == nil {
			refs = page.Refs
		}
		for _, ref := range refs {
			visit(ref.Path, nil)
		}
	}
	for _, t := range tests {
		if !t.Testdriver {
			visit(t.Path, t.Refs)
		}
	}
	sort.Strings(draw)
	renders := r.Render(ctx, draw)
	results := make([]Result, len(tests))
	next := make(chan int, len(tests))
	for i := range tests {
		next <- i
	}
	close(next)
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Go(func() {
			for i := range next {
				results[i] = decide(tests[i], pages, failed, renders)
			}
		})
	}
	wg.Wait()
	return results
}

// decide is t's outcome from the pages read and drawn.
func decide(t Test, pages map[string]*Page, failed map[string]error, renders map[string]Render) Result {
	res := Result{Path: t.Path}
	if t.Testdriver {
		res.Status, res.Reason = Skip, "testdriver"
		return res
	}
	if why, ok := unusable(t.Path, pages, failed, renders); !ok {
		res.Status, res.Reason = why.status, why.reason
		return res
	}
	if page := pages[t.Path]; page != nil {
		var ref *Page
		if len(t.Refs) > 0 {
			ref = pages[t.Refs[0].Path]
		}
		res.Properties = PropertiesOf(page, ref)
	}
	c := chain{pages: pages, failed: failed, renders: renders, fuzzy: t.Fuzzy, pictures: map[string]image.Image{}}
	ok, why := c.passes(t.Path, t.Refs, 0)
	switch {
	case ok:
		res.Status = Pass
	case why.status != "":
		res.Status, res.Reason = why.status, why.reason
	default:
		res.Status, res.Reason = Fail, why.reason
	}
	return res
}

type outcome struct {
	status Status
	reason string
}

// unusable says why the page at p can't be compared: it is skipped, or
// couldn't be read or drawn.
func unusable(p string, pages map[string]*Page, failed map[string]error, renders map[string]Render) (outcome, bool) {
	if err := failed[p]; err != nil {
		return outcome{Error, err.Error()}, false
	}
	if page := pages[p]; page != nil && page.Skip != "" {
		return outcome{Skip, page.Skip}, false
	}
	rd, ok := renders[p]
	switch {
	case !ok:
		return outcome{Error, "not drawn"}, false
	case rd.Kind == Skipped:
		return outcome{Skip, rd.Detail}, false
	case rd.Kind != Drawn:
		return outcome{Error, rd.Detail}, false
	}
	return outcome{}, true
}

// chain walks a test's references as WPT's runner does: the references
// of one page are alternatives, and a reference with its own continues
// the chain, all of whose links must pass.
type chain struct {
	pages    map[string]*Page
	failed   map[string]error
	renders  map[string]Render
	fuzzy    []Fuzzy
	pictures map[string]image.Image
}

// passes reports whether p relates to one of refs as it says; else the
// outcome of the last tried, with a status when it couldn't be compared.
func (c *chain) passes(p string, refs []Ref, depth int) (bool, outcome) {
	last := outcome{reason: "no reference"}
	if depth > 8 {
		return false, outcome{Error, "references chained too deep"}
	}
	for _, ref := range refs {
		if why, ok := unusable(ref.Path, c.pages, c.failed, c.renders); !ok {
			why.reason = "reference " + ref.Path + ": " + why.reason
			last = why
			continue
		}
		a, err := c.picture(p)
		if err != nil {
			return false, outcome{Error, err.Error()}
		}
		b, err := c.picture(ref.Path)
		if err != nil {
			return false, outcome{Error, err.Error()}
		}
		d := Compare(a, b)
		fz := FuzzyFor(c.fuzzy, ref.Path)
		if fz == nil {
			if page := c.pages[p]; page != nil {
				fz = FuzzyFor(page.Fuzzy, ref.Path)
			}
		}
		if !Passes(ref.Rel, d, fz) {
			last = outcome{reason: fmt.Sprintf("%s %s: %s", ref.Rel, ref.Path, d)}
			continue
		}
		next := c.pages[ref.Path].Refs
		if len(next) == 0 {
			return true, outcome{}
		}
		ok, why := c.passes(ref.Path, next, depth+1)
		if ok {
			return true, outcome{}
		}
		last = why
	}
	return false, last
}

func (c *chain) picture(p string) (image.Image, error) {
	if img, ok := c.pictures[p]; ok {
		return img, nil
	}
	img, err := LoadPNG(c.renders[p].Picture)
	if err != nil {
		return nil, err
	}
	c.pictures[p] = img
	return img, nil
}

// LoadPNG reads the picture at path.
func LoadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// Counts is how many tests of a directory end each way.
type Counts struct {
	Pass, Fail, Skip, Error int
	// ChromeMatch and ChromeDiffer count the tests compared with Chrome.
	ChromeMatch, ChromeDiffer int
}

// Total is every test counted.
func (c Counts) Total() int { return c.Pass + c.Fail + c.Skip + c.Error }

// ByDir counts results by directory, those of dirs first in their order.
func ByDir(results []Result) map[string]*Counts {
	by := map[string]*Counts{}
	for _, r := range results {
		d := Dir(r.Path)
		c := by[d]
		if c == nil {
			c = &Counts{}
			by[d] = c
		}
		switch r.Status {
		case Pass:
			c.Pass++
		case Fail:
			c.Fail++
		case Skip:
			c.Skip++
		case Error:
			c.Error++
		}
		if r.Chrome != nil {
			if r.Chrome.OK {
				c.ChromeMatch++
			} else {
				c.ChromeDiffer++
			}
		}
	}
	return by
}

// Group is a CSS property and the failing tests that use it.
type Group struct {
	Property string
	Tests    []string
}

// Groups ranks the properties of the tests failing as failing says, by
// how many use each, most first.
func Groups(results []Result, failing func(Result) bool) []Group {
	by := map[string][]string{}
	for _, r := range results {
		if !failing(r) {
			continue
		}
		props := r.Properties
		if len(props) == 0 {
			props = []string{"(none)"}
		}
		for _, p := range props {
			by[p] = append(by[p], r.Path)
		}
	}
	groups := make([]Group, 0, len(by))
	for p, tests := range by {
		groups = append(groups, Group{p, tests})
	}
	slices.SortFunc(groups, func(a, b Group) int {
		if len(a.Tests) != len(b.Tests) {
			return len(b.Tests) - len(a.Tests)
		}
		if a.Property < b.Property {
			return -1
		}
		return 1
	})
	return groups
}
