// Package reftest runs web-platform-tests' CSS reftests on ember's ui: a
// test page and its references, each read as a Document with the
// browser's sheets and the WPT fonts, drawn by ember at WPT's 800×600 and
// compared under the test's fuzzy rules, so both sides of a comparison
// are ember's own pictures (decision 0229). The tests come from a WPT
// checkout's MANIFEST.json; what ui cannot load (scripts, print, frames…)
// is skipped with its reason. Command run surveys directories and writes
// a summary; the gpu-tagged test holds the tests in testdata/expected.txt
// that pass, and, under the chrome tag, those that match Chrome.
package reftest

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Rel is how a picture relates to its reference: "==" for a match,
// "!=" for a mismatch.
type Rel string

const (
	Match    Rel = "=="
	Mismatch Rel = "!="
)

// Ref is a reference of a page: its path from the checkout's root
// (css/…, no leading slash) and how the page relates to it.
type Ref struct {
	Path string
	Rel  Rel
}

// Range is an inclusive range of a fuzzy rule.
type Range struct{ Min, Max int }

// Fuzzy is how far a picture may be from its reference and still match:
// the largest difference of a channel, and how many pixels differ. Ref
// names the reference it is for, "" for any.
type Fuzzy struct {
	Ref           string
	MaxDifference Range
	TotalPixels   Range
}

// Test is one reftest of the manifest.
type Test struct {
	// Path is the test's path from the checkout's root.
	Path  string
	Refs  []Ref
	Fuzzy []Fuzzy
	// Testdriver is set for a test that drives the page with
	// testdriver.js.
	Testdriver bool
}

// Dir is the test's directory under css/: css/CSS2 for
// css/CSS2/borders/x.xht.
func (t Test) Dir() string { return Dir(t.Path) }

// Dir is the directory under css/ of path, its first two parts.
func Dir(p string) string {
	parts := strings.SplitN(p, "/", 3)
	if len(parts) < 3 {
		return path.Dir(p)
	}
	return parts[0] + "/" + parts[1]
}

// manifest is the part of MANIFEST.json (version 9) read here.
type manifest struct {
	Version int `json:"version"`
	Items   struct {
		Reftest map[string]json.RawMessage `json:"reftest"`
	} `json:"items"`
}

// Load reads the reftests under dirs (css/CSS2…) from the manifest of the
// checkout at root, sorted by path. A directory the manifest has no test
// in is reported in missing.
func Load(root string, dirs []string) ([]Test, []string, error) {
	data, err := os.ReadFile(filepath.Join(root, "MANIFEST.json"))
	if err != nil {
		return nil, nil, err
	}
	var m manifest
	if err = json.Unmarshal(data, &m); err != nil {
		return nil, nil, fmt.Errorf("MANIFEST.json: %w", err)
	}
	if m.Version != 9 {
		return nil, nil, fmt.Errorf("MANIFEST.json is version %d, the runner reads version 9", m.Version)
	}
	all := map[string]Test{}
	for name, raw := range m.Items.Reftest {
		if err = walk(name, raw, all); err != nil {
			return nil, nil, err
		}
	}
	var tests []Test
	var missing []string
	for _, dir := range dirs {
		dir = strings.Trim(dir, "/")
		n := 0
		for p, t := range all {
			if strings.HasPrefix(p, dir+"/") {
				tests = append(tests, t)
				n++
			}
		}
		if n == 0 {
			missing = append(missing, dir)
		}
	}
	sort.Slice(tests, func(i, j int) bool { return tests[i].Path < tests[j].Path })
	return tests, missing, nil
}

// walk reads the manifest's tree at p: a directory is an object, a file
// an array of its hash then one entry per test it holds.
func walk(p string, raw json.RawMessage, into map[string]Test) error {
	if len(raw) > 0 && raw[0] == '{' {
		var dir map[string]json.RawMessage
		if err := json.Unmarshal(raw, &dir); err != nil {
			return err
		}
		for name, child := range dir {
			if err := walk(p+"/"+name, child, into); err != nil {
				return err
			}
		}
		return nil
	}
	var file []json.RawMessage
	if err := json.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	for _, entry := range file[1:] {
		t, err := entryTest(p, entry)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		into[t.Path] = t
	}
	return nil
}

// entryTest reads one entry: [url or null, [[ref, rel]…], {extras}].
func entryTest(p string, raw json.RawMessage) (Test, error) {
	var entry []json.RawMessage
	if err := json.Unmarshal(raw, &entry); err != nil || len(entry) < 3 {
		return Test{}, fmt.Errorf("an entry of %d fields", len(entry))
	}
	t := Test{Path: p}
	var url *string
	if err := json.Unmarshal(entry[0], &url); err != nil {
		return t, err
	}
	if url != nil {
		t.Path = strings.TrimPrefix(*url, "/")
	}
	var refs [][2]string
	if err := json.Unmarshal(entry[1], &refs); err != nil {
		return t, err
	}
	for _, r := range refs {
		t.Refs = append(t.Refs, Ref{Path: strings.TrimPrefix(r[0], "/"), Rel: Rel(r[1])})
	}
	var extras struct {
		Fuzzy      [][2]json.RawMessage `json:"fuzzy"`
		Testdriver bool                 `json:"testdriver"`
	}
	if err := json.Unmarshal(entry[2], &extras); err != nil {
		return t, err
	}
	t.Testdriver = extras.Testdriver
	for _, f := range extras.Fuzzy {
		var key []string
		_ = json.Unmarshal(f[0], &key)
		var ranges [2][2]int
		if err := json.Unmarshal(f[1], &ranges); err != nil {
			return t, err
		}
		fz := Fuzzy{
			MaxDifference: Range{ranges[0][0], ranges[0][1]},
			TotalPixels:   Range{ranges[1][0], ranges[1][1]},
		}
		if len(key) >= 2 {
			fz.Ref = strings.TrimPrefix(key[1], "/")
		}
		t.Fuzzy = append(t.Fuzzy, fz)
	}
	return t, nil
}

// FuzzyFor is the rule of fuzzy for the reference ref, nil for none.
func FuzzyFor(fuzzy []Fuzzy, ref string) *Fuzzy {
	for i := range fuzzy {
		if fuzzy[i].Ref == ref {
			return &fuzzy[i]
		}
	}
	for i := range fuzzy {
		if fuzzy[i].Ref == "" {
			return &fuzzy[i]
		}
	}
	return nil
}
