# sonora-cli

[![CI](https://github.com/raloonsoc/sonora-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/raloonsoc/sonora-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

A terminal music client for [Sonora](https://github.com/raloonsoc/sonora) and
any OpenSubsonic-compatible server. Cover art, synced lyrics, and
album-tinted UI — rendered directly in your terminal.

```
┌─ Browse ───────────────────────────────────────────────────────────┐
│  Artists                                                            │
│ ▸Kikagaku Moyo               12 albums                              │
│  Dripping Sun                 4 albums                              │
│  Khruangbin                   6 albums                              │
└───────────────────────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────────────────────┐
│ ▓▓▓  Sword of Doom              ▶ ━━━━━━━╺━━━━━━ 2:14/6:47  🔊80%  │
│ ▓▓▓  Kikagaku Moyo · Masana Temples          [L] lyrics            │
└──────────────────────────────────────────────────────────────────┘
```

A full-width browse pane on top, a compact now-playing bar docked at the
bottom — press `L` to switch to a fullscreen view with centered art and
large synced lyrics.

## Features

- **Works with any OpenSubsonic server.** Built against the standard contract,
  so it talks to Sonora, Navidrome, Airsonic, and anything else that speaks
  the protocol.
- **Cover art in the terminal.** Rendered as colored ASCII art, chosen over
  native graphics protocols (Kitty/iTerm2/Sixel) after they proved unreliable
  in a TUI that repaints its whole frame on every playback tick — see
  [Terminal support](#terminal-support).
- **Synced lyrics.** Millisecond-timestamped lines that follow along with
  playback, with the active line highlighted.
- **Album-derived accent color.** The UI tints itself to match the current
  album's artwork, the way Spotify tints its now-playing view. *(Sonora
  servers only — see [Accent color](#accent-color).)*
- **Playback via mpv.** No custom audio decoding; mpv streams directly from the
  server, including seeking via HTTP Range requests.
- **Single static binary.** No runtime dependencies beyond `mpv` and `ffmpeg`.

## Requirements

- `mpv` — audio playback
- `ffmpeg` — media handling
- A terminal with at least 16-color support (truecolor and graphics protocols
  are used opportunistically)
- An OpenSubsonic-compatible server

```bash
# macOS
brew install mpv ffmpeg

# Debian / Ubuntu
sudo apt install mpv ffmpeg

# Arch
sudo pacman -S mpv ffmpeg
```

## Installation

### Homebrew

```bash
brew install raloonsoc/tap/sonora-cli
```

### Arch (AUR)

```bash
yay -S sonora-cli-bin
```

### Go

```bash
go install github.com/raloonsoc/sonora-cli/cmd/sonora@latest
```

### Binaries

Prebuilt binaries for Linux and macOS (amd64 and arm64) are attached to each
[release](https://github.com/raloonsoc/sonora-cli/releases).

## Getting started

Run it. On first launch you'll be prompted for your server URL, username, and
password:

```bash
sonora
```

That writes `~/.config/sonora-cli/config.toml` with `0600` permissions, and
you're browsing your library.

## Configuration

`~/.config/sonora-cli/config.toml`

```toml
default_profile = "home"

[profiles.home]
url      = "https://music.example.com"
username = "raul"
password = "hunter2"        # see Password storage below
auth     = "subsonic"       # "subsonic" | "native"

[profiles.vps]
url      = "https://sonora.example.net"
username = "raul"
auth     = "native"         # stores a refresh token, no password at rest

[ui]
art    = "auto"             # "auto" | "ascii" | "off"
lyrics = true

[playback]
volume = 80
```

Switch profiles with `--profile`, or press `P` inside the app:

```bash
sonora --profile vps
```

### Flags

| Flag | Description |
|---|---|
| `--profile <name>` | Use a specific server profile |
| `--config <path>` | Override the config file location |
| `--version` | Print version and exit |

Every setting can also be overridden with an environment variable:
`SONORA_CLI_URL`, `SONORA_CLI_USERNAME`, `SONORA_CLI_PASSWORD`,
`SONORA_CLI_AUTH`, `SONORA_CLI_DEFAULT_PROFILE`, `SONORA_CLI_UI_ART`,
`SONORA_CLI_UI_LYRICS`, `SONORA_CLI_PLAYBACK_VOLUME`.

### Password storage

The Subsonic auth scheme computes a per-request token as `md5(password +
salt)`, which means the client needs the plaintext password to authenticate at
all. There is no way around this while using Subsonic auth.

sonora-cli handles it in order of preference:

1. **OS keychain**, when available — the password never touches disk in
   plaintext.
2. **Config file** with `0600` permissions — the documented fallback.

If your server supports Sonora's native JWT API, set `auth = "native"` on the
profile. Only a rotating refresh token is persisted; the access token lives in
memory and is refreshed transparently. No plaintext password at rest.

## Keybindings

Press `?` at any time for the in-app help.

### Global

| Key | Action |
|---|---|
| `q` | Quit |
| `?` | Toggle help |
| `/` | Search |
| `L` | Toggle lyrics view (centered art + large synced lyrics) |
| `P` | Switch profile |
| `esc` | Back |

### Library

| Key | Action |
|---|---|
| `↑` / `k`, `↓` / `j` | Move |
| `enter` | Open / play |
| `a` | Add to queue |
| `g` / `G` | Jump to top / bottom |

### Playback

These work everywhere — browsing the library, or inside the lyrics view.

| Key | Action |
|---|---|
| `space` | Play / pause |
| `n` / `p` | Next / previous track |
| `←` / `→` | Seek ∓10s |
| `-` / `+` | Volume down / up |

## Terminal support

Cover art renders as colored ASCII art everywhere — any 16-color-or-better
terminal works, no graphics-protocol support required.

Native graphics protocols (Kitty, iTerm2 inline images, Sixel) were tried
early on but pulled: they place images as state that persists outside the
normal text grid, separate from whatever the TUI's renderer believes is on
screen. Repainting the whole frame on every ~500ms playback tick — which
sonora-cli does, to drive the progress bar and lyrics highlight — desynced
the two, causing the previous placement to linger undeleted instead of the
new one replacing it in place. ASCII has no such out-of-band state, so it
repaints exactly as reliably as the rest of the UI.

`art = "auto" | "ascii" | "off"` in `[ui]` toggles art on/off; both non-"off"
values currently render the same ASCII output.

## Accent color

Sonora computes a vibrant color per album during ingestion and serves it
alongside the album metadata. sonora-cli reads that value and applies it to the
now-playing border, the progress bar, and the active lyric line.

This is **not** part of the standard OpenSubsonic contract. Against any other
server the field is simply absent and the UI falls back to a neutral accent —
everything else works identically.

## How it works

```
┌────────────┐   HTTP (OpenSubsonic)   ┌──────────────┐
│ sonora-cli │ ──────────────────────► │    server    │
└─────┬──────┘   metadata, art, lyrics └──────▲───────┘
      │                                       │
      │ JSON IPC                              │ HTTP + Range
      ▼                                       │
┌────────────┐                                │
│    mpv     │ ───────────────────────────────┘
└────────────┘        audio stream
```

sonora-cli never proxies audio bytes. It hands mpv a signed `stream.view` URL
and mpv fetches the audio itself, so seeking is a Range request straight to the
server. Playback is controlled over mpv's JSON IPC socket, and playback
position is polled to drive the progress bar and the lyrics highlight.

## Not included in v1

- Offline mode / local caching
- Library writes — playlist creation, tagging, and ratings are read-only
- Terminals without color output

## Contributing

Issues and PRs are welcome. To build from source:

```bash
git clone https://github.com/raloonsoc/sonora-cli
cd sonora-cli
make build
make test
```

`make lint` runs `gofmt`, `go vet`, and `golangci-lint`; CI enforces all
three on every pull request.

## License

[MIT](./LICENSE)
