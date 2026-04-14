package ethereal

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	etherealv1 "roundinternet.money/protos/gen/dex/ethereal/v1"
)

func TestIntent_MarshalEventData(t *testing.T) {
	sym := "BTC-PERP"
	subAcct := "sub-123"

	tests := []struct {
		name      string
		intent    Intent
		event     EventType
		to        string
		wantErr   error
		wantEvent string
		wantType  string
		wantSym   string // non-empty => expect "symbol" in data
		wantSub   string // non-empty => expect "subaccountId" in data
	}{
		{
			name: "subscribe_L2Book_symbol", intent: Sub, event: EventTypeL2Book, to: sym,
			wantEvent: "subscribe", wantType: etherealv1.EventType_json_name[EventTypeL2Book], wantSym: sym,
		},
		{
			name: "subscribe_Ticker_symbol", intent: Sub, event: EventTypeTicker, to: sym,
			wantEvent: "subscribe", wantType: etherealv1.EventType_json_name[EventTypeTicker], wantSym: sym,
		},
		{
			name: "subscribe_TradeFill_symbol", intent: Sub, event: EventTypeTradeFill, to: sym,
			wantEvent: "subscribe", wantType: etherealv1.EventType_json_name[EventTypeTradeFill], wantSym: sym,
		},
		{
			name: "subscribe_SubaccountLiquidation_subaccount", intent: Sub, event: EventTypeSubaccountLiquidation, to: subAcct,
			wantEvent: "subscribe", wantType: etherealv1.EventType_json_name[EventTypeSubaccountLiquidation], wantSub: subAcct,
		},
		{
			name: "subscribe_PositionUpdate_subaccount", intent: Sub, event: EventTypePositionUpdate, to: subAcct,
			wantEvent: "subscribe", wantType: etherealv1.EventType_json_name[EventTypePositionUpdate], wantSub: subAcct,
		},
		{
			name: "subscribe_OrderUpdate_subaccount", intent: Sub, event: EventTypeOrderUpdate, to: subAcct,
			wantEvent: "subscribe", wantType: etherealv1.EventType_json_name[EventTypeOrderUpdate], wantSub: subAcct,
		},
		{
			name: "subscribe_OrderFill_subaccount", intent: Sub, event: EventTypeOrderFill, to: subAcct,
			wantEvent: "subscribe", wantType: etherealv1.EventType_json_name[EventTypeOrderFill], wantSub: subAcct,
		},
		{
			name: "subscribe_TokenTransfer_subaccount", intent: Sub, event: EventTypeTokenTransfer, to: subAcct,
			wantEvent: "subscribe", wantType: etherealv1.EventType_json_name[EventTypeTokenTransfer], wantSub: subAcct,
		},
		{
			name: "unsubscribe_L2Book_symbol", intent: Unsub, event: EventTypeL2Book, to: sym,
			wantEvent: "unsubscribe", wantType: etherealv1.EventType_json_name[EventTypeL2Book], wantSym: sym,
		},
		{
			name: "unknown_event", intent: Sub,
			event:   etherealv1.EventType_EVENT_TYPE_EVENT_UNSPECIFIED,
			to:      sym,
			wantErr: ErrUnknownEvent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.intent.MarshalEventData(tt.event, tt.to)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("MarshalEventData err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("MarshalEventData: %v", err)
			}

			var wire struct {
				Event string          `json:"event"`
				Data  json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(got, &wire); err != nil {
				t.Fatalf("json.Unmarshal wire: %v", err)
			}
			if wire.Event != tt.wantEvent {
				t.Errorf("event field = %q, want %q", wire.Event, tt.wantEvent)
			}

			var data struct {
				Type          string `json:"type"`
				Symbol        string `json:"symbol"`
				SubaccountID  string `json:"subaccountId"`
				WantNoSymbol  bool   `json:"-"`
				WantNoSubacct bool   `json:"-"`
			}
			if err := json.Unmarshal(wire.Data, &data); err != nil {
				t.Fatalf("json.Unmarshal data: %v", err)
			}
			if data.Type != tt.wantType {
				t.Errorf("data.type = %q, want %q", data.Type, tt.wantType)
			}
			if tt.wantSym != "" {
				if data.Symbol != tt.wantSym {
					t.Errorf("data.symbol = %q, want %q", data.Symbol, tt.wantSym)
				}
				if data.SubaccountID != "" {
					t.Errorf("data.subaccountId = %q, want empty for symbol events", data.SubaccountID)
				}
			}
			if tt.wantSub != "" {
				if data.SubaccountID != tt.wantSub {
					t.Errorf("data.subaccountId = %q, want %q", data.SubaccountID, tt.wantSub)
				}
				if data.Symbol != "" {
					t.Errorf("data.symbol = %q, want empty for subaccount events", data.Symbol)
				}
			}
		})
	}
}

func TestUnmarshalEvent(t *testing.T) {
	opts := protojson.MarshalOptions{}
	mustJSON := func(m proto.Message) []byte {
		t.Helper()
		b, err := opts.Marshal(m)
		if err != nil {
			t.Fatalf("marshal fixture: %v", err)
		}
		return b
	}

	tests := []struct {
		name       string
		event      EventType
		data       []byte
		want       proto.Message
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "L2Book",
			event: EventTypeL2Book,
			data: mustJSON(&etherealv1.L2Book{
				E: etherealv1.EventType_json_name[EventTypeL2Book],
				T: 1,
				Data: &etherealv1.L2Book_Data{
					S: "s",
					T: 1,
				},
			}),
			want: &etherealv1.L2Book{},
		},
		{
			name: "Ticker",
			event: EventTypeTicker,
			data: mustJSON(&etherealv1.Ticker{
				E: etherealv1.EventType_json_name[EventTypeTicker],
				T: 1,
				Data: &etherealv1.Ticker_Data{
					S: "s",
					T: 1,
				},
			}),
			want: &etherealv1.Ticker{},
		},
		{
			name: "TradeFill",
			event: EventTypeTradeFill,
			data: mustJSON(&etherealv1.TradeFill{
				E: etherealv1.EventType_json_name[EventTypeTradeFill],
				T: 1,
				Data: &etherealv1.TradeFill_Data{
					S: "s",
					T: 1,
				},
			}),
			want: &etherealv1.TradeFill{},
		},
		{
			name: "SubaccountLiquidation",
			event: EventTypeSubaccountLiquidation,
			data: mustJSON(&etherealv1.SubaccountLiquidation{
				E: etherealv1.EventType_json_name[EventTypeSubaccountLiquidation],
				T: 1,
				Data: &etherealv1.SubaccountLiquidation_Data{
					Sid: "sid",
					T:   1,
				},
			}),
			want: &etherealv1.SubaccountLiquidation{},
		},
		{
			name: "PositionUpdate",
			event: EventTypePositionUpdate,
			data: mustJSON(&etherealv1.PositionUpdate{
				E: etherealv1.EventType_json_name[EventTypePositionUpdate],
				T: 1,
				Data: &etherealv1.PositionUpdate_Data{
					T: 1,
				},
			}),
			want: &etherealv1.PositionUpdate{},
		},
		{
			name: "OrderUpdate",
			event: EventTypeOrderUpdate,
			data: mustJSON(&etherealv1.OrderUpdate{
				E: etherealv1.EventType_json_name[EventTypeOrderUpdate],
				T: 1,
				Data: &etherealv1.OrderUpdate_Data{
					T: 1,
				},
			}),
			want: &etherealv1.OrderUpdate{},
		},
		{
			name: "OrderFill",
			event: EventTypeOrderFill,
			data: mustJSON(&etherealv1.OrderFill{
				E: etherealv1.EventType_json_name[EventTypeOrderFill],
				T: 1,
				Data: &etherealv1.OrderFill_Data{
					T: 1,
				},
			}),
			want: &etherealv1.OrderFill{},
		},
		{
			name: "TokenTransfer",
			event: EventTypeTokenTransfer,
			data: mustJSON(&etherealv1.TokenTransfer{
				E: etherealv1.EventType_json_name[EventTypeTokenTransfer],
				T: 1,
				Data: &etherealv1.TokenTransfer_Data{
					Id:  "id",
					Sid: "sid",
				},
			}),
			want: &etherealv1.TokenTransfer{},
		},
		{
			name:       "bad_json",
			event:      EventTypeL2Book,
			data:       []byte(`{`),
			wantAnyErr: true,
		},
		{
			name:    "unknown_event",
			event:   EventType(9999),
			data:    []byte(`{}`),
			wantErr: ErrUnknownEvent,
		},
		{
			name:    "unspecified_enum",
			event:   etherealv1.EventType_EVENT_TYPE_EVENT_UNSPECIFIED,
			data:    []byte(`{}`),
			wantErr: ErrUnknownEvent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalEvent(tt.event, tt.data)
			if tt.wantAnyErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if tt.wantErr != nil {
				if tt.wantErr == ErrUnknownEvent {
					if !errors.Is(err, ErrUnknownEvent) {
						t.Fatalf("err = %v, want ErrUnknownEvent", err)
					}
					return
				}
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalEvent: %v", err)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Fatalf("got type %T, want %T", got, tt.want)
			}
		})
	}
}
