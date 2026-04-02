// This package provides a WebSocket client for the Ethereal exchange API.
// It supports real-time streaming of market data and account events using Protocol Buffers.
// Some proto stubs are built for use with NATS, but integration is optional: examples using these are prefixed with "nats_".
package ethereal

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	etherealv1 "roundinternet.money/protos/gen/dex/ethereal/v1"
)

type Environment string

const (
	// Testnet is the WebSocket URL for the testnet environment.
	Testnet Environment = "wss://ws2.etherealtest.net/v1/stream"
	// Mainnet is the WebSocket URL for the mainnet environment.
	Mainnet Environment = "wss://ws2.ethereal.trade/v1/stream"
)

type Client struct {
	Con   *websocket.Conn
	conMu *sync.Mutex
	env   Environment

	streams []etherealv1.EventType

	l2bookCb                func(*etherealv1.L2Book)
	tickerCb                func(*etherealv1.Ticker)
	tradefillCb             func(*etherealv1.TradeFill)
	subaccountLiquidationCb func(*etherealv1.SubaccountLiquidation)
	positionUpdateCb        func(*etherealv1.PositionUpdate)
	orderUpdateCb           func(*etherealv1.OrderUpdate)
	orderFillCb             func(*etherealv1.OrderFill)
	tokenTransferCb         func(*etherealv1.TokenTransfer)

	hbCancel context.CancelCauseFunc
	pbOpts   *protojson.UnmarshalOptions
}

// NewClient creates a new WebSocket client connected to the specified environment.
// It establishes a connection and starts the keepalive mechanism.
func NewClient(parent context.Context, env Environment) *Client {
	ctx, cancel := context.WithCancelCause(parent)
	c, _, err := websocket.Dial(ctx, string(env), nil)
	if err != nil {
		log.Fatal(err)
	}

	cl := &Client{
		Con:   c,
		conMu: &sync.Mutex{},
		env:   env,

		streams: make([]etherealv1.EventType, 0),

		pbOpts: &protojson.UnmarshalOptions{DiscardUnknown: true},
	}

	cl.keepalive(ctx, cancel)
	cl.hbCancel = cancel

	return cl
}

// Subscribe sends a subscription request for the specified event type and symbol/subaccount.
func (c *Client) Subscribe(ctx context.Context, event EventType, to string) (err error) {
	var bytes []byte
	if bytes, err = Sub.MarshalEventData(event, to); err != nil {
		return
	}
	if err = c.Con.Write(ctx, websocket.MessageBinary, bytes); err != nil {
		c.streams = append(c.streams, event)
	}
	return
}

func (c *Client) Unsubscribe(ctx context.Context, event EventType, to string) (err error) {
	var bytes []byte
	if bytes, err = Unsub.MarshalEventData(event, to); err != nil {
		return err
	}
	return c.Con.Write(ctx, websocket.MessageBinary, bytes)
}

// Listen starts listening for incoming WebSocket messages and processes them.
// It blocks until the context is cancelled or an error occurs.
func (c *Client) Listen(parent context.Context) error {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	defer c.Close()

	for {
		_, data, err := c.Con.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return context.Cause(ctx)
			}
			cancel(err)
			return err
		}

		var e etherealv1.EventMessage
		if err := c.pbOpts.Unmarshal(data, &e); err != nil {
			if status := new(etherealv1.WebsocketStatus); c.pbOpts.Unmarshal(data, status) == nil {
				if !status.Status.Ok {
					return errors.New(status.String())
				}
			} else {
				return err
			}
			continue
		}

		event := etherealv1.EventType_json_value[e.E]

		switch event {
		case EventTypeL2Book:
			if c.l2bookCb != nil {
				var msg etherealv1.L2Book
				if err := c.pbOpts.Unmarshal(data, &msg); err != nil {
					cancel(err)
					return err
				}
				c.l2bookCb(&msg)
			}
		case EventTypeTicker:
			if c.tickerCb != nil {
				var msg etherealv1.Ticker
				if err := c.pbOpts.Unmarshal(data, &msg); err != nil {
					cancel(err)
					return err
				}
				c.tickerCb(&msg)
			}
		case EventTypeTradeFill:
			if c.tradefillCb != nil {
				var msg etherealv1.TradeFill
				if err := c.pbOpts.Unmarshal(data, &msg); err != nil {
					cancel(err)
					return err
				}
				c.tradefillCb(&msg)
			}
		case EventTypeSubaccountLiquidation:
			if c.subaccountLiquidationCb != nil {
				var msg etherealv1.SubaccountLiquidation
				if err := c.pbOpts.Unmarshal(data, &msg); err != nil {
					cancel(err)
					return err
				}
				c.subaccountLiquidationCb(&msg)
			}
		case EventTypePositionUpdate:
			if c.positionUpdateCb != nil {
				var msg etherealv1.PositionUpdate
				if err := c.pbOpts.Unmarshal(data, &msg); err != nil {
					cancel(err)
					return err
				}
				c.positionUpdateCb(&msg)
			}
		case EventTypeOrderUpdate:
			if c.orderUpdateCb != nil {
				var msg etherealv1.OrderUpdate
				if err := c.pbOpts.Unmarshal(data, &msg); err != nil {
					cancel(err)
					return err
				}
				c.orderUpdateCb(&msg)
			}
		case EventTypeOrderFill:
			if c.orderFillCb != nil {
				var msg etherealv1.OrderFill
				if err := c.pbOpts.Unmarshal(data, &msg); err != nil {
					cancel(err)
					return err
				}
				c.orderFillCb(&msg)
			}
		case EventTypeTokenTransfer:
			if c.tokenTransferCb != nil {
				var msg etherealv1.TokenTransfer
				if err := c.pbOpts.Unmarshal(data, &msg); err != nil {
					cancel(err)
					return err
				}
				c.tokenTransferCb(&msg)
			}
		}
	}
}

func (c *Client) OnL2Book(cb func(*etherealv1.L2Book)) {
	c.l2bookCb = cb
}

func (c *Client) OnTicker(cb func(*etherealv1.Ticker)) {
	c.tickerCb = cb
}

func (c *Client) OnTradeFill(cb func(*etherealv1.TradeFill)) {
	c.tradefillCb = cb
}

func (c *Client) OnSubaccountLiquidation(cb func(*etherealv1.SubaccountLiquidation)) {
	c.subaccountLiquidationCb = cb
}

func (c *Client) OnPositionUpdate(cb func(*etherealv1.PositionUpdate)) {
	c.positionUpdateCb = cb
}

func (c *Client) OnOrderUpdate(cb func(*etherealv1.OrderUpdate)) {
	c.orderUpdateCb = cb
}

func (c *Client) OnOrderFill(cb func(*etherealv1.OrderFill)) {
	c.orderFillCb = cb
}

func (c *Client) OnTokenTransfer(cb func(*etherealv1.TokenTransfer)) {
	c.tokenTransferCb = cb
}

func (c *Client) keepalive(ctx context.Context, cancel context.CancelCauseFunc) {
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				// Ping will return error if connection is dead
				if err := c.Con.Ping(ctx); err != nil {
					cancel(err)
					return
				}
			}
		}
	}()
}

func (c *Client) Resubscribe(parent context.Context) error {
	c.Close()

	c.conMu.Lock()
	defer c.conMu.Unlock()

	ctx, cancel := context.WithCancelCause(parent)

	// replace con and restart listener with new context
	var err error
	c.Con, _, err = websocket.Dial(ctx, string(c.env), nil)
	if err != nil {
		cancel(err)
		return err
	}
	c.hbCancel = cancel
	c.keepalive(ctx, cancel)

	return nil
}

// func (c *Client) UnsubscribeAll(ctx context.Context) (err error) {
// 	for _, s := range c.subscriptions {
// 		if err = c.Unsubscribe(ctx, s); err != nil {
// 			return err
// 		}
// 	}
// 	return
// }

func (c *Client) Close() {
	c.conMu.Lock()
	defer c.conMu.Unlock()
	if c.hbCancel != nil {
		c.hbCancel(nil)
	}
	if c.Con != nil {
		c.Con.Close(websocket.StatusNormalClosure, "closing")
	}
}
