package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lsherman98/yt-rss-cli/api"
)

type ViewState int

const (
	ViewSetAPIKey ViewState = iota
	ViewMainMenu
	ViewSelectPodcast
	ViewEnterURL
	ViewItemsTable
	ViewFatalError
)

type FatalErrorMsg struct {
	Err error
}

type ApiKeyCheckedMsg struct {
	HasKey bool
}

type PodcastsLoadedMsg struct {
	Podcasts []api.Podcast
	Err      error
}

type UrlAddedMsg struct {
	Item api.Item
	Err  error
}

type ItemsLoadedMsg struct {
	Items []api.Item
	Err   error
}

type UsageLoadedMsg struct {
	Usage *api.UsageResponse
	Err   error
}

type TickMsg time.Time

type menuItem string

func (i menuItem) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(menuItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	if index == m.Index() {
		fmt.Fprint(w, lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("> "+str))
	} else {
		fmt.Fprint(w, "  "+str)
	}
}

type Model struct {
	State           ViewState
	HasAPIKey       bool
	ApiKeyInput     textinput.Model
	UrlInput        textinput.Model
	MainMenu        list.Model
	PodcastTable    table.Model
	ItemsTable      table.Model
	Podcasts        []api.Podcast
	SelectedPodcast *api.Podcast
	Items           []api.Item
	Spinner         spinner.Model
	ProgressBar     progress.Model
	Usage           *api.UsageResponse
	Error           string
	Message         string
	Width           int
	Height          int
	Polling         bool
}

func InitialModel() Model {
	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "Enter your API key"
	apiKeyInput.Focus()
	apiKeyInput.CharLimit = 256
	apiKeyInput.Width = 50

	urlInput := textinput.New()
	urlInput.Placeholder = "Paste YouTube URL here"
	urlInput.CharLimit = 500
	urlInput.Width = 80

	items := []list.Item{
		menuItem("Add YouTube URL"),
		menuItem("Set API Key"),
	}
	mainMenu := list.New(items, itemDelegate{}, 30, 8)
	mainMenu.Title = "Main Menu"
	mainMenu.SetShowStatusBar(false)
	mainMenu.SetFilteringEnabled(false)
	mainMenu.SetShowHelp(false)
	mainMenu.Styles.Title = TitleStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	return Model{
		State:       ViewSetAPIKey,
		ApiKeyInput: apiKeyInput,
		UrlInput:    urlInput,
		MainMenu:    mainMenu,
		Spinner:     s,
		ProgressBar: prog,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(CheckAPIKey, m.Spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case FatalErrorMsg:
		m.Error = msg.Err.Error()
		m.State = ViewFatalError
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case ApiKeyCheckedMsg:
		m.HasAPIKey = msg.HasKey
		if msg.HasKey {
			m.State = ViewMainMenu
			return m, LoadUsage()
		} else {
			m.State = ViewSetAPIKey
			m.ApiKeyInput.Focus()
		}

	case UsageLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
		} else {
			m.Usage = msg.Usage
		}

	case PodcastsLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
			m.State = ViewMainMenu
		} else {
			m.Podcasts = msg.Podcasts
			m.Error = ""
			m.buildPodcastTable()
			m.State = ViewSelectPodcast
		}

	case UrlAddedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
		} else {
			m.Error = ""
			m.State = ViewItemsTable
			m.Polling = true
			cmds = append(cmds, LoadItems(m.SelectedPodcast.ID))
		}

	case ItemsLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
			m.Polling = false
		} else {
			m.Items = msg.Items
			m.buildItemsTable()

			hasCreated := false
			allSuccess := true
			for _, item := range m.Items {
				if item.Status == "CREATED" {
					hasCreated = true
					allSuccess = false
					break
				}
				if item.Status != "SUCCESS" {
					allSuccess = false
				}
			}

			if hasCreated && m.Polling {
				cmds = append(cmds, tick())
			} else {
				m.Polling = false
				if allSuccess && len(m.Items) > 0 {
					cmds = append(cmds, LoadUsage())
				}
			}
		}

	case TickMsg:
		if m.Polling && m.SelectedPodcast != nil {
			cmds = append(cmds, LoadItems(m.SelectedPodcast.ID))
		}

	case tea.KeyMsg:
		switch m.State {
		case ViewSetAPIKey:
			switch msg.String() {
			case "ctrl+c", "esc":
				if m.HasAPIKey {
					m.State = ViewMainMenu
					return m, nil
				}
				return m, tea.Quit
			case "ctrl+d":
				err := api.ClearApiKey()
				if err != nil {
					m.Error = err.Error()
				} else {
					m.HasAPIKey = false
					m.Message = "API key cleared successfully!"
					m.Error = ""
				}
				return m, nil
			case "enter":
				if m.ApiKeyInput.Value() != "" {
					err := api.SetApiKey(m.ApiKeyInput.Value())
					if err != nil {
						m.Error = err.Error()
					} else {
						m.HasAPIKey = true
						m.Message = "API key saved successfully!"
						m.ApiKeyInput.SetValue("")
						m.State = ViewMainMenu
						return m, LoadUsage()
					}
				}
				return m, nil
			}

		case ViewMainMenu:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				selected := m.MainMenu.SelectedItem()
				if selected != nil {
					switch selected.(menuItem) {
					case "Set API Key":
						m.State = ViewSetAPIKey
						m.ApiKeyInput.Focus()
						m.Error = ""
						m.Message = ""
					case "Add YouTube URL":
						m.State = ViewSelectPodcast
						m.Error = ""
						m.Message = ""
						return m, LoadPodcasts
					}
				}
			}

		case ViewSelectPodcast:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.State = ViewMainMenu
				return m, nil
			case "enter":
				if m.PodcastTable.Cursor() < len(m.Podcasts) {
					m.SelectedPodcast = &m.Podcasts[m.PodcastTable.Cursor()]
					m.State = ViewEnterURL
					m.UrlInput.Focus()
					m.UrlInput.SetValue("")
					return m, nil
				}
			}

		case ViewEnterURL:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.State = ViewSelectPodcast
				m.UrlInput.Blur()
				return m, nil
			case "enter":
				if m.UrlInput.Value() != "" && m.SelectedPodcast != nil {
					url := m.UrlInput.Value()
					m.UrlInput.SetValue("")
					return m, AddURL(m.SelectedPodcast.ID, url)
				}
			}

		case ViewItemsTable:
			switch msg.String() {
			case "ctrl+c", "q":
				m.Polling = false
				return m, tea.Quit
			case "a":
				m.State = ViewEnterURL
				m.UrlInput.Focus()
				m.UrlInput.SetValue("")
				m.Polling = false
				return m, nil
			case "m":
				m.State = ViewMainMenu
				m.Polling = false
				m.SelectedPodcast = nil
				return m, LoadUsage()
			}
		}
	}

	switch m.State {
	case ViewSetAPIKey:
		m.ApiKeyInput, cmd = m.ApiKeyInput.Update(msg)
		cmds = append(cmds, cmd)
	case ViewMainMenu:
		m.MainMenu, cmd = m.MainMenu.Update(msg)
		cmds = append(cmds, cmd)
	case ViewSelectPodcast:
		m.PodcastTable, cmd = m.PodcastTable.Update(msg)
		cmds = append(cmds, cmd)
	case ViewEnterURL:
		m.UrlInput, cmd = m.UrlInput.Update(msg)
		cmds = append(cmds, cmd)
	case ViewItemsTable:
		m.ItemsTable, cmd = m.ItemsTable.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.Spinner, cmd = m.Spinner.Update(msg)
	cmds = append(cmds, cmd)

	if m.State == ViewItemsTable && len(m.Items) > 0 {
		m.buildItemsTable()
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var s strings.Builder

	switch m.State {
	case ViewFatalError:
		s.WriteString(ErrorStyle.Render("Fatal Error: " + m.Error))
		s.WriteString("\n")
		s.WriteString(HelpStyle.Render("Press any key to exit"))

	case ViewSetAPIKey:
		s.WriteString(TitleStyle.Render("Set API Key"))
		s.WriteString("\n")
		if m.Message != "" {
			s.WriteString(SuccessStyle.Render(m.Message))
			s.WriteString("\n")
		}
		s.WriteString(m.ApiKeyInput.View())
		s.WriteString("\n")
		if m.Error != "" {
			s.WriteString(ErrorStyle.Render("Error: " + m.Error))
			s.WriteString("\n")
		}
		s.WriteString(HelpStyle.Render("Press Enter to save • Ctrl+d to clear API key • Esc to cancel"))

	case ViewMainMenu:
		if m.Message != "" {
			s.WriteString(SuccessStyle.Render(m.Message))
			s.WriteString("\n")
		}
		s.WriteString(m.MainMenu.View())
		s.WriteString("\n")

		if m.Usage != nil {
			s.WriteString("\n")
			usagePercent := 0.0
			if m.Usage.Limit > 0 {
				usagePercent = float64(m.Usage.Usage) / float64(m.Usage.Limit)
			}
			usageText := fmt.Sprintf("Usage: %s / %s",
				formatBytes(m.Usage.Usage),
				formatBytes(m.Usage.Limit),
			)
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(usageText))
			s.WriteString("\n")
			s.WriteString(m.ProgressBar.ViewAs(usagePercent))
			s.WriteString("\n")
		} else if m.Error != "" {
			s.WriteString(ErrorStyle.Render("Error: " + m.Error))
			s.WriteString("\n")
		}

		s.WriteString(HelpStyle.Render("↑/↓: Navigate • Enter: Select • q: Quit"))

	case ViewSelectPodcast:
		s.WriteString(TitleStyle.Render("Select a Podcast"))
		s.WriteString("\n")
		if len(m.Podcasts) == 0 {
			s.WriteString("No podcasts found.\n")
		} else {
			s.WriteString(m.PodcastTable.View())
		}
		s.WriteString("\n")
		if m.Error != "" {
			s.WriteString(ErrorStyle.Render("Error: " + m.Error))
			s.WriteString("\n")
		}
		s.WriteString(HelpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Back • q: Quit"))

	case ViewEnterURL:
		s.WriteString(TitleStyle.Render(fmt.Sprintf("Add URL to: %s", m.SelectedPodcast.Title)))
		s.WriteString("\n")
		s.WriteString(m.UrlInput.View())
		s.WriteString("\n")
		if m.Error != "" {
			s.WriteString(ErrorStyle.Render("Error: " + m.Error))
			s.WriteString("\n")
		}
		s.WriteString(HelpStyle.Render("Press Enter to add URL • Esc: Back • q: Quit"))

	case ViewItemsTable:
		s.WriteString(TitleStyle.Render(fmt.Sprintf("Items for: %s", m.SelectedPodcast.Title)))
		s.WriteString("\n")
		s.WriteString(m.ItemsTable.View())
		s.WriteString("\n")
		if m.Error != "" {
			s.WriteString(ErrorStyle.Render("Error: " + m.Error))
			s.WriteString("\n")
		}
		if m.Polling {
			s.WriteString(HelpStyle.Render("Polling for updates... • a: Add another URL • m: Main menu • q: Quit"))
		} else {
			s.WriteString(HelpStyle.Render("a: Add another URL • m: Main menu • q: Quit"))
		}
	}

	return s.String()
}
