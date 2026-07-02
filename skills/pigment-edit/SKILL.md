---
name: pigment-edit
description: >
  Use when the user asks to edit, modify, restyle, transform, remix, or combine
  existing images. Also use for image-to-image generation, inpainting requests,
  restyling photos, combining subjects from multiple images, iterating on a
  previously generated image, or applying visual changes described in text.
---

# pigment edit — image-to-image editing

## Command

```bash
pigment edit -i <image> "<instruction>"
```

At least one reference image (`-i` / `--ref`) is **required**. The instruction
describes what to change. Without `--json`, stdout is the saved file path.

```bash
# Restyle a photo
pigment edit -i photo.jpg "convert to watercolor painting style"

# Combine subjects from two images
pigment edit -i cat.png -i hat.png "put the hat on the cat"

# Iterate on a previous generation
pigment edit -i generated_sunset_abc123.png "add a sailboat in the foreground"
```

## Reference images

- Accepts **local file paths** or **http(s) URLs**.
- Up to **4 references** can be provided (repeat `-i`). If more are supplied,
  pigment keeps the first 4 (character refs prioritised) and warns on stderr.
- Use the output path from a previous `pigment gen` or `pigment edit` as a
  reference to iterate on your work.

## Use cases

| Scenario | Example |
|----------|---------|
| **Restyle** | `pigment edit -i photo.jpg "oil painting, impressionist style"` |
| **Combine subjects** | `pigment edit -i dog.png -i beach.jpg "the dog running on this beach"` |
| **Modify details** | `pigment edit -i logo.png "change the background to dark blue"` |
| **Iterate** | `pigment edit -i $(pigment gen "robot") "add glowing red eyes"` |
| **Format conversion** | `pigment edit -i input.png "same image" --format webp -o output.webp` |

## Shared flags

`pigment edit` accepts the same flags as `pigment gen`:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `-i` / `--ref` | `-i` | **(required)** | Reference image path or URL (repeatable) |
| `-o` / `--out` | `-o` | auto | Output file path |
| `--size` | | `auto` | Image dimensions |
| `--format` | | `png` | Output format: `png`, `jpeg`, `webp` |
| `--model` | | `gpt-5.5` | Model name |
| `--style` | | | Apply named style(s) |
| `--no-style` | | `false` | Suppress all styles |
| `--json` | | `false` | JSON output |
| `--timeout` | | `300` | Total timeout in seconds |
| `--stall-timeout` | | `120` | Stall timeout in seconds |
| `--open` | | `false` | Open result in viewer |

See the **pigment-generate** skill for the full JSON output contract,
error handling, and performance notes — they apply identically here.

## Tips

- When iterating, pipe the path: `pigment edit -i "$(pigment gen "castle")" "add a moat"`.
- For best results, describe **what you want changed** rather than describing
  the entire image from scratch.
- Combine with `--style` to apply a consistent aesthetic across edits.
