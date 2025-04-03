package dstarserial

import (
	"bufio"
	"time"

	"github.com/ashworthpayne/radiosubtext/proto"
	"github.com/tarm/serial"
)

type Modem struct {
	port *serial.Port
}

func New(portName string, baud int) (*Modem, error) {
	c := &serial.Config{Name: portName, Baud: baud, ReadTimeout: time.Second}
	p, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}
	return &Modem{port: p}, nil
}

func (m *Modem) Send(msg proto.Message) error {
	_, err := m.port.Write([]byte(msg.Encode() + "\n"))
	return err
}

func (m *Modem) Listen(outbox chan proto.Message) {
	scanner := bufio.NewScanner(m.port)
	for scanner.Scan() {
		line := scanner.Text()
		if msg, err := proto.Decode(line); err == nil {
			outbox <- msg
		}
	}
}
