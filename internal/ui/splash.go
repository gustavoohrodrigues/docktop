package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	splashFrames   = 32
	splashInterval = 75 * time.Millisecond
)

var pixelGlyphs = map[rune][]string{
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
}

func splashTick() tea.Cmd {
	return tea.Tick(splashInterval, func(t time.Time) tea.Msg { return splashFrameMsg(t) })
}

func (m *Model) splashView() string {
	width, height := max(m.w, 1), max(m.h, 1)
	bg := lipgloss.NewStyle().
		Background(m.th.Color(m.th.Background)).
		Foreground(m.th.Color(m.th.Text))

	if width < 54 || height < 18 {
		compact := lipgloss.NewStyle().Bold(true).Foreground(m.th.Color(m.th.Primary)).Render("◆ DOCKTOP")
		whale := lipgloss.NewStyle().Foreground(m.th.Color(m.th.Secondary)).Render("╲▄▟███████◉╮")
		return bg.Width(width).Height(height).Render(
			lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, compact+"\n\n"+whale),
		)
	}

	revealedRows := min(7, max(1, m.splashFrame/2))
	pixel := "█"
	if width >= 100 {
		pixel = "██"
	}
	logo := m.styledPixelLogo("DOCKTOP", revealedRows, pixel)

	seaWidth := min(width-4, 92)
	sea := m.splashSea(seaWidth)
	tagline := lipgloss.NewStyle().
		Foreground(m.th.Color(m.th.Muted)).
		Render(m.tr("splash_tagline"))
	skip := lipgloss.NewStyle().
		Foreground(m.th.Color(m.th.Border)).
		Render(m.tr("splash_skip"))

	content := logo + "\n" + sea + "\n" + tagline
	if height >= 27 {
		content += "\n\n" + skip
	}
	return bg.Width(width).Height(height).Render(
		lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content),
	)
}

func pixelLogo(text string, revealedRows int) string {
	return pixelLogoWith(text, revealedRows, "██")
}

func pixelLogoWith(text string, revealedRows int, pixel string) string {
	var rows []string
	for row := 0; row < 7; row++ {
		var line strings.Builder
		for index, char := range text {
			if index > 0 {
				line.WriteString("  ")
			}
			glyph := pixelGlyphs[char]
			for _, bit := range glyph[row] {
				if bit == '1' && row < revealedRows {
					line.WriteString(pixel)
				} else {
					line.WriteString("  ")
				}
			}
		}
		rows = append(rows, strings.TrimRight(line.String(), " "))
	}
	return strings.Join(rows, "\n")
}

func (m *Model) styledPixelLogo(text string, revealedRows int, pixel string) string {
	rows := strings.Split(pixelLogoWith(text, revealedRows, pixel), "\n")
	for i := range rows {
		color := m.th.Color(m.th.Primary)
		if i < 2 {
			color = m.th.Color(m.th.Secondary)
		}
		if i == 6 {
			color = m.th.Color(m.th.Focus)
		}
		rows[i] = lipgloss.NewStyle().Bold(true).Foreground(color).Render(rows[i])
	}
	accentWidth := lipgloss.Width(pixelLogoWith(text, 7, pixel))
	accent := lipgloss.NewStyle().
		Foreground(m.th.Color(m.th.Border)).
		Render("╺" + strings.Repeat("━", max(0, accentWidth-2)) + "╸")
	return strings.Join(rows, "\n") + "\n" + accent
}

func (m *Model) splashSea(width int) string {
	const whaleWidth = 68
	x := whalePosition(m.splashFrame, width, whaleWidth)
	whale := dockerWhale(m.splashFrame)
	lines := make([]string, 0, len(whale)+2)
	for row, art := range whale {
		left := x
		visible := art
		if left < 0 {
			cut := min(-left, len([]rune(visible)))
			visible = string([]rune(visible)[cut:])
			left = 0
		}
		if left >= width {
			visible = ""
		}
		room := max(0, width-left)
		runes := []rune(visible)
		if len(runes) > room {
			visible = string(runes[:room])
		}
		line := strings.Repeat(" ", left) + visible
		color := m.th.Color(m.th.Primary)
		if row < 5 {
			color = m.th.Color(m.th.Secondary)
		}
		if row == 0 {
			color = m.th.Color(m.th.NetworkChart)
		}
		if row == 11 || row == 12 {
			color = m.th.Color(m.th.Text)
		}
		if row == 13 {
			color = m.th.Color(m.th.Secondary)
		}
		lines = append(lines, lipgloss.NewStyle().Bold(row >= 5).Foreground(color).Render(line))
	}
	wavePatterns := []string{"≈~", "~≈", "∿~", "~∿"}
	wave := repeatToWidth(wavePatterns[(m.splashFrame/2)%len(wavePatterns)], width)
	lines = append(lines, lipgloss.NewStyle().Foreground(m.th.Color(m.th.NetworkChart)).Render(wave))
	progress := min(100, m.splashFrame*100/(splashFrames-1))
	status := fmt.Sprintf("[ %s %3d%% ]", m.tr("splash_boot"), progress)
	lines = append(lines, lipgloss.PlaceHorizontal(width, lipgloss.Center,
		lipgloss.NewStyle().Foreground(m.th.Color(m.th.Success)).Render(status)))
	return strings.Join(lines, "\n")
}

func whalePosition(frame, width, whaleWidth int) int {
	center := max(0, (width-whaleWidth)/2)
	switch {
	case frame < 10:
		return -whaleWidth + ((center + whaleWidth) * frame / 9)
	case frame < 23:
		return center + []int{0, 1, 1, 0, -1, -1}[frame%6]
	default:
		return center + ((width - center) * (frame - 22) / (splashFrames - 23))
	}
}

func repeatToWidth(pattern string, width int) string {
	runes := []rune(strings.Repeat(pattern, width/len([]rune(pattern))+1))
	return string(runes[:width])
}

func dockerWhale(frame int) []string {
	spout := []string{"  ·  °  ✦", " °  ·  ✧", "  ✦  ° ·", " ·  ✧  °"}[(frame/2)%4]
	tailTop, tailBottom := "╲", "╱"
	if (frame/3)%2 == 1 {
		tailTop, tailBottom = "╱", "╲"
	}
	eye := "◉"
	if frame%14 == 12 || frame%14 == 13 {
		eye = "━"
	}
	flipper := "╲▄▄▄▄▄╱"
	if (frame/3)%3 == 1 {
		flipper = "╲▄▄▄▄╱ "
	} else if (frame/3)%3 == 2 {
		flipper = " ╲▄▄▄▄╱"
	}
	trail := []string{"°  ·", " · °", "° · ", "  °·"}[frame%4]
	art := []string{
		"                                    " + spout,
		"                           ┌────┐        °",
		"                        ┌──┼────┼──┐    ╭╯",
		"                     ┌──┼──┼────┼──┼──┐ │",
		"                     └──┴──┴────┴──┴──┘ │",
		"   ▄▄          ▄▄            ╭──────────╯",
		" ▄████▄      ▄████▄      ▄▄▄▄╯       ▄▄▄▄▄▄▄",
		"██▀  ▀██▄▄▄▄██▀  ▀███████████████████████" + eye + "█╮  " + trail,
		"██▄          " + tailTop + "    ▄██████████████████████████╯",
		" ▀██▄▄          ▄████████████████████████▀",
		"    ▀██▄▄▄▄▄▄██████████████████▀▀▀▀▀▀",
		"       ▀▀███████████▀" + flipper,
		"           ▀▀▀▀▀▀       " + tailBottom + "▀▀",
	}
	if (frame/4)%2 == 1 {
		return append([]string{""}, art...)
	}
	return append(art, "")
}
