package asar

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"path"
)

type Archive struct {
	r    io.ReaderAt
	size int64

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
	var header [16]byte
	if _, err := a.r.ReadAt(header[:], 0); err != nil {
		return err
	}

	pickleLength := le.Uint32(header[0:4])
	if expected, got := uint32(4), pickleLength; expected != got {
		return fmt.Errorf("expected pickle size %q, got %q", expected, got)
	}

	indexLength := le.Uint32(header[4:8])
	// le.Uint32(header[8:12]) no idea what this is
	jsonLength := le.Uint32(header[12:16])
	r := io.NewSectionReader(a.r, 16, int64(jsonLength))

	var v file
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return err
	}

	// Walks over parsed json and build index.
	return a.recurse(v, "", int64(indexLength)+8)
}

func (a *Archive) recurse(v file, curpath string, dataOffset int64) error {
	for name, f := range v.Files {
		if name == ".." || name == "/" {
			// Use manual concatenation here because path.Join cleans paths.
			return fmt.Errorf("bad path: %s", curpath+"/"+name)
		}
		if f.Files != nil {
			if err := a.recurse(f, path.Join(curpath, name), dataOffset); err != nil {
				return err
			}
			continue
		}

		a.Files = append(a.Files, File{
			r:      a.r,
			path:   path.Join(curpath, name),
			offset: f.Offset + dataOffset,
			size:   f.Size,
		})
	}

	return nil
}

type file struct {
	Files  map[string]file `json:"files"`
	Offset int64           `json:"offset,string"`
	Size   int64           `json:"size"`
	Exec   bool            `json:"executable"`
}

var le = binary.LittleEndian
