package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	ws "github.com/roundinternetmoney/ethereal-wss/v2"
	etherealv1 "roundinternet.money/protos/gen/dex/ethereal/v1"
)

const bitcoinSymbol = "BTCUSD"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := ws.NewClient(ctx, ws.Mainnet)
	defer client.Close()

	// Set typed callbacks for each event type
	client.OnL2Book(func(msg *etherealv1.L2Book) {
		fmt.Printf("[L2Book] %v\n", msg)
	})
	client.OnTicker(func(msg *etherealv1.Ticker) {
		fmt.Printf("[Ticker] %v\n", msg)
	})
	client.OnTradeFill(func(msg *etherealv1.TradeFill) {
		fmt.Printf("[TradeFill] %v\n", msg)
	})

	events := []ws.EventType{
		ws.EventTypeL2Book,
		ws.EventTypeTicker,
		ws.EventTypeTradeFill,
	}

	for _, event := range events {
		if err := client.Subscribe(ctx, event, bitcoinSymbol); err != nil {
			log.Fatalf("subscribe %s: %v", event, err)
		}
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Listen(ctx) // blocking
	}()

	select {
	case <-ctx.Done():
		return
	case err := <-errCh:
		if err != nil && ctx.Err() == nil {
			log.Fatal(err)
		}
	}
}
