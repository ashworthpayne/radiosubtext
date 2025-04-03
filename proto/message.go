package proto

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	CmdMessage   = "MSG"
	CmdFingerReq = "FINGERREQ"
	CmdFingerRes = "FINGERRES"
)

type Message struct {
	From    string
	Group   string
	Cmd     string
	Body    string
	Created time.Time
}

func (m Message) Encode() string {
	return fmt.Sprintf("%s|%s|%s|%s", m.Cmd, m.From, m.Group, m.Body)
}

func Decode(line string) (Message, error) {
	parts := strings.SplitN(line, "|", 4)
	if len(parts) < 4 {
		return Message{}, errors.New("invalid message format")
	}
	return Message{
		Cmd:     parts[0],
		From:    parts[1],
		Group:   parts[2],
		Body:    parts[3],
		Created: time.Now(),
	}, nil
}
