---
name: pigment
description: >
  Use for any image work via the pigment CLI: generating images from text
  (logos, illustrations, posters, icons, banners, avatars, mockups, concept
  art, photo-realistic renders), editing/restyling/remixing/combining existing
  images, image-to-image transforms, and managing reusable style presets or
  recurring characters for visual consistency across generations.
---

# pigment — image generation, editing & styles

## Prerequisites

Run `pigment doctor` to verify credentials and connectivity. On auth failure,
tell the user to run `codex login` (install via `npm i -g @openai/codex` if
missing). Tokens are read from `~/.codex/auth.json` — no separate API key.

## Output contract

Without `--json`, stdout is **exactly the saved file path** (one line); all
progress goes to stderr — safe to capture in scripts:

```bash
path=$(pigment gen "a sunset over mountains" --format webp)
```

With `--json`, stdout is a single object:
`{"path", "model", "size", "format", "duration_ms", "prompt"}`.

Non-zero exit on error (message on stderr). Typical latency 60–90s.

## Generate (text-to-image)

```bash
pigment gen "<prompt>"
```

Prompt tips: be specific (subject, setting, mood); name a medium ("oil
painting", "3D render", "85mm photo"); describe lighting and composition.

## Edit (image-to-image)

```bash
pigment edit -i <image> "<instruction>"
```

At least one `-i`/`--ref` (local path or http(s) URL) is required; up to 4
(extras dropped with a warning). Describe **what to change**, not the whole
image. Use a previous output path as `-i` to iterate:

```bash
pigment edit -i photo.jpg "convert to watercolor painting style"
pigment edit -i cat.png -i hat.png "put the hat on the cat"
pigment edit -i "$(pigment gen "robot")" "add glowing red eyes"
```

## Shared flags (gen & edit)

| Flag | Default | Description |
|------|---------|-------------|
| `-i` / `--ref` | | Reference image path/URL (repeatable, max 4; required for edit) |
| `-o` / `--out` | auto | Output file path (auto-named if omitted) |
| `--size` | `auto` | `auto`, `1024x1024`, `1536x1024`, `1024x1536`, ... |
| `--format` | `png` | `png`, `jpeg`, `webp` |
| `--model` | `gpt-5.5` | Model name (env default: `PIGMENT_MODEL`) |
| `--style` | | Apply named style(s) from the library (repeatable) |
| `--no-style` | `false` | Suppress all styles (including active defaults) |
| `--json` | `false` | JSON output instead of plain path |
| `--timeout` | `300` | Total timeout (seconds) |
| `--stall-timeout` | `120` | Abort if no progress for N seconds |
| `--open` | `false` | Open result in default image viewer |

Concurrency is capped (default 4, `PIGMENT_CODEX_CONCURRENCY`); extra
requests queue with a waiting message on stderr.

## Style library (consistent styles & characters)

Save reusable **style presets** (aesthetic looks) and **characters**
(recurring subjects) with optional reference images. Stored in
`~/.config/pigment/styles/`; active defaults apply to every gen/edit.

| Command | Description |
|---------|-------------|
| `pigment style list` | List styles (`*` = active) |
| `pigment style show NAME` | Show kind, snippet, refs |
| `pigment style add NAME [SNIPPET] [--ref IMG...] [--kind style\|character] [--from-last]` | Add/overwrite |
| `pigment style add-ref NAME [IMG...] [--from-last]` | Add reference image(s) |
| `pigment style rm-ref NAME FILE` | Remove a reference image |
| `pigment style rm NAME` | Remove a style |
| `pigment style use NAME [NAME...]` | Set active default style(s) |
| `pigment style clear` | Clear active defaults |
| `pigment style reset [--yes]` | Reset to built-ins (deletes custom styles) |

**Kind rule of thumb**: describing *how* images look (colors, medium, mood)
→ `style`; describing *who/what* appears (person, mascot, object)
→ `character`.

```bash
# Character workflow: generate, save, reuse
pigment gen "a friendly robot named Max, blue metallic body, round head"
pigment style add max --kind character --from-last \
  "a friendly robot named Max with a blue metallic body and round head"
pigment gen "Max the robot exploring a jungle" --style max

# Style preset from references, set as default
pigment style add retro-poster "1950s travel poster, bold flat colors" \
  --ref poster1.jpg --ref poster2.jpg
pigment style use retro-poster
pigment gen "visit Mars"              # retro-poster applied automatically
pigment gen "a forest" --no-style     # override for one run
```
