package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ashworthpayne/radiosubtext/proto"
	"github.com/ashworthpayne/radiosubtext/radio"
	"github.com/ashworthpayne/radiosubtext/ui"
)

func main() {
	r, err := radio.OpenRadio("/dev/ttyUSB0", 9600)
	if err != nil {
		fmt.Println("Error opening radio:", err)
		os.Exit(1)
	}

	inbox := make(chan proto.Message)
	m := ui.NewModel(r, "N0CALL", "@CQ")

	// Incoming radio messages → UI inbox
	go r.Listen(inbox)
	go func() {
		for msg := range inbox {
			m.Push(msg)
		}
	}()

	// Outgoing messages → radio send
	go func() {
		for msg := range m.SendQueue {
			_ = r.Send(msg)
		}
	}()

	p := tea.NewProgram(m)
	if err := p.Start(); err != nil {
		fmt.Println("TUI error:", err)
		os.Exit(1)
	}
}
