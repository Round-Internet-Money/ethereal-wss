package etherealWss

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	etherealv1 "roundinternet.money/protos/gen/dex/ethereal/v1"
)

type Environment string

const (
	Testnet Environment = "wss://ws2.etherealtest.net/v1/stream"
	Mainnet Environment = "wss://ws2.ethereal.trade/v1/stream"
)

type Client struct {
	Con   *websocket.Conn
	conMu *sync.Mutex
	env   Environment

	streams []etherealv1.EventType

	callbacks map[etherealv1.EventType]func(proto.Message)
	hbCancel  context.CancelCauseFunc
	pbOpts    *protojson.UnmarshalOptions
}

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

		callbacks: make(map[etherealv1.EventType]func(proto.Message)),
		pbOpts:    &protojson.UnmarshalOptions{DiscardUnknown: true},
	}

	cl.keepalive(ctx, cancel)
	cl.hbCancel = cancel

	return cl
}

func (c *Client) Subscribe(ctx context.Context, event etherealv1.EventType, to string) (err error) {
	var bytes []byte
	if bytes, err = Sub.MarshalEventData(event, to); err != nil {
		return
	}
	fmt.Println(string(bytes))
	if err = c.Con.Write(ctx, websocket.MessageBinary, bytes); err != nil {
		c.streams = append(c.streams, event)
	}
	return
}

func (c *Client) Unsubscribe(ctx context.Context, event etherealv1.EventType, to string) (err error) {
	var bytes []byte
	if bytes, err = Unsub.MarshalEventData(event, to); err != nil {
		return err
	}
	return c.Con.Write(ctx, websocket.MessageBinary, bytes)
}

func (c *Client) SubscribeWithCallback(ctx context.Context, event etherealv1.EventType, to string, cb func(proto.Message)) (err error) {
	if err = c.Subscribe(ctx, event, to); err == nil {
		c.callbacks[event] = cb
	}
	return
}

func (c *Client) OnEvent(event etherealv1.EventType, cb func(proto.Message)) {
	c.callbacks[event] = cb
}

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
			fmt.Println(string(data))
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

		fmt.Println(string(data))

		if cb, ok := c.callbacks[event]; ok {
			if msg, err := UnmarshalEvent(event, data); err != nil {
				cancel(err)
				return err
			} else {
				cb(msg)
			}
		}
	}
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
