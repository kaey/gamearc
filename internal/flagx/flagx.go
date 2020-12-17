package flagx

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

func Version() string {
	ex, err := os.Executable()
	if err != nil {
		ex = os.Args[0]
	}

	res := new(strings.Builder)
	fmt.Fprintf(res, "%s: %s\n", ex, runtime.Version())

	mod, ok := debug.ReadBuildInfo()
	if !ok {
		return res.String()
	}

	fmt.Fprintf(res, "\tpath\t%s\n", mod.Path)
	fmt.Fprintf(res, "\tmod\t%s\t%s\t%s\n", mod.Main.Path, mod.Main.Version, mod.Main.Sum)

	for _, m := range mod.Deps {
		fmt.Fprintf(res, "\tdep\t%s\t%s\t%s\n", m.Path, m.Version, m.Sum)
		if m.Replace != nil {
			fmt.Fprintf(res, "\t=>\t%s\t%s\t%s\n", m.Replace.Path, m.Replace.Version, m.Replace.Sum)
		}
	}

	return res.String()
}

func Usage(u string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s\n", u)
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}
}

func Fail(err string) {
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n%s\n", err)
	os.Exit(2)
}
