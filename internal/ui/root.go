package ui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docktop/docktop/internal/audit"
	"github.com/docktop/docktop/internal/config"
	dock "github.com/docktop/docktop/internal/docker"
	"github.com/docktop/docktop/internal/i18n"
	"github.com/docktop/docktop/internal/registry"
	"github.com/docktop/docktop/internal/theme"
	"github.com/docktop/docktop/internal/utils"
)

var tabs = []string{"dashboard", "containers", "images", "registry", "volumes", "networks", "swarm", "services", "nodes", "stacks", "events", "audit", "settings"}

type snapshotMsg struct {
	s dock.Snapshot
	e error
}
type opMsg struct {
	action, resource, detail string
	e                        error
}
type searchMsg struct {
	results []registry.Result
	e       error
}
type shellMsg struct{ e error }
type auditMsg struct {
	entries []audit.Entry
	e       error
}
type tickMsg time.Time
type splashFrameMsg time.Time

type Model struct {
	ctx                                      context.Context
	cfg                                      config.Config
	engine                                   dock.Engine
	audit                                    *audit.Logger
	hub                                      *registry.Hub
	version                                  string
	w, h, tab, cursor, scroll, listOffset    int
	languageCursor                           int
	snap                                     dock.Snapshot
	results                                  []registry.Result
	err, status, mode, input, prompt, detail string
	loading, auto                            bool
	splash                                   bool
	splashFrame                              int
	targetID, targetName, action             string
	th                                       theme.Theme
	cpuHistory, memHistory                   []float64
	auditEntries                             []audit.Entry
}

func New(ctx context.Context, c config.Config, e dock.Engine, a *audit.Logger, v string) *Model {
	c.Language = i18n.Normalize(c.Language)
	return &Model{ctx: ctx, cfg: c, engine: e, audit: a, hub: registry.NewHub(), version: v, auto: true, loading: true, splash: true, th: theme.Get(c.Theme)}
}
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), m.loadAudit(), m.tick(), splashTick())
}
func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg { s, e := m.engine.Snapshot(m.ctx); return snapshotMsg{s, e} }
}
func (m *Model) tick() tea.Cmd {
	return tea.Tick(m.cfg.RefreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = x.Width, x.Height
	case snapshotMsg:
		m.loading = false
		m.err = ""
		if x.e != nil {
			m.err = i18n.LocalizeError(m.cfg.Language, x.e.Error())
		} else {
			m.snap = x.s
			m.captureHistory()
			m.clamp()
		}
	case searchMsg:
		m.loading = false
		m.results = x.results
		m.cursor = 0
		m.listOffset = 0
		if x.e != nil {
			m.err = i18n.LocalizeError(m.cfg.Language, x.e.Error())
		} else {
			m.status = fmt.Sprintf(m.tr("results_hub"), len(x.results))
		}
	case opMsg:
		m.loading = false
		m.status = fmt.Sprintf(m.tr("done"), x.action)
		m.detail = x.detail
		if x.e != nil {
			m.err = i18n.LocalizeError(m.cfg.Language, x.e.Error())
			m.status = fmt.Sprintf(m.tr("failed"), x.action)
		}
		result, errText := "ok", ""
		if x.e != nil {
			result = m.tr("error")
			errText = sanitize(x.e.Error())
		}
		_ = m.audit.Write(audit.Entry{Host: m.engine.Endpoint(), Action: x.action, Resource: x.resource, Result: result, Error: errText})
		if x.e == nil && x.detail != "" && (x.action == "logs" || x.action == "inspect" || x.action == "processes") {
			m.mode = "detail"
			m.scroll = 0
			return m, nil
		}
		return m, m.refresh()
	case shellMsg:
		if x.e != nil {
			m.err = i18n.LocalizeError(m.cfg.Language, x.e.Error())
		} else {
			m.status = m.tr("exec_closed")
		}
		return m, m.refresh()
	case auditMsg:
		m.loading = false
		m.auditEntries = x.entries
		if x.e != nil {
			m.err = i18n.LocalizeError(m.cfg.Language, x.e.Error())
		}
	case tickMsg:
		if m.auto && m.mode == "" {
			m.loading = true
			return m, tea.Batch(m.refresh(), m.tick())
		}
		return m, m.tick()
	case splashFrameMsg:
		if !m.splash {
			return m, nil
		}
		m.splashFrame++
		if m.splashFrame >= splashFrames {
			m.splash = false
			return m, nil
		}
		return m, splashTick()
	case tea.MouseMsg:
		if m.splash {
			m.splash = false
			return m, nil
		}
		return m.updateMouse(x)
	case tea.KeyMsg:
		if m.splash {
			if x.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m.splash = false
			return m, nil
		}
		if m.mode != "" {
			return m.updateOverlay(x)
		}
		return m.updateKeys(x)
	}
	return m, nil
}

func (m *Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.mode != "" {
		if msg.Button == tea.MouseButtonWheelUp {
			m.scroll = max(0, m.scroll-3)
		}
		if msg.Button == tea.MouseButtonWheelDown {
			m.scroll += 3
		}
		return m, nil
	}
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelUp {
		m.cursor = max(0, m.cursor-3)
		m.ensureListVisible()
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		m.cursor = min(max(0, m.count()-1), m.cursor+3)
		m.ensureListVisible()
		return m, nil
	}
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	if msg.Y == 1 {
		if tab, ok := m.tabAt(msg.X); ok {
			m.tab, m.cursor, m.scroll, m.listOffset = tab, 0, 0, 0
			if tab == 11 {
				return m, m.loadAudit()
			}
			return m, nil
		}
		if m.w > 100 && msg.X >= m.w-helpWidth(m.helpLabel())-1 {
			m.mode = "help"
			m.scroll = 0
		}
		return m, nil
	}
	if msg.Y == m.h-1 {
		if key, ok := m.footerActionAt(msg.X); ok {
			return m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		}
	}
	dataY := 4
	if m.err != "" || m.loading || m.status != "" {
		dataY++
	}
	if m.count() > m.listPageSize() {
		dataY++
	}
	visible := min(m.listPageSize(), max(0, m.count()-m.listOffset))
	if msg.Y >= dataY && msg.Y < dataY+visible {
		m.cursor = m.listOffset + msg.Y - dataY
		m.clamp()
		m.ensureListVisible()
	}
	return m, nil
}

func (m *Model) footerActionAt(x int) (string, bool) {
	right := fmt.Sprintf("? %s  Tab %s  r %s  R auto:%v  t %s  q %s ", m.tr("help"), m.tr("tabs"), m.tr("refresh"), m.auto, m.tr("theme"), m.tr("quit"))
	start := m.w - lipgloss.Width(right)
	items := []struct{ label, key string }{{"? " + m.tr("help"), "?"}, {"r " + m.tr("refresh"), "r"}, {"R auto:", "R"}, {"t " + m.tr("theme"), "t"}, {"q " + m.tr("quit"), "q"}}
	for _, item := range items {
		pos := strings.Index(right, item.label)
		if pos >= 0 && x >= start+pos && x < start+pos+len(item.label) {
			return item.key, true
		}
	}
	return "", false
}

func (m *Model) tabAt(x int) (int, bool) {
	start := max(0, m.tab-4)
	end := min(len(tabs), start+9)
	total := 0
	for i := start; i < end; i++ {
		total += lipgloss.Width(m.tabName(i)) + 2
	}
	area := m.w
	if m.w > 100 {
		area = m.w - helpWidth(m.helpLabel()) - 1
	}
	left := max(0, (area-total)/2)
	for i := start; i < end; i++ {
		width := lipgloss.Width(m.tabName(i)) + 2
		if x >= left && x < left+width {
			return i, true
		}
		left += width
	}
	return 0, false
}

func (m *Model) updateKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab", "right":
		m.tab = (m.tab + 1) % len(tabs)
		m.cursor = 0
		m.listOffset = 0
	case "shift+tab", "left":
		m.tab = (m.tab + len(tabs) - 1) % len(tabs)
		m.cursor = 0
		m.listOffset = 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.ensureListVisible()
	case "down", "j":
		if m.cursor < m.count()-1 {
			m.cursor++
		}
		m.ensureListVisible()
	case "pgup", "ctrl+u":
		m.cursor = max(0, m.cursor-m.listPageSize())
		m.ensureListVisible()
	case "pgdown", "ctrl+d":
		m.cursor = min(max(0, m.count()-1), m.cursor+m.listPageSize())
		m.ensureListVisible()
	case "g":
		m.cursor = 0
		m.ensureListVisible()
	case "G":
		m.cursor = max(0, m.count()-1)
		m.ensureListVisible()
	case "R":
		m.auto = !m.auto
	case "?", "f1":
		m.scroll = 0
		m.mode = "help"
	case "t":
		m.cycleTheme()
	case "L":
		if m.tab == 12 {
			m.openLanguageSelector()
		}
	case "/":
		if m.tab == 3 {
			m.openInput("search", m.tr("search_title"), "nginx, postgres, redis")
		}
	case "r":
		m.loading = true
		if m.tab == 11 {
			return m, m.loadAudit()
		}
		return m, m.refresh()
	case "x":
		if m.tab == 1 {
			return m, m.containerOp("restart")
		}
	case "S":
		return m, m.containerOp("start")
	case "T":
		return m, m.containerOp("stop")
	case "u":
		if m.tab == 1 {
			return m, m.containerOp("update-image")
		}
	case "s":
		if m.tab == 7 && len(m.snap.Services) > 0 && !m.readOnly() {
			m.openInput("scale-service", m.tr("scale_title"), m.tr("scale_prompt"))
		}
	case "A", "P", "D":
		if m.tab == 8 {
			return m.beginNodeAvailability(k.String())
		}
	case "enter":
		return m, m.inspectClusterSelection()
	case "p":
		switch m.tab {
		case 1:
			return m, m.containerOp("pause")
		case 2:
			m.openInput("pull", m.tr("pull_title"), "nginx:1.27")
		case 3:
			if len(m.results) > 0 {
				return m, m.pull(m.results[m.cursor].Name + ":latest")
			}
		}
	case "n":
		if m.readOnly() {
			break
		}
		switch m.tab {
		case 1:
			m.openInput("create-container", m.tr("create_container_title"), m.tr("create_container_prompt"))
		case 4:
			m.openInput("create-volume", m.tr("create_volume"), "name driver")
		case 5:
			m.openInput("create-network", m.tr("create_network"), "name driver: bridge|overlay|macvlan")
		}
	case "l":
		if m.tab == 1 {
			return m, m.containerDetail("logs")
		}
	case "i":
		if m.tab == 1 {
			return m, m.containerDetail("inspect")
		}
	case "o":
		if m.tab == 1 {
			return m, m.containerDetail("processes")
		}
	case "e":
		if m.tab == 1 {
			return m, m.execShell()
		}
	case "d":
		return m.beginRemove()
	}
	return m, nil
}

func (m *Model) updateOverlay(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == "language" {
		languages := i18n.Languages()
		switch k.String() {
		case "esc", "q":
			m.mode = ""
		case "up", "k":
			m.languageCursor = max(0, m.languageCursor-1)
		case "down", "j":
			m.languageCursor = min(len(languages)-1, m.languageCursor+1)
		case "enter":
			m.cfg.Language = languages[m.languageCursor].Code
			m.mode = ""
			if err := config.Save("", m.cfg); err != nil {
				m.err = fmt.Sprintf(m.tr("language_save_error"), err.Error())
			}
		}
		return m, nil
	}
	if m.mode == "detail" {
		switch k.String() {
		case "esc", "q", "enter":
			m.mode = ""
			m.detail = ""
		case "up", "k":
			m.scroll = max(0, m.scroll-1)
		case "down", "j":
			m.scroll++
		}
		return m, nil
	}
	if m.mode == "help" {
		switch k.String() {
		case "esc", "?", "f1", "q":
			m.mode = ""
		case "up", "k":
			m.scroll = max(0, m.scroll-1)
		case "down", "j":
			m.scroll++
		case "pgup":
			m.scroll = max(0, m.scroll-10)
		case "pgdown":
			m.scroll += 10
		case "home", "g":
			m.scroll = 0
		case "end", "G":
			m.scroll = 10000
		}
		return m, nil
	}
	if k.String() == "esc" {
		m.mode = ""
		m.input = ""
		return m, nil
	}
	if k.String() == "backspace" {
		if len(m.input) > 0 {
			_, n := utf8.DecodeLastRuneInString(m.input)
			m.input = m.input[:len(m.input)-n]
		}
		return m, nil
	}
	if k.String() == "enter" {
		return m.submitOverlay()
	}
	if k.Type == tea.KeyRunes {
		m.input += string(k.Runes)
	}
	return m, nil
}

func (m *Model) submitOverlay() (tea.Model, tea.Cmd) {
	mode, input := m.mode, strings.TrimSpace(m.input)
	if mode == "confirm" {
		if input != m.targetName {
			m.err = m.tr("confirm_mismatch")
			return m, nil
		}
		m.mode = ""
		return m, m.removeTarget()
	}
	if input == "" {
		m.err = m.tr("required")
		return m, nil
	}
	m.mode = ""
	m.input = ""
	m.loading = true
	switch mode {
	case "search":
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
			defer cancel()
			r, e := m.hub.Search(ctx, input)
			return searchMsg{r, e}
		}
	case "pull":
		return m, m.pull(input)
	case "create-container":
		req, parseErr := dock.ParseCreateSpec(input)
		if parseErr != nil {
			m.mode = mode
			m.input = input
			m.err = i18n.LocalizeError(m.cfg.Language, parseErr.Error())
			return m, nil
		}
		return m, m.createContainer(req)
	case "create-volume":
		p := strings.Fields(input)
		driver := "local"
		if len(p) > 1 {
			driver = p[1]
		}
		return m, m.resourceOp("create-volume", p[0], driver)
	case "create-network":
		p := strings.Fields(input)
		driver := "bridge"
		if len(p) > 1 {
			driver = p[1]
		}
		return m, m.resourceOp("create-network", p[0], driver)
	case "scale-service":
		if len(m.snap.Services) == 0 {
			return m, nil
		}
		var replicas uint64
		if _, err := fmt.Sscan(input, &replicas); err != nil {
			m.mode, m.input, m.err = mode, input, m.tr("replicas_invalid")
			return m, nil
		}
		svc := m.snap.Services[m.cursor]
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(m.ctx, 35*time.Second)
			defer cancel()
			e := m.engine.ScaleService(ctx, svc.ID, replicas)
			return opMsg{action: "scale-service", resource: svc.Name, e: e}
		}
	}
	return m, nil
}

func (m *Model) readOnly() bool {
	if m.cfg.ReadOnly {
		m.err = m.tr("readonly_block")
		return true
	}
	return false
}
func (m *Model) openInput(mode, title, prompt string) {
	m.mode = mode
	m.prompt = title + "\n" + prompt
	m.input = ""
	m.err = ""
}
func (m *Model) openLanguageSelector() {
	m.mode = "language"
	for i, language := range i18n.Languages() {
		if language.Code == m.cfg.Language {
			m.languageCursor = i
			break
		}
	}
}
func (m *Model) tr(key string) string     { return i18n.T(m.cfg.Language, key) }
func (m *Model) tabName(index int) string { return m.tr(tabs[index]) }
func (m *Model) helpLabel() string        { return "[F1 " + strings.ToUpper(m.tr("help")) + "]" }
func (m *Model) trState(value string) string {
	if value == "failed" {
		return m.tr("state_failed")
	}
	return m.tr(value)
}
func (m *Model) containerOp(a string) tea.Cmd {
	if m.tab != 1 || len(m.snap.Containers) == 0 || m.readOnly() {
		return nil
	}
	x := m.snap.Containers[m.cursor]
	if a == "pause" && x.State == "paused" {
		a = "unpause"
	}
	m.status = map[string]string{"start": m.tr("op_start"), "stop": m.tr("op_stop"), "restart": m.tr("op_restart"), "pause": m.tr("op_pause"), "unpause": m.tr("op_unpause"), "update-image": m.tr("op_update")}[a]
	return m.runContainer(a, x.ID)
}
func (m *Model) runContainer(a, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 35*time.Second)
		defer cancel()
		var e error
		switch a {
		case "start":
			e = m.engine.Start(ctx, id)
		case "stop":
			e = m.engine.Stop(ctx, id)
		case "restart":
			e = m.engine.Restart(ctx, id)
		case "pause":
			e = m.engine.Pause(ctx, id)
		case "unpause":
			e = m.engine.Unpause(ctx, id)
		case "update-image":
			detail, updateErr := m.engine.UpdateImage(ctx, id, nil)
			e = updateErr
			return opMsg{action: a, resource: id, detail: detail, e: e}
		}
		return opMsg{action: a, resource: id, e: e}
	}
}
func (m *Model) createContainer(req dock.CreateRequest) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 15*time.Minute)
		defer cancel()
		id, e := m.engine.CreateContainer(ctx, req)
		return opMsg{action: "create-container", resource: req.Name, detail: id, e: e}
	}
}
func (m *Model) pull(ref string) tea.Cmd {
	m.loading = true
	m.status = fmt.Sprintf(m.tr("pull_progress"), ref)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Minute)
		defer cancel()
		last := ""
		e := m.engine.Pull(ctx, ref, func(s string) { last = s })
		return opMsg{action: "pull", resource: ref, detail: last, e: e}
	}
}
func (m *Model) containerDetail(kind string) tea.Cmd {
	if len(m.snap.Containers) == 0 {
		return nil
	}
	id := m.snap.Containers[m.cursor].ID
	m.loading = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
		defer cancel()
		var text string
		var e error
		switch kind {
		case "logs":
			text, e = m.engine.Logs(ctx, id, 300)
		case "inspect":
			text, e = m.engine.Inspect(ctx, id)
		case "processes":
			text, e = m.engine.Processes(ctx, id)
		}
		return opMsg{action: kind, resource: id, detail: text, e: e}
	}
}
func (m *Model) execShell() tea.Cmd {
	if len(m.snap.Containers) == 0 {
		return nil
	}
	id := m.snap.Containers[m.cursor].ID
	cmd, e := m.engine.ShellCommand(m.ctx, id)
	if e != nil {
		m.err = i18n.LocalizeError(m.cfg.Language, e.Error())
		return nil
	}
	return tea.ExecProcess(cmd, func(e error) tea.Msg { return shellMsg{e} })
}
func (m *Model) resourceOp(action, name, arg string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
		defer cancel()
		var e error
		switch action {
		case "create-volume":
			e = m.engine.CreateVolume(ctx, name, arg)
		case "create-network":
			e = m.engine.CreateNetwork(ctx, name, arg)
		}
		return opMsg{action: action, resource: name, e: e}
	}
}

func (m *Model) beginRemove() (tea.Model, tea.Cmd) {
	if m.readOnly() {
		return m, nil
	}
	switch m.tab {
	case 1:
		if !m.cfg.DangerousActions.RemoveContainers {
			m.err = m.tr("policy_containers")
			return m, nil
		}
		if len(m.snap.Containers) > 0 {
			x := m.snap.Containers[m.cursor]
			m.targetID, m.targetName, m.action = x.ID, x.Name, "remove-container"
		}
	case 2:
		if !m.cfg.DangerousActions.RemoveImages {
			m.err = m.tr("policy_images")
			return m, nil
		}
		if len(m.snap.Images) > 0 {
			x := m.snap.Images[m.cursor]
			m.targetID, m.targetName, m.action = x.ID, shortID(x.ID), "remove-image"
		}
	case 4:
		if !m.cfg.DangerousActions.RemoveVolumes {
			m.err = m.tr("policy_volumes")
			return m, nil
		}
		if len(m.snap.Volumes) > 0 {
			x := m.snap.Volumes[m.cursor]
			m.targetID, m.targetName, m.action = x.Name, x.Name, "remove-volume"
		}
	case 5:
		if !m.cfg.DangerousActions.RemoveNetworks {
			m.err = m.tr("policy_networks")
			return m, nil
		}
		if len(m.snap.Networks) > 0 {
			x := m.snap.Networks[m.cursor]
			m.targetID, m.targetName, m.action = x.ID, x.Name, "remove-network"
		}
	default:
		return m, nil
	}
	if m.targetName == "" {
		return m, nil
	}
	m.mode = "confirm"
	m.input = ""
	m.prompt = fmt.Sprintf(m.tr("destructive"), m.targetName)
	return m, nil
}
func (m *Model) removeTarget() tea.Cmd {
	action, id, name := m.action, m.targetID, m.targetName
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 35*time.Second)
		defer cancel()
		var e error
		switch action {
		case "remove-container":
			e = m.engine.Remove(ctx, id, true)
		case "remove-image":
			e = m.engine.RemoveImage(ctx, id, true)
		case "remove-volume":
			e = m.engine.RemoveVolume(ctx, id, true)
		case "remove-network":
			e = m.engine.RemoveNetwork(ctx, id)
		case "node-active":
			e = m.engine.SetNodeAvailability(ctx, id, "active")
		case "node-pause":
			e = m.engine.SetNodeAvailability(ctx, id, "pause")
		case "node-drain":
			e = m.engine.SetNodeAvailability(ctx, id, "drain")
		}
		return opMsg{action: action, resource: name, e: e}
	}
}

func (m *Model) beginNodeAvailability(key string) (tea.Model, tea.Cmd) {
	if len(m.snap.Nodes) == 0 || m.readOnly() {
		return m, nil
	}
	if !m.cfg.DangerousActions.SwarmChanges {
		m.err = m.tr("policy_swarm")
		return m, nil
	}
	n := m.snap.Nodes[m.cursor]
	availability := map[string]string{"A": "active", "P": "pause", "D": "drain"}[key]
	m.targetID, m.targetName, m.action = n.ID, n.Hostname, "node-"+availability
	m.mode, m.input = "confirm", ""
	m.prompt = fmt.Sprintf(m.tr("node_change"), n.Hostname, availability, n.Hostname)
	return m, nil
}

func (m *Model) inspectClusterSelection() tea.Cmd {
	var kind, id string
	switch m.tab {
	case 7:
		if len(m.snap.Services) > 0 {
			kind, id = "service", m.snap.Services[m.cursor].ID
		}
	case 8:
		if len(m.snap.Nodes) > 0 {
			kind, id = "node", m.snap.Nodes[m.cursor].ID
		}
	default:
		return nil
	}
	if id == "" {
		return nil
	}
	m.loading = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
		defer cancel()
		detail, e := m.engine.ClusterInspect(ctx, kind, id)
		return opMsg{action: "inspect", resource: id, detail: detail, e: e}
	}
}

func (m *Model) loadAudit() tea.Cmd {
	return func() tea.Msg { entries, e := m.audit.Read(1000); return auditMsg{entries: entries, e: e} }
}

func (m *Model) captureHistory() {
	cpu, mem := 0.0, 0.0
	for _, v := range m.snap.Metrics {
		cpu += v.CPU
		mem += v.Memory
	}
	m.cpuHistory = appendLimit(m.cpuHistory, cpu, 90)
	m.memHistory = appendLimit(m.memHistory, mem, 90)
}
func appendLimit(v []float64, x float64, n int) []float64 {
	v = append(v, x)
	if len(v) > n {
		v = v[len(v)-n:]
	}
	return v
}
func (m *Model) count() int {
	switch m.tab {
	case 1:
		return len(m.snap.Containers)
	case 2:
		return len(m.snap.Images)
	case 3:
		return len(m.results)
	case 4:
		return len(m.snap.Volumes)
	case 5:
		return len(m.snap.Networks)
	case 7:
		return len(m.snap.Services)
	case 8:
		return len(m.snap.Nodes)
	case 9:
		return len(m.snap.Stacks)
	case 10:
		return len(m.snap.Events)
	case 11:
		return len(m.auditEntries)
	}
	return 0
}
func (m *Model) clamp() {
	m.cursor = min(m.cursor, max(0, m.count()-1))
	m.ensureListVisible()
}

func (m *Model) listPageSize() int {
	reserved := 12
	if m.tab == 7 {
		reserved = 20
	}
	if m.err != "" || m.loading || m.status != "" {
		reserved++
	}
	return max(1, m.h-reserved)
}

func (m *Model) ensureListVisible() {
	total := m.count()
	page := m.listPageSize()
	if total <= page {
		m.listOffset = 0
		return
	}
	if m.cursor < m.listOffset {
		m.listOffset = m.cursor
	}
	if m.cursor >= m.listOffset+page {
		m.listOffset = m.cursor - page + 1
	}
	m.listOffset = min(max(0, m.listOffset), max(0, total-page))
}

func (m *Model) visibleRange(total int) (int, int) {
	m.ensureListVisible()
	start := min(m.listOffset, total)
	return start, min(total, start+m.listPageSize())
}

func (m *Model) listProgress(start, end, total int) string {
	if total <= m.listPageSize() {
		return ""
	}
	return fmt.Sprintf("  ↑↓/PgUp/PgDn  %d–%d/%d\n", start+1, end, total)
}
func sanitize(s string) string {
	return utils.Sanitize(s)
}
func shortID(s string) string {
	s = strings.TrimPrefix(s, "sha256:")
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
func row(sel bool, s string) string {
	if sel {
		return "▸ " + s
	}
	return "  " + s
}

func (m *Model) View() string {
	if m.splash {
		return m.splashView()
	}
	if m.w < 76 || m.h < 20 {
		return "DockTop\n\n" + fmt.Sprintf(m.tr("minimum_terminal"), m.w, m.h)
	}
	bg := lipgloss.NewStyle().Background(m.th.Color(m.th.Background)).Foreground(m.th.Color(m.th.Text))
	out := m.top() + "\n" + m.body() + "\n" + m.footer()
	if m.mode != "" {
		out = m.overlay()
	}
	return bg.Width(m.w).Height(m.h).Render(out)
}
func (m *Model) top() string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(m.th.Color(m.th.Primary)).Render("◆ DockTop")
	tag := lipgloss.NewStyle().Foreground(m.th.Color(m.th.Muted)).Render("  " + m.tr("console_tagline"))
	head := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, logo+tag)
	var items string
	start := max(0, m.tab-4)
	end := min(len(tabs), start+9)
	for i := start; i < end; i++ {
		st := lipgloss.NewStyle().Padding(0, 1).Foreground(m.th.Color(m.th.Muted))
		if i == m.tab {
			st = st.Bold(true).Foreground(m.th.Color(m.th.Background)).Background(m.th.Color(m.th.Primary))
		}
		items += st.Render(m.tabName(i))
	}
	help := lipgloss.NewStyle().Bold(true).Foreground(m.th.Color(m.th.Warning)).Render(m.helpLabel())
	menu := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, items)
	if m.w > 100 {
		menu = lipgloss.PlaceHorizontal(m.w-helpWidth(help)-1, lipgloss.Center, items) + " " + help
	}
	return head + "\n" + menu
}
func (m *Model) body() string {
	h := m.h - 6
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(m.th.Color(m.th.Border)).Padding(0, 1).Width(m.w - 4).Height(h)
	head := ""
	if m.err != "" {
		head = lipgloss.NewStyle().Foreground(m.th.Color(m.th.Danger)).Bold(true).Render("⚠ "+m.err) + "\n"
	} else if m.loading {
		message := m.tr("sync_engine")
		if m.status != "" {
			message = m.status
		}
		head = lipgloss.NewStyle().Foreground(m.th.Color(m.th.Warning)).Render("◌ "+message) + "\n"
	} else if m.status != "" {
		head = lipgloss.NewStyle().Foreground(m.th.Color(m.th.Success)).Render("✓ "+m.status) + "\n"
	}
	var b string
	switch m.tab {
	case 0:
		b = m.dashboard()
	case 1:
		b = m.containers()
	case 2:
		b = m.images()
	case 3:
		b = m.registry()
	case 4:
		b = m.volumes()
	case 5:
		b = m.networks()
	case 6:
		b = m.swarmOverview()
	case 7:
		b = m.services()
	case 8:
		b = m.nodes()
	case 9:
		b = m.stacks()
	case 10:
		b = m.events()
	case 11:
		b = m.auditView()
	case 12:
		b = m.settings()
	}
	return box.Render(head + b)
}
func (m *Model) panel(title, body string, w, h int) string {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(m.th.Color(m.th.Border)).Padding(0, 1).Width(w).Height(h).Render(lipgloss.NewStyle().Bold(true).Foreground(m.th.Color(m.th.Secondary)).Render(title) + "\n" + body)
}
func (m *Model) dashboard() string {
	i := m.snap.Info
	running, stopped := 0, 0
	cpu, mem := 0.0, 0.0
	var memBytes uint64
	for _, c := range m.snap.Containers {
		if c.State == "running" {
			running++
		} else {
			stopped++
		}
	}
	for _, v := range m.snap.Metrics {
		cpu += v.CPU
		mem += v.Memory
		memBytes += v.MemoryBytes
	}
	inner := m.w - 8
	left := max(34, inner*55/100)
	right := inner - left - 1
	graphW := max(12, left-4)
	perf := fmt.Sprintf("%s  %6.1f%%  %s\n%s\n\n%s  %s  %5.1f%%\n%s", m.tr("cpu_containers"), cpu, meter(min(cpu, 100), 18, m.th.Color(m.th.Primary)), spark(m.cpuHistory, graphW, m.th.Color(m.th.Primary)), m.tr("ram_containers"), utils.Bytes(int64(memBytes)), mem, meter(min(mem, 100), 18, m.th.Color(m.th.Success)))
	engine := fmt.Sprintf("● %-18s %s\nEngine %-14s %s\n%s / %s\n%s: %s\nCPUs: %d   RAM: %s", i.Name, stateColor("online"), m.snap.Version, m.tr("api_negotiated"), i.OperatingSystem, i.Architecture, m.tr("storage"), i.Driver, i.NCPU, utils.Bytes(i.MemTotal))
	top := lipgloss.JoinHorizontal(lipgloss.Top, m.panel(m.tr("performance"), perf, left, 8), m.panel(m.tr("engine"), engine, right, 8))
	resources := fmt.Sprintf("%s  %d %s  %s %d %s  %s %d %s     %s %d     %s %d     %s %d", strings.ToUpper(m.tr("containers")), len(m.snap.Containers), m.tr("total"), stateColor("●"), running, m.tr("running"), lipgloss.NewStyle().Foreground(m.th.Color(m.th.Muted)).Render("●"), stopped, m.tr("stopped"), strings.ToUpper(m.tr("images")), len(m.snap.Images), strings.ToUpper(m.tr("volumes")), len(m.snap.Volumes), strings.ToUpper(m.tr("networks")), len(m.snap.Networks))
	return top + "\n" + m.panel(m.tr("resource_pulse"), resources, inner, 3) + "\n" + lipgloss.NewStyle().Foreground(m.th.Color(m.th.Muted)).Render(fmt.Sprintf(m.tr("updated"), m.snap.At.Format("15:04:05")))
}
func (m *Model) containers() string {
	b := m.tr("containers_header") + "\n"
	start, end := m.visibleRange(len(m.snap.Containers))
	b += m.listProgress(start, end, len(m.snap.Containers))
	for index, x := range m.snap.Containers[start:end] {
		i := start + index
		v := m.snap.Metrics[x.ID]
		b += row(i == m.cursor, fmt.Sprintf("%-18.18s %-12.12s %-22.22s %-10s %6.1f%% %7s  %s", x.Name, x.ID, x.Image, m.trState(x.State), v.CPU, utils.Bytes(int64(v.MemoryBytes)), x.Status)) + "\n"
	}
	if len(m.snap.Containers) == 0 {
		b += "\n  " + m.tr("container_empty")
	}
	return b + "\n  " + m.tr("containers_hint")
}
func (m *Model) images() string {
	b := m.tr("images_header") + "\n"
	start, end := m.visibleRange(len(m.snap.Images))
	b += m.listProgress(start, end, len(m.snap.Images))
	for index, x := range m.snap.Images[start:end] {
		i := start + index
		b += row(i == m.cursor, fmt.Sprintf("%-43.43s %-12.12s %s", x.Tags, shortID(x.ID), utils.Bytes(x.Size))) + "\n"
	}
	if len(m.snap.Images) == 0 {
		b += "\n  " + m.tr("image_empty")
	}
	return b + "\n  " + m.tr("images_hint")
}
func (m *Model) registry() string {
	b := m.tr("registry_header") + "\n"
	start, end := m.visibleRange(len(m.results))
	b += m.listProgress(start, end, len(m.results))
	for index, x := range m.results[start:end] {
		i := start + index
		kind := m.tr("community")
		if x.Official {
			kind = m.tr("official")
		}
		b += row(i == m.cursor, fmt.Sprintf("%-34.34s %7d %11d  %-10s  %.50s", x.Name, x.Stars, x.Pulls, kind, x.Description)) + "\n"
	}
	if len(m.results) == 0 {
		b += "\n  " + m.tr("registry_empty")
	}
	return b + "\n  " + m.tr("registry_hint")
}
func (m *Model) volumes() string {
	b := m.tr("volumes_header") + "\n"
	start, end := m.visibleRange(len(m.snap.Volumes))
	b += m.listProgress(start, end, len(m.snap.Volumes))
	for index, x := range m.snap.Volumes[start:end] {
		i := start + index
		b += row(i == m.cursor, fmt.Sprintf("%-41.41s %-12s %s", x.Name, x.Driver, x.Scope)) + "\n"
	}
	return b + "\n  " + m.tr("volumes_hint")
}
func (m *Model) networks() string {
	b := m.tr("networks_header") + "\n"
	start, end := m.visibleRange(len(m.snap.Networks))
	b += m.listProgress(start, end, len(m.snap.Networks))
	for index, x := range m.snap.Networks[start:end] {
		i := start + index
		b += row(i == m.cursor, fmt.Sprintf("%-28.28s %-12.12s %-12s %s", x.Name, x.ID, x.Driver, x.Scope)) + "\n"
	}
	return b + "\n  " + m.tr("networks_hint")
}
func (m *Model) swarmOverview() string {
	s := m.snap.Info.Swarm
	role := m.tr("standalone")
	guidance := m.tr("swarm_none")
	if string(s.LocalNodeState) == "active" {
		role = m.tr("worker")
		guidance = m.tr("swarm_worker")
		if s.ControlAvailable {
			role = m.tr("manager")
			guidance = m.tr("swarm_manager")
		}
	}
	clusterID := "-"
	if s.Cluster != nil {
		clusterID = shortID(s.Cluster.ID)
	}
	return fmt.Sprintf("%s\n\n  %-18s %s\n  %-18s %s\n  %-18s %s\n  %-18s %s\n  %-18s %d\n  %-18s %d\n  %-18s %v\n\n%s", m.tr("endpoint_state"), m.tr("local_role"), role, m.tr("node_state"), m.trState(string(s.LocalNodeState)), "node ID", shortID(s.NodeID), m.tr("cluster_id"), clusterID, m.tr("managers"), s.Managers, m.tr("nodes"), s.Nodes, m.tr("control_api"), s.ControlAvailable, guidance)
}
func (m *Model) services() string {
	b := m.tr("services_header") + "\n"
	start, end := m.visibleRange(len(m.snap.Services))
	b += m.listProgress(start, end, len(m.snap.Services))
	for index, x := range m.snap.Services[start:end] {
		i := start + index
		b += row(i == m.cursor, fmt.Sprintf("%-28.28s %-14.14s %-11s %4d/%-4d %-12s %.36s", x.Name, x.Stack, m.trState(x.Mode), x.Running, x.Desired, m.trState(x.Update), x.Image)) + "\n"
	}
	if len(m.snap.Services) == 0 {
		b += "\n  " + m.tr("service_empty") + "\n"
	} else {
		selected := m.snap.Services[m.cursor]
		b += "\n  " + fmt.Sprintf(m.tr("tasks_of"), selected.Name) + "\n"
		shown := 0
		for _, task := range m.snap.Tasks {
			if task.ServiceID != selected.ID || shown >= 6 {
				continue
			}
			errText := task.Error
			if errText == "" {
				errText = "-"
			}
			b += "  " + fmt.Sprintf(m.tr("task_line"), task.Slot, task.Node, m.trState(task.Desired), m.trState(task.State), errText) + "\n"
			shown++
		}
		if shown == 0 {
			b += "  " + m.tr("task_empty") + "\n"
		}
	}
	return b + "\n  " + m.tr("services_hint")
}
func (m *Model) nodes() string {
	b := m.tr("nodes_header") + "\n"
	start, end := m.visibleRange(len(m.snap.Nodes))
	b += m.listProgress(start, end, len(m.snap.Nodes))
	for index, x := range m.snap.Nodes[start:end] {
		i := start + index
		b += row(i == m.cursor, fmt.Sprintf("%-22.22s %-12.12s %-9s %-9s %-9s %-12s %3d  %8s  %s", x.Hostname, x.ID, m.trState(x.Role), m.trState(x.Availability), m.trState(x.State), m.trState(x.Manager), x.CPUs, utils.Bytes(x.Memory), x.Engine)) + "\n"
	}
	if len(m.snap.Nodes) == 0 {
		b += "\n  " + m.tr("node_empty") + "\n"
	}
	return b + "\n  " + m.tr("nodes_hint")
}
func (m *Model) stacks() string {
	b := m.tr("stacks_header") + "\n"
	start, end := m.visibleRange(len(m.snap.Stacks))
	b += m.listProgress(start, end, len(m.snap.Stacks))
	for index, x := range m.snap.Stacks[start:end] {
		i := start + index
		health := m.tr("healthy")
		if x.Failed > 0 || x.Running < x.Desired {
			health = m.tr("degraded")
		}
		b += row(i == m.cursor, fmt.Sprintf("%-35.35s %8d   %4d/%-4d   %7d  %s", x.Name, x.Services, x.Running, x.Desired, x.Failed, health)) + "\n"
	}
	if len(m.snap.Stacks) == 0 {
		b += "\n  " + m.tr("stack_empty") + "\n"
	}
	return b + "\n  " + m.tr("stacks_hint")
}
func (m *Model) events() string {
	b := m.tr("events_header") + "\n"
	start, end := m.visibleRange(len(m.snap.Events))
	b += m.listProgress(start, end, len(m.snap.Events))
	for index, x := range m.snap.Events[start:end] {
		i := start + index
		b += row(i == m.cursor, fmt.Sprintf("%-13s %-13s %-20.20s %-13.13s %s", x.Time.Format("15:04:05"), x.Type, x.Action, shortID(x.ID), x.Name)) + "\n"
	}
	if len(m.snap.Events) == 0 {
		b += "\n  " + m.tr("event_empty") + "\n"
	}
	return b + "\n  " + m.tr("events_hint")
}
func (m *Model) auditView() string {
	b := m.tr("audit_header") + "\n"
	start, end := m.visibleRange(len(m.auditEntries))
	b += m.listProgress(start, end, len(m.auditEntries))
	for index, x := range m.auditEntries[start:end] {
		i := start + index
		result := x.Result
		if result == "erro" || result == "error" {
			result = m.tr("error")
		}
		b += row(i == m.cursor, fmt.Sprintf("%-21s %-16.16s %-20.20s %-19.19s %s", x.Timestamp.Local().Format("2006-01-02 15:04:05"), x.User, x.Action, x.Resource, result)) + "\n"
	}
	if len(m.auditEntries) == 0 {
		b += "\n  " + m.tr("audit_empty") + "\n"
	}
	return b + "\n  " + m.tr("audit_hint")
}
func (m *Model) settings() string {
	return fmt.Sprintf("%s\n\n  %-20s %s\n  %-20s %s\n  %-20s %s\n  %-20s %s\n  %-20s %s\n  %-20s %v\n  %-20s %v\n  %-20s %v\n  %-20s %v\n  %-20s %v\n\n%s: %s\n\n  t %s  L %s", m.tr("effective_config"), m.tr("default_context"), m.cfg.DefaultContext, m.tr("endpoint"), m.engine.Endpoint(), m.tr("theme"), m.th.Name, m.tr("language"), m.cfg.Language, m.tr("refresh"), m.cfg.Refresh, m.tr("auto_refresh"), m.auto, m.tr("read_only"), m.cfg.ReadOnly, m.tr("mouse"), m.cfg.MouseEnabled, m.tr("audit"), m.cfg.Audit.Enabled, m.tr("telemetry"), m.cfg.TelemetryEnabled, m.tr("file"), config.Path(), m.tr("theme"), m.tr("select_language"))
}
func (m *Model) footer() string {
	mode := "RW"
	if m.cfg.ReadOnly {
		mode = "READ-ONLY"
	}
	transport := "TCP"
	if strings.HasPrefix(m.engine.Endpoint(), "unix:") {
		transport = "SOCKET"
	}
	if strings.Contains(m.engine.Endpoint(), "2376") {
		transport = "TLS"
	}
	left := fmt.Sprintf(" %s · %s · Engine %s · Swarm %s · %s ", m.cfg.DefaultContext, transport, m.snap.Version, m.snap.Info.Swarm.LocalNodeState, mode)
	right := fmt.Sprintf("? %s  Tab %s  r %s  R auto:%v  t %s  q %s ", m.tr("help"), m.tr("tabs"), m.tr("refresh"), m.auto, m.tr("theme"), m.tr("quit"))
	return left + strings.Repeat(" ", max(1, m.w-lipgloss.Width(left)-lipgloss.Width(right))) + right
}
func (m *Model) overlay() string {
	var content string
	if m.mode == "detail" {
		lines := strings.Split(m.detail, "\n")
		maxLines := m.h - 10
		m.scroll = min(m.scroll, max(0, len(lines)-maxLines))
		content = strings.Join(lines[m.scroll:min(len(lines), m.scroll+maxLines)], "\n") + "\n\n↑↓ " + m.tr("scroll") + " · Esc " + m.tr("close")
	} else if m.mode == "help" {
		lines := strings.Split(helpManual(m.cfg.Language), "\n")
		maxLines := m.h - 9
		m.scroll = min(m.scroll, max(0, len(lines)-maxLines))
		content = strings.Join(lines[m.scroll:min(len(lines), m.scroll+maxLines)], "\n") + "\n\n↑↓/PgUp/PgDn " + m.tr("scroll") + " · Home/End · F1/Esc " + m.tr("close")
	} else if m.mode == "language" {
		content = m.tr("select_language") + "\n\n"
		for i, language := range i18n.Languages() {
			content += row(i == m.languageCursor, language.Label+"  ["+language.Code+"]") + "\n"
		}
		content += "\n" + m.tr("confirm")
	} else {
		content = m.prompt + "\n\n> " + m.input + "█\n\nEnter " + m.tr("select") + " · Esc " + m.tr("cancel")
	}
	width := min(88, m.w-8)
	modal := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(m.th.Color(map[bool]string{true: m.th.Danger, false: m.th.Focus}[m.mode == "confirm"])).Background(m.th.Color(m.th.Panel)).Padding(1, 2).Width(width).MaxHeight(m.h - 4).Render(content)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, modal)
}
func (m *Model) cycleTheme() {
	n := theme.Names()
	for i, x := range n {
		if x == m.th.Name {
			m.th = theme.Get(n[(i+1)%len(n)])
			m.cfg.Theme = m.th.Name
			if err := config.Save("", m.cfg); err != nil {
				m.err = fmt.Sprintf(m.tr("theme_save_error"), err.Error())
			}
			m.status = fmt.Sprintf(m.tr("theme_changed"), m.th.Name)
			return
		}
	}
}
func stateColor(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#56F39A")).Render(s)
}
func helpWidth(s string) int { return lipgloss.Width(s) }
