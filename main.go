package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lsherman98/yt-rss-cli/ui"
	"github.com/lsherman98/yt-rss-cli/updater"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	updated, err := updater.CheckAndUpdate(version)
	if err != nil {
		fmt.Printf("⚠️  Update check failed: %v\n", err)
		fmt.Println("Continuing with current version...")
	}
	if updated {
		os.Exit(0)
	}

	p := tea.NewProgram(ui.InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}
