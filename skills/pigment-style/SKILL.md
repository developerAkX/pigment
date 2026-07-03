---
name: pigment-style
description: >
  Use when the user asks about consistent styles, character sheets, recurring
  characters, visual identity, style presets, saving a look, reusing a style
  across images, managing reference images for consistency, or any style/character
  library management for image generation.
---

# pigment style — consistent styles & characters

## Overview

The style library lets you save reusable **style presets** (aesthetic
directions like "Studio Ghibli watercolor") and **character definitions**
(recurring subjects like "Max the robot") with optional reference images.
Styles are stored in `~/.config/pigment/styles/` and automatically applied
to `gen` and `edit` commands when set as active defaults.

## Subcommands

| Command | Description |
|---------|-------------|
| `pigment style list` | List all styles (active ones marked with `*`) |
| `pigment style show NAME` | Show details: kind, snippet, refs, asset path |
| `pigment style add NAME [SNIPPET] [--ref IMG...] [--kind style\|character] [--from-last]` | Add or overwrite a style |
| `pigment style add-ref NAME [IMG...] [--from-last]` | Add reference image(s) to an existing style |
| `pigment style rm-ref NAME FILE` | Remove a reference image from a style |
| `pigment style rm NAME` | Remove a style entirely |
| `pigment style use NAME [NAME...]` | Set the active default style(s) |
| `pigment style clear` | Clear the active default set (no styles applied by default) |
| `pigment style reset [--yes]` | Reset to built-in styles (destructive, deletes all custom styles) |

## Style vs Character

| | `--kind style` (default) | `--kind character` |
|---|---|---|
| **Purpose** | Aesthetic direction (colors, medium, mood) | Recurring subject (person, mascot, object) |
| **Snippet** | Appended to every prompt as style guidance | Appended to describe the subject |
| **Refs** | Attached as style-reference images | Attached as character-reference images |
| **When to use** | You want all images to share a visual look | You want the same subject to appear consistently across images |

**Rule of thumb**: if you're describing *how* the image looks → **style**.
If you're describing *who/what* appears → **character**.

## Using styles with gen/edit

- **`--style NAME`**: apply specific style(s) for this run (repeatable).
- **`--no-style`**: suppress all styles (including active defaults) for this run.
- **Active defaults**: styles set via `pigment style use` are applied
  automatically to every `gen`/`edit` unless overridden by `--style` or
  `--no-style`.

```bash
# Apply explicitly
pigment gen "a castle" --style ghibli

# Set as default and forget
pigment style use ghibli
pigment gen "a castle"        # ghibli applied automatically
pigment gen "a forest" --no-style   # override: no style this time
```

## Workflow examples

### Create a character from a generated image

```bash
# Generate a character concept
pigment gen "a friendly robot named Max, blue metallic body, round head"
# Save it as a character using the last output
pigment style add max --kind character --from-last \
  "a friendly robot named Max with a blue metallic body and round head"
# Now use Max in new scenes
pigment gen "Max the robot exploring a jungle" --style max
```

### Create a style from reference images

```bash
pigment style add retro-poster \
  "1950s travel poster style, bold flat colors, vintage typography" \
  --ref poster1.jpg --ref poster2.jpg
pigment style use retro-poster
pigment gen "visit Mars"
```

### Iterate and refine

```bash
pigment gen "a cozy cabin in snow"
pigment style add-ref cozy-cabin --from-last
pigment edit -i "$(pigment gen "a cozy cabin" --style cozy-cabin)" \
  "add warm light glowing from the windows"
```
