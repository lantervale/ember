// Command run surveys WPT's CSS reftests on ember's ui (package reftest):
// every test and reference of the directories given is drawn by ember in
// worker processes, compared under the test's fuzzy rules, and summed up
// in summary.md and results.json in -out, each test's pictures beside
// them. A results.json with Chrome's comparisons (the chrome-tagged test
// writes one) is merged with -chrome.
//
//	go run ./ui/internal/reftest/run -out /tmp/reftests   # light's checkout beside ember, or -wpt dir
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lantervale/ember/ui/internal/reftest"
)

// lightWPT is light's checkout, beside ember, from ember's root.
const lightWPT = "../light/.wpt/wpt"

func main() {
	wpt := flag.String("wpt", os.Getenv("EMBER_WPT"), "the WPT checkout (EMBER_WPT)")
	dirs := flag.String("dirs", strings.Join(reftest.Dirs, ","), "the directories of the checkout to run, by commas")
	out := flag.String("out", filepath.Join(os.TempDir(), "ember-reftests"), "where the pictures and the summary go")
	workers := flag.Int("j", 4, "worker processes at once")
	timeout := flag.Duration("timeout", 30*time.Second, "the longest a page may take to draw")
	only := flag.String("only", "", "an expectations file: run only the tests it lists")
	chrome := flag.String("chrome", "", "a results.json with Chrome's comparisons to merge")
	expect := flag.String("expectations", "", "write the expectations of the tests that pass to this file")
	worker := flag.Bool("worker", false, "draw the pages read on the standard input (the runner's own use)")
	flag.Parse()
	if *wpt == "" {
		if _, err := os.Stat(lightWPT); err != nil {
			log.Fatal("no WPT checkout: give -wpt or EMBER_WPT, or run task wpt:setup in light beside ember")
		}
		*wpt = lightWPT
	}
	if *worker {
		work(*wpt, os.Stdin, os.Stdout)
		return
	}
	self, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	start := time.Now()
	r := &reftest.Runner{
		Root: *wpt, Dirs: strings.Split(*dirs, ","), Only: *only,
		Renderer: reftest.Renderer{
			Binary: self, Args: []string{"-worker", "-wpt", *wpt}, Root: *wpt, Out: *out,
			Workers: *workers, Timeout: *timeout,
			Progress: func(done, total int) {
				if done%200 == 0 || done == total {
					log.Printf("drawn %d of %d pages", done, total)
				}
			},
		},
	}
	results, missing, err := r.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	s := reftest.Summary{Checkout: *wpt, Commit: reftest.Commit(*wpt), Missing: missing,
		Took: time.Since(start).Round(time.Second).String(), Top: 20}
	if *chrome != "" {
		if s.ChromeVersion, err = reftest.MergeChrome(results, *chrome); err != nil {
			log.Fatal(err)
		}
	}
	if err = write(*out, s, results); err != nil {
		log.Fatal(err)
	}
	if *expect != "" {
		f, createErr := os.Create(*expect)
		if createErr != nil {
			log.Fatal(createErr)
		}
		header := fmt.Sprintf("The WPT reftests ember's ui passes, at WPT %s (decision 0229).\n"+
			"A path, then ref when it matches its references (task reftest), chrome when it matches Chrome %d\n"+
			"(task reftest:chrome). Rewritten by go run ./ui/internal/reftest/run -expectations <this file>\n"+
			"-chrome <the EMBER_REFTEST_CHROME file of a survey of TestReftestsMatchChrome>.", s.Commit, s.ChromeVersion)
		err = reftest.WriteExpectations(f, header, results)
		f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("%d tests in %s; summary in %s", len(results), s.Took, filepath.Join(*out, "summary.md"))
}

func write(out string, s reftest.Summary, results []reftest.Result) error {
	f, err := os.Create(filepath.Join(out, "summary.md"))
	if err != nil {
		return err
	}
	reftest.WriteSummary(f, s, results)
	if err = f.Close(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(results, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "results.json"), data, 0o600)
}
