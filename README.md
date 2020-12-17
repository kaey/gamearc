[![GoDoc](https://godoc.org/github.com/kaey/gamearc?status.svg)](https://godoc.org/github.com/kaey/gamearc)

Gamearc
=======

Command-line unpackers for several archive formats used in games.

Unpolished and mostly untested.

Currently supports:

- Inform 7 (blorb)
- RPG Maker VX Ace (rgss3a, v3 only)
- RPG Maker MV (rpgmvp, rpgmvm, rpgmvo)
- Ren'py (rpa, v3 only)
- Wolf RPG (dxa, v6 only, compression unsupported)


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
- open console or powershell
- run `go env -w GO111MODULE=on`
- run `go get github.com/kaey/gamearc/cmd/...`
- binaries will appear in `$HOME/go/bin` (you can verify its location by running `go env GOPATH`)

