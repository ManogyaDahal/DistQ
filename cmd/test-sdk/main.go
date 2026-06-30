package main

import (
	"context"

	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

func main() {

	// SDK client that talks to DistQ API
	sdk := redisclient.NewSDK("http://localhost:8080")

	ctx := context.Background()

	// Run every integration test
	RunAllTests(ctx, sdk)
}
