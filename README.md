# 🐝 buzzybox: Portable shell utilities

`buzzybox` is a multicall binary like `busybox` that brings a subset of shell
utilities to multiple platforms. It can be run as a standalone program
(`buzzybox`) or imported as a library (`lesiw.io/buzzybox/hive`).

## Features

* Memory safe.
* [No dependencies.](go.mod)
* [tinygo](https://tinygo.org/) compatible.
* Friendly license (MIT).

## Installation

``` sh
go install lesiw.io/buzzybox@latest
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
| `basename` | ✅    | ✅      | ✅    | ✅     |
| `false`    | ✅    | ✅      | ✅    | ✅     |
| `true`     | ✅    | ✅      | ✅    | ✅     |
