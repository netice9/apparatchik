package util

//go:generate gospecific -pkg=github.com/netice9/notifier-go -specific-type=[]string -out-dir=.
//go:generate mv notifier.go output_tracker_notifier.go
//go:generate sed -i "s/^package notifier/package util/" output_tracker_notifier.go
//go:generate sed -i "s/Notifier/OutputTrackerNotifier/g" output_tracker_notifier.go

import (
	"bytes"
	"io"
	"sync"
)

type OutputTracker struct {
	sync.Mutex
	MaxLines    int
	Lines       []string
	CurrentLine *bytes.Buffer
	*OutputTrackerNotifier
	Closed bool
}

func NewOutputTracker(maxLines int) *OutputTracker {
	return &OutputTracker{
		MaxLines:              maxLines,
		CurrentLine:           &bytes.Buffer{},
		OutputTrackerNotifier: NewOutputTrackerNotifier([]string{}),
	}
}

func (o *OutputTracker) Close() {
	o.Lock()
	defer o.Unlock()
	o.Closed = true
	o.OutputTrackerNotifier.Close()
}

func (o *OutputTracker) Write(p []byte) (int, error) {
	o.Lock()
	defer o.Unlock()

	if o.Closed {
		return 0, io.EOF
	}

	n, err := o.CurrentLine.Write(p)
	changed := false
	for {
		b := o.CurrentLine.Bytes()
		if bytes.IndexByte(b, 0xa) != -1 {
			line, err := o.CurrentLine.ReadBytes(0xa)
			if err != nil {
				return 0, err
			}
			o.Lines = append(o.Lines, string(line[:len(line)-1]))
			if len(o.Lines) > o.MaxLines {
				o.Lines = o.Lines[1:]
			}
			changed = true
		} else {
			break
		}
	}

	if changed {
		o.Notify(o.Lines)
	}

	return n, err
}
