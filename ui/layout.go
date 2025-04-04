package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ashworthpayne/radiosubtext/modems"
	"github.com/ashworthpayne/radiosubtext/proto"
	"github.com/charmbracelet/lipgloss"
)

var (
	topBarStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240")).Padding(0, 1)
	userListStyle = lipgloss.NewStyle().Width(20).Padding(0, 1).BorderRight(true)
	chatStyle     = lipgloss.NewStyle().Padding(0, 0)
	inputStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1).BorderTop(true)
)

type Model struct {
	width, height int
	messages      []proto.Message
	input         textinput.Model
	viewport      viewport.Model
	radio         modems.Modem
	callSign      string
	group         string
	SendQueue     chan proto.Message

	rxActive bool
	txActive bool
	lastSeen map[string]time.Time
}

func NewModel(r modems.Modem, callSign, group string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type message…"
	ti.Focus()
	ti.Width = 40

	vp := viewport.New(80, 20)
	vp.SetContent("")

	return Model{
		messages:  []proto.Message{},
		input:     ti,
		viewport:  vp,
		radio:     r,
		callSign:  callSign,
		group:     group,
		SendQueue: make(chan proto.Message, 10),
		lastSeen:  make(map[string]time.Time),
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Push(msg proto.Message) {
	m.messages = append(m.messages, msg)
	if len(m.messages) > 100 {
		m.messages = m.messages[len(m.messages)-100:]
	}

	if msg.Cmd == proto.CmdFingerRes {
		cache, _ := proto.LoadFingerCache()
		cache[msg.From] = proto.FingerEntry{
			Callsign:     msg.From,
			LastResponse: msg.Body,
			Updated:      time.Now(),
		}
		_ = proto.SaveFingerCache(cache)
	}

	if msg.From != m.callSign {
		m.rxActive = true
		m.lastSeen[msg.From] = time.Now()
	}

	m.viewport.SetContent(DrawScrollback(m.messages, m.callSign))
	m.viewport.GotoBottom()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		usable := msg.Width - userListStyle.GetWidth() - 2
		m.input.Width = usable
		m.viewport.Width = usable
		m.viewport.Height = msg.Height - 5
		m.viewport.SetContent(DrawScrollback(m.messages, m.callSign))
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			body := strings.TrimSpace(m.input.Value())
			if body == "" {
				return m, nil
			}
			if body == "/quit" {
				quitMsg := proto.Message{
					From:    m.callSign,
					Group:   m.group,
					Cmd:     proto.CmdMessage,
					Body:    "📡 signed off.",
					Created: time.Now(),
				}
				_ = m.radio.Send(quitMsg)
				m.Push(quitMsg)
				return m, tea.Quit
			}
			if strings.HasPrefix(body, "/finger") {
				args := strings.Fields(body)
				if len(args) >= 2 {
					target := args[1]
					msg := proto.Message{
						From:  m.callSign,
						Group: m.group,
						Cmd:   proto.CmdFingerReq,
						Body:  target,
					}
					m.SendQueue <- msg
					m.Push(msg)
					m.txActive = true
					m.input.Reset()
					return m, nil
				}
			}
			if strings.HasPrefix(body, "/whois") {
				args := strings.Fields(body)
				if len(args) >= 2 {
					target := strings.ToUpper(args[1])
					cache, _ := proto.LoadFingerCache()
					entry, ok := cache[target]
					if ok {
						ago := time.Since(entry.Updated).Round(time.Second)
						m.Push(proto.Message{
							From:  "CACHE",
							Group: "@local",
							Cmd:   "WHOIS",
							Body:  fmt.Sprintf("%s (%s ago)", entry.LastResponse, ago),
						})
					} else {
						m.Push(proto.Message{
							From:  "CACHE",
							Group: "@local",
							Cmd:   "WHOIS",
							Body:  "No cache entry.",
						})
					}
					m.input.Reset()
					return m, nil
				}
			}
			msgOut := proto.Message{
				From:    m.callSign,
				Group:   m.group,
				Cmd:     proto.CmdMessage,
				Body:    body,
				Created: time.Now(),
			}
			m.SendQueue <- msgOut
			m.Push(msgOut)
			m.txActive = true
			m.input.Reset()
			return m, nil
		}
	case tea.MouseMsg:
		m.viewport, cmd = m.viewport.Update(msg)
	}
	m.viewport, _ = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	topBar := DrawTopBar(m.group, m.width)
	userList := DrawUserList(m.KnownUsers(), m.MailMap(), m.CacheMap())
	chat := m.viewport.View()
	input := DrawInputBox(m.input.View(), m.rxActive, m.txActive, m.width-userListStyle.GetWidth()-2)

	main := lipgloss.JoinVertical(lipgloss.Left, topBar, chat, input)
	return lipgloss.JoinHorizontal(lipgloss.Top, userList, main)
}

func DrawScrollback(msgs []proto.Message, self string) string {
	var b strings.Builder
	wrap := lipgloss.NewStyle().Width(80)

	for _, msg := range msgs {
		var prefix string
		switch msg.Cmd {
		case proto.CmdMessage:
			if msg.From == self {
				prefix = "You: "
				b.WriteString(wrap.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(prefix + msg.Body)))
			} else {
				b.WriteString(wrap.Render(fmt.Sprintf("%s: %s", msg.From, msg.Body)))
			}
		case proto.CmdFingerRes:
			b.WriteString(wrap.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("finger reply: " + msg.Body)))
		case proto.CmdFingerReq:
			b.WriteString(wrap.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("finger → " + msg.Body)))
		case "WHOIS":
			b.WriteString(wrap.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("whois → " + msg.Body)))
		}
		b.WriteString("\n")
	}
	return chatStyle.Render(b.String())
}

func DrawInputBox(input string, rxActive, txActive bool, width int) string {
	prefix := "⚫️"
	if txActive {
		prefix = "🔴"
	} else if rxActive {
		prefix = "🟢"
	}
	return inputStyle.Width(width).Render(fmt.Sprintf("%s %s", prefix, input))
}

func DrawTopBar(currentGroup string, width int) string {
	utc := time.Now().UTC().Format("15:04 UTC")
	local := time.Now().Format("15:04 MST")
	center := fmt.Sprintf("@%s", currentGroup)

	left := topBarStyle.Render("🕒 " + utc)
	right := topBarStyle.Render(local + " 🕒")
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	centerStyle := topBarStyle.Copy().Align(lipgloss.Center).Width(width - leftWidth - rightWidth)
	centerBlock := lipgloss.JoinHorizontal(lipgloss.Top, left, centerStyle.Render(center), right)

	return lipgloss.PlaceHorizontal(width, lipgloss.Center, centerBlock)
}

func DrawUserList(known []string, mail map[string]bool, cached map[string]bool) string {
	var b strings.Builder
	b.WriteString("@CQ\n")
	for _, call := range known {
		line := call
		if mail[call] {
			line = "✉️  " + line
		}
		if cached[call] {
			line = "⭐ " + line
		}
		b.WriteString(line + "\n")
	}
	return userListStyle.Render(b.String())
}

func (m Model) KnownUsers() []string {
	set := make(map[string]bool)
	for _, msg := range m.messages {
		set[msg.From] = true
	}
	delete(set, m.callSign)
	var list []string
	for k := range set {
		list = append(list, k)
	}
	sort.Strings(list)
	return list
}

func (m Model) MailMap() map[string]bool {
	return map[string]bool{} // Placeholder for future feature
}

func (m Model) CacheMap() map[string]bool {
	cache, _ := proto.LoadFingerCache()
	out := make(map[string]bool)
	for k := range cache {
		out[k] = true
	}
	return out
}
