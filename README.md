# pigment

**Image generation and editing CLI powered by your ChatGPT subscription.**

No API key needed. Pigment uses codex OAuth to access image generation.

## Quick install (binary + agent skills)

One command installs the `pigment` binary **and** registers the agent skills
with your coding agent (via [`npx skills add`](https://skills.sh)):

```bash
curl -fsSL https://raw.githubusercontent.com/developerAkX/pigment/main/install.sh | bash
```

```powershell
# Windows
irm https://raw.githubusercontent.com/developerAkX/pigment/main/install.ps1 | iex
```

By default skills are installed for **opencode**. Choose another agent with
`PIGMENT_SKILLS_AGENT=claude-code` (or `'*'` for all), or skip skills with
`PIGMENT_NO_SKILLS=1`.

## Install

### Homebrew (macOS)

```bash
brew install developerAkX/tap/pigment
```

### Scoop (Windows)

```powershell
scoop bucket add developerAkX https://github.com/developerAkX/scoop-bucket
scoop install pigment
```

### Shell script (macOS / Linux)

Installs the binary and the agent skills, then runs `pigment doctor`:

```bash
curl -fsSL https://raw.githubusercontent.com/developerAkX/pigment/main/install.sh | bash
```

### PowerShell (Windows)

```powershell
irm https://raw.githubusercontent.com/developerAkX/pigment/main/install.ps1 | iex
```

### Go install

```bash
go install github.com/developerAkX/pigment/cmd/pigment@latest
```

Then install the skills separately (see [Agent Skills](#agent-skills)).

## Quickstart

```bash
# 1. Authenticate (requires npm i -g @openai/codex)
codex login

# 2. Verify setup
pigment doctor

# 3. Generate an image
pigment gen "a cozy cabin in a snowy forest, warm light from windows"

# 4. Edit an image
pigment edit -i cabin.png "add northern lights in the sky"
```

## Command Reference

| Command | Description |
|---------|-------------|
| `pigment gen "<prompt>"` | Generate an image from a text prompt |
| `pigment edit -i IMG "<prompt>"` | Edit an image with reference(s) and instruction |
| `pigment style list` | List all saved styles |
| `pigment style show NAME` | Show style details |
| `pigment style add NAME [SNIPPET]` | Add a style (with `--ref`, `--kind`, `--from-last`) |
| `pigment style add-ref NAME IMG...` | Add reference images to a style |
| `pigment style rm-ref NAME FILE` | Remove a reference image |
| `pigment style rm NAME` | Remove a style |
| `pigment style use NAME...` | Set active default style(s) |
| `pigment style clear` | Clear active defaults |
| `pigment style reset` | Reset to built-in styles |
| `pigment doctor` | Check system readiness |
| `pigment auth status` | Show authentication status |
| `pigment auth login` | Instructions to authenticate |
| `pigment auth logout` | Instructions to log out |
| `pigment version` | Print version |
| `pigment upgrade` | Upgrade to latest release (`--check` for dry run) |
| `pigment skill list` | List embedded agent skills |
| `pigment skill install` | Install agent skills (`--target`, `--dir`, `--force`) |

## Agent Skills

Pigment ships three embedded agent skills that teach AI coding assistants
(opencode, Claude Code, etc.) how to use pigment:

- **pigment-generate** — text-to-image generation
- **pigment-edit** — image-to-image editing
- **pigment-style** — style/character library management

### Recommended: `npx skills add`

The skills are published to the [skills.sh](https://skills.sh) registry
straight from this repository. Install all three into your agent with:

```bash
# Install for opencode (globally, no prompts)
npx skills add developerAkX/pigment --skill '*' --agent opencode --global --yes

# Or pick specific skills / agents
npx skills add developerAkX/pigment --skill pigment-generate --agent claude-code
npx skills add developerAkX/pigment --all      # every skill, every detected agent
```

Supported agents include opencode, claude-code, cursor, codex, github-copilot,
windsurf, gemini, and more.

### Alternative: embedded installer

The binary also carries the skills, so you can install them offline:

```bash
pigment skill install                 # opencode (default)
pigment skill install --target claude # Claude Code
pigment skill install --dir PATH      # custom directory
```

Once installed, your AI assistant can generate and edit images on your behalf
using pigment commands.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PIGMENT_MODEL` | Default model for generation | `gpt-5.5` |
| `PIGMENT_CONFIG_DIR` | Config directory override | `~/.config/pigment` |
| `PIGMENT_CODEX_CONCURRENCY` | Max concurrent codex requests | `4` |
| `PIGMENT_NO_COLOR` | Disable color output | unset |
| `PIGMENT_NO_UPDATE_CHECK` | Disable upgrade check | unset |
| `NO_COLOR` | Standard no-color flag | unset |

## JSON Output Contract

When `--json` is passed to `gen` or `edit`, stdout is a single JSON object:

```json
{
  "path": "generated_sunset_abc123.png",
  "model": "gpt-5.5",
  "size": "1024x1024",
  "format": "png",
  "duration_ms": 72000,
  "prompt": "a sunset over mountains"
}
```

Without `--json`, stdout is exactly the saved file path (one line). All
progress and status messages go to stderr.

## Disclaimer

Pigment uses an **unofficial ChatGPT backend** (codex OAuth) for image
generation. This is not an official OpenAI product. The backend may change
or break at any time without notice. Use at your own risk.

## Credits

Inspired by [leeguooooo/chatgpt-imagegen](https://github.com/leeguooooo/chatgpt-imagegen)
(MIT License) as the original reference implementation for ChatGPT image
generation via codex OAuth.

## License

MIT License. Copyright (c) Ayush Sharma.

See [LICENSE](LICENSE) for details.
