package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaey/gamearc/internal/flagx"
)

func main() {
	keyFilePath := flag.String("key-file", "", "Path to system.json")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Usage = flagx.Usage("gamearc-rpgmv [FLAGS] SRCDIR DSTDIR")
	flag.Parse()

	if *versionFlag {
		fmt.Fprintf(os.Stderr, "%s", flagx.Version())
		os.Exit(0)
	}

	srcfile := flag.Arg(0)
	if srcfile == "" {
		flagx.Fail("Specify SRCDIR and DSTDIR")
	}

	dstdir := flag.Arg(1)
	if dstdir == "" {
		flagx.Fail("Specify DSTDIR")
	}

	if err := Main(srcfile, dstdir, *keyFilePath); err != nil {
		log.Fatalln(err)
	}
}

func Main(srcdir, dstdir, keyFilePath string) error {
	if keyFilePath == "" {
		dir, err := filepath.Abs(srcdir)
		if err != nil {
			return fmt.Errorf("src absolute path error: %w", err)
		}
		for { // walk the tree up until we find System.json or hit /
			ndir := filepath.Dir(dir)
			if ndir == dir {
				return fmt.Errorf("can't find data/System.json in any of the parent directories")
			}
			dir = ndir
			keyFilePath = filepath.Join(dir, "data/System.json") // TODO: use case insensitive search.
			if _, err := os.Stat(keyFilePath); err == nil {
				break
			}
		}
	}
	keyFileData, err := os.ReadFile(keyFilePath)
	if err != nil {
		return fmt.Errorf("key-file read error: %w", err)
	}

	// strip utf-8 byte order mark.
	keyFileData, _ = bytes.CutPrefix(keyFileData, []byte("\ufeff"))

	keyFile := struct {
		Key string `json:"encryptionKey"`
	}{}

	// TODO: might be lz compressed https://github.com/pieroxy/lz-string-go
	if err := json.Unmarshal(keyFileData, &keyFile); err != nil {
		return fmt.Errorf("key-file decode error: %w", err)
	}

	var key [16]byte
	if _, err := hex.Decode(key[:], []byte(keyFile.Key)); err != nil {
		return fmt.Errorf("malformed key: %w", err)
	}

	if err := os.MkdirAll(dstdir, 0o755); err != nil {
		return fmt.Errorf("DST create error: %w", err)
	}

	srcroot, err := os.OpenRoot(srcdir)
	if err != nil {
		return err
	}
	defer srcroot.Close()

	dstroot, err := os.OpenRoot(dstdir)
	if err != nil {
		return err
	}
	defer dstroot.Close()

	return fs.WalkDir(srcroot.FS(), ".", func(srcpath string, srcentry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if srcentry.IsDir() {
			return dstroot.MkdirAll(srcpath, 0o755)
		}

		ext := filepath.Ext(srcentry.Name())        // extension with dot (for ex .rpgmvp)
		dstpath := strings.TrimSuffix(srcpath, ext) // relative path without extension (for ex w04_16)

		switch ext {
		case ".rpgmvp", ".png_":
			ext = ".png"
		case ".rpgmvm", ".m4a_":
			ext = ".m4a"
		case ".rpgmvo", ".ogg_":
			ext = ".ogg"
		default:
			// Not encrypted.
			return nil
		}

		r, err := srcroot.Open(srcpath)
		if err != nil {
			return err
		}
		defer r.Close()

		w, err := dstroot.Create(dstpath + ext)
		if err != nil {
			return err
		}
		defer w.Close()

		if err := copyDecrypt(key, r, w); err != nil {
			return err
		}

		return nil
	})
}

func verifyHeader(_ [16]byte) error {
	// TODO: maybe implement
	return nil
}

func copyDecrypt(key [16]byte, r io.Reader, w io.Writer) error {
	var header [16]byte

	if _, err := r.Read(header[:]); err != nil {
		return err
	}

	if err := verifyHeader(header); err != nil {
		return err
	}

	// Only first 16 bytes are xor-encrypted.
	var start [16]byte
	if _, err := r.Read(start[:]); err != nil {
		return err
	}

	for i := range start {
		start[i] ^= key[i]
	}

	if _, err := w.Write(start[:]); err != nil {
		return err
	}

	// Just copy the rest of the file.
	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return nil
}
