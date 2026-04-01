# Golang Websocket Client for Ethereal API

[![Go Reference](https://pkg.go.dev/badge/github.com/roundinternetmoney/ethereal-wss/v2.svg)](https://pkg.go.dev/github.com/roundinternetmoney/ethereal-wss/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/roundinternetmoney/ethereal-wss/v2)](https://goreportcard.com/report/github.com/roundinternetmoney/ethereal-wss/v2)
[![CI](https://github.com/roundinternetmoney/ethereal-wss/actions/workflows/ci.yml/badge.svg)](https://github.com/roundinternetmoney/ethereal-wss/actions/workflows/ci.yml)


## Features
- Protobuf support.
- Minimal dependencies

## Getting started

- Requires Go 1.25+.
- Install from GitHub: `go get github.com/roundinternetmoney/ethereal-wss/v2`

## Example Usage

```go
package main

import (
    "context"
    "log"

    "github.com/roundinternetmoney/ethereal-wss/v2"
    etherealv1 "roundinternet.money/protos/gen/dex/ethereal/v1"
)

func main() {
    client := ethereal.NewClient(context.Background(), ethereal.Mainnet)
    defer client.Close()

    // Subscribe to ticker updates for BTCUSD
    err := client.Subscribe(context.Background(), ethereal.EventTypeTicker, "BTCUSD")
    if err != nil {
        log.Fatal(err)
    }

    // typed callback for events
    client.OnTicker(func(msg *etherealv1.Ticker) {
        // message as typed struct
        log.Printf("Received ticker: %v", msg)
    })

    // Start listening
    log.Fatal(client.Listen(context.Background()))
}
```

For more examples, see the `examples/` directory.

## Modifying the package
- This client depends on protobuf wrappers from [pkg.go.dev/roundinternet.money/protos](https://pkg.go.dev/roundinternet.money/protos)
- If you want to extend the `.proto` files directly, see the Buf module at [buf.build/round-internet-money/dex](https://buf.build/round-internet-money/dex)
- Otherwise, use or fork [github.com/roundinternetmoney/protos](https://github.com/roundinternetmoney/protos)

Contributing
-------------
Contributions are welcome! Please open issues or pull requests as needed.


## Todo

- Add a `resubscribe` intent helper.
