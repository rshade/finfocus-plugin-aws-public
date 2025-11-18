package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rshade/pulumicost-core/pkg/pluginsdk"
	"github.com/rshade/pulumicost-plugin-aws-public/internal/plugin"
	"github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
)

func main() {
	// Initialize pricing client
	pricingClient, err := pricing.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Failed to initialize pricing client: %v\n", err)
		os.Exit(1)
	}
	region := pricingClient.Region()

	// Create plugin instance
	awsPlugin := plugin.NewAWSPublicPlugin(region, pricingClient)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Received shutdown signal\n")
		cancel()
	}()

	// Serve using pluginsdk
	config := pluginsdk.ServeConfig{
		Plugin: awsPlugin,
		Port:   0, // Use PORT env var or random port
	}
	if err := pluginsdk.Serve(ctx, config); err != nil {
		fmt.Fprintf(os.Stderr, "[pulumicost-plugin-aws-public] Error: %v\n", err)
		os.Exit(1)
	}
}
