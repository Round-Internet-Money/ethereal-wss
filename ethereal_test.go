package ethereal

import "testing"

func TestMarshalEventData(t *testing.T) {
	tests := []struct {
		intent Intent
		event  EventType
		to     string
		want   string
	}{
		{Sub, EventTypeTicker, "BTCUSD", `{"event":"subscribe","data":{"type":"Ticker","symbol":"BTCUSD"}}`},
		{Unsub, EventTypeL2Book, "ETHUSD", `{"event":"unsubscribe","data":{"type":"L2Book","symbol":"ETHUSD"}}`},
	}

	for _, tt := range tests {
		got, err := tt.intent.MarshalEventData(tt.event, tt.to)
		if err != nil {
			t.Errorf("MarshalEventData() error = %v", err)
			continue
		}
		if string(got) != tt.want {
			t.Errorf("MarshalEventData() = %v, want %v", string(got), tt.want)
		}
	}
}

func TestEnvironment(t *testing.T) {
	if string(Testnet) != "wss://ws2.etherealtest.net/v1/stream" {
		t.Errorf("Testnet URL incorrect")
	}
	if string(Mainnet) != "wss://ws2.ethereal.trade/v1/stream" {
		t.Errorf("Mainnet URL incorrect")
	}
}
