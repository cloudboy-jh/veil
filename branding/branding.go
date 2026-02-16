package branding

import "github.com/charmbracelet/lipgloss"

const WordmarkSmall = `           _ __
 _  _____ (_) /
| |/ / -_) / /
|___/\__/_/_/`

var (
	Violet   = lipgloss.Color("#8B5CF6")
	White    = lipgloss.Color("#F8FAFC")
	Slate    = lipgloss.Color("#94A3B8")
	Charcoal = lipgloss.Color("#171717")
	Amber    = lipgloss.Color("#F59E0B")
	Emerald  = lipgloss.Color("#10B981")
)

var (
	LogoStyle = lipgloss.NewStyle().
			Foreground(Violet).
			Bold(true)

	TaglineStyle = lipgloss.NewStyle().
			Foreground(Slate).
			Italic(true)
)

func Render() string {
	logo := LogoStyle.Render(WordmarkSmall)
	tagline := TaglineStyle.Render("encrypted secrets for developers")
	return logo + "\n" + tagline + "\n"
}
