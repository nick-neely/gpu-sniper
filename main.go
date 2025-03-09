package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"gpu-sniper/alerts"
	"gpu-sniper/config"
	"gpu-sniper/stock"
	"gpu-sniper/ui"
)

func main() {
	// Display application header
	ui.PrintHeader()
	
	// Initialize audio system for alerts
	alerts.InitializeSoundSystem()
	
	// Setup graceful shutdown handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Use context for cancellation of countdown goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Add WaitGroup to track goroutines
	var wg sync.WaitGroup

	// Track current progress
	var currentTracker *ui.ProgressTracker

	// Handle shutdown signals
	go func() {
		<-signalChan
		fmt.Println("\nShutting down gracefully...")
		cancel() // Cancel the context
		// Wait for all goroutines to finish
		wg.Wait()
		os.Exit(0)
	}()

	// Run initial check
	if stock.CheckStock() {
		alerts.TriggerPurchase()
	}

	// Create channel for control flow
	done := make(chan bool)
	
	// Start first countdown with context
	wg.Add(1)
	go func() {
		currentTracker = ui.Countdown(ctx, config.PollingInterval, done)
		// Update the global tracker in the stock package
		stock.CurrentProgressTracker = currentTracker
		wg.Done()
	}()
	
	for range done {
		// Clear any residual progress bar
		fmt.Printf("\r\033[K")
		
		// Run the check and trigger purchase if in stock
		if stock.CheckStock() {
			alerts.TriggerPurchase()
		}
		fmt.Println(strings.Repeat("─", 50))
		
		// Start a new countdown cycle with jitter using the same context
		nextInterval := stock.GetNextPollingInterval()
		wg.Add(1)
		go func() {
			currentTracker = ui.Countdown(ctx, nextInterval, done)
			// Update the global tracker in the stock package
			stock.CurrentProgressTracker = currentTracker
			wg.Done()
		}()
	}
}
