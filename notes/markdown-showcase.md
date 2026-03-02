---
id: markdown-showcase
title: Markdown Rendering Showcase
created_at: 2026-02-26T10:00:00-03:00
updated_at: 2026-02-26T10:00:00-03:00
tags:
  - example
  - markdown
---

# Markdown Rendering Showcase

This note demonstrates all the inline markdown features supported by lumi's TUI.

## Emphasis

- *asterisk italic* and _underscore italic_
- **asterisk bold** and __underscore bold__
- ***asterisk bold italic*** and ___underscore bold italic___
- ~~strikethrough text~~
- Words_with_underscores stay normal (no false italic)

## Code

Inline: use `fmt.Println("hello")` to print.

```go
func main() {
    fmt.Println("Hello from lumi!")
}
```

```python
def greet(name):
    return f"Hello, {name}!"
```

## Links

- Standard link: [Lumi on GitHub](https://github.com/vinizap/lumi)
- External link: [Go Documentation](https://go.dev/doc/)
- Wikilink to another note: [[welcome]]
- Wikilink: [[vim-tips]]

Press `enter` on a link to follow it. Press `x` on an external URL to open in browser.
Press `s` on a wikilink to open in horizontal split, `S` for vertical.

## Tables

| Feature       |  Status   |                 Notes |
| ------------- | :-------: | --------------------: |
| Bold/Italic   |   Done    | Asterisk + underscore |
| Tables        |   Done    |       Aligned columns |
| Checkboxes    |   Done    |     Toggle with enter |
| Links         |   Done    |        Follow + split |
| Code blocks   |   Done    |    Language highlight |

## Lists

1. First ordered item
2. Second ordered item
3. Third ordered item

- Unordered item one
- Unordered item two
  - Nested item

> This is a blockquote for reference text.

--[Markdown Rendering Showcase](markdown-showcase.md)-

See also: [[todo-example]]