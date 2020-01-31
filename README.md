# gotham

[![Build Status](https://travis-ci.com/sleep2death/gotham.svg?branch=master)](https://travis-ci.com/sleep2death/gotham)
[![Go Report Card](https://goreportcard.com/badge/github.com/sleep2death/gotham)](https://goreportcard.com/report/github.com/sleep2death/gotham)
[![codecov](https://codecov.io/gh/sleep2death/gotham/branch/master/graph/badge.svg)](https://codecov.io/gh/sleep2death/gotham)

A well-tested/high-performace tcp router written by go, with protbuf supported, highly inspired by this [post](https://sahilm.com/tcp-servers-that-run-like-clockwork/), and the source of standard http library:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go).

## content
- [Installation](#installation)
- [Quick start](#quick-start)

## Installation

To install gotham package, you need to install Go and set your Go workspace first.

1. The first need [Go](https://golang.org/) installed (**version 1.11+ is required**), then you can use the below Go command to install Gin.

```sh
$ go get -u github.com/sleep2death/gotham
```

2. Import it in your code:

```go
import "github.com/sleep2death/gotham"
```

## Quick start
