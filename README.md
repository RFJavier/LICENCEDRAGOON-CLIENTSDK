# Dragoon Licence SDK (Go)

Go SDK client for Dragoon License Server.

## Install

```bash
go get github.com/RFJavier/dragoon-licence-sdk/license
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/RFJavier/dragoon-licence-sdk/license"
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

    sdk.OnGracePeriodStart(func() { log.Println("grace period started") })
    sdk.OnHeartbeatError(func(err error) { log.Printf("heartbeat error: %v", err) })
    sdk.OnBlocked(func() { log.Println("license blocked") })

    if _, err := sdk.Activate("LIC-DRAGOON-12345"); err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go sdk.StartHeartbeat(ctx)

    select {}
}
```

## Server API Contract

The SDK expects:

- POST /v1/license/activate
- POST /v1/license/heartbeat

Headers sent by SDK:

- X-CLIENT-ID
- X-CLIENT-SECRET
- X-TIMESTAMP
- X-NONCE

## Folder Structure

- license: public SDK package.
- examples: runnable integration sample.
- docs/api/openapi.yaml: official v1 HTTP contract for Go SDK.
- docs/api/signature-vectors.md: signature compatibility vectors.
- fixtures/signature/*.json: official conformance vectors used by tests.
- docs/FASE5_FASE6_GO.md: implementation guide for phase 5 and 6.
- docs/FASE8_GO_OFICIAL.md: phase 8 execution for Go-only release line.
- scripts/smoke_test.ps1: smoke build for Windows.
- scripts/smoke_test.sh: smoke build for Linux/macOS.
- CHANGELOG.md: release history.
- LICENSE: legal license.
- MAINTENANCE.md: repository maintenance and release guide.

## Versioning

Semantic versioning is used:

- v0.9.x beta line.
- v1.0.0 first stable public release.

## Conformance

Phase 8 introduces official fixture-based conformance tests for Go:

- `license/phase8_conformance_test.go`
- `fixtures/signature/activate.json`
- `fixtures/signature/heartbeat.json`
