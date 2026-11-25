package shard_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/shard"
)

func TestEveryTestRunsInExactlyOneShardButTheSplitOnes(t *testing.T) {
	var names []string
	for k := range 400 {
		names = append(names, fmt.Sprintf("TestCase%d", k))
	}
	split := []string{"TestCase7"}
	const n = 4
	seen := map[string]int{}
	for i := range n {
		picked := shard.Pick(names, i, n, split)
		assert.Contains(t, picked, "TestCase7", "a split test runs in every shard")
		assert.Greater(t, len(picked), 400/n/2, "shard %d has its share", i)
		assert.Less(t, len(picked), 400/n*2, "shard %d has its share", i)
		for _, name := range picked {
			seen[name]++
		}
	}
	for _, name := range names {
		want := 1
		if name == "TestCase7" {
			want = n
		}
		assert.Equal(t, want, seen[name], name)
	}
}

func TestOneShardRunsEverything(t *testing.T) {
	names := []string{"TestA", "TestB", "ExampleC"}
	assert.Equal(t, names, shard.Pick(names, 0, 1, nil))
}

func TestThePatternRunsExactlyTheNames(t *testing.T) {
	re := regexp.MustCompile(shard.Pattern([]string{"TestA", "TestA2", "TestB.c"}))
	for name, want := range map[string]bool{
		"TestA": true, "TestA2": true, "TestB.c": true, "TestAB": false, "TestBxc": false, "XTestA": false,
	} {
		assert.Equal(t, want, re.MatchString(name), name)
	}
}

func TestParse(t *testing.T) {
	i, n, ok := shard.Parse("2/4")
	assert.True(t, ok)
	assert.Equal(t, [2]int{2, 4}, [2]int{i, n})
	for _, bad := range []string{"", "4/4", "-1/4", "1/0", "a/b", "3"} {
		_, _, ok = shard.Parse(bad)
		assert.False(t, ok, bad)
	}
	assert.Equal(t, []string{"TestA", "TestB"}, shard.Split(" TestA, ,TestB"))
	assert.Empty(t, shard.Split(""))
}

func TestASplitTestsSubtestsSpreadOverTheShards(t *testing.T) {
	split := []string{"TestPages"}
	const n = 3
	ran := map[string]int{}
	for i := range n {
		for k := range 60 {
			name := fmt.Sprintf("TestPages/page-%d", k)
			why, err := shard.Skip(name, i, n, split)
			require.NoError(t, err)
			if why == "" {
				ran[name]++
			}
		}
	}
	assert.Len(t, ran, 60, "every subtest ran")
	for name, times := range ran {
		assert.Equal(t, 1, times, name)
	}
	_, err := shard.Skip("TestOther/page", 0, n, split)
	assert.Error(t, err, "a test splitting itself unnamed would lose subtests")
}
