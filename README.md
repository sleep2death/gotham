# gotham

[![Build Status](https://travis-ci.com/sleep2death/gotham.svg?branch=master)](https://travis-ci.com/sleep2death/gotham)
[![Go Report Card](https://goreportcard.com/badge/github.com/sleep2death/gotham)](https://goreportcard.com/report/github.com/sleep2death/gotham)
[![codecov](https://codecov.io/gh/sleep2death/gotham/branch/master/graph/badge.svg)](https://codecov.io/gh/sleep2death/gotham)

A well-tested/high-performace tcp/protobuf router written in go, highly inspired by [Gin](https://github.com/gin-gonic/gin), and the source of standard http library:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go).

## content

-   [Installation](#installation)
-   [Quick start](#quick-start)

## Installation

To install gotham package, you need to install Go and set your Go workspace first.

1. The first need [Go](https://golang.org/) installed (**version 1.11+ is required**), then you can use the below Go command to install Gotham.

```sh
$ go get -u github.com/sleep2death/gotham
```

2. Import it in your code:

```go
import "github.com/sleep2death/gotham"
```

## Quick start

All examples you can find is in `/examples` folder.

```go
package main

import (
	"log"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"github.com/sleep2death/gotham"
	"github.com/sleep2death/gotham/examples/pb"
)

func main() {
	// SERVER
	// Starts a new gotham instance without any middleware attached.
	router := gotham.New()

	// Define your handlers.
	router.Handle("/pb/EchoMessage", func(c *gotham.Context) {
		message := new(pb.EchoMessage)

		// If some error fires, you can abort the request.
		if err := proto.Unmarshal(c.Data(), message); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// log.Printf("Ping request received at %s", ptypes.TimestampString(message.Ts))
		message.Message = "Pong"
		message.Ts = ptypes.TimestampNow()
		c.Write(message)
	})

	// Run, gotham, Run...
	addr := ":9090"
	log.Fatal(router.Run(addr))
}
```
