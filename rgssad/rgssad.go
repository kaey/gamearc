package rgssad

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"strings"
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
	key    uint32
}

func (f *File) Path() string {
	return f.path
}

func (f *File) Reader() *io.SectionReader {
	return io.NewSectionReader(&decryptReaderAt{r: f.r, key: f.key, startOffset: f.offset}, f.offset, f.size)
}

func OpenArchive(r io.ReaderAt, size int64) (*Archive, error) {
	a := &Archive{r: r, size: size}
	if err := a.readIndex(); err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	return a, nil
}

func (a *Archive) readIndex() error {
	var header [12]byte
	if _, err := a.r.ReadAt(header[:], 0); err != nil {
		return err
	}

	if expected, got := []byte("RGSSAD\000"), header[0:7]; !bytes.Equal(expected, got) {
		return fmt.Errorf("expected rgss header %q, got %q", expected, got)
	}

	if expected, got := byte(0x3), header[7]; expected != got {
		return fmt.Errorf("expected rgss version %q, got %q", expected, got)
	}

	key := le.Uint32(header[8:12])*9 + 3

	offset := int64(12)
	for {
		var entry [16]byte
		if _, err := a.r.ReadAt(entry[:], offset); err != nil {
			return err
		}
		offset += 16

		fileoffset := le.Uint32(entry[0:4]) ^ key
		filesize := le.Uint32(entry[4:8]) ^ key
		filekey := le.Uint32(entry[8:12]) ^ key
		pathlen := le.Uint32(entry[12:16]) ^ key

		if fileoffset == 0 {
			return nil
		}

		// Read path.
		pathb := make([]byte, pathlen)
		if _, err := a.r.ReadAt(pathb, offset); err != nil {
			return err
		}
		offset += int64(pathlen)

		// Decrypt path and replace backslashes with forward slashes.
		keyb := [...]byte{byte(key), byte(key >> 8), byte(key >> 16), byte(key >> 24)}
		for i := range pathb {
			pathb[i] ^= keyb[i%4]
			if pathb[i] == '\\' {
				pathb[i] = '/'
			}
		}

		p := path.Clean(string(pathb))
		if path.IsAbs(p) {
			return fmt.Errorf("archive contains a file with absolute path: %q", p)
		}
		if strings.Split(p, "/")[0] == ".." {
			return fmt.Errorf("archive contains a file with path that leads outside of its root: %q", p)
		}
		a.Files = append(a.Files, File{
			r:      a.r,
			path:   p,
			offset: int64(fileoffset),
			size:   int64(filesize),
			key:    filekey,
		})
	}
}

type decryptReaderAt struct {
	r           io.ReaderAt
	key         uint32
	startOffset int64
}

func (r *decryptReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = r.r.ReadAt(p, off)

	// 4 bytes of a file is encrypted with initial key, then key is incremented (k*7 + 3).
	// Since we support random reads key has to be recalculated every time.
	ro := int(off - r.startOffset) // relative offset of a file
	ko := uint32(ro / 4)           // key offset, one key encrypts 4 bytes
	k := r.key                     // initial key
	for i := uint32(0); i < ko; i++ {
		// Can't simply multiply this by ko because of overflow semantics.
		k = k*7 + 3
	}

	// Usual XOR decryption routine, except key is incremented every 4 bytes.
	// Also since reading may start at any offset, not necessarily multiple of 4,
	// we have to account for it.
	for i := range p[:n] {
		if i > 0 && (i+ro)%4 == 0 {
			k = k*7 + 3
		}
		kb := [...]byte{byte(k), byte(k >> 8), byte(k >> 16), byte(k >> 24)}
		p[i] ^= kb[(i+ro)%4]
	}

	return n, err
}

var le = binary.LittleEndian
