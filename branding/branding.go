package branding

import "github.com/charmbracelet/lipgloss"

const Wordmark = `
●           ●   ●●●●●●●   ●●●●●   ●
 ●         ●    ●           ●     ●
  ●       ●     ●           ●     ●
   ●     ●      ●●●●●       ●     ●
    ●   ●       ●           ●     ●
     ● ●        ●           ●     ●
      ●         ●●●●●●●   ●●●●●   ●●●●●●●
`

const WordmarkCompact = `
●       ●  ●●●●●  ●●●  ●
 ●     ●   ●       ●   ●
  ●   ●    ●●●●    ●   ●
   ● ●     ●       ●   ●
    ●      ●●●●●  ●●●  ●●●●●
`

const LogoFull = `
    ● ● ●
   ●     ●
  ●       ●
  ●       ●      ●           ●   ●●●●●●●   ●●●●●   ●
  ●       ●       ●         ●    ●           ●     ●
   ●     ●         ●       ●     ●           ●     ●
    ● ● ●           ●     ●      ●●●●●       ●     ●
     ● ●             ●   ●       ●           ●     ●
     ● ●              ● ●        ●           ●     ●
     ● ● ● ●           ●         ●●●●●●●   ●●●●●   ●●●●●●●
     ● ●
     ● ● ● ●
     ● ●
`

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
	logo := LogoStyle.Render(WordmarkCompact)
	tagline := TaglineStyle.Render("  Encrypted secrets for developers")
	return logo + "\n" + tagline + "\n"
}

func RenderFull() string {
	logo := LogoStyle.Render(LogoFull)
	tagline := TaglineStyle.Render("  Encrypted secrets for developers")
	return logo + "\n" + tagline + "\n"
}
