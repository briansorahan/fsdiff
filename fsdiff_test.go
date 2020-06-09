package fsdiff_test

import (
	"os"
	"testing"

	"github.com/briansorahan/fsdiff"
)

func TestNew(t *testing.T) {
	t.Run("missing Root option", func(t *testing.T) {
		_, err := fsdiff.New(fsdiff.Recursive())
		if err == nil {
			t.Fatal("expected an error")
		}
		if got, expect := err.Error(), "Root option is required"; got != expect {
			t.Fatalf("expected %s, got %s", expect, got)
		}
	})
	t.Run("Root directory does not exist", func(t *testing.T) {
		_, err := fsdiff.New(
			fsdiff.Root("nonexistent"),
			fsdiff.Recursive(),
		)
		if err == nil {
			t.Fatal("expected an error")
		}
		if got, expect := err.Error(), "getting initial file system snapshot: walking file system: lstat nonexistent: no such file or directory"; got != expect {
			t.Fatalf("expected %s, got %s", expect, got)
		}
	})
}

func TestCreate(t *testing.T) {
	differ, err := fsdiff.New(
		fsdiff.Root("testdata"),
		fsdiff.Recursive(),
	)
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
		t.Fatalf("expected events[0].Op to be Write, got %s", events[0].Op.String())
	}
	if events[0].Path != "testdata" {
		t.Fatalf("expected events[0].Path to be testdata, got %s", events[0].Path)
	}
	if events[1].Op != fsdiff.Create {
		t.Fatalf("expected events[1].Op to be Create, got %s", events[1].Op.String())
	}
	if events[1].Path != "testdata/foo" {
		t.Fatalf("expected events[1].Path to be testdata/foo, got %s", events[1].Path)
	}
}

func TestDelete(t *testing.T) {
	f, err := os.Create("testdata/foo")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()                 // Best effort.
	defer os.Remove("testdata/foo") // Best effort.

	if _, err := f.Write([]byte("bar")); err != nil {
		t.Fatal(err)
	}
	differ, err := fsdiff.New(
		fsdiff.Root("testdata"),
		fsdiff.Recursive(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove("testdata/foo"); err != nil {
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
		t.Fatalf("expected events[0].Op to be Write, got %s", events[0].Op.String())
	}
	if events[0].Path != "testdata" {
		t.Fatalf("expected events[0].Path to be testdata, got %s", events[0].Path)
	}
	if events[1].Op != fsdiff.Remove {
		t.Fatalf("expected events[1].Op to be Remove, got %s", events[1].Op.String())
	}
	if events[1].Path != "testdata/foo" {
		t.Fatalf("expected events[1].Path to be testdata/foo, got %s", events[1].Path)
	}
}

func TestPollError(t *testing.T) {
	if err := os.Mkdir("testdata/temp", 0o755); err != nil {
		t.Fatal(err)
	}
	// TODO
	differ, err := fsdiff.New(
		fsdiff.Root("testdata/temp"),
		fsdiff.Recursive(),
	)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create("testdata/temp/foo")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() // Best effort.

	if _, err := f.Write([]byte("bar")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll("testdata/temp"); err != nil {
		t.Fatal(err)
	}
	events, err := differ.Poll()
	if err == nil {
		t.Fatal("expected an error")
	}
	if expect, got := "getting file system snapshot: walking file system: lstat testdata/temp: no such file or directory", err.Error(); expect != got {
		t.Fatalf("expected %s, got %s", expect, got)
	}
	if len(events) > 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestRename(t *testing.T) {
	f, err := os.Create("testdata/foo")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()                 // Best effort.
	defer os.Remove("testdata/foo") // Best effort.
	defer os.Remove("testdata/bar") // Best effort.

	if _, err := f.Write([]byte("bar")); err != nil {
		t.Fatal(err)
	}
	differ, err := fsdiff.New(
		fsdiff.Root("testdata"),
		fsdiff.Recursive(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename("testdata/foo", "testdata/bar"); err != nil {
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
		t.Fatalf("expected events[0].Op to be Write, got %s", events[0].Op.String())
	}
	if events[0].Path != "testdata" {
		t.Fatalf("expected events[0].Path to be testdata, got %s", events[0].Path)
	}
	if events[1].Op != fsdiff.Rename {
		t.Fatalf("expected events[1].Op to be Rename, got %s", events[1].Op.String())
	}
	if events[1].Path != "testdata/bar" {
		t.Fatalf("expected events[1].Path to be testdata/bar, got %s", events[1].Path)
	}
	if events[1].OldPath != "testdata/foo" {
		t.Fatalf("expected events[1].OldPath to be testdata/foo, got %s", events[1].Path)
	}
}

func TestOpString(t *testing.T) {
	for i, testcase := range []struct {
		Expect string
		Op     fsdiff.Op
	}{
		{
			Expect: "CREATE",
			Op:     fsdiff.Create,
		},
		{
			Expect: "REMOVE",
			Op:     fsdiff.Remove,
		},
		{
			Expect: "RENAME",
			Op:     fsdiff.Rename,
		},
		{
			Expect: "UNKNOWN",
			Op:     1e6,
		},
		{
			Expect: "WRITE",
			Op:     fsdiff.Write,
		},
	} {
		if exp, got := testcase.Expect, testcase.Op.String(); exp != got {
			t.Fatalf("testcase %d: expected %s, got %s", i, exp, got)
		}
	}
}

func TestUpdateError(t *testing.T) {
	if err := os.Mkdir("testdata/temp", 0o755); err != nil {
		t.Fatal(err)
	}
	differ, err := fsdiff.New(
		fsdiff.Root("testdata/temp"),
		fsdiff.Recursive(),
	)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create("testdata/temp/foo")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() // Best effort.

	differ.Update()

	if _, err := f.Write([]byte("bar")); err != nil {
		t.Fatal(err)
	}
	differ.Update()

	if err := os.RemoveAll("testdata/temp"); err != nil {
		t.Fatal(err)
	}
	differ.Update()

	events, err := differ.Poll()
	if err == nil {
		t.Fatal("expected an error")
	}
	if expect, got := "walking file system: lstat testdata/temp: no such file or directory", err.Error(); expect != got {
		t.Fatalf("expected %s, got %s", expect, got)
	}
	if len(events) > 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}
