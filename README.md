# 🐝 buzzybox: Portable shell utilities

`buzzybox` is a multicall binary like `busybox` that brings a subset of shell
utilities to multiple platforms. It can be run as a standalone program
(`buzzybox`) or imported as a library (`lesiw.io/buzzybox/hive`).

## Features

* Written in a [memory safe language](https://media.defense.gov/2023/Dec/06/2003352724/-1/-1/0/THE-CASE-FOR-MEMORY-SAFE-ROADMAPS-TLP-CLEAR.PDF#page=19).
* Uses only [Go Project](https://go.dev/project) [dependencies](go.mod).
* Compatible with [tinygo](https://tinygo.org/).
* Open-sourced under a [permissive license](LICENSE).

## Installation

``` sh
go install lesiw.io/buzzybox@latest
```

## Usage

### Subcommand

```sh
echo "hello embedded world" | buzzybox awk '{ print $1, $3 }'
```

### Symlink

```sh
ln -s "$(which buzzybox)" awk
echo "hello embedded world" | ./awk '{ print $1, $3 }'
```

### Library

```go
package main

import (
	"strings"

	"lesiw.io/buzzybox/hive"
)

func main() {
	cmd := hive.Command("awk", "{ print $1, $3 }")
	cmd.Stdin = strings.NewReader("hello embedded world")
	cmd.Run()
}
```

[▶️ Run this example on the Go Playground](https://go.dev/play/p/NI5W18yuX8A)

### Docker

```sh
echo "hello embedded world" | docker run -i lesiw/buzzybox awk '{ print $1, $3 }'
```

## App criteria for inclusion

One of the following:
1. The app is defined in the POSIX standard.
2. The app isn’t in POSIX, but is found in multiple *nix environments, like
   MacOS, busybox, and at least one major non-busybox Linux distribution. (`tar`
   is a good example of a non-POSIX utility that is near-ubiquitous.)

And all of the following:
1. [Orthogonal](https://go.dev/talks/2010/ExpressivenessOfGo-2010.pdf): the app
   solves a problem that cannot reasonably be solved by using the existing apps
   in combination.

## Support matrix

| App        | Linux | Windows | MacOS | TinyGo |
|:-----------|:------|:--------|:------|--------|
| `arch`     | ✅    | ✅      | ✅    | ✅     |
| `awk`      | ✅    | ✅      | ✅    | ✅     |
| `basename` | ✅    | ✅      | ✅    | ✅     |
| `false`    | ✅    | ✅      | ✅    | ✅     |
| `true`     | ✅    | ✅      | ✅    | ✅     |
