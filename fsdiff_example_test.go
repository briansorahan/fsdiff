package fsdiff_test

import (
	"encoding/json"
	"log"
	"os"

	"github.com/briansorahan/fsdiff"
)

func ExampleDiffer() {
	// Initialize the differ.
	// This will read the current state of the file system
	// at the specified root directory.
	diff, err := fsdiff.New(fsdiff.Root("testdata"))
	if err != nil {
		log.Fatal(err)
	}
	// Create a file and update the differ.
	f, err := os.Create("testdata/foo")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	diff.Update()

	// Write data to the new file and update the differ.
	if _, err := f.Write([]byte("blah")); err != nil {
		log.Fatal(err)
	}
	diff.Update()

	// Rename a file and update the differ.
	if err := os.Rename("testdata/foo", "testdata/bar"); err != nil {
		log.Fatal(err)
	}
	diff.Update()

	// Remove a file and update the differ.
	if err := os.Remove("testdata/bar"); err != nil {
		log.Fatal(err)
	}
	diff.Update()

	// Poll all the events the differ has seen.
	events, err := diff.Poll()
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(os.Stdout)

	// Encode the events as JSON on stdout.
	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			log.Fatal(err)
		}
		if _, err := f.Write([]byte{0x0A}); err != nil {
			log.Fatal(err)
		}
	}
	// Output:
	// {"oldpath":"","op":"WRITE","path":"testdata"}
	// {"oldpath":"","op":"CREATE","path":"testdata/foo"}
	// {"oldpath":"","op":"WRITE","path":"testdata/foo"}
	// {"oldpath":"","op":"WRITE","path":"testdata"}
	// {"oldpath":"testdata/foo","op":"RENAME","path":"testdata/bar"}
	// {"oldpath":"","op":"WRITE","path":"testdata"}
	// {"oldpath":"","op":"REMOVE","path":"testdata/bar"}
}
