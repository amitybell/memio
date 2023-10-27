package memio

import (
	"fmt"
	"io"
	"io/fs"
	"slices"
	"time"
	"unsafe"
)

// File implements file-like methods on an in-memory buffer
type File struct {
	pos int
	buf []byte
}

// Len returns the length of the internal buffer
func (f *File) Len() int {
	return len(f.buf)
}

// Offset returns the current read/write position of the internal buffer
func (f *File) Offset() int64 {
	return int64(f.pos)
}

// Bytes returns the internal buffer
func (f *File) Bytes() []byte {
	return f.buf
}

// String returns a copy of the internal buffer as a string
func (f *File) String() string {
	return string(f.buf)
}

// StringRef returns a reference to the internal buffer as a string
func (f *File) StringRef() string {
	return unsafe.String(unsafe.SliceData(f.buf), len(f.buf))
}

// Reset sets the internal buffer to p and the internal offset to 0
func (f *File) Reset(p []byte) {
	f.pos = 0
	f.buf = p
}

// Read implements io.Reader
func (f *File) Read(p []byte) (int, error) {
	if f.pos >= len(f.buf) {
		return 0, io.EOF
	}
	n := copy(p, f.buf[f.pos:])
	f.pos += n
	return n, nil
}

// expand grows the internal buffer to fill n bytes and sets pos to the end
//
// It returns a slice that should be filled with n bytes of content
func (f *File) expand(n int) []byte {
	n += f.pos
	f.buf = slices.Grow(f.buf, n)
	if n > len(f.buf) {
		f.buf = f.buf[:n]
	}
	s := f.buf[f.pos:n]
	f.pos = n
	return s
}

// Seek implements io.Writer
func (f *File) Write(p []byte) (int, error) {
	return copy(f.expand(len(p)), p), nil
}

// WriteString implements io.StringWriter
func (f *File) WriteString(p string) (int, error) {
	s := f.expand(len(p))
	n := copy(s, p)
	return n, nil
}

// WriteByte implements io.ByteWriter
func (f *File) WriteByte(p byte) error {
	s := f.expand(1)
	s[0] = p
	return nil
}

// Seek implements io.Seeker
//
// If the final offset is greater than Len(), the internal buffer is expanded accordingly
func (f *File) Seek(offset int64, whence int) (int64, error) {
	var sp int64
	switch whence {
	case io.SeekStart:
		sp = offset
	case io.SeekCurrent:
		sp = int64(f.pos) + offset
	case io.SeekEnd:
		sp = int64(len(f.buf)) + offset
	default:
		return 0, fmt.Errorf("File.Seek: invalid whence(%d): %w", whence, fs.ErrInvalid)
	}
	if sp < 0 {
		return 0, fmt.Errorf("File.Seek: negative offset(%d): %w", sp, fs.ErrInvalid)
	}
	f.pos = int(sp)
	// simulates creating "holes" in files
	if f.pos > len(f.buf) {
		f.buf = slices.Grow(f.buf, f.pos)[:f.pos]
	}
	return sp, nil
}

// Stat implements the fs.File.Stat interface
//
// It always returns itself
func (f *File) Stat() (fs.FileInfo, error) {
	return f, nil
}

// Close implements the fs.File.Close interface
//
// It always returns nil
func (f *File) Close() error {
	return nil
}

// Name implements the fs.FileInfo.Name interface
//
// It always returns ""
func (f *File) Name() string {
	return ""
}

// Size implements the fs.FileInfo.Size interface
//
// It returns the length of the internal buffer
func (f *File) Size() int64 {
	return int64(len(f.buf))
}

// Mode implements the fs.FileInfo.Mode interface
//
// It always returns fs.ModeIrregular
func (f *File) Mode() fs.FileMode {
	return fs.ModeIrregular
}

// ModTime implements the fs.FileInfo.ModTime interface
//
// It always returns time.Time{}
func (f *File) ModTime() time.Time {
	return time.Time{}
}

// IsDir implements the fs.FileInfo.IsDir interface
//
// It always returns false.
func (f *File) IsDir() bool {
	return false
}

// Sys implements the fs.FileInfo.Sys interface
//
// It always returns nil
func (f *File) Sys() any {
	return nil
}

// NewFile returns a new File instance with the internal buffer set to s
func NewFile(s []byte) *File {
	return &File{buf: s}
}
