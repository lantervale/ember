package reftest

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
)

// Expectation is what a test is held to: matching its references, and
// matching Chrome.
type Expectation struct {
	Ref, Chrome bool
}

// The words of an expectations line after its path.
const (
	expectRef    = "ref"
	expectChrome = "chrome"
)

// ReadExpectations reads an expectations file: one test path a line,
// then "ref" when it matches its references and "chrome" when it matches
// Chrome; # starts a comment.
func ReadExpectations(path string) (map[string]Expectation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]Expectation{}
	s := bufio.NewScanner(f)
	for n := 1; s.Scan(); n++ {
		line, _, _ := strings.Cut(s.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		var e Expectation
		for _, w := range fields[1:] {
			switch w {
			case expectRef:
				e.Ref = true
			case expectChrome:
				e.Chrome = true
			default:
				return nil, fmt.Errorf("%s:%d: %q is neither ref nor chrome", path, n, w)
			}
		}
		out[fields[0]] = e
	}
	return out, s.Err()
}

// WriteExpectations writes the tests of results that pass either way,
// after header, a comment.
func WriteExpectations(w io.Writer, header string, results []Result) error {
	for _, line := range strings.Split(strings.TrimSpace(header), "\n") {
		if _, err := fmt.Fprintln(w, "# "+line); err != nil {
			return err
		}
	}
	for _, r := range results {
		var words []string
		if r.Status == Pass {
			words = append(words, expectRef)
		}
		if r.Chrome != nil && r.Chrome.OK {
			words = append(words, expectChrome)
		}
		if len(words) == 0 {
			continue
		}
		if _, err := fmt.Fprintln(w, r.Path, strings.Join(words, " ")); err != nil {
			return err
		}
	}
	return nil
}

// Summary is what a survey's summary says besides its results.
type Summary struct {
	Checkout, Commit string
	Missing          []string
	Took             string
	ChromeVersion    int
	// Top is how many property groups each ranking lists.
	Top int
}

// WriteSummary writes a Markdown summary of results.
func WriteSummary(w io.Writer, s Summary, results []Result) {
	p := func(format string, args ...any) { fmt.Fprintf(w, format, args...) }
	p("# WPT reftests on ember's ui\n\n")
	p("Checkout %s at %s; every test and reference drawn by ember at %d×%d, 1 pixel a point; took %s.\n\n",
		s.Checkout, s.Commit, Width, Height, s.Took)
	chrome := slices.ContainsFunc(results, func(r Result) bool { return r.Chrome != nil })
	if chrome {
		p("Compared with Chrome %d too (mean difference of a channel at most %.4f, the pixel pages' rule). "+
			"Where a reference and Chrome disagree, Chrome is right.\n\n", s.ChromeVersion, ChromeThreshold)
	}
	if len(s.Missing) > 0 {
		p("Not in the checkout (sparse): %s.\n\n", strings.Join(s.Missing, ", "))
	}
	table(w, results, chrome)
	top := max(s.Top, 1)
	if chrome {
		p("\n## Differing from Chrome, by property (top %d)\n\n", top)
		groups(w, Groups(results, func(r Result) bool { return r.Chrome != nil && !r.Chrome.OK }), top)
	}
	p("\n## Failing against their references, by property (top %d)\n\n", top)
	groups(w, Groups(results, func(r Result) bool { return r.Status == Fail }), top)
	if chrome {
		disagreements(w, results)
	}
	p("\n## Skipped, by reason\n\n")
	reasons(w, results, Skip)
	p("\n## Errors, by reason\n\n")
	reasons(w, results, Error)
}

// table writes the counts of results by directory, and of them all.
func table(w io.Writer, results []Result, chrome bool) {
	p := func(format string, args ...any) { fmt.Fprintf(w, format, args...) }
	by := ByDir(results)
	dirs := make([]string, 0, len(by))
	for d := range by {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	p("## By directory\n\n| directory | tests | pass | fail | skip | error | pass rate (drawn) |")
	if chrome {
		p(" matches Chrome | differs from Chrome |")
	}
	p("\n|---|---:|---:|---:|---:|---:|---:|")
	if chrome {
		p("---:|---:|")
	}
	p("\n")
	var total Counts
	row := func(name string, c Counts) {
		drawn := c.Pass + c.Fail
		rate := 0.0
		if drawn > 0 {
			rate = 100 * float64(c.Pass) / float64(drawn)
		}
		p("| %s | %d | %d | %d | %d | %d | %.1f%% |", name, c.Total(), c.Pass, c.Fail, c.Skip, c.Error, rate)
		if chrome {
			p(" %d | %d |", c.ChromeMatch, c.ChromeDiffer)
		}
		p("\n")
	}
	for _, d := range dirs {
		c := *by[d]
		row(d, c)
		total.Pass += c.Pass
		total.Fail += c.Fail
		total.Skip += c.Skip
		total.Error += c.Error
		total.ChromeMatch += c.ChromeMatch
		total.ChromeDiffer += c.ChromeDiffer
	}
	row("**all**", total)
}

func groups(w io.Writer, gs []Group, top int) {
	fmt.Fprintf(w, "| # | property | tests | examples |\n|---:|---|---:|---|\n")
	for i, g := range gs[:min(top, len(gs))] {
		ex := g.Tests[:min(3, len(g.Tests))]
		fmt.Fprintf(w, "| %d | `%s` | %d | %s |\n", i+1, g.Property, len(g.Tests), strings.Join(ex, ", "))
	}
}

// disagreements lists the tests whose reference and Chrome say
// otherwise: those that match their reference but not Chrome (ember
// draws both wrong alike), and those that match Chrome but not their
// reference (the reference is drawn wrong, or its comparison is too
// strict).
func disagreements(w io.Writer, results []Result) {
	var refOnly, chromeOnly []string
	for _, r := range results {
		if r.Chrome == nil {
			continue
		}
		switch {
		case r.Status == Pass && !r.Chrome.OK:
			refOnly = append(refOnly, fmt.Sprintf("%s (mean %.4f)", r.Path, r.Chrome.Mean))
		case r.Status == Fail && r.Chrome.OK:
			chromeOnly = append(chromeOnly, r.Path+" ("+r.Reason+")")
		}
	}
	fmt.Fprintf(w, "\n## Where the reference and Chrome disagree\n\n")
	fmt.Fprintf(w, "Matching the reference but not Chrome (%d; ember draws the test and its reference wrong alike):\n\n",
		len(refOnly))
	list(w, refOnly, 50)
	fmt.Fprintf(w, "\nMatching Chrome but not the reference (%d; the reference is drawn wrong, or the test differs "+
		"by less than Chrome's threshold lets through on a page of 800×600, a few hundred pixels):\n\n",
		len(chromeOnly))
	list(w, chromeOnly, 50)
}

func list(w io.Writer, items []string, most int) {
	for _, it := range items[:min(most, len(items))] {
		fmt.Fprintf(w, "- %s\n", it)
	}
	if len(items) > most {
		fmt.Fprintf(w, "- … %d more\n", len(items)-most)
	}
}

func reasons(w io.Writer, results []Result, status Status) {
	count := map[string]int{}
	for _, r := range results {
		if r.Status == status {
			why := r.Reason
			if i := strings.Index(why, ": "); i > 0 && strings.HasPrefix(why, "reference ") {
				why = "reference: " + why[i+2:]
			}
			if len(why) > 120 {
				why = why[:120] + "…"
			}
			count[why]++
		}
	}
	type kv struct {
		k string
		v int
	}
	var all []kv
	for k, v := range count {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v || all[i].v == all[j].v && all[i].k < all[j].k })
	fmt.Fprintf(w, "| reason | tests |\n|---|---:|\n")
	for _, e := range all[:min(30, len(all))] {
		fmt.Fprintf(w, "| %s | %d |\n", strings.ReplaceAll(e.k, "|", "\\|"), e.v)
	}
}
