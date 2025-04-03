package modems

import "github.com/ashworthpayne/radiosubtext/proto"

type Modem interface {
	Send(proto.Message) error
	Listen(chan proto.Message)
}
