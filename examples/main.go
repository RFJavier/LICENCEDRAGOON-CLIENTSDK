package main

import (
	"context"
	"log"
	"time"

	"github.com/RFJavier/LICENCEDRAGOON-CLIENTSDK/sdk"
)

func main() {
	sdk, err := license.New(license.Config{
		APIURL:       "http://localhost:8080",
		ClientID:     "your-client-id",
		ClientSecret: "your-client-secret",
		Interval:     30 * time.Second,
		Timeout:      10 * time.Second,
		StoragePath:  ".license/state.json",
		PublicKey:    "your-ed25519-public-key-hex-64-chars",
		GracePeriod:  10 * time.Minute,
		MaxRetries:   3,
	})
	if err != nil {
		log.Fatal(err)
	}

	sdk.OnGracePeriodStart(func() {
		log.Println("grace period started: using cached validation")
	})
	sdk.OnHeartbeatError(func(err error) {
		log.Printf("heartbeat error: %v\n", err)
	})
	sdk.OnBlocked(func() {
		log.Println("license blocked or grace period expired")
	})

	if _, err := sdk.Activate("LIC-DRAGOON-12345"); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sdk.StartHeartbeat(ctx)

	select {}
}
