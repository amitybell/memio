package memio

import (
	"encoding/binary"
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

// ReadByte implements io.ByteReader
func (f *File) ReadByte() (byte, error) {
	if f.pos >= len(f.buf) {
		return 0, io.EOF
	}
	c := f.buf[f.pos+1]
	f.pos += 1
	return c, nil
}

// ReadFull wraps io.ReadFull(f, p)
func (f *File) ReadFull(p []byte) (int, error) {
	n, err := io.ReadFull(f, p)
	if err != nil {
		return n, fmt.Errorf("File.ReadFull: %w", err)
	}
	return n, nil
}

// ReadUint16 reads a 16-bit number in the byte order specified by o
func (f *File) ReadUint16(o binary.ByteOrder) (uint16, error) {
	p := [2]byte{}
	if _, err := f.ReadFull(p[:]); err != nil {
		return 0, fmt.Errorf("File.ReadUint16: %w", err)
	}
	return o.Uint16(p[:]), nil
}

// ReadUint32 reads a 32-bit number in the byte order specified by o
func (f *File) ReadUint32(o binary.ByteOrder) (uint32, error) {
	p := [4]byte{}
	if _, err := f.ReadFull(p[:]); err != nil {
		return 0, fmt.Errorf("File.ReadUint32: %w", err)
	}
	return o.Uint32(p[:]), nil
}

// ReadUint64 reads a 64-bit number in the byte order specified by o
func (f *File) ReadUint64(o binary.ByteOrder) (uint64, error) {
	p := [8]byte{}
	if _, err := f.ReadFull(p[:]); err != nil {
		return 0, fmt.Errorf("File.ReadUint64: %w", err)
	}
	return o.Uint64(p[:]), nil
}

// Expand grows the internal buffer to fill n bytes and sets pos to the end
//
// It returns a slice that should be filled with n bytes of content
func (f *File) Expand(n int) []byte {
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
	return copy(f.Expand(len(p)), p), nil
}

// WriteString implements io.StringWriter
func (f *File) WriteString(p string) (int, error) {
	s := f.Expand(len(p))
	n := copy(s, p)
	return n, nil
}

// WriteByte implements io.ByteWriter
func (f *File) WriteByte(p byte) error {
	s := f.Expand(1)
	s[0] = p
	return nil
}

// WriteUint16 writes n in the byte order specified by o
func (f *File) WriteUint16(o binary.ByteOrder, n uint16) {
	s := f.Expand(2)
	o.PutUint16(s, n)
}

// WriteUint32 writes n in the byte order specified by o
func (f *File) WriteUint32(o binary.ByteOrder, n uint32) {
	s := f.Expand(4)
	o.PutUint32(s, n)
}

// WriteUint64 writes n in the byte order specified by o
func (f *File) WriteUint64(o binary.ByteOrder, n uint64) {
	s := f.Expand(8)
	o.PutUint64(s, n)
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
