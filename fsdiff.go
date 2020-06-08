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
	latest snapshot
	root   string
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
	snap, err := newSnapshot(d.root)
	if err != nil {
		return nil, errors.Wrap(err, "getting initial file system snapshot")
	}
	d.latest = snap

	return d, nil
}

// Poll diffs the current state of the file system against the previous state.
func (d *Differ) Poll() ([]Event, error) {
	curr, err := newSnapshot(d.root)
	if err != nil {
		return nil, errors.Wrap(err, "getting file system snapshot")
	}
	events := diff(d.latest, curr)

	d.latest = curr

	return events, nil
}

// Event defines an event on a file system.
type Event struct {
	Info    os.FileInfo
	OldPath string // Only populated for a rename event.
	Op      Op
	Path    string
}

// Op defines an operation on a file.
type Op int

// Operations
const (
	Create Op = iota
	Write
	Remove
	Rename
)

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

type snapshot map[string]os.FileInfo

func newSnapshot(root string) (snapshot, error) {
	snap := snapshot{}

	if err := filepath.Walk(root, snap.visit); err != nil {
		return nil, errors.Wrap(err, "walking file system")
	}
	return snap, nil
}

func (s snapshot) visit(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	s[path] = info

	return nil
}

// diff 2 file system snapshots.
// x is assumed to be earlier y.
func diff(x, y snapshot) []Event {
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
					Info: cinfo,
					Op:   Rename,
					Path: cpath,
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

// Root specifies the root of the file system that a differ tracks.
func Root(root string) Option {
	return func(d *Differ) {
		d.root = root
	}
}
