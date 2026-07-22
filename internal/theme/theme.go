package theme

import (
	"errors"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sort"
)

type Theme struct {
	Name         string `yaml:"name"`
	Background   string `yaml:"background"`
	Panel        string `yaml:"panel"`
	Border       string `yaml:"border"`
	Primary      string `yaml:"primary"`
	Secondary    string `yaml:"secondary"`
	Success      string `yaml:"success"`
	Warning      string `yaml:"warning"`
	Danger       string `yaml:"danger"`
	Muted        string `yaml:"muted"`
	Text         string `yaml:"text"`
	Selection    string `yaml:"selection"`
	Focus        string `yaml:"border_focus"`
	CPUChart     string `yaml:"cpu_chart"`
	MemoryChart  string `yaml:"memory_chart"`
	NetworkChart string `yaml:"network_chart"`
	DiskChart    string `yaml:"disk_chart"`
}

var themes = map[string]Theme{
	"dark-ops":         basic("dark-ops", "#080B12", "#0D1421", "#293750", "#37E6D3", "#4C9DFF", "#56F39A", "#FFD166", "#FF5470", "#72809A", "#D7E3F4", "#17394A", "#35D9FF"),
	"btop-classic":     basic("btop-classic", "#101014", "#17171D", "#45455A", "#00FFFF", "#A970FF", "#70FF70", "#FFFF70", "#FF5070", "#808090", "#EEEEF5", "#303044", "#FF80D0"),
	"nord":             basic("nord", "#2E3440", "#3B4252", "#4C566A", "#88C0D0", "#81A1C1", "#A3BE8C", "#EBCB8B", "#BF616A", "#7B88A1", "#ECEFF4", "#434C5E", "#8FBCBB"),
	"gruvbox-dark":     basic("gruvbox-dark", "#1D2021", "#282828", "#504945", "#83A598", "#D3869B", "#B8BB26", "#FABD2F", "#FB4934", "#928374", "#EBDBB2", "#3C3836", "#8EC07C"),
	"dracula":          basic("dracula", "#282A36", "#303341", "#6272A4", "#8BE9FD", "#BD93F9", "#50FA7B", "#F1FA8C", "#FF5555", "#7C8196", "#F8F8F2", "#44475A", "#FF79C6"),
	"catppuccin-mocha": basic("catppuccin-mocha", "#11111B", "#181825", "#45475A", "#89DCEB", "#89B4FA", "#A6E3A1", "#F9E2AF", "#F38BA8", "#7F849C", "#CDD6F4", "#313244", "#CBA6F7"),
	"high-contrast":    basic("high-contrast", "#000000", "#080808", "#FFFFFF", "#00FFFF", "#00AAFF", "#00FF00", "#FFFF00", "#FF3030", "#B0B0B0", "#FFFFFF", "#003A3A", "#FFFFFF"),
}

func basic(n, bg, panel, border, primary, secondary, success, warning, danger, muted, text, selection, focus string) Theme {
	return Theme{Name: n, Background: bg, Panel: panel, Border: border, Primary: primary, Secondary: secondary, Success: success, Warning: warning, Danger: danger, Muted: muted, Text: text, Selection: selection, Focus: focus, CPUChart: primary, MemoryChart: success, NetworkChart: secondary, DiskChart: warning}
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

func LoadFile(path string) (Theme, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return Theme{}, e
	}
	var t Theme
	if e = yaml.Unmarshal(b, &t); e != nil {
		return Theme{}, e
	}
	if t.Name == "" {
		t.Name = filepath.Base(path[:len(path)-len(filepath.Ext(path))])
	}
	if e = t.Validate(); e != nil {
		return Theme{}, e
	}
	return t, nil
}
func (t Theme) Validate() error {
	if t.Name == "" || t.Background == "" || t.Panel == "" || t.Border == "" || t.Focus == "" || t.Text == "" || t.Primary == "" || t.Danger == "" {
		return errors.New("tema não contém todas as cores semânticas obrigatórias")
	}
	return nil
}
func LoadDirectories(dirs ...string) map[string]Theme {
	out := map[string]Theme{}
	for name, t := range themes {
		out[name] = t
	}
	for _, dir := range dirs {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
		sort.Strings(matches)
		for _, p := range matches {
			if t, e := LoadFile(p); e == nil {
				out[t.Name] = t
			}
		}
	}
	return out
}
