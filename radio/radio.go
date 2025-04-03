package radio

import (
	"bufio"
	"time"

	"github.com/ashworthpayne/radiosubtext/proto"
	"github.com/tarm/serial"
)

type Radio struct {
	port *serial.Port
}

// Opens the radio's TTY interface
func OpenRadio(dev string, baud int) (*Radio, error) {
	c := &serial.Config{
		Name:        dev,
		Baud:        baud,
		ReadTimeout: time.Second,
	}
	p, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}
	return &Radio{port: p}, nil
}

// Sends a structured message over RF
func (r *Radio) Send(msg proto.Message) error {
	_, err := r.port.Write([]byte(msg.Encode() + "\n"))
	return err
}

// Listens for incoming lines from radio and pushes to inbox
func (r *Radio) Listen(inbox chan proto.Message) {
	scanner := bufio.NewScanner(r.port)
	for scanner.Scan() {
		line := scanner.Text()
		if msg, err := proto.Decode(line); err == nil {
			inbox <- msg
		}
	}
}
