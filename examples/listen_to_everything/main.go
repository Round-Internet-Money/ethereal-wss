package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	ws "github.com/roundinternetmoney/ethereal-wss"
	etherealv1 "roundinternet.money/protos/gen/dex/ethereal/v1"

	"google.golang.org/protobuf/proto"
)

const bitcoinSymbol = "BTCUSD"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := ws.NewClient(ctx, ws.Mainnet)
	defer client.Close()

	events := []etherealv1.EventType{
		etherealv1.EventType_EVENT_TYPE_L2_BOOK,
		etherealv1.EventType_EVENT_TYPE_TICKER,
		etherealv1.EventType_EVENT_TYPE_TRADE_FILL,
	}

	for _, event := range events {
		if err := client.SubscribeWithCallback(ctx, event, bitcoinSymbol, func(msg proto.Message) {
			fmt.Printf("[%s] %T %v\n", event.String(), msg, msg)
		}); err != nil {
			log.Fatalf("subscribe %s: %v", event.String(), err)
		}
	}

	subscriptionPayload, err := ws.Sub.MarshalEventData(etherealv1.EventType_EVENT_TYPE_TICKER, bitcoinSymbol)
	if err != nil {
		log.Fatalf("unable to build ticker subscription: %v", err)
	}
	fmt.Printf("ticker subscribe payload: %s\n", subscriptionPayload)

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
