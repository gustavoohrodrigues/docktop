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
	"github.com/docktop/docktop/internal/registry"
	"github.com/docktop/docktop/internal/theme"
	"github.com/docktop/docktop/internal/utils"
)

var tabs = []string{"Dashboard", "Containers", "Images", "Registry", "Volumes", "Networks", "Swarm", "Services", "Nodes", "Stacks", "Events", "Audit", "Settings"}

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
type tickMsg time.Time

type Model struct {
	ctx                                      context.Context
	cfg                                      config.Config
	engine                                   dock.Engine
	audit                                    *audit.Logger
	hub                                      *registry.Hub
	version                                  string
	w, h, tab, cursor, scroll                int
	snap                                     dock.Snapshot
	results                                  []registry.Result
	err, status, mode, input, prompt, detail string
	loading, auto                            bool
	targetID, targetName, action             string
	th                                       theme.Theme
	cpuHistory, memHistory                   []float64
}

func New(ctx context.Context, c config.Config, e dock.Engine, a *audit.Logger, v string) *Model {
	return &Model{ctx: ctx, cfg: c, engine: e, audit: a, hub: registry.NewHub(), version: v, auto: true, loading: true, th: theme.Get(c.Theme)}
}
func (m *Model) Init() tea.Cmd { return tea.Batch(m.refresh(), m.tick()) }
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
			m.err = x.e.Error()
		} else {
			m.snap = x.s
			m.captureHistory()
			m.clamp()
		}
	case searchMsg:
		m.loading = false
		m.results = x.results
		m.cursor = 0
		if x.e != nil {
			m.err = x.e.Error()
		} else {
			m.status = fmt.Sprintf("%d resultados no Docker Hub", len(x.results))
		}
	case opMsg:
		m.loading = false
		m.status = x.action + " concluído"
		m.detail = x.detail
		if x.e != nil {
			m.err = x.e.Error()
			m.status = x.action + " falhou"
		}
		result, errText := "ok", ""
		if x.e != nil {
			result = "erro"
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
			m.err = x.e.Error()
		} else {
			m.status = "sessão exec encerrada; TUI restaurada"
		}
		return m, m.refresh()
	case tickMsg:
		if m.auto && m.mode == "" {
			m.loading = true
			return m, tea.Batch(m.refresh(), m.tick())
		}
		return m, m.tick()
	case tea.KeyMsg:
		if m.mode != "" {
			return m.updateOverlay(x)
		}
		return m.updateKeys(x)
	}
	return m, nil
}

func (m *Model) updateKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab", "right":
		m.tab = (m.tab + 1) % len(tabs)
		m.cursor = 0
	case "shift+tab", "left":
		m.tab = (m.tab + len(tabs) - 1) % len(tabs)
		m.cursor = 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < m.count()-1 {
			m.cursor++
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = max(0, m.count()-1)
	case "R":
		m.auto = !m.auto
	case "?", "f1":
		m.scroll = 0
		m.mode = "help"
	case "t":
		m.cycleTheme()
	case "/":
		if m.tab == 3 {
			m.openInput("search", "Pesquisar Docker Hub", "ex.: nginx, postgres, redis")
		}
	case "r":
		if m.tab == 1 {
			return m, m.containerOp("restart")
		}
		m.loading = true
		return m, m.refresh()
	case "S":
		return m, m.containerOp("start")
	case "T":
		return m, m.containerOp("stop")
	case "p":
		switch m.tab {
		case 1:
			return m, m.containerOp("pause")
		case 2:
			m.openInput("pull", "Baixar imagem", "ex.: nginx:1.27")
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
			m.openInput("create-container", "CRIAR E INICIAR CONTAINER", "Formato (7 campos separados por |):\nnome | imagem | portas | volumes | ambiente | restart | comando\n\nExemplo nginx:\nweb | nginx:alpine | 8080:80 | /srv/site:/usr/share/nginx/html:ro | APP_ENV=prod | unless-stopped |\n\nCampos opcionais podem ficar vazios. A imagem será baixada automaticamente se necessário.")
		case 4:
			m.openInput("create-volume", "Criar volume", "nome driver(opcional)")
		case 5:
			m.openInput("create-network", "Criar rede", "nome driver: bridge|overlay|macvlan")
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
			m.err = "confirmação não corresponde ao recurso"
			return m, nil
		}
		m.mode = ""
		return m, m.removeTarget()
	}
	if input == "" {
		m.err = "valor obrigatório"
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
			m.err = parseErr.Error()
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
	}
	return m, nil
}

func (m *Model) readOnly() bool {
	if m.cfg.ReadOnly {
		m.err = "operação bloqueada: modo read-only"
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
func (m *Model) containerOp(a string) tea.Cmd {
	if m.tab != 1 || len(m.snap.Containers) == 0 || m.readOnly() {
		return nil
	}
	x := m.snap.Containers[m.cursor]
	if a == "pause" && x.State == "paused" {
		a = "unpause"
	}
	m.status = map[string]string{"start": "Iniciando container selecionado…", "stop": "Solicitando parada graciosa (timeout 10s)…", "restart": "Reiniciando container selecionado…", "pause": "Pausando processos do container…", "unpause": "Retomando processos do container…"}[a]
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
	m.status = "Baixando " + ref + " pela Docker Engine API; aguarde as camadas…"
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
		m.err = e.Error()
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
			m.err = "remoção de containers bloqueada pela política"
			return m, nil
		}
		if len(m.snap.Containers) > 0 {
			x := m.snap.Containers[m.cursor]
			m.targetID, m.targetName, m.action = x.ID, x.Name, "remove-container"
		}
	case 2:
		if !m.cfg.DangerousActions.RemoveImages {
			m.err = "remoção de imagens bloqueada pela política"
			return m, nil
		}
		if len(m.snap.Images) > 0 {
			x := m.snap.Images[m.cursor]
			m.targetID, m.targetName, m.action = x.ID, shortID(x.ID), "remove-image"
		}
	case 4:
		if !m.cfg.DangerousActions.RemoveVolumes {
			m.err = "remoção de volumes bloqueada pela política"
			return m, nil
		}
		if len(m.snap.Volumes) > 0 {
			x := m.snap.Volumes[m.cursor]
			m.targetID, m.targetName, m.action = x.Name, x.Name, "remove-volume"
		}
	case 5:
		if !m.cfg.DangerousActions.RemoveNetworks {
			m.err = "remoção de redes bloqueada pela política"
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
	m.prompt = "AÇÃO DESTRUTIVA\nImpacto: o recurso poderá ser perdido ou causar indisponibilidade.\nDigite exatamente " + m.targetName
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
		}
		return opMsg{action: action, resource: name, e: e}
	}
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
	}
	return 0
}
func (m *Model) clamp() { m.cursor = min(m.cursor, max(0, m.count()-1)) }
func sanitize(s string) string {
	for _, p := range []string{"password=", "token=", "authorization=", "key="} {
		if i := strings.Index(strings.ToLower(s), p); i >= 0 {
			s = s[:i] + p + "[redacted]"
		}
	}
	return s
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
	if m.w < 76 || m.h < 20 {
		return "DockerMin\n\nTerminal mínimo: 76x20. Atual: " + fmt.Sprintf("%dx%d", m.w, m.h)
	}
	bg := lipgloss.NewStyle().Background(m.th.Color(m.th.Background)).Foreground(m.th.Color(m.th.Text))
	out := m.top() + "\n" + m.body() + "\n" + m.footer()
	if m.mode != "" {
		out = m.overlay()
	}
	return bg.Width(m.w).Height(m.h).Render(out)
}
func (m *Model) top() string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(m.th.Color(m.th.Primary)).Render("◆ DockerMin")
	tag := lipgloss.NewStyle().Foreground(m.th.Color(m.th.Muted)).Render("  Docker & Swarm Operations Console")
	head := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, logo+tag)
	var items string
	start := max(0, m.tab-4)
	end := min(len(tabs), start+9)
	for i := start; i < end; i++ {
		st := lipgloss.NewStyle().Padding(0, 1).Foreground(m.th.Color(m.th.Muted))
		if i == m.tab {
			st = st.Bold(true).Foreground(m.th.Color(m.th.Background)).Background(m.th.Color(m.th.Primary))
		}
		items += st.Render(tabs[i])
	}
	help := lipgloss.NewStyle().Bold(true).Foreground(m.th.Color(m.th.Warning)).Render("[F1 AJUDA]")
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
		message := "sincronizando com Docker Engine…"
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
	default:
		b = m.infoTab()
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
	perf := fmt.Sprintf("CPU containers  %6.1f%%  %s\n%s\n\nRAM containers  %s  %5.1f%%\n%s", cpu, meter(min(cpu, 100), 18, m.th.Color(m.th.Primary)), spark(m.cpuHistory, graphW, m.th.Color(m.th.Primary)), utils.Bytes(int64(memBytes)), mem, meter(min(mem, 100), 18, m.th.Color(m.th.Success)))
	engine := fmt.Sprintf("● %-18s %s\nEngine %-14s API negociada\n%s / %s\nStorage: %s\nCPUs: %d   RAM: %s", i.Name, stateColor("online"), m.snap.Version, i.OperatingSystem, i.Architecture, i.Driver, i.NCPU, utils.Bytes(i.MemTotal))
	top := lipgloss.JoinHorizontal(lipgloss.Top, m.panel("PERFORMANCE", perf, left, 8), m.panel("ENGINE", engine, right, 8))
	resources := fmt.Sprintf("CONTAINERS  %d total  %s %d running  %s %d stopped     IMAGES %d     VOLUMES %d     NETWORKS %d", len(m.snap.Containers), stateColor("●"), running, lipgloss.NewStyle().Foreground(m.th.Color(m.th.Muted)).Render("●"), stopped, len(m.snap.Images), len(m.snap.Volumes), len(m.snap.Networks))
	return top + "\n" + m.panel("RESOURCE PULSE", resources, inner, 3) + "\n" + lipgloss.NewStyle().Foreground(m.th.Color(m.th.Muted)).Render("Amostras reais de stats · picos normalizados no histórico · atualizado "+m.snap.At.Format("15:04:05"))
}
func (m *Model) containers() string {
	b := "   NOME               ID           IMAGEM                 ESTADO      CPU     RAM      STATUS\n"
	for i, x := range m.snap.Containers {
		v := m.snap.Metrics[x.ID]
		b += row(i == m.cursor, fmt.Sprintf("%-18.18s %-12.12s %-22.22s %-10s %6.1f%% %7s  %s", x.Name, x.ID, x.Image, x.State, v.CPU, utils.Bytes(int64(v.MemoryBytes)), x.Status)) + "\n"
	}
	if len(m.snap.Containers) == 0 {
		b += "\n  Nenhum container. Pressione n para criar e iniciar um."
	}
	return b + "\n  n criar  S start  T stop  r restart  p pause  l logs  i inspect  o processos  e shell  d remover"
}
func (m *Model) images() string {
	b := "   REPOSITÓRIO/TAG                              ID           TAMANHO\n"
	for i, x := range m.snap.Images {
		b += row(i == m.cursor, fmt.Sprintf("%-43.43s %-12.12s %s", x.Tags, shortID(x.ID), utils.Bytes(x.Size))) + "\n"
	}
	if len(m.snap.Images) == 0 {
		b += "\n  Nenhuma imagem local."
	}
	return b + "\n  p pull por referência  d remover com confirmação"
}
func (m *Model) registry() string {
	b := "   REPOSITÓRIO                         STARS       PULLS  TIPO        DESCRIÇÃO\n"
	for i, x := range m.results {
		kind := "community"
		if x.Official {
			kind = "official"
		}
		b += row(i == m.cursor, fmt.Sprintf("%-34.34s %7d %11d  %-10s  %.50s", x.Name, x.Stars, x.Pulls, kind, x.Description)) + "\n"
	}
	if len(m.results) == 0 {
		b += "\n  Pressione / para pesquisar imagens no Docker Hub."
	}
	return b + "\n  / pesquisar  p baixar selecionada (latest)"
}
func (m *Model) volumes() string {
	b := "   NOME                                      DRIVER       SCOPE\n"
	for i, x := range m.snap.Volumes {
		b += row(i == m.cursor, fmt.Sprintf("%-41.41s %-12s %s", x.Name, x.Driver, x.Scope)) + "\n"
	}
	return b + "\n  n criar volume  d remover (confirmação reforçada)"
}
func (m *Model) networks() string {
	b := "   NOME                         ID           DRIVER       SCOPE\n"
	for i, x := range m.snap.Networks {
		b += row(i == m.cursor, fmt.Sprintf("%-28.28s %-12.12s %-12s %s", x.Name, x.ID, x.Driver, x.Scope)) + "\n"
	}
	return b + "\n  n criar rede  d remover (confirmação digitada)"
}
func (m *Model) infoTab() string {
	return lipgloss.NewStyle().Bold(true).Foreground(m.th.Color(m.th.Primary)).Render(tabs[m.tab]) + "\n\nMódulo reservado para a próxima etapa operacional. Nenhuma ação simulada é exibida nesta versão."
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
	right := fmt.Sprintf("? ajuda  Tab abas  r refresh  R auto:%v  t tema  q sair ", m.auto)
	return left + strings.Repeat(" ", max(1, m.w-lipgloss.Width(left)-lipgloss.Width(right))) + right
}
func (m *Model) overlay() string {
	var content string
	if m.mode == "detail" {
		lines := strings.Split(m.detail, "\n")
		maxLines := m.h - 10
		m.scroll = min(m.scroll, max(0, len(lines)-maxLines))
		content = strings.Join(lines[m.scroll:min(len(lines), m.scroll+maxLines)], "\n") + "\n\n↑↓ rolar · Esc fechar"
	} else if m.mode == "help" {
		lines := strings.Split(helpManual, "\n")
		maxLines := m.h - 9
		m.scroll = min(m.scroll, max(0, len(lines)-maxLines))
		content = strings.Join(lines[m.scroll:min(len(lines), m.scroll+maxLines)], "\n") + "\n\n↑↓/PgUp/PgDn rolar · Home/End · F1 ou Esc fechar"
	} else {
		content = m.prompt + "\n\n> " + m.input + "█\n\nEnter confirma · Esc cancela"
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
			m.status = "tema: " + m.th.Name
			return
		}
	}
}
func stateColor(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#56F39A")).Render(s)
}
func helpWidth(s string) int { return lipgloss.Width(s) }
