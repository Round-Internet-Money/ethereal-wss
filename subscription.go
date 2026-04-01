// Package ethereal provides a WebSocket client for the Ethereal exchange API.
// It supports real-time streaming of market data and account events using Protocol Buffers.
package ethereal

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	etherealv1 "roundinternet.money/protos/gen/dex/ethereal/v1"
)

// EventType represents the type of event for subscriptions.
type EventType = etherealv1.EventType

// Event type constants for easy access without importing protos.
const (
	EventTypeL2Book                EventType = etherealv1.EventType_EVENT_TYPE_L2_BOOK
	EventTypeTicker                EventType = etherealv1.EventType_EVENT_TYPE_TICKER
	EventTypeTradeFill             EventType = etherealv1.EventType_EVENT_TYPE_TRADE_FILL
	EventTypeSubaccountLiquidation EventType = etherealv1.EventType_EVENT_TYPE_SUBACCOUNT_LIQUIDATION
	EventTypePositionUpdate        EventType = etherealv1.EventType_EVENT_TYPE_POSITION_UPDATE
	EventTypeOrderUpdate           EventType = etherealv1.EventType_EVENT_TYPE_ORDER_UPDATE
	EventTypeOrderFill             EventType = etherealv1.EventType_EVENT_TYPE_ORDER_FILL
	EventTypeTokenTransfer         EventType = etherealv1.EventType_EVENT_TYPE_TOKEN_TRANSFER
)

var ErrUnknownEvent = errors.New("unknown event")

type Intent string

const (
	Sub   Intent = "subscribe"
	Unsub Intent = "unsubscribe"
)

type WebsocketRequest struct {
	I Intent      `json:"event"`
	D interface{} `json:"data"`
}

func (i Intent) MarshalEventData(event EventType, to string) ([]byte, error) {
	var data interface{}
	switch event {
	case etherealv1.EventType_EVENT_TYPE_L2_BOOK,
		etherealv1.EventType_EVENT_TYPE_TICKER,
		etherealv1.EventType_EVENT_TYPE_TRADE_FILL:
		data = struct {
			T string `json:"type"`
			S string `json:"symbol"`
		}{etherealv1.EventType_json_name[event], to}
	case etherealv1.EventType_EVENT_TYPE_SUBACCOUNT_LIQUIDATION,
		etherealv1.EventType_EVENT_TYPE_POSITION_UPDATE,
		etherealv1.EventType_EVENT_TYPE_ORDER_UPDATE,
		etherealv1.EventType_EVENT_TYPE_ORDER_FILL,
		etherealv1.EventType_EVENT_TYPE_TOKEN_TRANSFER:
		data = struct {
			T string `json:"type"`
			S string `json:"subaccountId"`
		}{etherealv1.EventType_json_name[event], to}
	default:
		return nil, ErrUnknownEvent
	}
	return json.Marshal(WebsocketRequest{i, data})
}

func UnmarshalEvent(event EventType, data []byte) (proto.Message, error) {
	var m proto.Message
	switch event {
	case etherealv1.EventType_EVENT_TYPE_L2_BOOK:
		m = new(etherealv1.L2Book)
	case etherealv1.EventType_EVENT_TYPE_TICKER:
		m = new(etherealv1.Ticker)
	case etherealv1.EventType_EVENT_TYPE_TRADE_FILL:
		m = new(etherealv1.TradeFill)
	case etherealv1.EventType_EVENT_TYPE_SUBACCOUNT_LIQUIDATION:
		m = new(etherealv1.SubaccountLiquidation)
	case etherealv1.EventType_EVENT_TYPE_POSITION_UPDATE:
		m = new(etherealv1.PositionUpdate)
	case etherealv1.EventType_EVENT_TYPE_ORDER_UPDATE:
		m = new(etherealv1.OrderUpdate)
	case etherealv1.EventType_EVENT_TYPE_ORDER_FILL:
		m = new(etherealv1.OrderFill)
	case etherealv1.EventType_EVENT_TYPE_TOKEN_TRANSFER:
		m = new(etherealv1.TokenTransfer)
	default:
		return nil, ErrUnknownEvent
	}
	if err := protojson.Unmarshal(data, m); err != nil {
		return nil, err
	}
	return m, nil
}

// type Subscription[Proto proto.Message, Data EventData] struct {
// 	*websocket.Conn
// 	eventName string
// 	data      Proto
// 	Callback  func(Proto)

// 	eventType EventData
// }

// func NewSubscription[P proto.Message, D EventData](callback func(P)) *Subscription[P, D] {
// 	return &Subscription[P, D]{}
// }

// func (c *Subscription[_, EventData]) Subscribe(ctx context.Context) error {
// 	if bytes, err := json.Marshal(&SubscriptionIntent[EventData]{I: Sub, D: c.eventType}); err != nil {
// 		return err
// 	} else {
// 		return c.Write(ctx, websocket.MessageBinary, bytes)
// 	}
// }

// func (c *Subscription[_, EventData]) Unsubscribe(ctx context.Context) error {
// 	if bytes, err := json.Marshal(&SubscriptionIntent[EventData]{I: Unsub, D: c.eventType}); err != nil {
// 		return err
// 	} else {
// 		return c.Write(ctx, websocket.MessageBinary, bytes)
// 	}
// }
