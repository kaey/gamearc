package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/kaey/gamearc/blorb"
	"github.com/kaey/gamearc/internal/flagx"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Usage = flagx.Usage("gamearc-blorb [FLAGS] SRCFILE DSTDIR")
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

func Main(srcfile, dstdir string) error {
	r, err := os.Open(srcfile)
	if err != nil {
		return err
	}

	ri, err := r.Stat()
	if err != nil {
		return err
	}

	arc, err := blorb.OpenArchive(r, ri.Size())
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dstdir, 0755); err != nil {
		return err
	}

	for _, f := range arc.Pics {
		name := fmt.Sprintf("%04d.%s", f.ID(), f.Format())

		r := f.Reader()
		w, err := os.Create(filepath.Join(dstdir, name))
		if err != nil {
			return err
		}

		if _, err := io.Copy(w, r); err != nil {
			return err
		}

		if err := w.Close(); err != nil {
			return err
		}
	}

	return nil
}
