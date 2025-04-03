# 📻 RadioSubtext

**A terminal-based ham radio chat client for digital modes with personality, plugins, and pluck.**

Built for RF nerds, by RF nerds. It talks over anything with a serial port or a plugin—and logs your contacts, caches your station info, and just plain feels good to use.

---

## 💡 Features

- `/finger <CALLSIGN>` — Ask any station to share their gear/grid profile
- `/whois <CALLSIGN>` — View cached station info (works offline!)
- Scrollable chat log with group-based messages (`@CQ`, `@TEST`)
- Modular modem interface (serial, fake, future plugins)
- Local cache: `~/.radiosubtext/finger.json`
- Built-in fake net for development and offline play
- Plug-and-play architecture for future modems (JS8Call, TCP, satellites)
- Beautiful Bubble Tea-powered terminal UI

---

## 🔧 Getting Started

### Prereqs

- Go 1.20+
- A brain
- Maybe a radio

### Clone & Run

```bash
git clone https://github.com/ashworthpayne/radiosubtext.git
cd radiosubtext
go run ./cmd/radiosubtext --fake --callsign N0CALL
```

### Talk to the fake net

```text
/finger KJ4XYZ
/whois KJ4XYZ
```

---

## 🧪 Development & Testing

Run in loopback mode with a fake modem that simulates RF traffic:

```bash
go run ./cmd/radiosubtext --fake --callsign N0CALL
```

All `/finger` responses are auto-cached to `~/.radiosubtext/finger.json`.

---

## 💬 Planned Commands

- `/mail <CALLSIGN>` — Send long-form messages
- `/stations` — List all known cached callsigns
- `/setfreq <MHz>` — Tune radio (if supported)
- `/relay on/off` — Allow relaying others’ mail
- `/clear` — Wipe scrollback

---

## ✨ Example Finger Response

```json
{
  "KJ4XYZ": {
    "callsign": "KJ4XYZ",
    "last_response": "Gear: IC-9700 | Grid: EM65 | VFO chaos mode: ✅",
    "updated": "2025-04-03T14:45:56Z"
  }
}
```

---

## 🛠️ Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — for that sweet terminal UI
- [tarm/serial](https://pkg.go.dev/github.com/tarm/serial) — for serial radio support

---

## 👋 Contact

Created by [Ashworth Payne](https://github.com/ashworthpayne)  
DMs open. Radios on. Let's build weird stuff.
