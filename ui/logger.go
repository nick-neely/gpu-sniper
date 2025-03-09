package ui

import (
	"fmt"
	"strings"
	"time"

	"gpu-sniper/config"
)

// LogInfo logs an informational message with timestamp
func LogInfo(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	timeStr := config.TimeColor.Sprintf("[%s]", timestamp)
	prefix := config.InfoColor.Sprint("INFO  ")
	fmt.Printf("%s %s %s\n", timeStr, prefix, fmt.Sprintf(format, args...))
}

// LogSuccess logs a success message with timestamp
func LogSuccess(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	timeStr := config.TimeColor.Sprintf("[%s]", timestamp)
	prefix := config.SuccessColor.Sprint("OK    ")
	fmt.Printf("%s %s %s\n", timeStr, prefix, fmt.Sprintf(format, args...))
}

// LogError logs an error message with timestamp
func LogError(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	timeStr := config.TimeColor.Sprintf("[%s]", timestamp)
	prefix := config.ErrorColor.Sprint("ERROR ")
	fmt.Printf("%s %s %s\n", timeStr, prefix, fmt.Sprintf(format, args...))
}

// LogWarning logs a warning message with timestamp
func LogWarning(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	timeStr := config.TimeColor.Sprintf("[%s]", timestamp)
	prefix := config.WarningColor.Sprint("WARN  ")
	fmt.Printf("%s %s %s\n", timeStr, prefix, fmt.Sprintf(format, args...))
}

// PrintHeader prints the application header with styling
func PrintHeader() {
	fmt.Println()
	config.HeaderColor.Printf("🔍 GPU SNIPER - Monitoring for %s\n", config.TargetGPU)
	config.HeaderColor.Printf("🔗 Retailer URL: %s\n", config.RetailerURL)
	config.HeaderColor.Printf("💻 By: nick-neely (github)\n")
	config.HeaderColor.Printf("⏱️  Default check interval: %s (adjusts automatically)\n", config.DefaultPollingInterval)
	config.HeaderColor.Printf("🛡️  Anti-bot measures: Random user agents, jittered timing, related page visits\n")
	fmt.Println(strings.Repeat("═", 50))
}

// PrintStatusUpdate prints a periodic status update with current program state
func PrintStatusUpdate() {
	if config.CheckCount > 0 {
		timeSinceLastCheck := time.Since(config.LastCheckTime)
		fmt.Printf("\r\033[K") // Clear the current line
		fmt.Printf("[STATUS] Checks: %d | Current interval: %v | Last check: %v ago\n",
			config.CheckCount, 
			config.PollingInterval,
			timeSinceLastCheck.Round(time.Second))
	}
}
