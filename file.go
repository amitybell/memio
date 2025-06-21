package memio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"slices"
	"time"
	"unicode/utf8"
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

// Reset is equivalent to Truncate(0)
func (f *File) Reset() *File {
	f.pos = 0
	f.buf = f.buf[:0]
	return f
}

// Truncate sets the internal offset and buffer size to n
func (f *File) Truncate(n int) *File {
	f.Seek(int64(n), io.SeekStart)
	f.buf = f.buf[:n]
	return f
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
	c := f.buf[f.pos]
	f.pos += 1
	return c, nil
}

// readBytes implements ReadBytes and ReadString, returning a slice to the internal buffer
func (f *File) readBytes(delim byte) ([]byte, error) {
	if f.pos >= len(f.buf) {
		return nil, io.ErrUnexpectedEOF
	}
	if i := bytes.IndexByte(f.buf[f.pos:], delim); i >= 0 {
		s := f.buf[f.pos : f.pos+i]
		f.pos += i + 1 // skip over delim
		return s, nil
	}
	s := f.buf[f.pos:]
	f.pos = len(f.buf)
	return s, io.ErrUnexpectedEOF
}

// ReadBytes reads bytes up to and excluding delim
// An error (wrapping io.ErrUnexpectedEOF) is returned iff delim is not found
func (f *File) ReadBytes(delim byte) ([]byte, error) {
	p, err := f.readBytes(delim)
	q := append([]byte(nil), p...)
	if err != nil {
		return q, fmt.Errorf("File.ReadBytes: %w", err)
	}
	return q, nil
}

// ReadString reads bytes up to and excluding delim
// An error (wrapping io.ErrUnexpectedEOF) is returned iff delim is not found
func (f *File) ReadString(delim byte) (string, error) {
	p, err := f.readBytes(delim)
	q := string(p)
	if err != nil {
		return q, fmt.Errorf("File.ReadString: %w", err)
	}
	return q, nil
}

// ReadFull fills buffer p, or returns the number of bytes read and error io.ErrUnexpectedEOF
func (f *File) ReadFull(p []byte) (int, error) {
	if f.pos >= len(f.buf) {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, f.buf[f.pos:])
	f.pos += n
	if n < len(p) {
		return n, io.ErrUnexpectedEOF
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

// ReadInt16 is a wrapper around int16(ReadUint16)
func (f *File) ReadInt16(o binary.ByteOrder) (int16, error) {
	n, err := f.ReadUint16(o)
	return int16(n), err
}

// ReadInt32 is a wrapper around int32(ReadUint32)
func (f *File) ReadInt32(o binary.ByteOrder) (int32, error) {
	n, err := f.ReadUint32(o)
	return int32(n), err
}

// ReadInt64 is a wrapper around int64(ReadUint64)
func (f *File) ReadInt64(o binary.ByteOrder) (int64, error) {
	n, err := f.ReadUint64(o)
	return int64(n), err
}

// ReadFloat32 reads a 32-bit floating point number in the byte order specified by o
func (f *File) ReadFloat32(o binary.ByteOrder) (float32, error) {
	n, err := f.ReadUint32(o)
	if err != nil {
		return 0, fmt.Errorf("File.ReadFloat32: %w", err)
	}
	return math.Float32frombits(n), nil
}

// ReadFloat64 reads a 64-bit floating point number in the byte order specified by o
func (f *File) ReadFloat64(o binary.ByteOrder) (float64, error) {
	n, err := f.ReadUint64(o)
	if err != nil {
		return 0, fmt.Errorf("File.ReadFloat64: %w", err)
	}
	return math.Float64frombits(n), nil
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

// Grow increases the capacity of the internal buffer to guarantee space for another n byte without reallocation
func (f *File) Grow(n int) *File {
	f.buf = slices.Grow(f.buf, f.pos+n)
	return f
}

// ReadFrom implements io.ReaderFrom
func (f *File) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		f.Grow(1 << 10)
		m, err := r.Read(f.buf[f.pos:cap(f.buf)])
		if m < 0 {
			panic(fmt.Sprintf("%T.Read() returned negative count %d", r, m))
		}
		f.buf = f.buf[:f.pos+m]
		n += int64(m)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return n, nil
			}
			return n, err
		}
	}
}

// WriteTo implements io.WriterTo
func (f *File) WriteTo(w io.Writer) (int64, error) {
	if f.pos >= len(f.buf) {
		return 0, fmt.Errorf("File.WriteTo: %w", io.ErrUnexpectedEOF)
	}
	s := f.buf[f.pos:]
	n, err := w.Write(s)
	f.pos += n
	if err != nil {
		return int64(n), fmt.Errorf("File.WriteTo: %w", err)
	}
	if n < len(s) {
		return int64(n), fmt.Errorf("File.WriteTo: %w", io.ErrShortWrite)
	}
	return int64(n), nil
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

// WriteInt16 is a wrapper around f.WriteUint16(o, uint16(n))
func (f *File) WriteInt16(o binary.ByteOrder, n int16) {
	f.WriteUint16(o, uint16(n))
}

// WriteInt32 is a wrapper around f.WriteUint32(o, uint32(n))
func (f *File) WriteInt32(o binary.ByteOrder, n int32) {
	f.WriteUint32(o, uint32(n))
}

// WriteInt64 is a wrapper around f.WriteUint64(o, uint64(n))
func (f *File) WriteInt64(o binary.ByteOrder, n int64) {
	f.WriteUint64(o, uint64(n))
}

// WriteFloat64 is a wrapper around f.WriteUint64(o, math.Float64bits(n))
func (f *File) WriteFloat64(o binary.ByteOrder, n float64) {
	f.WriteUint64(o, math.Float64bits(n))
}

// WriteFloat32 is a wrapper around f.WriteUint32(o, math.Float32bits(n))
func (f *File) WriteFloat32(o binary.ByteOrder, n float32) {
	f.WriteUint32(o, math.Float32bits(n))
}

func (f *File) Printf(format string, args ...any) *File {
	fmt.Fprintf(f, format, args...)
	return f
}

// PrintRune appends the UTF-8 encoding of r to the buffer at the current position
func (f *File) PrintRune(r rune) *File {
	s := utf8.AppendRune(f.buf[:f.pos], r)
	if len(s) > len(f.buf) {
		f.buf = s
	}
	f.pos = len(s)
	return f
}

func (f *File) PrintString(a ...string) *File {
	n := 0
	for _, s := range a {
		n += len(s)
	}
	f.Grow(n)
	for _, s := range a {
		f.buf = append(f.buf[:f.pos], s...)
		f.pos += len(s)
	}
	return f
}

func (f *File) PrintBytes(s []byte) *File {
	f.buf = append(f.buf[:f.pos], s...)
	f.pos += len(s)
	return f
}

func (f *File) PrintByte(b byte) *File {
	f.buf = append(f.buf[:f.pos], b)
	f.pos += 1
	return f
}

func (f *File) PrintFunc(size int, fn func(s []byte) []byte) *File {
	f.Grow(size)
	s := f.buf[f.pos : f.pos : f.pos+size]
	t := fn(s)
	if unsafe.SliceData(s) == unsafe.SliceData(t) {
		f.pos += len(t)
		f.buf = f.buf[:f.pos]
	} else {
		f.buf = append(f.buf[:f.pos], t...)
		f.pos = len(f.buf)
	}
	return f
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

// Rewind sets the Read position of the internal buffer back to the start
//
// It's equivalent to Seek(0, io.SeekStart) or Seek(0, 0)
func (f *File) Rewind() *File {
	f.pos = 0
	return f
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
