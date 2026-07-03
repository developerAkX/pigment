---
name: pigment-generate
description: >
  Use when the user asks to generate an image, create artwork, make a logo,
  illustration, poster, icon, banner, thumbnail, avatar, mockup, concept art,
  photo-realistic render, or any visual asset from a text description.
---

# pigment generate — text-to-image

## Prerequisites

Run `pigment doctor` first to verify credentials and network connectivity.
If doctor reports auth issues, tell the user to run `codex login` (requires
`npm i -g @openai/codex` if not installed). pigment reads tokens from
`~/.codex/auth.json` — no separate API key is needed.

## Command

```bash
pigment gen "<prompt>"
```

Without `--json`, stdout is **exactly the saved file path** (one line, no
decoration). All progress/status goes to stderr. This makes it safe to
capture the path in scripts:

```bash
path=$(pigment gen "a sunset over mountains" --format webp)
echo "Saved to: $path"
```

With `--json`, stdout is a single JSON object (see JSON contract below).

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--size` | | `auto` | Image dimensions: `auto`, `1024x1024`, `1536x1024`, `1024x1536`, etc. |
| `--format` | | `png` | Output format: `png`, `jpeg`, `webp` |
| `-o` / `--out` | `-o` | auto | Output file path (auto-generates a name if omitted) |
| `--model` | | `gpt-5.5` | Model name. Override default via `PIGMENT_MODEL` env var. |
| `--timeout` | | `300` | Total timeout in seconds |
| `--stall-timeout` | | `120` | Stall (no-progress) timeout in seconds |
| `-i` / `--ref` | `-i` | | Reference image path or URL (repeatable, up to 4) |
| `--style` | | | Apply named style(s) from the style library (repeatable) |
| `--no-style` | | `false` | Suppress all styles for this run |
| `--json` | | `false` | Output JSON object instead of plain path |
| `--no-progress` | | `false` | Suppress progress output on stderr |
| `--open` | | `false` | Open result in default image viewer |
| `--quiet` | | `false` | Suppress update notices |

## JSON contract

When `--json` is passed, stdout is a single JSON object:

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

Fields: `path` (string, saved file), `model` (string), `size` (string),
`format` (string, png|jpeg|webp), `duration_ms` (int64, wall-clock
milliseconds), `prompt` (string, the original user prompt).

## Prompt-writing tips

- **Be specific**: "a golden retriever puppy sitting on a red cushion, soft
  window light, shallow depth of field" beats "a dog".
- **Include style/medium**: "oil painting", "3D render", "watercolor sketch",
  "photograph with 85mm lens".
- **Describe lighting**: "dramatic side-lighting", "overcast diffuse light",
  "neon glow".
- **Mention composition**: "close-up", "wide establishing shot", "top-down
  flat lay", "rule of thirds".
- **Iterate**: use the output path from a previous `gen` as a `--ref` input
  to `edit` for refinement.

## Performance & error handling

- **Typical latency**: 60–90 seconds depending on model, size, and server load.
  The `--timeout` flag (default 300s) caps wall-clock time.
- **Exit codes**: non-zero on any error. The error message is printed to stderr.
- **Auth errors**: if pigment reports an authentication failure, instruct the
  user to run `codex login` to refresh their ChatGPT session tokens.
- **Concurrency**: pigment enforces a concurrency limit (default 4, configurable
  via `PIGMENT_CODEX_CONCURRENCY`). Extra requests queue with a waiting message.
- **Stall detection**: if the backend stops sending data for `--stall-timeout`
  seconds (default 120), the request is aborted.
