package reftest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Dirs are the directories of css/ surveyed by default.
//
//nolint:gochecknoglobals // the default list
var Dirs = []string{
	"css/CSS2", "css/css-flexbox", "css/css-grid", "css/css-backgrounds", "css/css-text",
	"css/css-writing-modes", "css/css-tables", "css/css-values", "css/css-color", "css/css-position",
	"css/css-overflow", "css/css-transforms", "css/css-multicol", "css/css-ui", "css/css-fonts",
}

// Runner surveys the reftests of Dirs in the checkout at Root, only
// those the expectations file Only lists when it is set.
type Runner struct {
	Root     string
	Dirs     []string
	Only     string
	Renderer Renderer
}

// Run draws and decides the tests; missing are the directories with
// none in the checkout.
func (r *Runner) Run(ctx context.Context) ([]Result, []string, error) {
	tests, missing, err := Load(r.Root, r.Dirs)
	if err != nil {
		return nil, nil, err
	}
	if r.Only != "" {
		want, readErr := ReadExpectations(r.Only)
		if readErr != nil {
			return nil, nil, readErr
		}
		kept := tests[:0]
		for _, t := range tests {
			if _, ok := want[t.Path]; ok {
				kept = append(kept, t)
			}
		}
		tests = kept
	}
	if err = os.MkdirAll(r.Renderer.Out, 0o750); err != nil {
		return nil, nil, err
	}
	return Survey(ctx, r.Root, tests, &r.Renderer), missing, nil
}

// Commit is the commit the checkout at root is at, "unknown" when its
// HEAD can't be read.
func Commit(root string) string {
	head, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return "unknown"
	}
	ref, isRef := strings.CutPrefix(strings.TrimSpace(string(head)), "ref: ")
	if !isRef {
		return ref
	}
	at, err := os.ReadFile(filepath.Join(root, ".git", filepath.FromSlash(ref))) //nolint:gosec // the checkout's own ref
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(at))
}

// ChromeResults are Chrome's comparisons as the chrome-tagged test
// writes them: the Chrome they were made with, and each test's.
type ChromeResults struct {
	Chrome  int                      `json:"chrome"`
	Results map[string]*ChromeResult `json:"results"`
}

// MergeChrome sets the Chrome comparisons of the file at path on the
// results they are of, and returns the Chrome's version.
func MergeChrome(results []Result, path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var c ChromeResults
	if err = json.Unmarshal(data, &c); err != nil {
		return 0, err
	}
	for i := range results {
		if cr, ok := c.Results[results[i].Path]; ok {
			results[i].Chrome = cr
		}
	}
	return c.Chrome, nil
}
