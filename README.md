# lego [![CircleCI](https://circleci.com/gh/stairlin/lego.svg?style=svg)](https://circleci.com/gh/stairlin/lego) [![Go Report Card](https://goreportcard.com/badge/github.com/stairlin/lego)](https://goreportcard.com/report/github.com/stairlin/lego)

## Why

> People don't buy painting, they buy painted walls.

Go is gaining popularity at exponential speed, but adopting the language to build web applications at the early stage of a company can be challenging due to the lack of ready-to-use tools. LEGO has been solving that problem with a framework that contains the tools required to build robust distributed services. LEGO made most of the decisions for you, so that you can focus on bringing more values to your products.

## Manifesto

	1. Grow with the product
	2. Defer decisions
	3. Not for everyone

### 1. Grow with the product
LEGO is a framework designed to grow with developers from the first service to multiple resilient microservices at descent scale.

### 2. Defer decisions
Making technical decisions can be needlessly time consuming, especially at the early stage of a product development. That is the reason why LEGO made a lot of these decisions for you and as trivial as possible. That means you won't be locked-in into a specific vendor technology.

### 3. Not for everyone
Even though LEGO can grow with your product, it does not necessarily mean that it is the right choice for you. LEGO primarily solves Stairlin's problems and may discard very important problems in your product. Nevertheless, LEGO is open to new ideas and contributions as long as they are consistent with our philosophy.

## Demo

Start a simple HTTP server

```shell
$ git clone https://github.com/stairlin/lego.git
$ cd lego/example
$ CONFIG_URI=file://${PWD}/config.json go run http_server.go
```

Send a request

```shell
$ curl -v http://127.0.0.1:3000/ping
```

## Example

### Simple HTTP server

This code creates a LEGO instance and attach and HTTP handler to it with one route `/ping`.

```go
package main

import (
	"fmt"
	"os"

	"github.com/stairlin/lego"
	"github.com/stairlin/lego/handler/http"
	"github.com/stairlin/lego/log"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

func main() {
	// Create lego
	config := &AppConfig{}
	app, err := lego.New("api", config)
	if err != nil {
		fmt.Println("Problem initialising lego", err)
		os.Exit(1)
	}

	// Register HTTP handler
	h := http.NewHandler()
	h.Handle("/ping", http.GET, &Ping{})
	app.RegisterHandler("127.0.0.1:3000", h)

	// Start serving requests
	err = app.Serve()
	if err != nil {
		fmt.Println("Problem serving requests", err)
		os.Exit(1)
	}
}

// HTTP handler example
type Ping struct{}

func (a *Ping) Call(c *http.Context) http.Renderer {
	c.Ctx.Trace("action.ping", "Simple request", log.Time("start_at", c.StartAt))
	return c.Head(http.StatusOK)
}

```

### Config

Example of a configuration file

```json
{
    "node": "node.test",
    "version": "test",
    "log": {
        "level": "trace",
        "formatter": {
            "adapter": "logf"
        },
        "printer": {
            "adapter": "stdout"
        }
    },
    "stats": {
        "on": false
    },
    "request": {
        "timeout_ms": 500
    },
    "app": {
        "foo": "bar"
    }
}
```
