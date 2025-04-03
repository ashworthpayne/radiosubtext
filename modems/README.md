# 📡 Modems in RadioSubtext

This directory contains modular **radio interface drivers** (aka *modems*) used by RadioSubtext to send and receive messages over different transports like D-STAR, serial, or simulated backends.

---

## 🧩 Modem Interface

To create a new modem, implement this interface:

```go
type Modem interface {
    Send(msg proto.Message) error
    Listen(outbox chan proto.Message)
}
```

- `Send(msg)`: Called when the user sends a message from the UI.
- `Listen(outbox)`: Push messages into the `outbox` channel to simulate receiving them from the air.

---

## 📁 Included Modems

### `fake/`
A fully self-contained test modem that:
- Responds to `/finger` requests
- Periodically emits chat messages
- Doesn’t require any hardware

Use it with:
```bash
go run ./cmd/radiosubtext --fake
```

### `dstarserial/`
Communicates with a D-STAR radio over `/dev/ttyUSBx` or COM port using standard serial input/output.

---

## ➕ Writing Your Own Modem

1. Create a folder like `modems/mycoolmodem/`
2. Create a struct that implements the `Modem` interface
3. Register it in `main.go` using a CLI flag or config value

Example:

```go
type Modem struct {}

func (m *Modem) Send(msg proto.Message) error {
    // Send msg.Body over RF or a socket, etc.
    return nil
}

func (m *Modem) Listen(outbox chan proto.Message) {
    go func() {
        for {
            // Receive data, convert to proto.Message
            outbox <- msg
        }
    }()
}
```

---

## 🔮 Future Ideas

- JS8Call modem via UNIX socket
- TCP/UDP modem for bridging across the internet
- SDR-based modem (e.g. PlutoSDR)
- Satellite repeater interface
- External modem via stdin/stdout

---

## 🛠️ Gotchas

- Messages must be line-based and UTF-8 safe
- Use newline `\n` as a message delimiter if working over raw streams
- Avoid buffering delays in `Listen()` loops

---

Want to contribute a modem driver? Open a PR!
