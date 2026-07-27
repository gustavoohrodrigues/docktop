package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPixelLogoContainsVisiblePixels(t *testing.T) {
	logo := pixelLogo("DOCKTOP", 7)
	if !strings.Contains(logo, "██") || len(strings.Split(logo, "\n")) != 7 {
		t.Fatalf("invalid pixel logo:\n%s", logo)
	}
}

func TestDockerWhaleHasContainersAndEye(t *testing.T) {
	whale := strings.Join(dockerWhale(0), "\n")
	for _, element := range []string{"┌────┐", "┴", "◉", "████", "╲▄▄▄▄▄╱"} {
		if !strings.Contains(whale, element) {
			t.Fatalf("Docker whale is missing %q:\n%s", element, whale)
		}
	}
	if whale == strings.Join(dockerWhale(7), "\n") {
		t.Fatal("whale animation frames should differ")
	}
}

func TestWhalePositionEntersCentersAndExits(t *testing.T) {
	const width, whaleWidth = 92, 68
	if got := whalePosition(0, width, whaleWidth); got != -whaleWidth {
		t.Fatalf("initial position = %d", got)
	}
	center := (width - whaleWidth) / 2
	if got := whalePosition(16, width, whaleWidth); got < center-1 || got > center+1 {
		t.Fatalf("middle position = %d, expected around %d", got, center)
	}
	if got := whalePosition(splashFrames-1, width, whaleWidth); got != width {
		t.Fatalf("final position = %d, expected %d", got, width)
	}
}

func TestRepeatToWidth(t *testing.T) {
	if got := []rune(repeatToWidth("≈~", 17)); len(got) != 17 {
		t.Fatalf("wave width = %d", len(got))
	}
}

func TestSplashCanBeSkipped(t *testing.T) {
	m := &Model{splash: true}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.splash {
		t.Fatal("splash should close on keyboard input")
	}
	if cmd != nil {
		t.Fatal("skipping splash must not trigger an action")
	}
}

func TestSplashFinishesAtLastFrame(t *testing.T) {
	m := &Model{splash: true, splashFrame: splashFrames - 1}
	_, _ = m.Update(splashFrameMsg{})
	if m.splash {
		t.Fatal("splash should finish after its final frame")
	}
}
