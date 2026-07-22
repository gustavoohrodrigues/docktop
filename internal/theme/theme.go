package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct{ Name, Background, Panel, Border, Primary, Secondary, Success, Warning, Danger, Muted, Text, Selection, Focus string }

var themes = map[string]Theme{
	"dark-ops":         {"dark-ops", "#080B12", "#0D1421", "#293750", "#37E6D3", "#4C9DFF", "#56F39A", "#FFD166", "#FF5470", "#72809A", "#D7E3F4", "#17394A", "#35D9FF"},
	"btop-classic":     {"btop-classic", "#101014", "#17171D", "#45455A", "#00FFFF", "#A970FF", "#70FF70", "#FFFF70", "#FF5070", "#808090", "#EEEEF5", "#303044", "#FF80D0"},
	"nord":             {"nord", "#2E3440", "#3B4252", "#4C566A", "#88C0D0", "#81A1C1", "#A3BE8C", "#EBCB8B", "#BF616A", "#7B88A1", "#ECEFF4", "#434C5E", "#8FBCBB"},
	"gruvbox-dark":     {"gruvbox-dark", "#1D2021", "#282828", "#504945", "#83A598", "#D3869B", "#B8BB26", "#FABD2F", "#FB4934", "#928374", "#EBDBB2", "#3C3836", "#8EC07C"},
	"dracula":          {"dracula", "#282A36", "#303341", "#6272A4", "#8BE9FD", "#BD93F9", "#50FA7B", "#F1FA8C", "#FF5555", "#7C8196", "#F8F8F2", "#44475A", "#FF79C6"},
	"catppuccin-mocha": {"catppuccin-mocha", "#11111B", "#181825", "#45475A", "#89DCEB", "#89B4FA", "#A6E3A1", "#F9E2AF", "#F38BA8", "#7F849C", "#CDD6F4", "#313244", "#CBA6F7"},
	"high-contrast":    {"high-contrast", "#000000", "#080808", "#FFFFFF", "#00FFFF", "#00AAFF", "#00FF00", "#FFFF00", "#FF3030", "#B0B0B0", "#FFFFFF", "#003A3A", "#FFFFFF"},
}

func Get(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["dark-ops"]
}
func Names() []string {
	return []string{"dark-ops", "btop-classic", "nord", "gruvbox-dark", "dracula", "catppuccin-mocha", "high-contrast"}
}
func (t Theme) Color(v string) lipgloss.Color { return lipgloss.Color(v) }
