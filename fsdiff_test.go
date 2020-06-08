package fsdiff_test

import (
	"os"
	"testing"

	"github.com/briansorahan/fsdiff"
)

func TestCreate(t *testing.T) {
	differ, err := fsdiff.New(fsdiff.Root("testdata"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create("testdata/foo")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()                 // Best effort.
	defer os.Remove("testdata/foo") // Best effort.

	if _, err := f.Write([]byte("bar")); err != nil {
		t.Fatal(err)
	}
	events, err := differ.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Op != fsdiff.Write {
		t.Fatalf("expected events[0].Op to be Write, got %s", events[0].Op)
	}
	if events[0].Path != "testdata" {
		t.Fatalf("expected events[0].Path to be testdata, got %s", events[0].Path)
	}
	if events[1].Op != fsdiff.Create {
		t.Fatalf("expected events[1].Op to be Create, got %s", events[1].Op)
	}
	if events[1].Path != "testdata/foo" {
		t.Fatalf("expected events[1].Path to be testdata/foo, got %s", events[1].Path)
	}
}
