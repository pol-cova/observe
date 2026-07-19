package tui

import "github.com/charmbracelet/lipgloss"

var (
	title    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	muted    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warning  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	critical = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	good     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	panel    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1)
)
