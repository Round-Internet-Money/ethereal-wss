// example nats client using nats/micro and protoc-gen-nats
package main

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
	basev1 "roundinternet.money/protos/gen/dex/base/v1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	var con *nats.Conn
	if con, err = nats.Connect("nats://0.0.0.0:4222"); err != nil {
		return
	}

	client := basev1.NewReaderServiceNatsClient(con)

	var assets *basev1.DexAssetResponse
	if assets, err = client.DexAsset(ctx, &basev1.DexAssetRequest{
		Dexes:         []basev1.Dex{basev1.Dex_DEX_ETHEREAL},
		RelatedAssets: []string{},
	}); err != nil {
		panic(err)
	}

	log.Printf("  ← payload=%v", assets.A)

	var stream *basev1.ReaderService_Stream_ClientStream
	if stream, err = client.Stream(ctx, &basev1.StreamRequest{
		Dexes: []basev1.Dex{basev1.Dex_DEX_ETHEREAL},
		Kinds: []basev1.PayloadKind{basev1.PayloadKind_PAYLOAD_KIND_ASSET},
	}); err != nil {
		panic(err)
	}

	for {
		resp, err := stream.Recv(ctx)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Fatalf("recv failed: %v", err)
		}
		log.Printf("  ← kind=%v payload=%v", resp.Kind, resp.Payload)
	}
	log.Println("  ✓ Stream complete")
}
