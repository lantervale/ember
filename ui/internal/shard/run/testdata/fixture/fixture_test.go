// Package fixture is the shard runner's test package: each test logs its
// name and the shard it ran in to FIXTURE_LOG, TestTable's subtests
// spread over the shards, and TestFails fails when FIXTURE_FAIL is set.
package fixture

import (
	"fmt"
	"os"
	"testing"

	"github.com/lantervale/ember/ui/internal/shard"
)

func ran(t *testing.T) {
	f, err := os.OpenFile(os.Getenv("FIXTURE_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", t.Name(), os.Getenv(shard.Env))
}

func TestA(t *testing.T) { ran(t) }
func TestB(t *testing.T) { ran(t) }
func TestC(t *testing.T) { ran(t) }
func TestD(t *testing.T) { ran(t) }
func TestE(t *testing.T) { ran(t) }
func TestF(t *testing.T) { ran(t) }
func TestG(t *testing.T) { ran(t) }
func TestH(t *testing.T) { ran(t) }

func TestTable(t *testing.T) {
	for k := range 12 {
		t.Run(fmt.Sprint("row", k), func(t *testing.T) {
			if i, n, ok := shard.Parse(os.Getenv(shard.Env)); ok {
				why, err := shard.Skip(t.Name(), i, n, shard.Split(os.Getenv(shard.SplitEnv)))
				if err != nil {
					t.Fatal(err)
				}
				if why != "" {
					t.Skip(why)
				}
			}
			ran(t)
		})
	}
}

func TestFails(t *testing.T) {
	ran(t)
	if os.Getenv("FIXTURE_FAIL") != "" {
		t.Log("the fixture's failure")
		t.Fail()
	}
}
