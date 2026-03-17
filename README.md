# Golang Websocket Client for Ethereal API

## Features
- Protobuf support.
- Minimal dependencies

## Getting started

- Requires Go 1.25+.
- Install: `go get https://github.com/Round-Internet-Money/ethereal-wss`

## Example Usage

From the client directory:

- `make examples`
- `bin/example_listen_to_everything`

## Modifying the package
- Because the client uses protobufs, modifications must depend on them as well.
- If you do want to extend the `.proto` files directly, see this buf.build [repo](https://buf.build/round-internet-money/dex). 
- Otherwise, see this repo for instruction on  how to fork or include it [github.com/Round-Internet-Money/pb-dex](https://github.com/Round-Internet-Money/pb-dex)

Contributing
-------------
Contributions are welcome! Please open issues or pull requests as needed.


Todo
-----
```
- "resubscribe" event method
```