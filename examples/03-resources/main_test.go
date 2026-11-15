package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestTheRisingWindPutsTheFlamesOut(t *testing.T) {
	want, err := os.ReadFile("output.txt")
	if err != nil {
		t.Fatal(err)
	}
	w := newWorld()
	got := strings.Join(stdoutOf(t, func() { w.RunFrames(tenSeconds) }), "\n") + "\n"
	if !w.ShouldQuit() {
		t.Fatal("the World still runs after ten seconds")
	}
	if got != string(want) {
		t.Fatalf("the World prints\n%s\nwant output.txt\n%s", got, want)
	}
}

// tenSeconds is ten seconds of frames at the fixed step, more than the
// World needs: RunFrames stops when it quits.
const tenSeconds = 600

func stdoutOf(t *testing.T, run func()) []string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	run()
	os.Stdout = stdout
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}
