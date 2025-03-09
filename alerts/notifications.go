package alerts

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/fatih/color"

	"gpu-sniper/config"
	"gpu-sniper/ui"
)

// PlaySound plays a WAV file and returns after it's finished
func PlaySound(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	streamer, _, err := wav.Decode(f)
	if err != nil {
		return err
	}
	defer streamer.Close()

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() { done <- true })))
	<-done
	return nil
}

// OpenURL attempts to open the provided URL in the default browser.
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// For Windows, use rundll32 to open the URL
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		// For macOS
		cmd = exec.Command("open", url)
	default:
		// For Linux and other OSes
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// TriggerPurchase performs all actions when a product is detected in stock
func TriggerPurchase() {
	// Construct the direct add-to-cart link using the productID and AssociateTag
	addToCartURL := "https://www.amazon.com/gp/aws/cart/add-res.html?ASIN.1=" + config.ProductID + "&Quantity.1=1&AssociateTag=nisdisatc-20"

	// Immediately display the alert and URL
	alertMsg := color.New(color.FgHiGreen, color.Bold).Sprintf("🚨 ALERT: %s IS IN STOCK! 🚨", config.TargetGPU)
	addToCartMsg := color.New(color.FgHiYellow, color.Bold).Sprintf("Direct Add-to-Cart: %s", addToCartURL)
	fmt.Println("\n" + alertMsg)
	fmt.Println(addToCartMsg + "\n")

	// Automatically open the URL in the default browser
	if err := OpenURL(addToCartURL); err != nil {
		ui.LogError("Failed to open add-to-cart link: %v", err)
	}

	// Play sound alerts in a goroutine
	go func() {
		const soundFile = "beep.wav" // Ensure this file exists in your project directory
		for i := 0; i < 3; i++ {
			if err := PlaySound(soundFile); err != nil {
				ui.LogError("Failed to play alert sound: %v", err)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
}

// InitializeSoundSystem prepares the audio system for alerts
func InitializeSoundSystem() bool {
	ui.LogInfo("Initializing audio system...")
	f, err := os.Open("beep.wav")
	if err != nil {
		ui.LogError("Failed to open sound file for initialization: %v", err)
		return false
	}
	defer f.Close()
	
	streamer, format, err := wav.Decode(f)
	if err != nil {
		ui.LogError("Failed to decode sound file: %v", err)
		return false
	}
	defer streamer.Close()
	
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		ui.LogError("Failed to initialize audio system: %v", err)
		return false
	}
	ui.LogSuccess("Audio system initialized")
	return true
}
