package ethereal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	etherealv1 "roundinternet.money/protos/gen/dex/ethereal/v1"
)

func newTestClient(conn *websocket.Conn) *Client {
	return &Client{
		Con:   conn,
		conMu: &sync.Mutex{},
		pbOpts: &protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

func TestNewTestClient_ProtojsonDiscardUnknown(t *testing.T) {
	c := newTestClient(nil)
	if c.pbOpts == nil || !c.pbOpts.DiscardUnknown {
		t.Fatal("expected protojson UnmarshalOptions with DiscardUnknown: true")
	}
}

// wsTestPair returns a client *websocket.Conn (for the Client under test), a server
// *websocket.Conn to inject frames, and cleanup which closes resources.
func wsTestPair(t *testing.T) (clientConn, serverConn *websocket.Conn, cleanup func()) {
	t.Helper()

	serverCh := make(chan *websocket.Conn, 1)
	done := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("Accept: %v", err)
			return
		}
		serverCh <- c
		<-done
		_ = c.Close(websocket.StatusNormalClosure, "")
	}))

	cleanup = func() {
		close(done)
		srv.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		cleanup()
		t.Fatalf("Dial: %v", err)
	}

	select {
	case serverConn = <-serverCh:
	case <-time.After(3 * time.Second):
		cleanup()
		t.Fatal("timeout waiting for server websocket")
	}

	prevCleanup := cleanup
	cleanup = func() {
		_ = clientConn.Close(websocket.StatusNormalClosure, "")
		prevCleanup()
	}

	return clientConn, serverConn, cleanup
}

func TestSubscribe_Unsubscribe_MessageBinary(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	ctx := context.Background()

	if err := c.Subscribe(ctx, EventTypeL2Book, "BTC-PERP"); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	typ, data, err := serverConn.Read(ctx)
	if err != nil {
		t.Fatalf("server Read: %v", err)
	}
	if typ != websocket.MessageBinary {
		t.Fatalf("message type = %v, want MessageBinary", typ)
	}
	var subWire struct {
		Event string `json:"event"`
		Data  struct {
			Type   string `json:"type"`
			Symbol string `json:"symbol"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &subWire); err != nil {
		t.Fatalf("json: %v", err)
	}
	if subWire.Event != "subscribe" || subWire.Data.Type != "L2Book" || subWire.Data.Symbol != "BTC-PERP" {
		t.Fatalf("unexpected subscribe payload: %+v", subWire)
	}

	if err := c.Unsubscribe(ctx, EventTypeOrderUpdate, "sub-9"); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}
	typ, data, err = serverConn.Read(ctx)
	if err != nil {
		t.Fatalf("server Read 2: %v", err)
	}
	if typ != websocket.MessageBinary {
		t.Fatalf("message type 2 = %v, want MessageBinary", typ)
	}
	var unsubWire struct {
		Event string `json:"event"`
		Data  struct {
			Type         string `json:"type"`
			SubaccountID string `json:"subaccountId"`
			Symbol       string `json:"symbol"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &unsubWire); err != nil {
		t.Fatalf("json 2: %v", err)
	}
	if unsubWire.Event != "unsubscribe" || unsubWire.Data.Type != "OrderUpdate" || unsubWire.Data.SubaccountID != "sub-9" || unsubWire.Data.Symbol != "" {
		t.Fatalf("unexpected unsubscribe payload: %+v", unsubWire)
	}
}

var pbMarshaler = protojson.MarshalOptions{}

func mustProtoJSON(t *testing.T, m proto.Message) []byte {
	t.Helper()
	b, err := pbMarshaler.Marshal(m)
	if err != nil {
		t.Fatalf("protojson.Marshal: %v", err)
	}
	return b
}

func TestListen_DispatchEachEventType(t *testing.T) {
	cases := []struct {
		name string
		send proto.Message
		reg  func(*Client, *atomic.Int32)
	}{
		{
			name: "L2Book",
			send: &etherealv1.L2Book{
				E: etherealv1.EventType_json_name[EventTypeL2Book],
				T: 42,
				Data: &etherealv1.L2Book_Data{
					S: "sym",
					T: 1,
				},
			},
			reg: func(c *Client, n *atomic.Int32) {
				c.OnL2Book(func(b *etherealv1.L2Book) {
					if b.GetE() == etherealv1.EventType_json_name[EventTypeL2Book] && b.GetData().GetS() == "sym" {
						n.Add(1)
					}
				})
			},
		},
		{
			name: "Ticker",
			send: &etherealv1.Ticker{
				E: etherealv1.EventType_json_name[EventTypeTicker],
				T: 1,
				Data: &etherealv1.Ticker_Data{
					S: "sym",
					T: 1,
				},
			},
			reg: func(c *Client, n *atomic.Int32) {
				c.OnTicker(func(x *etherealv1.Ticker) {
					if x.GetData().GetS() == "sym" {
						n.Add(1)
					}
				})
			},
		},
		{
			name: "TradeFill",
			send: &etherealv1.TradeFill{
				E: etherealv1.EventType_json_name[EventTypeTradeFill],
				T: 1,
				Data: &etherealv1.TradeFill_Data{
					S: "sym",
					T: 1,
				},
			},
			reg: func(c *Client, n *atomic.Int32) {
				c.OnTradeFill(func(x *etherealv1.TradeFill) {
					if x.GetData().GetS() == "sym" {
						n.Add(1)
					}
				})
			},
		},
		{
			name: "SubaccountLiquidation",
			send: &etherealv1.SubaccountLiquidation{
				E: etherealv1.EventType_json_name[EventTypeSubaccountLiquidation],
				T: 1,
				Data: &etherealv1.SubaccountLiquidation_Data{
					Sid: "sid",
					T:   1,
				},
			},
			reg: func(c *Client, n *atomic.Int32) {
				c.OnSubaccountLiquidation(func(x *etherealv1.SubaccountLiquidation) {
					if x.GetData().GetSid() == "sid" {
						n.Add(1)
					}
				})
			},
		},
		{
			name: "PositionUpdate",
			send: &etherealv1.PositionUpdate{
				E: etherealv1.EventType_json_name[EventTypePositionUpdate],
				T: 1,
				Data: &etherealv1.PositionUpdate_Data{
					T: 1,
				},
			},
			reg: func(c *Client, n *atomic.Int32) {
				c.OnPositionUpdate(func(x *etherealv1.PositionUpdate) {
					n.Add(1)
				})
			},
		},
		{
			name: "OrderUpdate",
			send: &etherealv1.OrderUpdate{
				E: etherealv1.EventType_json_name[EventTypeOrderUpdate],
				T: 1,
				Data: &etherealv1.OrderUpdate_Data{
					T: 1,
				},
			},
			reg: func(c *Client, n *atomic.Int32) {
				c.OnOrderUpdate(func(x *etherealv1.OrderUpdate) {
					n.Add(1)
				})
			},
		},
		{
			name: "OrderFill",
			send: &etherealv1.OrderFill{
				E: etherealv1.EventType_json_name[EventTypeOrderFill],
				T: 1,
				Data: &etherealv1.OrderFill_Data{
					T: 1,
				},
			},
			reg: func(c *Client, n *atomic.Int32) {
				c.OnOrderFill(func(x *etherealv1.OrderFill) {
					n.Add(1)
				})
			},
		},
		{
			name: "TokenTransfer",
			send: &etherealv1.TokenTransfer{
				E: etherealv1.EventType_json_name[EventTypeTokenTransfer],
				T: 1,
				Data: &etherealv1.TokenTransfer_Data{
					Id:  "id1",
					Sid: "sid",
				},
			},
			reg: func(c *Client, n *atomic.Int32) {
				c.OnTokenTransfer(func(x *etherealv1.TokenTransfer) {
					if x.GetData().GetId() == "id1" {
						n.Add(1)
					}
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clientConn, serverConn, cleanup := wsTestPair(t)
			defer cleanup()

			var hits atomic.Int32
			c := newTestClient(clientConn)
			tc.reg(c, &hits)

			payload := mustProtoJSON(t, tc.send)
			ctx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error, 1)
			go func() { errCh <- c.Listen(ctx) }()

			if err := serverConn.Write(ctx, websocket.MessageText, payload); err != nil {
				t.Fatalf("server write: %v", err)
			}
			waitFor(t, func() bool { return hits.Load() >= 1 }, 2*time.Second, "callback")
			cancel()
			err := <-errCh
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("Listen: %v", err)
			}
			if hits.Load() != 1 {
				t.Fatalf("callback hits = %d, want 1", hits.Load())
			}
		})
	}
}

func TestListen_NilCallback_NoPanic(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	// no OnL2Book registered

	payload := mustProtoJSON(t, &etherealv1.L2Book{
		E: etherealv1.EventType_json_name[EventTypeL2Book],
		T: 1,
		Data: &etherealv1.L2Book_Data{
			S: "s",
			T: 1,
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	if err := serverConn.Write(ctx, websocket.MessageText, payload); err != nil {
		t.Fatalf("server write: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	cancel()
	err := <-errCh
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Listen: %v", err)
	}
}

func waitFor(t *testing.T, cond func() bool, d time.Duration, what string) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func TestListen_WebsocketStatus_okFalse_ReturnsError(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	// EventMessage fails (e is not a string); WebsocketStatus succeeds with ok false.
	payload := []byte(`{"e":1,"status":{"ok":false}}`)

	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	if err := serverConn.Write(ctx, websocket.MessageText, payload); err != nil {
		t.Fatalf("server write: %v", err)
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected error from WebsocketStatus with ok false")
	}
}

func TestListen_WebsocketStatus_okTrue_Continues(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	var hits atomic.Int32
	c.OnL2Book(func(b *etherealv1.L2Book) { hits.Add(1) })

	// EventMessage fails; WebsocketStatus ok true -> continue; then valid L2Book.
	statusThenBook := []byte(`{"e":1,"status":{"ok":true}}`)
	l2 := mustProtoJSON(t, &etherealv1.L2Book{
		E: etherealv1.EventType_json_name[EventTypeL2Book],
		T: 1,
		Data: &etherealv1.L2Book_Data{S: "x", T: 1},
	})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	if err := serverConn.Write(ctx, websocket.MessageText, statusThenBook); err != nil {
		t.Fatalf("server write status: %v", err)
	}
	if err := serverConn.Write(ctx, websocket.MessageText, l2); err != nil {
		t.Fatalf("server write l2: %v", err)
	}
	waitFor(t, func() bool { return hits.Load() >= 1 }, 2*time.Second, "L2Book after status")
	cancel()
	<-errCh
	if hits.Load() != 1 {
		t.Fatalf("expected L2Book callback after status ok, hits=%d", hits.Load())
	}
}

func TestListen_MalformedJSON_ReturnsError(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	if err := serverConn.Write(ctx, websocket.MessageText, []byte(`not json`)); err != nil {
		t.Fatalf("server write: %v", err)
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListen_PayloadUnmarshalError_Returns(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	c.OnL2Book(func(*etherealv1.L2Book) { t.Error("should not invoke callback on bad payload") })

	// Valid EventMessage discriminator but invalid L2Book (t must be int).
	payload := []byte(`{"e":"L2Book","t":"nope"}`)

	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	if err := serverConn.Write(ctx, websocket.MessageText, payload); err != nil {
		t.Fatalf("server write: %v", err)
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestListen_DiscardUnknown_ExtraKeys(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	var hits atomic.Int32
	c := newTestClient(clientConn)
	c.OnL2Book(func(*etherealv1.L2Book) { hits.Add(1) })

	raw := string(mustProtoJSON(t, &etherealv1.L2Book{
		E: etherealv1.EventType_json_name[EventTypeL2Book],
		T: 1,
		Data: &etherealv1.L2Book_Data{
			S: "s",
			T: 1,
		},
	}))
	payload := []byte(`{"zzzUnknown":true,` + raw[1:]) // prepend unknown key to JSON object

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	if err := serverConn.Write(ctx, websocket.MessageText, payload); err != nil {
		t.Fatalf("server write: %v", err)
	}
	waitFor(t, func() bool { return hits.Load() >= 1 }, 2*time.Second, "DiscardUnknown callback")
	cancel()
	<-errCh
	if hits.Load() != 1 {
		t.Fatalf("expected callback with DiscardUnknown, hits=%d", hits.Load())
	}
}

func TestListen_ReadError_Returns(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	_ = serverConn.Close(websocket.StatusGoingAway, "bye")

	err := <-errCh
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestListen_ContextCancel(t *testing.T) {
	clientConn, _, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	cancel()
	err := <-errCh
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancel cause, got %v", err)
	}
}

func TestListen_UnhandledEventString_ContinuesUntilCancel(t *testing.T) {
	clientConn, serverConn, cleanup := wsTestPair(t)
	defer cleanup()

	c := newTestClient(clientConn)
	var hits atomic.Int32
	c.OnL2Book(func(*etherealv1.L2Book) { hits.Add(1) })

	// Valid JSON for EventMessage with unknown e — not in EventType_json_value as a key? "bogus" missing -> map lookup 0, no switch arm.
	payload := []byte(`{"e":"bogus"}`)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Listen(ctx) }()

	if err := serverConn.Write(ctx, websocket.MessageText, payload); err != nil {
		t.Fatalf("server write: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if hits.Load() != 0 {
		t.Fatal("did not expect L2Book callback")
	}
	cancel()
	<-errCh
}
