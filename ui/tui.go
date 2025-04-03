package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ashworthpayne/radiosubtext/modems"
	"github.com/ashworthpayne/radiosubtext/proto"
)

type Model struct {
	messages     []proto.Message
	input        textinput.Model
	radio        modems.Modem
	callSign     string
	group        string
	SendQueue    chan proto.Message
	scrollOffset int
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
		scrollOffset: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			body := strings.TrimSpace(m.input.Value())
			if body == "" {
				break
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
					m.input.Reset()
					break
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
					break
				}
			}

			msg := proto.Message{
				From:  m.callSign,
				Group: m.group,
				Cmd:   proto.CmdMessage,
				Body:  body,
			}
			m.SendQueue <- msg
			m.Push(msg)
			m.input.Reset()

		case "up", "k":
			if m.scrollOffset < len(m.messages)-1 {
				m.scrollOffset++
			}
		case "down", "j":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("📻  RadioSubtext\n")
	b.WriteString("----------------------------------\n")

	visible := 15
	start := len(m.messages) - visible - m.scrollOffset
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > len(m.messages) {
		end = len(m.messages)
	}

	for _, msg := range m.messages[start:end] {
		var prefix string

		switch msg.Cmd {
		case proto.CmdMessage:
			prefix = fmt.Sprintf("[%s] %s: ", msg.Group, msg.From)
		case proto.CmdFingerRes:
			prefix = fmt.Sprintf("[💁 %s] ", msg.From)
		case "WHOIS":
			prefix = fmt.Sprintf("[📓 %s] ", msg.From)
		case proto.CmdFingerReq:
			prefix = fmt.Sprintf("[📨 finger→%s] ", msg.Body)
		default:
			prefix = fmt.Sprintf("[?? %s] ", msg.From)
		}

		b.WriteString(fmt.Sprintf("%s%s\n", prefix, msg.Body))
	}

	b.WriteString("\n↑/↓ or j/k to scroll | q to quit\n")
	b.WriteString("> " + m.input.View())
	return b.String()
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

}
