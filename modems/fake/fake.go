package fake

import (
	"math/rand"
	"strings"
	"time"

	"github.com/ashworthpayne/radiosubtext/proto"
)

type Modem struct {
	Callsign string
	Inbox    chan proto.Message
}

func New() *Modem {
	return &Modem{
		Callsign: "KJ4XYZ",
		Inbox:    make(chan proto.Message, 10),
	}
}

func (f *Modem) Send(msg proto.Message) error {

	f.Inbox <- msg
	return nil
}

func (f *Modem) Listen(outbox chan proto.Message) {
	go func() {
		for {
			select {
			case msg := <-f.Inbox:

				if msg.Cmd == proto.CmdFingerReq &&
					strings.EqualFold(msg.Body, f.Callsign) {

					outbox <- proto.Message{
						From:    f.Callsign,
						Group:   msg.Group,
						Cmd:     proto.CmdFingerRes,
						Body:    "Gear: IC-9700 | Grid: EM65 | VFO chaos mode: ✅",
						Created: time.Now(),
					}
				}

			default:
				if rand.Intn(50) == 0 {
					outbox <- proto.Message{
						From:    "W1AW",
						Group:   "@CQ",
						Cmd:     proto.CmdMessage,
						Body:    "🔊 Test net starting soon!",
						Created: time.Now(),
					}
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}()
}
