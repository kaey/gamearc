package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/kaey/gamearc/internal/flagx"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Usage = flagx.Usage("gamearc-zip [FLAGS] SRCFILE DSTDIR")
	flag.Parse()

	if *versionFlag {
		fmt.Fprintf(os.Stderr, "%s", flagx.Version())
		os.Exit(0)
	}

	srcfile := flag.Arg(0)
	if srcfile == "" {
		flagx.Fail("Specify SRCFILE and DSTDIR")
	}

	dstdir := flag.Arg(1)
	if dstdir == "" {
		flagx.Fail("Specify DSTDIR")
	}

	if err := Main(srcfile, dstdir); err != nil {
		log.Fatalln(err)
	}
}

var shiftJIS = japanese.ShiftJIS.NewDecoder()

func Main(srcfile, dstdir string) error {
	r, err := os.Open(srcfile)
	if err != nil {
		return err
	}

	ri, err := r.Stat()
	if err != nil {
		return err
	}

	arc, err := zip.NewReader(r, ri.Size())
	if err != nil {
		return err
	}

	for _, f := range arc.File {
		path := f.Name
		if f.NonUTF8 {
			p, _, err := transform.String(shiftJIS, path)
			if err != nil {
				return err
			}
			path = p
		}
		dst := filepath.Join(dstdir, filepath.FromSlash(path))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				log.Fatalln("DST create error:", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			log.Fatalln("DST create error:", err)
		}
		fi, err := f.Open()
		if err != nil {
			return err
		}
		fo, err := os.Create(dst)
		if err != nil {
			return err
		}

		if _, err := io.Copy(fo, fi); err != nil {
			return err
		}

		if err := fi.Close(); err != nil {
			return err
		}
		if err := fo.Close(); err != nil {
			return err
		}
	}

	return nil
}
