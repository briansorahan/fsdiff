// Package fsdiff returns events that show changes on a given file system
// from one point in time to the next.
package fsdiff

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Differ diffs a file system at two points in time.
type Differ struct {
	err       error // Errors encountered during Update.
	events    []Event
	latest    Snapshot
	recursive bool
	root      string
}

// New creates a new Differ.
func New(options ...Option) (*Differ, error) {
	d := &Differ{}

	for _, opt := range options {
		opt(d)
	}
	if len(d.root) == 0 {
		return nil, errors.New("Root option is required")
	}
	snap, err := NewSnapshot(d.root, d.recursive)
	if err != nil {
		return nil, errors.Wrap(err, "getting initial file system snapshot")
	}
	d.latest = snap

	return d, nil
}

// Latest returns the latest filesystem snapshot.
func (d *Differ) Latest() Snapshot {
	return d.latest
}

// Poll diffs the current state of the file system against the previous state.
func (d *Differ) Poll() ([]Event, error) {
	if d.err != nil {
		return nil, d.err
	}
	println("fsdiff: creating new snapshot")

	curr, err := NewSnapshot(d.root, d.recursive)
	if err != nil {
		return nil, errors.Wrap(err, "getting file system snapshot")
	}
	println("fsdiff: created new snapshot")

	events := append(d.events, Diff(d.latest, curr)...)

	d.events = nil
	d.latest = curr

	return events, nil
}

// Update updates the internal slice that tracks file system events.
// If this method encounters an error, the error will be returned from
// the Poll method when you try to see all the events that have been tracked.
func (d *Differ) Update() {
	// TODO: early return if there has already been an error?
	curr, err := NewSnapshot(d.root, d.recursive)
	if err != nil {
		d.err = err
		return
	}
	d.events = append(d.events, Diff(d.latest, curr)...)
	d.latest = curr
}

// Event defines an event on a file system.
type Event struct {
	Info    os.FileInfo `json:"-"`
	OldPath string      `json:"oldpath"` // Only populated for a rename event.
	Op      Op          `json:"op"`
	Path    string      `json:"path"`
}

// Op defines an operation on a file system.
type Op int

// File system operations.
const (
	Create Op = iota
	Write
	Remove
	Rename
)

// MarshalJSON returns a JSON representation of the Op.
func (o Op) MarshalJSON() ([]byte, error) {
	return []byte(`"` + o.String() + `"`), nil
}

// String converts the Op to a string.
func (o Op) String() string {
	switch o {
	case Create:
		return "CREATE"
	case Write:
		return "WRITE"
	case Remove:
		return "REMOVE"
	case Rename:
		return "RENAME"
	default:
		return "UNKNOWN"
	}
}

// Snapshot indexes file metadata.
// Keys are file paths, values are the metadata associated with the file.
type Snapshot map[string]os.FileInfo

// NewSnapshot creates a new snapshot.
func NewSnapshot(root string, recursive bool) (Snapshot, error) {
	snap := Snapshot{}

	if recursive {
		if err := filepath.Walk(root, snap.Visit); err != nil {
			return nil, errors.Wrap(err, "walking file system")
		}
	} else {
		f, err := os.Open(root)
		if err != nil {
			return nil, errors.Wrap(err, "opening root directory")
		}
		defer f.Close()

		infos, err := f.Readdir(-1)
		if err != nil {
			return nil, errors.Wrap(err, "reading files in directory")
		}
		for _, info := range infos {
			println("fsdiff: visited " + info.Name())
			println("fsdiff: joined with root " + filepath.Join(root, info.Name()))
			snap[filepath.Join(root, info.Name())] = info
		}
	}
	return snap, nil
}

// Visit is a WalkFunc useful via the stdlib path/filepath package.
func (s Snapshot) Visit(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	println("fsdiff: visited " + path)

	s[path] = info

	return nil
}

// Diff 2 file system snapshots.
// x is assumed to be earlier y.
func Diff(x, y Snapshot) []Event {
	var (
		events  []Event
		creates = map[string]os.FileInfo{}
		deletes = map[string]os.FileInfo{}
	)
	for filename, yinfo := range y {
		if xinfo, ok := x[filename]; !ok {
			creates[filename] = yinfo
		} else if yinfo.ModTime().After(xinfo.ModTime()) {
			events = append(events, Event{
				Info: yinfo,
				Op:   Write,
				Path: filename,
			})
		}
	}
	for filename, xinfo := range x {
		if _, ok := y[filename]; !ok {
			deletes[filename] = xinfo
		}
	}
	for dpath, dinfo := range deletes {
		for cpath, cinfo := range creates {
			if os.SameFile(dinfo, cinfo) {
				events = append(events, Event{
					Info:    cinfo,
					OldPath: dpath,
					Op:      Rename,
					Path:    cpath,
				})
				delete(deletes, dpath)
				delete(creates, cpath)
			}
		}
	}
	for filepath, info := range creates {
		events = append(events, Event{
			Info: info,
			Op:   Create,
			Path: filepath,
		})
	}
	for filepath, info := range deletes {
		events = append(events, Event{
			Info: info,
			Op:   Remove,
			Path: filepath,
		})
	}
	return events
}

// Option defines an option on a differ.
type Option func(*Differ)

// Recursive causes a Differ to descend into child directories.
func Recursive() Option {
	return func(d *Differ) {
		d.recursive = true
	}
}

// Root specifies the root of the file system that a differ tracks.
func Root(root string) Option {
	return func(d *Differ) {
		d.root = root
	}
}
