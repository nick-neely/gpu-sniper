package ui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gpu-sniper/config"
)

// ProgressTracker monitors and updates progress on an ongoing operation
type ProgressTracker struct {
	startTime time.Time
	endTime   time.Time
	mu        sync.Mutex
	status    string        // Current status message
	interval  time.Duration // Current polling interval
}

// NewProgressTracker creates a new progress tracker with the given duration
func NewProgressTracker(duration time.Duration) *ProgressTracker {
	now := time.Now()
	return &ProgressTracker{
		startTime: now,
		endTime:   now.Add(duration),
		status:    "Waiting",
		interval:  duration,
	}
}

// UpdateStatus sets the current status message
func (pt *ProgressTracker) UpdateStatus(status string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.status = status
}

// UpdateEndTime updates the end time for the progress tracking
func (pt *ProgressTracker) UpdateEndTime(newEndTime time.Time) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.endTime = newEndTime
}

// UpdateDuration adds more time to the current end time
func (pt *ProgressTracker) UpdateDuration(newDuration time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	now := time.Now()
	pt.interval = newDuration
	
	// If countdown is already over, set a new end time from now
	if now.After(pt.endTime) {
		pt.startTime = now
		pt.endTime = now.Add(newDuration)
	} else {
		pt.endTime = now.Add(newDuration)
	}
}

// Remaining gets the remaining time until the end time
func (pt *ProgressTracker) Remaining() time.Duration {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	now := time.Now()
	if now.After(pt.endTime) {
		return 0
	}
	return pt.endTime.Sub(now)
}

// Elapsed gets the elapsed time since start
func (pt *ProgressTracker) Elapsed() time.Duration {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return time.Since(pt.startTime)
}

// Total gets the total duration
func (pt *ProgressTracker) Total() time.Duration {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.endTime.Sub(pt.startTime)
}

// IsComplete checks if the countdown is complete
func (pt *ProgressTracker) IsComplete() bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return time.Now().After(pt.endTime)
}

// GetStatus returns the current status message
func (pt *ProgressTracker) GetStatus() string {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.status
}

// GetInterval returns the current interval duration
func (pt *ProgressTracker) GetInterval() time.Duration {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.interval
}

// DrawProgressBar displays a progress bar for remaining time with enhanced status information
func DrawProgressBar(tracker *ProgressTracker) {
	// Calculate percentage
	elapsed := tracker.Elapsed()
	total := tracker.Total()
	percent := float64(elapsed) / float64(total)
	if percent > 1.0 {
		percent = 1.0
	}

	// Calculate filled width
	filled := int(percent * float64(config.ProgressWidth))

	// Format remaining time
	remaining := tracker.Remaining()
	remainingStr := fmt.Sprintf("%02d:%02d", int(remaining.Minutes()), int(remaining.Seconds())%60)

	// Draw progress bar
	bar := "["
	for i := 0; i < config.ProgressWidth; i++ {
		if i < filled {
			bar += "■"
		} else {
			bar += "·"
		}
	}
	bar += "]"

	// Clear line and print progress with additional status information
	fmt.Printf("\r\033[K") // Clear the current line
	
	// Get the check count from stock package
	checkCount := "0"
	if config.CheckCount > 0 {
		checkCount = fmt.Sprintf("%d", config.CheckCount)
	}
	
	// Get current interval formatted nicely
	interval := tracker.GetInterval()
	intervalStr := fmt.Sprintf("%ds", int(interval.Seconds()))
	if interval >= time.Minute {
		intervalStr = fmt.Sprintf("%dm%ds", int(interval.Minutes()), int(interval.Seconds())%60)
	}
	
	statusStr := tracker.GetStatus()
	
	// Print all the information in one line
	fmt.Printf("Status: %s | Checks: %s | Interval: %s | Next: %s %s", 
		config.InfoColor.Sprint(statusStr),
		config.WarningColor.Sprint(checkCount),
		config.InfoColor.Sprint(intervalStr),
		config.TimeColor.Sprint(remainingStr), 
		config.ProgressColor.Sprint(bar))
}

// Countdown manages a countdown timer with visual progress bar
func Countdown(ctx context.Context, duration time.Duration, done chan<- bool) *ProgressTracker {
	tracker := NewProgressTracker(duration)
	ticker := time.NewTicker(500 * time.Millisecond)
	
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if tracker.IsComplete() {
					tracker.UpdateStatus("Checking")
					// Clear the progress bar line
					fmt.Printf("\r\033[K")
					done <- true
					return
				}
				DrawProgressBar(tracker)
			}
		}
	}()
	
	return tracker
}
