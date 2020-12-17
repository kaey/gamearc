package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

	if *keyFilePath == "" {
		flagx.Fail("Specify -key-file")
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
	keyFileData, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		return fmt.Errorf("key-file read error: %w", err)
	}

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

	if err := os.MkdirAll(dstdir, 0755); err != nil {
		return fmt.Errorf("DST create error: %w", err)
	}

	// TODO: support single src file
	srcfiles, err := ioutil.ReadDir(srcdir)
	if err != nil {
		return err
	}

filesLoop:
	for _, srcfile := range srcfiles {
		if srcfile.IsDir() {
			continue filesLoop
		}

		ext := filepath.Ext(srcfile.Name())                            // extension with dot (for ex .rpgmvp)
		base := strings.TrimSuffix(filepath.Base(srcfile.Name()), ext) // basename without extension (for ex w04_16)

		switch ext {
		case ".rpgmvp":
			ext = ".png"
		case ".rpgmvm":
			ext = ".m4a"
		case ".rpgmvo":
			ext = ".ogg"
		default:
			// Not encrypted.
			continue filesLoop
		}

		r, err := os.Open(filepath.Join(srcdir, srcfile.Name()))
		if err != nil {
			return err
		}

		w, err := os.Create(filepath.Join(dstdir, base+ext))
		if err != nil {
			return err
		}

		if err := decrypt(key, r, w); err != nil {
			return err
		}

		if err := r.Close(); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
	}

	return nil
}

func verifyHeader(b [16]byte) error {
	// TODO: maybe implement
	return nil
}

func decrypt(key [16]byte, r io.Reader, w io.Writer) error {
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
