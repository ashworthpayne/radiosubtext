# 🧠 Notes

## 📁 Directory Tree
⚠️ Some of this may need to be cleaned up ⚠️ 

```
.
├── README.md
├── cmd
│   └── radiosubtext
│       └── main.go
├── go.mod
├── go.sum
├── modems
│   ├── README.md
│   ├── dstarserial
│   │   └── dstar.go
│   ├── fake
│   │   └── fake.go
│   └── interface.go
├── proto
│   ├── finger.go
│   ├── handler.go
│   └── message.go
├── radio
│   └── radio.go
├── scratchfile.txt
└── ui
    └── tui.go

9 directories, 14 files
```

## 🤔 Thoughts

* The UI needs to be built out. The current one is ick.
* I'd like a red/green emoji dot to indicated send/rec to the modem. Just a bit of eye candy.
    🔴 🟢 ⚫️
* Change /finger and /whois to just /whois, and logically switch functions as-needed
* local config file with call, grid, radio, ??. Used to respond to finger commands.
