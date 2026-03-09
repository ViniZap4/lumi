---
id: media-showcase
title: Media Embedding Showcase
created_at: 2026-03-06T10:00:00-03:00
updated_at: 2026-03-06T10:00:00-03:00
tags:
  - example
  - media
---

# Media Embedding Showcase

Lumi supports images, video, PDF, and embeds using standard `![alt](src)` syntax.

## Images

![Test Image](images/test1.png)

![Screenshot](images/screenshot.png)

![Architecture Diagram](images/diagram.png)

## YouTube Embeds

![Big Buck Bunny](https://www.youtube.com/watch?v=aqz-KE-bpKQ)

![Sintel Trailer](https://youtu.be/eRsGyueVLvQ)

## Vimeo Embeds

![Vimeo Example](https://vimeo.com/76979871)

## Supported Formats

| Media Type | Syntax | Web | TUI |
| ---------- | :----: | :-: | :-: |
| Image | `![alt](file.png)` | Rendered | Rendered |
| Video | `![alt](file.mp4)` | Player | Thumbnail |
| PDF | `![alt](file.pdf)` | Viewer | Placeholder |
| YouTube | `![alt](youtube.com/...)` | Embed | Placeholder |
| Vimeo | `![alt](vimeo.com/...)` | Embed | Placeholder |

Video extensions: `.mp4`, `.webm`, `.mov`, `.avi`, `.mkv`, `.ogg`, `.ogv`

See also: [[image-test]] and [[markdown-showcase]]
