package tui

import "github.com/charmbracelet/lipgloss"

type palette struct {
	title, muted, warning, critical, good, panel lipgloss.Style
}

var (
	title, muted, warning, critical, good, panel lipgloss.Style
)

func init() {
	applyTheme("default")
}

func applyTheme(name string) {
	p := paletteFor(name)
	title, muted, warning, critical, good, panel = p.title, p.muted, p.warning, p.critical, p.good, p.panel
}

func paletteFor(name string) palette {
	switch name {
	case "light":
		return palette{
			title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("24")),
			muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
			warning:  lipgloss.NewStyle().Foreground(lipgloss.Color("172")),
			critical: lipgloss.NewStyle().Foreground(lipgloss.Color("160")),
			good:     lipgloss.NewStyle().Foreground(lipgloss.Color("28")),
			panel:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("24")).Padding(0, 1),
		}
	case "mono":
		return palette{
			title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")),
			muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
			warning:  lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
			critical: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")),
			good:     lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
			panel:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("245")).Padding(0, 1),
		}
	default:
		return defaultPalette()
	}
}

func defaultPalette() palette {
	return palette{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		warning:  lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		critical: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		good:     lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		panel:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1),
	}
}
