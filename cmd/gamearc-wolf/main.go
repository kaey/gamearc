package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/kaey/gamearc/internal/flagx"
	"github.com/kaey/gamearc/wolf"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Usage = flagx.Usage("gamearc-wolf [FLAGS] SRCFILE DSTDIR")
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

	arc, err := wolf.OpenArchive(r, ri.Size())
	if err != nil {
		return err
	}

	for _, f := range arc.Files {
		dst := filepath.Join(dstdir, filepath.FromSlash(f.Path()))
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			log.Fatalln("DST create error:", err)
		}

		data, err := f.Data()
		if err != nil {
			log.Fatalln(err)
		}

		if err := ioutil.WriteFile(dst, data, 0644); err != nil {
			log.Fatalln(err)
		}
	}

	return nil
}
