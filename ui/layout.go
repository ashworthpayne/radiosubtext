package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ashworthpayne/radiosubtext/modems"
	"github.com/ashworthpayne/radiosubtext/proto"
	"github.com/charmbracelet/lipgloss"
)

var (
	topBarStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240")).Padding(0, 1)
	userListStyle = lipgloss.NewStyle().Width(20).Padding(0, 1).BorderRight(true)
	chatStyle     = lipgloss.NewStyle().Padding(0, 1).MaxWidth(0)
	inputStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1).BorderTop(true)
)

func DrawTopBar(currentGroup string, width int) string {
	utc := time.Now().UTC().Format("15:04 UTC")
	local := time.Now().Format("15:04 MST")
	center := fmt.Sprintf("@%s", currentGroup)

	left := topBarStyle.Render("🕒 " + utc)
	right := topBarStyle.Render(local + " 🕒")
	centerStyle := topBarStyle.Copy().Align(lipgloss.Center).Width(width - len(left) - len(right) - 4)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		centerStyle.Render(center),
		right,
	)
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

func DrawScrollback(msgs []proto.Message, self string) string {
	var b strings.Builder
	for _, msg := range msgs {
		var prefix string
		switch msg.Cmd {
		case proto.CmdMessage:
			if msg.From == self {
				prefix = "You: "
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(prefix + msg.Body + "\n"))
			} else {
				b.WriteString(fmt.Sprintf("%s: %s\n", msg.From, msg.Body))
			}
		case proto.CmdFingerRes:
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("finger reply: " + msg.Body + "\n"))
		case "WHOIS":
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("whois: " + msg.Body + "\n"))
		default:
			b.WriteString(fmt.Sprintf("%s: %s\n", msg.From, msg.Body))
		}
	}
	return chatStyle.Render(b.String())
}

func DrawInputBox(input string, rxActive, txActive bool, width int) string {
	rx := "⚫ RX"
	tx := "⚫ TX"
	if rxActive {
		rx = "🟢 RX"
	}
	if txActive {
		tx = "🔴 TX"
	}
	status := fmt.Sprintf("%s  %s", rx, tx)
	return inputStyle.Width(width).Render(fmt.Sprintf("%s> %s", status, input))
}

type Model struct {
	width        int
	height       int
	messages     []proto.Message
	input        textinput.Model
	radio        modems.Modem
	callSign     string
	group        string
	SendQueue    chan proto.Message
	ScrollOffset int
	rxActive     bool
	txActive     bool
	lastSeen     map[string]time.Time
}

func NewModel(r modems.Modem, callSign, group string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type message…"
	ti.Focus()
	ti.Width = 40

	return Model{
		messages:     []proto.Message{},
		input:        ti,
		radio:        r,
		callSign:     callSign,
		group:        group,
		SendQueue:    make(chan proto.Message, 10),
		ScrollOffset: 0,
		lastSeen:     make(map[string]time.Time),
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Push(msg proto.Message) {
	wasAtBottom := m.ScrollOffset == 0
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
		if wasAtBottom {
			m.ScrollOffset = 0
		} else {
			m.ScrollOffset++
		}
		m.rxActive = true
		m.lastSeen[msg.From] = time.Now()
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.ScrollOffset < len(m.messages)-1 {
				m.ScrollOffset++
			}
			return m, nil
		case "down", "j":
			if m.ScrollOffset > 0 {
				m.ScrollOffset--
			}
			return m, nil
		case "enter":
			body := strings.TrimSpace(m.input.Value())
			if body == "/quit" {
				quitMsg := proto.Message{
					From:    m.callSign,
					Group:   m.group,
					Cmd:     proto.CmdMessage,
					Body:    "📡 signed off.",
					Created: time.Now(),
				}
				m.SendQueue <- quitMsg
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
							Body:  fmt.Sprintf("No cached entry for %s", target),
						})
					}
					m.input.Reset()
					return m, nil
				}
			}

			// Default message handling
			msg := proto.Message{
				From:  m.callSign,
				Group: m.group,
				Cmd:   proto.CmdMessage,
				Body:  body,
			}
			m.SendQueue <- msg
			m.Push(msg)
			m.txActive = true
			m.input.Reset()
			return m, nil
		}

	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	maxHeight := m.height
	if maxHeight == 0 {
		maxHeight = 24 // fallback for non-TTY environments
	}
	top := DrawTopBar(m.group, m.width)

	cutoff := time.Now().Add(-5 * time.Minute)
	var known []string
	for callsign, t := range m.lastSeen {
		if t.After(cutoff) {
			known = append(known, callsign)
		}
	}
	sort.Strings(known)

	mail := map[string]bool{}
	cache, _ := proto.LoadFingerCache()
	cached := make(map[string]bool)
	for _, c := range known {
		if _, ok := cache[c]; ok {
			cached[c] = true
		}
	}

	left := DrawUserList(known, mail, cached)
	scrollLimit := maxHeight - 6 // space for top, input, padding
	if scrollLimit > len(m.messages) {
		scrollLimit = len(m.messages)
	}
	// Clamp scroll offset to prevent underflow
	if m.ScrollOffset > len(m.messages)-scrollLimit {
		m.ScrollOffset = len(m.messages) - scrollLimit
	}
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}
	start := len(m.messages) - scrollLimit - m.ScrollOffset
	if start < 0 {
		start = 0
	}
	right := DrawScrollback(m.messages[start:start+scrollLimit], m.callSign)
	bottom := DrawInputBox(m.input.View(), m.rxActive, m.txActive, m.width)

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return top + "\n" + content + "\n" + bottom
}
