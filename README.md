# go-feature

[![Build Status](https://travis-ci.org/spreadshirt/go-feature.svg?branch=master)](https://travis-ci.org/spreadshirt/go-feature)
[![GoDoc](https://godoc.org/github.com/spreadshirt/go-feature?status.svg)](https://godoc.org/github.com/spreadshirt/go-feature)

> Package feature provides a simple mechanism for creating and managing feature flags.

See [./examples/hello](./examples/hello/hello.go) for an example.

## Trying it out

- run `make && ./example-hello`
- visit <http://localhost:22022/hello> in your browser
- run `curl -XPOST 'http://localhost:22022/features/scream?enabled=true'`
- visit the `/hello` page again, and it will display a "louder" message
  (i.e. with more UPPERCASE letters)

Additional things to try out:

- <http://localhost:22022/features>: Displays the list of features, and
  their state.  If visited in a browser, it will provide a tiny
  user-interface for setting parameters.
- `curl -XPOST 'http://localhost:22022/features/surprise?enabled=true&ratio=0.75'`
  (note the "ratio" parameter)

## Building and testing

The Makefile is set up to work with a project-local GOPATH.  Thus you
can just run `make` to build the example, or `make test` to run the
tests.
