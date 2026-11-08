# Install guide for a Claude agent

Instructions for a Claude Code agent setting up this WhatsApp MCP server on a
user's Mac. Follow the steps in order. The only step that needs the human is the
QR scan in step 4.

This fork adds two things on top of [Sealjay/mcp-whatsapp](https://github.com/Sealjay/mcp-whatsapp):
- a redesigned pairing page (`/pair`, light and dark mode);
- a loopback guard: when no auth token is set, requests whose `Host` or `Origin`
  is not a loopback name are rejected, which blocks DNS-rebinding attacks from
  web pages.

## 1. Prerequisites

```bash
go version   # needs Go 1.26+; install with: brew install go
```

FFmpeg is optional (only for sending non-Opus voice notes): `brew install ffmpeg`.

## 2. Clone and build

```bash
mkdir -p ~/mcp && cd ~/mcp
git clone https://github.com/Vicfou-dev/mcp-whatsapp.git whatsapp
cd whatsapp
make build          # produces ./bin/whatsapp-mcp
go test ./...       # optional sanity check
```

## 3. Run the daemon (auto-start with launchd)

The MCP client connects over HTTP, so the daemon must already be running when
Claude Code starts. Otherwise the server shows up as `ECONNREFUSED`. Install it
as a launchd agent so it starts at login and restarts if it crashes:

```bash
cat > ~/Library/LaunchAgents/com.whatsapp-mcp.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>com.whatsapp-mcp</string>
  <key>WorkingDirectory</key><string>$HOME/mcp/whatsapp</string>
  <key>ProgramArguments</key><array>
    <string>$HOME/mcp/whatsapp/bin/whatsapp-mcp</string><string>serve</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>$HOME/mcp/whatsapp/whatsapp.log</string>
  <key>StandardErrorPath</key><string>$HOME/mcp/whatsapp/whatsapp.log</string>
</dict></plist>
EOF
launchctl load ~/Library/LaunchAgents/com.whatsapp-mcp.plist
```

Check that it answers:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8765/pair   # expect 200
```

To run it without launchd, start `./bin/whatsapp-mcp serve` from the repo
directory instead. Only one instance can run per `store/` directory (it is
locked with `store/.lock`).

## 4. Pair the phone (human step)

```bash
open http://127.0.0.1:8765/pair
```

Ask the user to scan the QR: WhatsApp → **Settings** → **Linked Devices** →
**Link a Device**. The page refreshes the QR every 5 seconds and shows
"Already paired" once linked. If the user can't scan, the page has a fallback
that pairs with the phone number instead.

The session is stored in `store/whatsapp.db`. WhatsApp rotates it about every
20 days, and `/pair` then shows a new QR to scan.

## 5. Register the server in Claude Code

```bash
claude mcp add --scope user --transport http whatsapp http://127.0.0.1:8765/mcp
```

Then the user runs `/mcp` in Claude Code (or restarts it) and checks that
`whatsapp` is connected. Smoke test without Claude:

```bash
curl -s -X POST http://127.0.0.1:8765/mcp \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | head -c 300
```

## Troubleshooting

| Symptom | Fix |
|---|---|
| `ECONNREFUSED` in `/mcp` | The daemon isn't running. Start it (step 3), then `/mcp` → whatsapp → Reconnect. |
| `another whatsapp-mcp instance is already running` | One is already up (`pgrep -fl whatsapp-mcp`). If none is, delete `store/.lock`. |
| `forbidden host` / `forbidden origin` | Use `127.0.0.1` or `localhost`, not a LAN IP or hostname. For remote access, set an auth token (see README). |
| `/pair` shows a QR again after weeks | The session rotated. Rescan. |

## Safety rules for the agent

- Never commit or share `store/` or `media/`. They hold the WhatsApp session and
  the message history (both are git-ignored).
- Never send a WhatsApp message unless the user explicitly asks for it and has
  approved the recipient and the text.
