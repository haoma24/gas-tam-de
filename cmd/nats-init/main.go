// Command nats-init ensures Gas Tam Đệ JetStream domain streams exist on a local NATS.
//
// Usage:
//
//	go run ./cmd/nats-init
//	NATS_URL=nats://127.0.0.1:4222 go run ./cmd/nats-init
package main

import (
	"fmt"
	"os"

	"gas-tam-de/pkg/config"
	"gas-tam-de/pkg/natsx"
)

func main() {
	url := config.Get("NATS_URL", "nats://127.0.0.1:4222")
	nc, js, err := natsx.ConnectJS(url)
	if err != nil {
		fail(err)
	}
	defer nc.Close()

	if err := natsx.PingJS(js); err != nil {
		fail(err)
	}
	if err := natsx.EnsureStreams(js); err != nil {
		fail(err)
	}

	fmt.Printf("NATS JetStream OK at %s\n", url)
	for _, s := range natsx.DomainStreams() {
		info, err := js.StreamInfo(s.Name)
		if err != nil {
			fail(err)
		}
		fmt.Printf("  stream %-10s subjects=%v messages=%d\n",
			info.Config.Name, info.Config.Subjects, info.State.Msgs)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "nats-init: %v\n", err)
	os.Exit(1)
}
