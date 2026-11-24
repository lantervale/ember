// Package shard splits the ui Chrome tests into shards that run at once:
// CI's jobs, or the Chrome slots of one machine. A test falls in the shard
// its name hashes to; a test split by its subtests (Split) runs in every
// shard, each of its subtests where its own name hashes to, so one long
// table of pages spreads as the rest does. ui/internal/shard/run runs
// them.
package shard

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Env says, in a test process, which shard of how many it runs: "i/n",
// i from 0.
const Env = "EMBER_CHROME_SHARD"

// SplitEnv names, comma-separated, the tests split by their subtests.
const SplitEnv = "EMBER_CHROME_SPLIT"

// Of is the shard of n that name falls in.
func Of(name string, n int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return int(h.Sum32() % uint32(n)) //nolint:gosec // n is a small positive count
}

// Parse reads Env's "i/n", and whether it is one: i from 0 and under n.
func Parse(s string) (int, int, bool) {
	a, b, found := strings.Cut(s, "/")
	if !found {
		return 0, 0, false
	}
	i, errI := strconv.Atoi(a)
	n, errN := strconv.Atoi(b)
	if errI != nil || errN != nil || n < 1 || i < 0 || i >= n {
		return 0, 0, false
	}
	return i, n, true
}

// Split is SplitEnv's list.
func Split(env string) []string {
	var names []string
	for name := range strings.SplitSeq(env, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// Pick is the tests of names shard i of n runs: those that hash to it, and
// every split one.
func Pick(names []string, i, n int, split []string) []string {
	var picked []string
	for _, name := range names {
		if slices.Contains(split, name) || Of(name, n) == i {
			picked = append(picked, name)
		}
	}
	return picked
}

// Pattern is the -run pattern of exactly names.
func Pattern(names []string) string {
	quoted := make([]string, len(names))
	for k, name := range names {
		quoted[k] = regexp.QuoteMeta(name)
	}
	return "^(" + strings.Join(quoted, "|") + ")$"
}

// Skip says why the subtest name doesn't run in shard i of n, empty when
// it does: one of a test split by its subtests runs where its own name
// hashes to. A test splitting its subtests that split doesn't name is an
// error, as its subtests of the other shards would run nowhere.
func Skip(name string, i, n int, split []string) (string, error) {
	top, _, _ := strings.Cut(name, "/")
	if !slices.Contains(split, top) {
		return "", fmt.Errorf("shard: %s splits its subtests across shards, but %s doesn't name it: "+
			"its other shards' subtests would run nowhere", top, SplitEnv)
	}
	if s := Of(name, n); s != i {
		return fmt.Sprintf("in shard %d of %d", s, n), nil
	}
	return "", nil
}
