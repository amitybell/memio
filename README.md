# Intro

Package memio implements common types and functions used for in-memory I/O.

# Install

    go get github.com/amitybell/memio

# Usage

```go
    package main

    import (
        "github.com/amitybell/memio"
        "io"
        "fmt"
    )

    func main() {
        // since File is purely in-memory, as long as all inputs are valid (e.g. Seek offsets/whence),
        // all method calls will succeed.
        f := &memio.File{}
        f.WriteString("hello world")
        // reset back to the begining of the file to begin reading
        f.Seek(0, 0)
        s, _ := io.ReadAll(f)
        fmt.Printf("%s\n", s)
    }
```
