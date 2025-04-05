package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ashworthpayne/radiosubtext/modems"
	"github.com/ashworthpayne/radiosubtext/modems/dstarserial"
	"github.com/ashworthpayne/radiosubtext/modems/fake"
	"github.com/ashworthpayne/radiosubtext/proto"
	"github.com/ashworthpayne/radiosubtext/ui"
)

func main() {
	// CLI flags
	port := flag.String("port", "/dev/ttyUSB0", "Serial port for D-STAR modem")
	callsign := flag.String("callsign", "N0CALL", "Your station callsign")
	group := flag.String("group", "@CQ", "Default group name")
	useFake := flag.Bool("fake", false, "Use fake modem instead of serial")
	flag.Parse()

	var modem modems.Modem

	if *useFake {
		fmt.Println("🔧 Running in FAKE mode — no serial hardware required.")
		modem = fake.New()
	} else {
		dstar, err := dstarserial.New(*port, 9600)
		if err != nil {
			fmt.Println("Error opening D-STAR modem:", err)
			os.Exit(1)
		}
		modem = dstar
	}

	// Init UI
	model := ui.NewModel(modem, *callsign, *group)
	inbox := make(chan proto.Message)

	// Connect modem → UI
	go modem.Listen(inbox)

	// Connect UI → modem
	go func() {
		for msg := range model.SendQueue {
			_ = modem.Send(msg)
		}
	}()

	// Connect inbound messages into UI's RecvQueue
	go func() {
		for msg := range inbox {
			model.RecvQueue <- msg
		}
	}()

	// Launch TUI
	p := tea.NewProgram(model)
	if err := p.Start(); err != nil {
		fmt.Println("TUI error:", err)
		os.Exit(1)
	}
}
