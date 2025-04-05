package fake

import (
	"math/rand"
	"time"

	"github.com/ashworthpayne/radiosubtext/proto"
)

type Modem struct {
	inbox chan proto.Message
}

func New() *Modem {
	m := &Modem{
		inbox: make(chan proto.Message, 10),
	}
	return m
}

func (f *Modem) Send(msg proto.Message) error {
	// Don't echo back your own messages in fake mode
	return nil
}

func (f *Modem) Listen(outbox chan proto.Message) {
	go func() {
		println("📡 Fake modem started. Simulating traffic...")

		// Define our simulated stations and their groups
		cqStations := []string{"K5TEST", "N0BOT", "W1LITE"}
		radioStations := []string{"KJ4QAC", "W4MW", "WB4IXU", "N4JS", "KX4DX"}

		// Define some realistic ham radio messages
		cqMessages := []string{
			"CQ CQ CQ",
			"QSL on your last",
			"Running 100W to a dipole",
			"Conditions are good today",
			"Thanks for the contact",
			"How copy?",
			"Weather here is clear, temp 72F",
		}

		radioMessages := []string{
			"Net control checking in",
			"Traffic for Atlanta",
			"Emergency power test successful",
			"ARES net starting in 10",
			"Switching to digital mode",
			"Testing new antenna setup",
			"Mobile station, standing by",
			"Field day site secured",
			"Weather net active",
		}

		// Simulate traffic with random intervals
		for {
			// Random delay between 2-5 seconds
			time.Sleep(time.Duration(2000+rand.Intn(3000)) * time.Millisecond)

			// Randomly choose which group to send to
			if rand.Float32() < 0.6 { // 60% chance for CQ traffic
				station := cqStations[rand.Intn(len(cqStations))]
				msg := cqMessages[rand.Intn(len(cqMessages))]
				outbox <- proto.Message{
					From:    station,
					Group:   "@CQ",
					Cmd:     proto.CmdMessage,
					Body:    msg,
					Created: time.Now(),
				}
			} else { // 40% chance for Radio group traffic
				station := radioStations[rand.Intn(len(radioStations))]
				msg := radioMessages[rand.Intn(len(radioMessages))]
				outbox <- proto.Message{
					From:    station,
					Group:   "@Radio",
					Cmd:     proto.CmdMessage,
					Body:    msg,
					Created: time.Now(),
				}
			}
		}
	}()

	// pump inbox to outbox (e.g., loopback or echo)
	go func() {
		for msg := range f.inbox {
			outbox <- msg
		}
	}()
}
