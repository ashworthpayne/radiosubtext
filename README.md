# 📻 RadioSubtext

**A terminal-based RF messaging system for ham radio nerds.**  
Uses D-STAR's data channel to send UTF-8 messages, chat in groups, and build a playful, modern protocol layer over the air.

---

## 🚀 Features

- Terminal chat UI with emoji, Markdown-style formatting, and scrollback
- Write messages over `/dev/ttyUSBx` using D-STAR digital data
- Message types: `MSG`, `MAIL`, `FINGER`, `CHANGEFREQ`, and more
- Optional peer relaying, contact metadata, and group chat
- Lightweight line-based protocol for easy expansion
- Compatible with Icom radios that expose a serial interface

---

## 📦 Install

```bash
git clone https://github.com/yourname/RadioSubtext.git
cd RadioSubtext
go build ./cmd/radiosubtext

## usage 
./radiosubtext --port /dev/ttyUSB0 --callsign N0CALL
