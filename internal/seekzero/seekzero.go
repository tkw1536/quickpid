//spellchecker:words seekzero
package seekzero

//spellchecker:words errors
import (
	"errors"
	"fmt"
	"io"
)

// MakeOnceSeekable ensures that the given [io.Reader] implements [io.ReadSeeker],
// and can be sought to zero at least once.
//
// If the reader already implements [io.ReadSeeker] (and is not a [OnceSeekStartReader]), then the reader is returned as is.
// Otherwise a new [OnceSeekStartReader] is returned.
func MakeOnceSeekable(reader io.Reader) io.ReadSeeker {
	if seeker, ok := reader.(io.ReadSeeker); ok {
		if _, ok := seeker.(*OnceSeekStartReader); !ok {
			return seeker
		}
	}
	return NewOnceSeekStartReader(reader)
}

// OnceSeekStartReader is a reader that can be sought to the beginning of the input once.
// Seeking to zero means that the reader will re-read the entire input from the beginning.
type OnceSeekStartReader struct {
	didSeekToZero bool
	buffer        []byte
	reader        io.Reader
}

func NewOnceSeekStartReader(reader io.Reader) *OnceSeekStartReader {
	return &OnceSeekStartReader{reader: reader}
}

var _ io.Seeker = (*OnceSeekStartReader)(nil)

var (
	errAlreadySoughtOnce = errors.New("OnceSeekStartReader: already sought to zero")
	errInvalidSeek       = errors.New("OnceSeekStartReader: invalid seek")
)

// SeekToStart seeks to the beginning of the input.
// It can only be called once.
func (r *OnceSeekStartReader) SeekToStart() error {
	if r.didSeekToZero {
		return errAlreadySoughtOnce
	}
	r.didSeekToZero = true
	return nil
}

// Seek implements the [io.Seeker] interface.
// This method only supports seeking to whence == io.SeekStart and offset == 0 once.
func (r *OnceSeekStartReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart && offset == 0 {
		return 0, r.SeekToStart()
	}
	return 0, errInvalidSeek
}

func (r *OnceSeekStartReader) SeekToOffset(offset int64) error {
	if r.didSeekToZero {
		return errAlreadySoughtOnce
	}
	r.didSeekToZero = true
	return nil
}

func (r *OnceSeekStartReader) Read(p []byte) (int, error) {
	if !r.didSeekToZero {
		// Before the reset: read directly from the reader
		// But keep a copy of the bytes in the buffer.
		n, err := r.reader.Read(p)
		r.buffer = append(r.buffer, p[:n]...)
		return r.wrapReadError(n, err)
	}

	// After reset: Drain the buffer first.
	// We reset the buffer to nil if it's empty, to free up the memory and help the GC.
	if len(r.buffer) > 0 {
		n := copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		if len(r.buffer) == 0 {
			r.buffer = nil
		}
		return n, nil
	}

	// Clear the buffer, and read from the reader.
	return r.wrapReadError(r.reader.Read(p))
}

// wrapReadError wraps an (int, error) pair from the underlying reader.
func (*OnceSeekStartReader) wrapReadError(n int, err error) (int, error) {
	//spellchecker:words nolint errorlint
	//nolint:errorlint // to guarantee implementing [io.Reader] properly, we want to retain the sentinel [io.EOF] value.
	if err == nil || err == io.EOF {
		return n, err
	}
	return n, fmt.Errorf("underlying reader returned an error: %w", err)
}
