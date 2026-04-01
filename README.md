# Golang Websocket Client for Ethereal API

[![Go Reference](https://pkg.go.dev/badge/github.com/roundinternetmoney/ethereal-wss.svg)](https://pkg.go.dev/github.com/roundinternetmoney/ethereal-wss)


## Features
- Protobuf support.
- Minimal dependencies

## Getting started

- Requires Go 1.25+.
- Install from GitHub: `go get github.com/roundinternetmoney/ethereal-wss`

## Example Usage

From the client directory:

- `make examples`
- `bin/example_account_balance`

## Modifying the package
- This client depends on protobuf wrappers from [pkg.go.dev/roundinternet.money/protos](https://pkg.go.dev/roundinternet.money/pb-dex)
- If you want to extend the `.proto` files directly, see the Buf module at [buf.build/round-internet-money/dex](https://buf.build/round-internet-money/dex)
- Otherwise, use or fork [github.com/roundinternetmoney/protos](https://github.com/Round-Internet-Money/pb-dex)

Contributing
-------------
Contributions are welcome! Please open issues or pull requests as needed.


## Todo

- Add a `resubscribe` intent helper.
