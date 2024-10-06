package memio

import (
	"bytes"
	"io"
	"testing"
)

func TestWriteSeeker(t *testing.T) {
	final := "Hello, World!!"
	f := &File{}
	if _, err := f.Write(bytes.ToLower([]byte(`hello??world?`))); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Seek(-1, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("!"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if err := f.WriteByte('H'); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Seek(4, io.SeekCurrent); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(", W"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("!"); err != nil {
		t.Fatal(err)
	}

	if exp, got := final, f.String(); got != exp {
		t.Fatalf("Expected `%s`; Got `%s`", exp, got)
	}
}

func TestReader(t *testing.T) {
	final := "Hello, World!"
	f := &File{}
	if _, err := f.WriteString(final); err != nil {
		t.Fatal(err)
	}

	s, _ := io.ReadAll(f)
	if exp, got := "", string(s); got != exp {
		t.Fatalf("Expected `%s`; Got `%s`", exp, got)
	}

	if _, err := f.Seek(7, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	s, _ = io.ReadAll(f)
	if exp, got := "World!", string(s); got != exp {
		t.Fatalf("Expected `%s`; Got `%s`", exp, got)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	s, _ = io.ReadAll(f)
	if exp, got := final, string(s); got != exp {
		t.Fatalf("Expected `%s`; Got `%s`", exp, got)
	}
}

func TestTruncate(t *testing.T) {
	f := File{}

	f.WriteString("hello")
	if s := f.StringRef(); s != "hello" {
		t.Fatalf("Expected %q; Got %q", "hello", s)
	}

	f.Truncate(4).WriteString("world")
	if s := f.StringRef(); s != "hellworld" {
		t.Fatalf("Expected %q; Got %q", "hellworld", s)
	}

	f.Reset().WriteString("world")
	if s := f.StringRef(); s != "world" {
		t.Fatalf("Expected %q; Got %q", "world", s)
	}
}
