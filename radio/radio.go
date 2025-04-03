package radio

import (
	"bufio"
	"time"

	"radiosubtext/proto"

	"github.com/tarm/serial"
)

type Radio struct {
	port *serial.Port
}

// OpenRadio connects to serial device.
func OpenRadio(dev string, baud int) (*Radio, error) {
	c := &serial.Config{Name: dev, Baud: baud, ReadTimeout: time.Second * 1}
	p, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}
	return &Radio{port: p}, nil
}

func (r *Radio) Send(msg proto.Message) error {
	raw := msg.Encode()
	_, err := r.port.Write([]byte(raw + "\n"))
	return err
}

func (r *Radio) Listen(inbox chan proto.Message) {
	scanner := bufio.NewScanner(r.port)
	for scanner.Scan() {
		line := scanner.Text()
		if msg, err := proto.Decode(line); err == nil {
			inbox <- msg
		}
	}
}
