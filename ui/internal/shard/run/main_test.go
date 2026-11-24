package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixture = "./testdata/fixture"

// shardRun runs the runner, built for t, on the fixture with args; it
// returns what each test logged, its output and whether it failed.
func shardRun(t *testing.T, fail bool, args ...string) (map[string][]string, string, bool) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "run")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "%s", out)
	log := filepath.Join(t.TempDir(), "log")
	cmd := exec.Command(bin, append(args, fixture)...)
	cmd.Env = append(os.Environ(), "FIXTURE_LOG="+log, "EMBER_CHROME_SHARD=", "EMBER_CHROME_SPLIT=")
	if fail {
		cmd.Env = append(cmd.Env, "FIXTURE_FAIL=1")
	}
	out, err = cmd.CombinedOutput()
	logged, _ := os.ReadFile(log)
	where := map[string][]string{}
	for line := range strings.Lines(string(logged)) {
		name, shard, _ := strings.Cut(strings.TrimSpace(line), " ")
		where[name] = append(where[name], shard)
	}
	return where, string(out), err != nil
}

func TestEveryShardTogetherRunsEachTestOnce(t *testing.T) {
	where, out, failed := shardRun(t, false, "-shards", "3", "-split", "TestTable")
	require.False(t, failed, out)
	shards := map[string]bool{}
	for _, name := range []string{"TestA", "TestB", "TestC", "TestD", "TestE", "TestF", "TestG", "TestH", "TestFails"} {
		require.Len(t, where[name], 1, "%s: %v", name, where)
		shards[where[name][0]] = true
	}
	assert.Len(t, shards, 3, "the tests spread over the three shards: %v", where)
	rows := 0
	for name, in := range where {
		if strings.HasPrefix(name, "TestTable/") {
			assert.Len(t, in, 1, name)
			rows++
		}
	}
	assert.Equal(t, 12, rows, "each of the split test's subtests ran once")
	assert.Contains(t, out, "the slowest tests")
}

func TestOneShardRunsItsTestsAlone(t *testing.T) {
	all, _, _ := shardRun(t, false, "-shards", "3", "-split", "TestTable")
	one, out, failed := shardRun(t, false, "-shards", "3", "-only", "1", "-split", "TestTable")
	require.False(t, failed, out)
	for name, in := range all {
		if in[0] == "1/3" {
			assert.Equal(t, []string{"1/3"}, one[name], name)
		} else {
			assert.NotContains(t, one, name)
		}
	}
	assert.NotEmpty(t, one)
}

func TestAFailingTestFailsTheRunAndShowsItsOutput(t *testing.T) {
	_, out, failed := shardRun(t, true, "-shards", "2", "-split", "TestTable")
	assert.True(t, failed)
	assert.Contains(t, out, "the fixture's failure")
	assert.Contains(t, out, "--- FAIL: TestFails")
	assert.NotContains(t, out, "--- PASS", "passing tests stay quiet, as without -v")
}

func TestATestSplittingItselfUnnamedFails(t *testing.T) {
	_, out, failed := shardRun(t, false, "-shards", "2")
	assert.True(t, failed, "TestTable splits its rows but -split doesn't name it")
	assert.Contains(t, out, "would run nowhere")
}
