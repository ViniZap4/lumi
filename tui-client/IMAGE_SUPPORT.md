# Image Support in Lumi TUI

## Overview

The TUI client now supports inline image rendering using the Kitty graphics protocol.

## Supported Terminals

- **Kitty** - Full support
- **WezTerm** - Supports Kitty protocol
- **iTerm2** - Partial support (may need configuration)
- Other terminals - Will show `[Image not found: path]` fallback

## Usage

Use standard markdown image syntax:

```markdown
![Alt text](path/to/image.png)
```

### Relative Paths

Images are resolved relative to the note's location:

```markdown
# In notes/project.md
![Diagram](images/diagram.png)
# Resolves to: notes/images/diagram.png
```

### Absolute Paths

```markdown
![Photo](/Users/username/Pictures/photo.jpg)
```

## Supported Formats

- PNG (`.png`)
- JPEG (`.jpg`, `.jpeg`)
- GIF (`.gif`)

## Testing

A test note with sample images is available:

```bash
cd tui-client
./lumi ../notes
# Navigate to "image-test" note
```

## Implementation

- **Detection**: Regex pattern matching for `![alt](path)`
- **Protocol**: Kitty graphics protocol with base64 encoding
- **Fallback**: Error message for missing images or unsupported terminals

## Limitations

- Terminal must support Kitty graphics protocol
- Large images may cause rendering delays
- No image resizing (uses original dimensions)
- No caching (re-encodes on each render)

## Future Enhancements

- [ ] Image caching
- [ ] Automatic resizing to terminal width
- [ ] iTerm2 inline images protocol
- [ ] Sixel protocol support
- [ ] Image preview in tree view
