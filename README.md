[![Go Reference](https://pkg.go.dev/badge/github.com/kaey/gamearc.svg)](https://pkg.go.dev/github.com/kaey/gamearc)

Gamearc
=======

Command-line unpackers for several archive formats used in games.

Currently supports:

- Inform 7 (blorb)
- RPG Maker VX Ace (rgss3a, v3 only)
- RPG Maker MV (rpgmvp, rpgmvm, rpgmvo)
- RPG Maker MZ (png_, m4a_, ogg_)
- Ren'py (rpa, v3 only)
- Wolf RPG (dxa, v6 only, compression unsupported)
- Electron (asar)
- zip (decodes non-utf8 filenames as shift-jis)


Usage
-----

```
Usage:
  gamearc-blorb [FLAGS] SRCFILE DSTDIR

Flags:
  -version
    	Print version and exit

Specify SRCFILE and DSTDIR
```


Releases
-----

https://github.com/kaey/gamearc/releases


Building from source
-----

- install go compiler toolchain https://golang.org/dl
- run `go install github.com/kaey/gamearc/cmd/...@latest`
- binaries will appear in `$HOME/go/bin` (you can verify its location by running `go env GOPATH`)

Similar projects
-----

- https://notabug.org/gameripper/gameripper
