package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lsherman98/yt-rss-cli/api"
)

func CheckAPIKey() tea.Msg {
	_, err := api.GetApiKey()
	return ApiKeyCheckedMsg{HasKey: err == nil}
}

func LoadPodcasts() tea.Msg {
	podcasts, err := api.ListPodcasts()
	return PodcastsLoadedMsg{Podcasts: podcasts, Err: err}
}

func AddURL(podcastID, url string) tea.Cmd {
	return func() tea.Msg {
		item, err := api.AddUrlToPodcast(podcastID, url)
		return UrlAddedMsg{Item: item, Err: err}
	}
}

func LoadItems(podcastID string) tea.Cmd {
	return func() tea.Msg {
		items, err := api.GetPodcastItems(podcastID)
		return ItemsLoadedMsg{Items: items, Err: err}
	}
}

func LoadUsage() tea.Cmd {
	return func() tea.Msg {
		usage, err := api.GetUsage()
		return UsageLoadedMsg{Usage: usage, Err: err}
	}
}

func parseCreatedTime(created string) time.Time {
	if created == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999Z",
		"2006-01-02 15:04:05Z",
		"2006-01-02 15:04:05.999Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, created); err == nil {
			return t
		}
	}

	return time.Time{}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func formatBytes(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	if bytes >= GB {
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	} else if bytes >= MB {
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	} else if bytes >= KB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	}
	return fmt.Sprintf("%d B", bytes)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *Model) buildItemsTable() {
	columns := []table.Column{
		{Title: "Title", Width: 60},
		{Title: "Status", Width: 20},
		{Title: "Created", Width: 30},
	}

	sortedItems := make([]api.Item, len(m.Items))
	copy(sortedItems, m.Items)
	sort.Slice(sortedItems, func(i, j int) bool {
		timeI := parseCreatedTime(sortedItems[i].Created)
		timeJ := parseCreatedTime(sortedItems[j].Created)

		if timeI.IsZero() && timeJ.IsZero() {
			return false
		}
		if timeI.IsZero() {
			return false
		}
		if timeJ.IsZero() {
			return true
		}

		return timeI.After(timeJ)
	})

	rows := []table.Row{}
	for _, item := range sortedItems {
		status := item.Status
		switch item.Status {
		case "CREATED":
			status = m.Spinner.View() + " PROCESSING"
		case "ERROR":
			status = "❌ ERROR"
		case "SUCCESS":
			status = "✓ SUCCESS"
		}

		title := item.Title
		if title == "" {
			if item.Status == "CREATED" {
				title = "Processing..."
			} else {
				title = "(No title)"
			}
		}

		created := item.Created
		if created != "" {
			t := parseCreatedTime(created)
			if !t.IsZero() {
				created = t.Local().Format("Jan 2, 2006 3:04 PM")
			}
		} else {
			created = "-"
		}

		rows = append(rows, table.Row{title, status, created})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(min(len(rows)+2, 20)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		BorderBottom(true).
		Bold(true)

	t.SetStyles(s)
	m.ItemsTable = t
}

func (m *Model) buildPodcastTable() {
	columns := []table.Column{
		{Title: "Title", Width: 60},
	}

	rows := []table.Row{}
	for _, p := range m.Podcasts {
		rows = append(rows, table.Row{p.Title})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(false)

	t.SetStyles(s)
	m.PodcastTable = t
}
