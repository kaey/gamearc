package blorb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Archive struct {
	r    io.ReaderAt
	size int64

	Pics  []File
	Snds  []File
	Datas []File
	Execs []File
	Gluls []File
}

type File struct {
	r      io.ReaderAt
	id     int
	format string
	offset int64
	size   int64
}

func (f *File) ID() int {
	return f.id
}

func (f *File) Format() string {
	return f.format
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
	var header [24]byte
	if _, err := a.r.ReadAt(header[:], 0); err != nil {
		return err
	}

	// https://en.wikipedia.org/wiki/Interchange_File_Format
	if expected, got := []byte("FORM"), header[0:4]; !bytes.Equal(expected, got) {
		return fmt.Errorf("expected group chunk %q, got %q", expected, got)
	}

	_ = header[4:8] // int32 chunk len, we expect only one form in file so ignore for now.

	if expected, got := []byte("IFRS"), header[8:12]; !bytes.Equal(expected, got) {
		return fmt.Errorf("expected form type %q, got %q", expected, got)
	}

	if expected, got := []byte("RIdx"), header[12:16]; !bytes.Equal(expected, got) {
		return fmt.Errorf("expected first chunk type %q, got %q", expected, got)
	}

	size := be.Uint32(header[16:20])
	num := be.Uint32(header[20:24]) // Number of entries

	// Each index entry is 12 bytes, num (which takes 4 bytes) is also included in total size.
	if expectedSize := 4 + num*12; expectedSize != size {
		return fmt.Errorf("expected index size %d, got %d (num: %d)", expectedSize, size, num)
	}

	idx := make([]byte, size-4)
	if _, err := a.r.ReadAt(idx, 24); err != nil {
		return err
	}

	for i := 0; i < int(num); i++ {
		idx := idx[i*12:]

		typ := idx[0:4]
		id := int(be.Uint32(idx[4:8]))
		offset := int64(be.Uint32(idx[8:12]))

		var buf [8]byte
		if _, err := a.r.ReadAt(buf[:], offset); err != nil {
			return err
		}

		format := string(buf[0:4])
		switch {
		case bytes.Equal([]byte("PNG "), buf[0:4]):
			format = "png"
		case bytes.Equal([]byte("JPEG"), buf[0:4]):
			format = "jpg"
		case bytes.Equal([]byte("GLUL"), buf[0:4]):
			format = "glul"
		}
		size := int64(be.Uint32(buf[4:8]))

		file := File{
			r:      a.r,
			id:     id,
			format: format,
			offset: offset + 8,
			size:   size,
		}

		switch {
		case bytes.Equal([]byte("Pict"), typ):
			a.Pics = append(a.Pics, file)
		case bytes.Equal([]byte("Snd "), typ):
			a.Snds = append(a.Snds, file)
		case bytes.Equal([]byte("Data"), typ):
			a.Datas = append(a.Datas, file)
		case bytes.Equal([]byte("Exec"), typ):
			a.Execs = append(a.Execs, file)
		case bytes.Equal([]byte("GLUL"), typ):
			a.Gluls = append(a.Gluls, file)
		default:
			return fmt.Errorf("expected valid resource type, got %q", typ)
		}
	}

	return nil
}

var be = binary.BigEndian
