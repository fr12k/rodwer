package rodwer

import (
	"log"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	// Start Xvfb if no DISPLAY is set (e.g., in headless CI containers)
	if os.Getenv("DISPLAY") == "" {
		if _, err := exec.LookPath("Xvfb"); err == nil {
			xvfb := exec.Command("Xvfb", ":99", "-screen", "0", "1024x768x24", "-nolisten", "tcp")
			if err := xvfb.Start(); err == nil {
				if err := os.Setenv("DISPLAY", ":99"); err != nil {
					log.Printf("failed to set DISPLAY: %v", err)
				}
				defer func() {
					if err := xvfb.Process.Kill(); err != nil {
						log.Printf("failed to kill Xvfb: %v", err)
					}
				}()
			}
		}
	}

	os.Exit(m.Run())
}
