package multiprogressbar

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/schollz/progressbar/v3"
)

// MultiProgressBar manages multiple progress bars.
type MultiProgressBar struct {
	curLine int
	bars    []*progressbar.ProgressBar
	guard   sync.Mutex
	output  *bufio.Writer
}

// New creates a new MultiProgressBar with default options.
func New() *MultiProgressBar {
	return NewOptions()
}

// NewOptions creates a new MultiProgressBar with the provided options.
func NewOptions(options ...Option) *MultiProgressBar {
	mpb := &MultiProgressBar{
		curLine: 0,
		bars:    []*progressbar.ProgressBar{},
		guard:   sync.Mutex{},
		output:  bufio.NewWriter(os.Stdout),
	}
	for _, opt := range options {
		opt(mpb)
	}
	return mpb
}

// Add adds a progress bar to the MultiProgressBar.
// This changes the writer of the progress bar. Do not change the writer afterwards!
// Not thread safe.
// Returns the added progress bar.
func (mpb *MultiProgressBar) Add(pBar *progressbar.ProgressBar) *progressbar.ProgressBar {
	progressbar.OptionSetWriter(&multiProgressBarWriter{
		MultiProgressBar: mpb,
		idx:              len(mpb.bars),
	})(pBar)
	mpb.bars = append(mpb.bars, pBar)
	return pBar
}

// Get returns the progressbar.ProgressBar at the given index.
// Panics if the index does not exist.
func (mpb *MultiProgressBar) Get(idx int) *progressbar.ProgressBar {
	return mpb.bars[idx]
}

// BarCount returns the number of progress bars.
func (mpb *MultiProgressBar) BarCount() int {
	return len(mpb.bars)
}

// RenderBlank calls RenderBlank on all progress bars.
// If an error occurs, RenderBlank might not be called on all bars.
func (mpb *MultiProgressBar) RenderBlank() error {
	for _, pbar := range mpb.bars {
		if err := pbar.RenderBlank(); err != nil {
			return err
		}
	}
	return nil
}

// Finish calls Finish on all progress bars.
// If an error occurs, Finish might not be called on all bars.
// This also calls End.
func (mpb *MultiProgressBar) Finish() error {
	for _, pbar := range mpb.bars {
		if err := pbar.Finish(); err != nil {
			return err
		}
	}
	return mpb.End()
}

// End moves the cursor to the end of the progress bars.
// Not thread safe.
func (mpb *MultiProgressBar) End() error {
	_, err := mpb.move(len(mpb.bars), mpb.output)
	if err != nil {
		return err
	}
	return mpb.output.Flush()
}

// move moves the cursor to the beginning of the current progress bar.
func (mpb *MultiProgressBar) move(id int, writer io.Writer) (int, error) {
	bias := mpb.curLine - id
	mpb.curLine = id
	if bias > 0 {
		// move up
		return fmt.Fprintf(writer, "\r\033[%dA", bias)
	} else if bias < 0 {
		// move down
		return fmt.Fprintf(writer, "\r\033[%dB", -bias)
	}
	return 0, nil
}

// Option is the type all options need to adhere to.
type Option func(p *MultiProgressBar)

// OptionSetWriter sets the output writer.
// Behavior is undefined if called while using the MultiProgressBar.
func OptionSetWriter(writer io.Writer) Option {
	return func(mpb *MultiProgressBar) {
		mpb.output = bufio.NewWriter(writer)
	}
}

// multiProgressBarWriter is an io.Writer wrapper to know which progress bar wants to write.
type multiProgressBarWriter struct {
	*MultiProgressBar
	idx int
}

func (lw *multiProgressBarWriter) Write(p []byte) (n int, err error) {
	lw.guard.Lock()
	defer lw.guard.Unlock()
	n, err = lw.move(lw.idx, lw.output)
	if err != nil {
		return n, err
	}
	return lw.output.Write(p)
}
