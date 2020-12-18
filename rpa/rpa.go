package rpa

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"math/big"
	"path"
	"strconv"
	"strings"

	pickle "github.com/kisielk/og-rek"
)

type Archive struct {
	r     io.ReaderAt
	size  int64
	Files []File
}

type File struct {
	r      io.ReaderAt
	path   string
	offset int64
	size   int64
}

func (f *File) Path() string {
	return f.path
}

func (f *File) Reader() *io.SectionReader {
	return io.NewSectionReader(f.r, f.offset, f.size)
}

func OpenArchive(r io.ReaderAt, size int64) (*Archive, error) {
	a := &Archive{r: r, size: size}
	if err := a.readIndex(); err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	return a, nil
}

func (a *Archive) readIndex() error {
	var header [34]byte
	if _, err := a.r.ReadAt(header[:], 0); err != nil {
		return nil
	}

	if expected, got := []byte("RPA-3.0 "), header[0:8]; !bytes.Equal(expected, got) {
		return fmt.Errorf("expected %q, got %q", expected, got)
	}

	trailerOffset, err := strconv.ParseInt(string(header[8:24]), 16, 64)
	if err != nil {
		return fmt.Errorf("trailer offset: %w", err)
	}
	if trailerOffset >= a.size {
		return fmt.Errorf("trailer offset beyond file size, offset: %v, size: %v", trailerOffset, a.size)
	}

	key, err := strconv.ParseInt(string(header[25:33]), 16, 64)
	if err != nil {
		return fmt.Errorf("key: %w", err)
	}
	if header[33] != '\n' {
		return fmt.Errorf("incomplete header")
	}

	tr := io.NewSectionReader(a.r, trailerOffset, a.size)
	tzr, err := zlib.NewReader(tr)
	if err != nil {
		return fmt.Errorf("trailer decompress: %w", err)
	}
	defer tzr.Close()

	td := pickle.NewDecoder(tzr)
	trailer, err := td.Decode()
	if err != nil {
		return fmt.Errorf("trailer decode: %w", err)
	}

	// The rest of this bullshit code walks over pickle data and converts it into usable struct.
	trailerMap, ok := trailer.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("expected trailer to be a map, got %T", trailer)
	}
	a.Files = make([]File, 0, len(trailerMap))

	for k, v := range trailerMap {
		p, ok := k.(string)
		if !ok {
			return fmt.Errorf("expected string path, got: %q", k)
		}
		// Strip first part (it matches archive name), clean then run checks.
		pp := strings.Split(path.Clean(p), "/")
		p = path.Join(pp[1:]...)
		if path.IsAbs(p) {
			return fmt.Errorf("archive contains a file with absolute path: %q", p)
		}
		if strings.Split(p, "/")[0] == ".." {
			return fmt.Errorf("archive contains a file with path that leads outside of its root: %q", p)
		}

		v2, ok := v.([]interface{})
		if !ok {
			return fmt.Errorf("expected slice value, got %T", v2)
		}
		if len(v2) != 1 {
			return fmt.Errorf("expected val length to be 1, got %v", len(v2))
		}

		tuple, ok := v2[0].(pickle.Tuple)
		if !ok {
			return fmt.Errorf("expected tuple value, got %T", v2)
		}
		if len(tuple) != 3 {
			return fmt.Errorf("expected 3 items in a tuple, got %v", len(tuple))
		}

		offsetBig, ok := tuple[0].(*big.Int)
		if !ok {
			return fmt.Errorf("expected offset big.Int value, got %T", tuple[0])
		}
		offset := offsetBig.Int64() ^ key
		if offset >= a.size {
			return fmt.Errorf("file offset beyond file size, offset: %v, size: %v", offset, a.size)
		}

		size, ok := tuple[1].(int64)
		if !ok {
			return fmt.Errorf("expected size int64 value, got %T", tuple[1])
		}
		size ^= key

		a.Files = append(a.Files, File{
			r:      a.r,
			path:   p,
			offset: offset,
			size:   size,
		})
	}

	return nil
}
