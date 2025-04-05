package ui

import (
	"fmt"
	"log"
	"os"
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
	// Styles for different message types
	systemStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")) // Yellow for system messages
	selfStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#42A5F5")) // Blue for own messages
	otherStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")) // White for other messages
	userListStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(20)
	topBarStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240")).Padding(0, 1)
	chatStyle   = lipgloss.NewStyle().Padding(0, 0)
	inputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1).BorderTop(true)
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
	RecvQueue     chan proto.Message

	rxActive      bool
	txActive      bool
	lastRX        time.Time
	lastTX        time.Time
	statusTimeout time.Duration
	lastSeen      map[string]time.Time
}

func init() {
	// Create debug log file
	f, _ := os.OpenFile("/tmp/ui_debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if f != nil {
		log.SetOutput(f)
	}
}

func NewModel(r modems.Modem, callSign, group string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type message…"
	ti.Focus()
	ti.Width = 40

	vp := viewport.New(80, 20)
	vp.SetContent("")

	return Model{
		messages:      []proto.Message{},
		input:         ti,
		viewport:      vp,
		radio:         r,
		callSign:      callSign,
		group:         group,
		SendQueue:     make(chan proto.Message, 10),
		RecvQueue:     make(chan proto.Message, 10),
		statusTimeout: 500 * time.Millisecond,
		lastSeen:      make(map[string]time.Time),
	}
}

func checkMessages() tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		checkMessages(),
	)
}

func (m Model) Push(msg proto.Message) Model {
	// Debug log to file instead of terminal
	log.Printf("DEBUG: Received message - From: %s, Group: %s, Body: %s\n", msg.From, msg.Group, msg.Body)

	// Create a copy to modify
	copy := m

	copy.messages = append(copy.messages, msg)
	if len(copy.messages) > 100 {
		copy.messages = copy.messages[len(copy.messages)-100:]
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

	if msg.From != copy.callSign {
		copy.rxActive = true
		copy.lastRX = time.Now()
		copy.lastSeen[msg.From] = time.Now()
	}

	content := DrawScrollback(copy.messages, copy.callSign, copy.group)
	log.Printf("DEBUG: Drawing scrollback with %d messages for group %s\n", len(copy.messages), copy.group)
	copy.viewport.SetContent(content)
	copy.viewport.GotoBottom()

	return copy
}

func (m Model) statusIndicator() string {
	now := time.Now()

	// Check if we're in TX timeout period
	if m.txActive && now.Sub(m.lastTX) < m.statusTimeout {
		return "🔴"
	}

	// Check if we're in RX timeout period
	if m.rxActive && now.Sub(m.lastRX) < m.statusTimeout {
		return "🟢"
	}

	// Default to idle
	return "⚫️"
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// Create a copy we can modify
	copy := m

	select {
	case newMsg := <-m.RecvQueue:
		// Add message to our buffer
		copy.messages = append(copy.messages, newMsg)
		if len(copy.messages) > 100 {
			copy.messages = copy.messages[len(copy.messages)-100:]
		}

		if newMsg.From != copy.callSign {
			copy.rxActive = true
			copy.lastRX = time.Now()
			copy.lastSeen[newMsg.From] = time.Now()
		}

		// Update viewport
		copy.viewport.SetContent(DrawScrollback(copy.messages, copy.callSign, copy.group))
		copy.viewport.GotoBottom()
	default:
		// No message waiting
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		copy.width = msg.Width
		copy.height = msg.Height
		usable := msg.Width - userListStyle.GetWidth() - 2
		copy.input.Width = usable
		copy.viewport.Width = usable
		copy.viewport.Height = msg.Height - 5
		copy.viewport.SetContent(DrawScrollback(copy.messages, copy.callSign, copy.group))
		copy.viewport.GotoBottom()

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			body := strings.TrimSpace(copy.input.Value())
			if body == "" {
				return copy, nil
			}

			// Handle commands
			if strings.HasPrefix(body, "/") {
				switch {
				case body == "/quit":
					quitMsg := proto.Message{
						From:    copy.callSign,
						Group:   copy.group,
						Cmd:     proto.CmdMessage,
						Body:    "📡 signed off.",
						Created: time.Now(),
					}
					_ = copy.radio.Send(quitMsg)
					copy = copy.Push(proto.Message{
						From:    "SYSTEM",
						Group:   "@local",
						Cmd:     proto.CmdMessage,
						Body:    fmt.Sprintf("Leaving group %s", copy.group),
						Created: time.Now(),
					})
					return copy, tea.Quit

				case strings.HasPrefix(body, "/join"):
					args := strings.Fields(body)
					if len(args) >= 2 {
						newGroup := args[1]
						if !strings.HasPrefix(newGroup, "@") {
							newGroup = "@" + newGroup
						}
						oldGroup := copy.group
						copy.group = newGroup

						// Notify both groups about the change
						leaveMsg := proto.Message{
							From:    copy.callSign,
							Group:   oldGroup,
							Cmd:     proto.CmdMessage,
							Body:    "📡 leaving channel.",
							Created: time.Now(),
						}
						joinMsg := proto.Message{
							From:    copy.callSign,
							Group:   newGroup,
							Cmd:     proto.CmdMessage,
							Body:    "📡 joined channel.",
							Created: time.Now(),
						}
						sysMsg := proto.Message{
							From:    "SYSTEM",
							Group:   "@local",
							Cmd:     proto.CmdMessage,
							Body:    fmt.Sprintf("Switched from %s to %s", oldGroup, newGroup),
							Created: time.Now(),
						}

						_ = copy.radio.Send(leaveMsg)
						_ = copy.radio.Send(joinMsg)
						copy = copy.Push(sysMsg)

						// Update viewport for new group
						copy.viewport.SetContent(DrawScrollback(copy.messages, copy.callSign, copy.group))
						copy.viewport.GotoBottom()
					}
					copy.input.Reset()
					return copy, nil

				case strings.HasPrefix(body, "/finger"):
					args := strings.Fields(body)
					if len(args) >= 2 {
						target := args[1]
						msg := proto.Message{
							From:  copy.callSign,
							Group: copy.group,
							Cmd:   proto.CmdFingerReq,
							Body:  target,
						}
						copy.SendQueue <- msg
						copy = copy.Push(msg)
						copy.txActive = true
						copy.lastTX = time.Now()
						copy.input.Reset()
						return copy, nil
					}
				case strings.HasPrefix(body, "/whois"):
					args := strings.Fields(body)
					if len(args) >= 2 {
						target := strings.ToUpper(args[1])
						cache, _ := proto.LoadFingerCache()
						entry, ok := cache[target]
						if ok {
							ago := time.Since(entry.Updated).Round(time.Second)
							copy = copy.Push(proto.Message{
								From:  "CACHE",
								Group: "@local",
								Cmd:   "WHOIS",
								Body:  fmt.Sprintf("%s (%s ago)", entry.LastResponse, ago),
							})
						} else {
							copy = copy.Push(proto.Message{
								From:  "CACHE",
								Group: "@local",
								Cmd:   "WHOIS",
								Body:  "No cache entry.",
							})
						}
						copy.input.Reset()
						return copy, nil
					}
				}
			} else {
				// Regular message
				msgOut := proto.Message{
					From:    copy.callSign,
					Group:   copy.group,
					Cmd:     proto.CmdMessage,
					Body:    body,
					Created: time.Now(),
				}
				copy.SendQueue <- msgOut
				copy = copy.Push(msgOut)
				copy.txActive = true
				copy.lastTX = time.Now()
			}
			copy.input.Reset()
			return copy, nil
		}
	}

	// Clear status indicators after timeout
	now := time.Now()
	if copy.txActive && now.Sub(copy.lastTX) >= copy.statusTimeout {
		copy.txActive = false
	}
	if copy.rxActive && now.Sub(copy.lastRX) >= copy.statusTimeout {
		copy.rxActive = false
	}

	// Handle viewport updates
	copy.viewport, cmd = copy.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Handle input updates
	copy.input, cmd = copy.input.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Always check for more messages
	cmds = append(cmds, checkMessages())

	return copy, tea.Batch(cmds...)
}

func (m Model) View() string {
	topBar := DrawTopBar(m.group, m.width)
	userList := DrawUserList(m.KnownUsers(), m.MailMap(), m.CacheMap())
	chat := m.viewport.View()
	input := DrawInputBox(m.input.View(), m.rxActive, m.txActive, m.width-userListStyle.GetWidth()-2)

	main := lipgloss.JoinVertical(lipgloss.Left, topBar, chat, input)
	return lipgloss.JoinHorizontal(lipgloss.Top, userList, main)
}

func DrawScrollback(messages []proto.Message, callSign, group string) string {
	var out strings.Builder

	for _, msg := range messages {
		// Skip messages not in our group
		if msg.Group != group && msg.Group != "@local" {
			continue
		}

		// Format the message
		line := fmt.Sprintf("%s: %s", msg.From, msg.Body)

		// Style based on message type
		switch {
		case msg.Group == "@local":
			// System messages in yellow
			line = systemStyle.Render(line)
		case strings.HasSuffix(strings.TrimSpace(msg.Body), "?"):
			// Questions in red
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(line)
		case msg.From == callSign:
			// Our messages in blue
			line = selfStyle.Render(line)
		default:
			// Other messages in default color
			line = otherStyle.Render(line)
		}

		out.WriteString(line + "\n")
	}

	return out.String()
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
