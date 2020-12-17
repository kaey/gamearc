package wolf

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

type Archive struct {
	r     io.ReaderAt
	size  int64
	key   [12]byte
	Files []File
}

type File struct {
	r      io.ReaderAt
	key    [12]byte
	path   string
	offset int64
	size   int64
}

func (f *File) Path() string {
	return f.path
}

func (f *File) Data() ([]byte, error) {
	data := make([]byte, int(f.size))
	_, err := f.r.ReadAt(data, f.offset)
	if err != nil {
		return nil, err
	}
	for i := range data {
		data[i] ^= f.key[(int(f.size)+i)%len(f.key)]
	}

	return data, nil
}

func OpenArchive(r io.ReaderAt, size int64) (*Archive, error) {
	a := &Archive{r: r, size: size}
	if err := a.readIndex(); err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	return a, nil
}

func (a *Archive) readIndex() error {
	header := make([]byte, 48)
	if _, err := a.r.ReadAt(header[:], 0); err != nil {
		return err
	}

	copy(a.key[0:4], header[12:16])
	copy(a.key[4:8], header[28:32])
	copy(a.key[8:12], header[20:24])

	for i := range header {
		header[i] ^= a.key[i%len(a.key)]
	}

	if header[0] != 'D' || header[1] != 'X' {
		return fmt.Errorf("file header must start with DX, got %q", header[0:2])
	}

	version := getUint16(header, 2)
	if version != 6 {
		return fmt.Errorf("expected version 6, got: %d", version)
	}

	trailerSize := getUint32(header, 4)
	dataOffset := getUint64(header, 8)
	trailerOffset := getUint64(header, 16)
	fileTableOffset := getUint64(header, 24)
	dirTableOffset := getUint64(header, 32)
	codepage := getUint64(header, 40)
	if codepage != 932 {
		// https://docs.microsoft.com/en-us/windows/win32/intl/code-page-identifiers
		return fmt.Errorf("unsupported codepage, expected 932(shift-jis), got: %v", codepage)
	}

	trailer := make([]byte, trailerSize)
	if _, err := a.r.ReadAt(trailer, int64(trailerOffset)); err != nil {
		return err
	}
	for i := range trailer {
		trailer[i] ^= a.key[i%len(a.key)]
	}

	tr := japanese.ShiftJIS.NewDecoder()
	if err := a.decodeDir(trailer, dataOffset, fileTableOffset, dirTableOffset, 0, tr, ""); err != nil {
		return fmt.Errorf("decode first dir: %w", err)
	}

	return nil
}

func (a *Archive) decodeDir(b []byte, dataOffset, fileTableOffset, dirTableOffset, curDirOffset int, tr transform.Transformer, path string) error {
	//fileOffset := getUint64(b, dirTableOffset+curDirOffset+0)
	//parentOffset := getUint64(b, dirTableOffset+curDirOffset+8)
	nFiles := getUint64(b, dirTableOffset+curDirOffset+16)
	filelistOffset := getUint64(b, dirTableOffset+curDirOffset+24)

	for i := 0; i < nFiles; i++ {
		nameOffset := getUint64(b, fileTableOffset+filelistOffset+i*64+0)
		attr := getUint64(b, fileTableOffset+filelistOffset+i*64+8)
		//createTime := getUint64(b, fileTableOffset+filelistOffset+i*64+16)
		//accessTime := getUint64(b, fileTableOffset+filelistOffset+i*64+24)
		//writeTime := getUint64(b, fileTableOffset+filelistOffset+i*64+32)
		filedataOffset := getUint64(b, fileTableOffset+filelistOffset+i*64+40)
		size := getUint64(b, fileTableOffset+filelistOffset+i*64+48)
		compressedDataSize := getUint64(b, fileTableOffset+filelistOffset+i*64+56)

		if compressedDataSize > -1 {
			continue
			//return fmt.Errorf("compressed archives not supported")
		}

		nameLength := getUint16(b, nameOffset) * 4
		//nameParity := getUint16(b, nameOffset+2)
		name, _, err := transform.String(tr, strings.Trim(string(b[nameOffset+4+nameLength:nameOffset+4+nameLength*2]), "\x00"))
		if err != nil {
			return err
		}

		if name == ".." || strings.Contains(name, "/") {
			return fmt.Errorf("bad path: %q", name)
		}

		if attr&0x10 > 0 { // is a directory, https://docs.microsoft.com/en-us/windows/win32/fileio/file-attribute-constants
			if err := a.decodeDir(b, dataOffset, fileTableOffset, dirTableOffset, filedataOffset, tr, filepath.Join(path, name)); err != nil {
				return err
			}

			continue
		}

		a.Files = append(a.Files, File{
			r:      a.r,
			key:    a.key,
			path:   filepath.Join(path, name),
			offset: int64(dataOffset + filedataOffset),
			size:   int64(size),
		})
	}

	return nil
}

func getUint16(b []byte, offset int) int {
	b = b[offset:]
	_ = b[1]
	return int(b[0]) | int(b[1])<<8
}

func getUint32(b []byte, offset int) int {
	b = b[offset:]
	_ = b[3]
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
}

func getUint64(b []byte, offset int) int {
	b = b[offset:]
	_ = b[7]
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24 |
		int(b[4])<<32 | int(b[5])<<40 | int(b[6])<<48 | int(b[7])<<56
}
