# lumi design language

Status: v2, 2026-11-26 (motion v2, rich prose palette, space pass, Livros shelf). v1 was 2026-11-25. This applies to every lumi client: web (reference implementation), Apple (macOS first) and TUI. It refines SPEC-PARTE2 §1 ("look and motion") and keeps its limits.

Source of truth in the web client:

- `src/styles/tokens.css`: every colour, depth, type, space, shape and motion token.
- `src/lib/motion.ts`: the spring numbers (with a test that keeps the CSS `linear()` strings in sync), the spring integrator and the shared rAF engine (motion v2), and the motion level.
- `src/lib/motion-actions.ts`: the choreographies built on it (cursor slab, block rise, carousel, origin of overlays).
- `src/lib/prose-palette.ts`: the rich prose palette, contrast-checked per theme.
- `src/lib/ambient.ts`: ambient light and the pause while the tab is hidden.

The owner's brief (2026-11-25), in Portuguese: *"quero um design vivo, quero sentir que lumi é um app vivo, as animações precisam ser boas; mas a navegação está boa, e o split também."* In English: the design should feel alive, the animations have to be good, and the navigation and splits are already good. So this document changes look and motion only.

Owner feedback on v1 (2026-11-26): *"as animações e a sensação do lumi ser vivo não aconteceu; a mudança para um design mais moderno sim, mas ainda precisa melhorar mais"* — the motion was too subtle and short to be perceived (≤ 220 ms, 1 % overshoot, tiny transforms, close instant, a cross-fade on note open). Motion v2 (§6) keeps every key instant but makes each response *felt*: real springs with visible physics, choreography with an origin and a destination, and exits that are quick but visible. Also: *"o html do markdown renderizado não está bom, e está só com uma cor"* → the rich prose palette (§2.5). Navigation (the floating explorer) and the split model stay exactly as SPEC-PARTE2 §2–3 define them.

---

## 1. Personality: alive, calm, precise

| | means | never |
|---|---|---|
| **Alive** | Things respond. The cursor glides, focus blooms, presence breathes, a remote edit glows once. The app shows it is awake without being asked. | Gimmicks, perpetual motion in the reading area, confetti. |
| **Calm** | Motion is slow where it is ambient (≥ 2.4 s periods) and quick where it is a response: 90 % there in ≈ 90–160 ms, the spring's tail (≤ 720 ms) is the physics you feel, never a wait. Colour is the theme's own, used sparingly. | Flashing, bouncing for its own sake, saturated fills. |
| **Precise** | Every movement explains a change of state: where something came from, where it went, what has focus. Keys are never delayed. | Decorative motion that hides state, any animation on the typing path that blocks input. |

Test for any new piece of motion: *does it tell the user something (focus, origin, result, presence)?* If not, it must be ambient and ≥ 2.4 s, or it goes.

---

## 2. Colour

### 2.1 Inputs

Each theme in the shared catalogue (`themes.ts` = `tui-client/theme/builtin.go`) provides 15 colours: primary, secondary, accent, muted, background, selectedBg, overlayBg, text, textDim, border, separator, error, warning, info, plus the logo colours. Components never use these raw.

### 2.2 Derived ramps (oklch)

All ramps are `color-mix(in oklch, …)` of the inputs. oklch mixes are perceptually even, so every theme, light or dark, gets the same structure.

| token | recipe | use |
|---|---|---|
| `--surface-sunk` | overlayBg 55% + background | wells, inputs, the resting-pane canvas |
| `--surface-0` | background | the page, the reader |
| `--surface-1` | text 3.5% over background | bars, lanes, raised rows |
| `--surface-2` | text 6% | active segment, keycaps |
| `--surface-3` | text 9% | switch track |
| `--ink-1` | text | body, titles |
| `--ink-2` | text 78% | secondary text, resting pane titles |
| `--ink-3` | text 58% (66% light) | metadata, labels. Replaces the theme's `textDim` for anything a person must read (textDim is often below 3:1). |
| `--ink-4` | text 36% | decoration only: separators, counts, glyphs |
| `--accent` | primary | edges, carets, the cursor bar |
| `--accent-ink` | primary 82% + text | accent text (more legible than raw primary on dark) |
| `--accent-wash` / `-soft` / `-line` / `-glow` | primary at 9 / 16 / 45 / 34 % alpha | fills, selected fills, rings, light |
| `--hairline-soft` / `--hairline` / `--hairline-strong` | border at 36 / 62 / 90 % | dividers, by importance |
| `--mode-normal/insert/visual/replace` | primary / accent / secondary / error | vim mode colour (§6.7) |

### 2.3 Ambient accent

The accent is light, not paint. It appears as:

- thin lines (the 2 px focus edge, the cursor bar);
- washes (≤ 16 % alpha);
- glows (`0 0 Npx var(--accent-glow)`) under things that have focus or are being moved.

Never use a solid accent fill except on the one primary button of a view.

### 2.4 Light and dark

The same tokens apply in both, with these changes for light themes (`[data-theme='light']`):

- shadows use an ink-tinted colour (`rgb(40 48 80)`) at about 45 % of the dark strength;
- the top edge light is a white line;
- glass is denser (78–90 % over overlayBg);
- surfaces derive from overlayBg rather than adding text.

Contrast floors are 4.5:1 for body text (`ink-1`, `ink-2`) and 3:1 for UI text and icons (`ink-3`). `ink-4` is never the only carrier of meaning.

### 2.5 Prose palette (rendered markdown)

Rendered markdown is not one colour. `lib/prose-palette.ts` derives, per theme, a palette that every client can reproduce, and pulls each colour toward the theme's ink until it clears its contrast floor on the surface it sits on (dark themes mix toward `text`; light themes toward `text` darkened 45 % with black, which keeps the hue's chroma). Tint strengths shrink until body text keeps 4.5:1 on them. A vitest checks every theme.

| element | source | floor |
|---|---|---|
| h1 | primary (+ a 3 px primary→secondary bar under it) | 4.5 |
| h2 | secondary (rule starts in its colour, fades into a hairline) | 4.5 |
| h3 / h4 / h5-6 | accent 62 % · primary 42 % · secondary 40 % over text; h5/h6 caps | 4.5 |
| bold / italic | text + 30 % warning (warm) / text + 30 % secondary | 4.5 |
| strike | ink at 62 %, line in error | 3 |
| ==mark== | warning tint (≤ 38 %), text on it | 4.5 on the tint |
| link | accent; resting 1 px line at 40 %, a solid line draws in from the left on hover (glide) | 4.5 |
| inline code | a theme green (the theme's own green when it has one, else #9ece6a / #3f7d32) on a 12 % tint pill | 4.5 on the pill |
| code block | primary-tinted surface; syntax: keyword secondary, string green, number warning, function primary, type accent, property info, tag error, comment ink | 4.5 (comment 3) |
| quote | secondary→primary bar, secondary 6 % wash, italic | 4.5 on the wash |
| callouts | note/info = info, todo = primary, abstract = accent, tip/success = green, question/example = secondary, warning = warning, danger = error, quote = muted; title in the colour, body in text | 4.5 on the tint |
| list markers, task boxes | primary | 3 |
| #tags | chips; hue = FNV-1a(name) mod 5 over primary, secondary, accent, green, warning | 4.5 on the chip |
| tables | header primary 10 % tint + primary ink; zebra primary 3.5 % | 4.5 |
| hr, footnote refs | primary→secondary gradient; accent pill refs | — |

Mac: compute the same values once per theme (the algorithm is 40 lines; no runtime colour mixing per draw). TUI: the heading ramp, link, code and callout colours map 1:1 to lipgloss foregrounds; tints become nothing (terminals have no alpha).

---

## 3. Depth: elevation, glass, light

Elevation means layered shadows: a tight contact shadow, a wide soft ambient one, and a hairline ring. There are five levels:

| level | token | used by |
|---|---|---|
| 0 | none | the page, panes |
| 1 | `--elev-1` | cards (kanban), raised rows |
| 2 | `--elev-2` | hovered cards, canvas nodes, the drop label |
| 3 | `--elev-3` | menus, toasts, hovered canvas nodes, kanban landing |
| 4 | `--elev-4` | floating explorer, palette, hover card, the dragged kanban card |

- **Edge light.** Every raised surface adds `--edge-light`, a 1 px inset highlight on the top edge. Surfaces then read as lit from above, like glass.
- **Glass** (`--glass-bg` + `--glass-blur` = blur 22 px, saturate 1.5) is only for surfaces that float over content: the explorer, palette, hover card, context menu, toasts and the drop label. Rules:
  1. never stack two glass layers (the explorer scrim is a plain tint with a vignette, not a blur);
  2. glass always has an elevation-4 or elevation-3 shadow and the edge light;
  3. the panel may carry a faint accent wash on its top edge (the explorer does).
- **Scrim.** A tint plus a radial vignette. No blur.

---

## 4. Typography

There are two systems that never mix.

**UI (chrome).** System sans in px, dense, never scaled by the prose setting.

| token | px | use |
|---|---|---|
| `--text-2xs` | 11 | status bar, legend, counts (mono) |
| `--text-xs` | 12 | pane bars, buttons (sm), metadata |
| `--text-sm` | 13 | rows, buttons, crumbs |
| `--text-md` | 14 | base UI |
| `--text-lg` | 16 | dialog titles |
| `--text-xl` | 20 | preview card title |
| `--text-2xl` / `--text-3xl` | 28 / 40 | the home title |

Tracking: `--tracking-ui` (−0.006 em) on labels, `--tracking-tight` (−0.022 to −0.032 em) on display sizes, `--tracking-caps` (+0.08 em) on mono eyebrows. Use mono only for paths, keys, counts and the status line.

**Prose (`.lumi-prose`).** One system for the reader, split preview, Live Preview, hover card, explorer preview, kanban cards and canvas text. Faces are sans, serif or mono; sizes are 16 / 18 / 14.5 px with all rhythm in em; the measure is 68ch. Headings are 720 weight with tight tracking. h2 carries a rule that starts with a hint of accent. Links lift their underline and gain a faint accent wash on hover. Blockquotes are italic. Task checks pop on the bouncy spring.

---

## 5. Space, radius, hairlines, icons

- **Grid.** 4 px (`--sp-1` … `--sp-7` = 4 8 12 16 24 32 48).
- **Radii grow with the surface.** xs 4 (keycaps, chips), sm 6 (rows, buttons), 8 (cards), lg 12 (lanes, canvas nodes), xl 16 (floating panels), pill (badges, toasts, presence).
- **Hairlines.** 1 px, from the three hairline tokens. A 2 px line always means focus or mode.
- **Icons.** 16-unit grid, 1.2–1.5 px strokes with round caps and joins, `currentColor`, 12–15 px rendered. Folders are tinted secondary, notes are ink-3, and the cursor row's icon turns accent.

---

**Space (2026-11-26, owner: *"não está tendo um bom aproveitamento de espaço"*).** Reading width is a setting ("Largura de leitura": estreita 68ch · média 84ch, default · larga 110ch · total), one `--md-measure` for the reader, Live Preview and split preview; wide screens show an outline rail in the gutter when it fits. Row density is a setting ("Densidade": confortável 30 px · compacto 24 px). Chrome is one row: top bar 38 px, note bar 36, pane header 32, one 24 px status line that carries the legend. The explorer takes up to 90 % × 86 % of the viewport.

**Keys (SPEC-PARTE2 §5b).** `-` always opens the explorer, including from a focused canvas or board in any mode. Canvas zoom: in `+` `=` `zi` ⌘/Ctrl-`+`; out `_` `zo` ⌘/Ctrl-`-`; fit `0` `zf`.

---

## 6. Motion language

### 6.1 Springs (v2)

Every response uses one of five named springs (mass 1). v1's springs were critically damped and ≤ 320 ms: correct, and invisible. v2 keeps the *rise* short (90 % of the way in ≈ 90–160 ms, so nothing feels slow) but lets the tail carry visible physics. The web samples each into a CSS `linear()` easing (`--spring-*`, paired with `--spring-*-ms`) and also integrates them for real in JS (§6.10).

| name | stiffness | damping | ζ | ω0 | settle (ms) | overshoot | SwiftUI `.spring(response:dampingFraction:)` | use |
|---|---|---|---|---|---|---|---|---|
| **snappy** | 700 | 38 | 0.72 | 26.5 | 320 | 3.6 % | (0.237, 0.72) | press, small UI, pills |
| **glide** | 380 | 26 | 0.67 | 19.5 | 410 | 5.8 % | (0.322, 0.67) | things that *travel*: cursor slab, folder carousel, pane sizes, focus light, landing |
| **gentle** | 260 | 28 | 0.87 | 16.1 | 370 | 0.2 % | (0.390, 0.87) | content rising in, panes, views |
| **bouncy** | 400 | 20 | 0.50 | 20.0 | 560 | 16 % | (0.314, 0.50) | things that *pop*: overlays opening, toasts, badges, names, nodes, drops |
| **soft** | 150 | 22 | 0.90 | 12.2 | 520 | 0 | (0.513, 0.90) | veils, large surfaces, ambient |

Exact equivalent: SwiftUI `.interpolatingSpring(mass: 1, stiffness: k, damping: c)`. TUI (harmonica): `NewSpring(FPS(60), ω0, ζ)` with the ω0 and ζ above.

**Motion level** ("Movimento", Settings → Appearance; palette "Movimento: …"):

| level | meaning |
|---|---|
| **completo** (default) | everything below |
| **suave** | the same choreography; travel × 0.45 and no overshoot (every spring's damping raised to ≥ 0.95 × critical; CSS `--spring-snappy/glide/bouncy` take gentle's curve) |
| **reduzido** | instant state changes; no transforms or fades. The OS *prefers-reduced-motion* always maps here. |

Durations that are not springs:

| token | ms | use |
|---|---|---|
| tap | 90 | press feedback, hover tints |
| fade | 140 | colour and opacity changes |
| exit | 120 | every exit; always faster than its entrance |
| flash | 700 | remote-change flash (sync ping rings: 900 ms, a second ring 160 ms later) |
| breath | 2400 | presence dots, sync breathing |
| ambient | 26000 | the home light drift (v1: 48 s, too slow to notice) |

### 6.2 Choreography rules

1. **Enter on a spring, exit quick but visible.** Entrances use gentle, glide or bouncy. Exits are never instant in completo: a *ghost* (a snapshot clone) plays the exit on snappy while the real DOM is already in its final state, so the key path never waits on an exit.
8. **Interruptible, never queued.** Every JS-driven motion is a spring integrated per frame; a new target mid-flight keeps position *and velocity* (holding `j` makes the cursor slab stream, repeated `l` adds momentum to the carousel).
2. **Origin is truth.** Things come from where they were. An explorer row grows into the note title (shared element), a link morphs into its hover card, a split opens from the side it was asked for, a zoomed pane slides from its old box, and a kanban card lands from where it was dropped.
3. **Stagger is a ripple, not a queue.** 16–55 ms steps (rows 22, blocks 26, columns 55), at most 240 ms in total, never more than 14 items, and items are interactive from frame 0.
4. **Follow-through.** Things with mass overshoot by about 1 % (glide) or lean with velocity (the kanban card tilts by up to 7°, proportional to horizontal pointer velocity, and settles upright when still). Canvas pans coast with exponential friction (τ ≈ 260 ms).
5. **Anticipation is press.** A press scales to 0.96–0.985 at tap speed and springs back on release (snappy). Nothing winds up before acting.
6. **One thing leads.** In a compound change, the element the user acted on moves first, and the context follows 40–70 ms later (folder travel: the column first, the preview after; note open: the old page leaves, the new arrives 70 ms later, so two texts never cross-dissolve).
7. **Sequential, not ghosted.** Never cross-fade two different texts at 50 %. Leave first, then arrive.

### 6.3 Focus between panes

The focused pane is *lit*:

- a 1 px accent-line ring;
- a 2 px top edge of light that breathes (opacity 1 → 0.55, 5.2 s);
- an accent wash fading down 90 px.

Resting panes sit under an 18 % veil of the background colour. On focus change the ring blooms in on the gentle spring, while the old one leaves at exit speed and the veil crossfades on soft. Both are opacity-only. Zoomed panes have no ring and no veil.

### 6.4 Ambient life

All ambient effects are cheap, pause while the window is hidden (`:root[data-hidden]`), and stop under Reduce Motion.

- **Home light.** Two very soft blooms of primary and secondary, each 64–78 vmax, as radial gradients with no blur filter. They drift and rotate on a 48 s and a 61 s cycle (alternate, transform-only), with a 3.5 % grain overlay against banding.
- **Wordmark.** The mark's conic light turns once a minute (an `@property` angle).
- **Cursor-proximity light.** Interactive surfaces marked `.glow` carry a 160 px radial accent light under the pointer. One passive listener, one write per frame, only on the hovered element.
- **Breathing.** Live presence dots, the syncing dot, the focused pane's top edge, the kanban drop slot.

### 6.5 Feedback micro-interactions

| moment | motion |
|---|---|
| hover | 0.5–2 px lift (snappy) + tint (tap). Rows nudge their title 2 px. |
| press | scale 0.96 (buttons), 0.985 (rows and cards), tap speed; release on snappy |
| success | the check pops (scale 0.6 → 1, bouncy); a toast with a tone dot slides up (bouncy) |
| error | one damped shake (360 ms, 5 → 4 → 2.5 → 1 px); the error toast shakes once; the delete bar shakes on arrival |
| undo | the restored element re-enters from where it went (FLIP), on gentle |
| drag lift | scale 1.035, elevation 4, accent rim and under-glow, lean with velocity |
| drop target | the receiving container glows (accent line and 32 px glow); the slot breathes |
| landing | from the drop point into the slot on glide (1 % overshoot reads as weight) |
| canvas | nodes spawn on bouncy (scale 0.92 → 1); edges draw from their source (dash offset on soft) and arrow heads fade in as the line arrives; a released pan coasts |

### 6.6 Presence

- **Avatars.** A 22 px disc in the collaborator's colour, with a lit highlight and a 2 px ring in their colour. A live dot breathes beside it. Avatars pop in on bouncy and lift on hover.
- **Remote cursors.** A 2 px caret in the collaborator's colour with a head dot. A name tag pops in (bouncy) on every move, holds about 2.2 s, then fades. A caret that stays idle dims to 45 % over 12 s. Hovering a caret shows the name again.
- **Remote edits.** The changed range flashes the accent once (700 ms) and the sync dot pings.

### 6.7 Mode changes (vim)

The editing pane's **left edge** (2 px, fading downwards) and the note bar's bottom line take the mode colour: normal = primary, insert = accent, visual = secondary, replace = error. The colour cross-fades in 140 ms. The mode pill's width glides and its label cross-fades. The caret blink breathes (eased) instead of switching hard. The caret glides between positions in ≤ 80 ms.

### 6.8 Loading → content

Skeletons have one compositor sweep (a translated highlight on a pseudo-element, 1.4 s). Real content resolves out of the skeleton: opacity, 3 px rise and a 2 px blur clearing, on gentle. The explorer preview and the reader do this. Layout never shifts: skeletons reserve the final metrics (CLS < 0.01).

### 6.9 Empty states

Empty states keep the ambient light (the home), keep their keys visible, and have one primary action. No illustrations.

### 6.10 Motion v2 choreography (what you should *see*)

Engine: `Spring` (semi-implicit Euler, ≤ 4 ms sub-steps, frame-rate independent, rests by distance and speed) and one shared rAF loop (`engine`) that writes only `transform` and `opacity`, stops when everything rests, clamps frame gaps to 64 ms, and hands elements back to CSS at rest. `exitGhost()` clones an element for an exit. Frame capture can freeze and seek the clock (`window.__lumiMotion`).

| # | moment | before (v1) | after (v2) | springs / timing | Mac (SwiftUI) | TUI |
|---|---|---|---|---|---|---|
| 1 | Explorer open | panel fades in at scale .965 (≈ 3 % change), scrim 140 ms fade | panel GROWS from its trigger (explorer button or the pointer's last trigger), scale .80 → 1 with 16 % overshoot; columns dealt in from the left 55 ms apart; the scrim and its vignette ease in (no second blur layer: it doubled frame cost) | bouncy (panel) · glide (columns, x −26) · soft (scrim) | `.scaleEffect(anchor: triggerUnitPoint)` on bouncy; columns `.offset(x:)` with `.delay(i*0.055)` | box draws at once; columns appear 1 frame apart |
| 2 | Explorer close (Esc, outside) | instant | ghost collapses toward the trigger (scale .86, fade) while the overlay is already gone | snappy | `.transition(.scale(0.86, anchor:).combined(with: .opacity))` | instant |
| 3 | Open note from explorer | row-name morph + cross-fade | ghost panel collapses INTO the opened row (scale .5 at the row), the row name flies to the note title (shared element), old page lifts away (110 ms), new page's blocks rise | snappy · glide · gentle | `matchedGeometryEffect` row→title + panel scale toward row | instant; title flashes accent 2 frames |
| 4 | Folder travel (l / h) | columns shift 14 px | carousel: columns pushed ±90 px toward where you came from and spring home with overshoot; repeated presses ADD (momentum); preview arrives 45 ms later from further out; rows ripple in from the side | glide | `.offset(x:)` with velocity-preserving spring | columns redraw; 2-frame shift |
| 5 | Cursor travel (j / k) | slab slides with a CSS transition | a spring slab that STRETCHES along its travel with speed (scaleY up to 1.55 from the trailing edge) and settles to its own height; holding j streams; long jumps start ≤ 6 rows away | glide | `.offset(y:)` + `scaleEffect(y: 1 + min(.55, |v|/2600), anchor: trailing)` | highlight moves; previous row dims 1 frame |
| 6 | Note opens (content) | page cross-fade | content RISES by blocks: first 10 visible blocks, y 18 → 0 and fade, 26 ms apart (≤ 240 ms), after a 40 ms beat; only when a different note arrives (edits never) | gentle | `.offset(y:)`+opacity per paragraph with `.delay` | lines revealed 1 frame apart (≤ 6) |
| 7 | Pane split | new pane slides 18 px | the new pane UNFOLDS from the divider (scaleX/Y .4 → 1 from the shared edge, overshoots); neighbours spring to their new sizes (FLIP) | glide | `.transition(.scale(0.4, anchor: .leading))` + frame animation | border draws from the split side (3 frames) |
| 8 | Pane focus | ring fades in | a bar of LIGHT travels along the top edges from the old pane to the new one, then fades into the new pane's lit edge; ring blooms | glide (travel) · soft (fade) | a `Capsule` overlay with `matchedGeometryEffect` between panes | border colour moves |
| 9 | Pane zoom | slide from old box + clip | the pane expands from its box (clip + translate); the others RECEDE (ghosts scale .9 and fade); unzoom: the pane FLIPs back into its slot and the others come forward one by one (45 ms apart) | glide | `matchedGeometryEffect` + others `.scaleEffect(0.9).opacity(0)` | instant + badge |
| 10 | Vim mode change | edge colour cross-fade 140 ms | the new mode's colour SWEEPS down the editor's edge (a lit 3 px bar, 520 ms), edge colour eases over 320 ms, the mode pill gets a bouncy kick (scale 1.18 → 1) | bouncy (pill) · ease (sweep) | `Rectangle().offset(y:)` sweep + pill `.scaleEffect` | gutter colour = mode colour |
| 11 | Presence | name tag pops at .92 | name tags pop from .55 (bouncy) and hold 3.6 s; a peer's caret arrives with a short grow (scaleY .3 → 1) | bouncy · glide | overlay label `.transition(.scale)` | "● Maya" in the status line |
| 12 | Sync / remote change | 700 ms ring to 2.8× | the dot itself takes the hit (scale 1.9 → 1, bouncy), two rings ripple out (to 3.6×, 900 ms, 160 ms apart), the whole badge pulses (1.08) | bouncy · 900 ms | `.scaleEffect` + `Circle().stroke` rings | dot glyph flips 2 frames |
| 13 | Toasts | fly 14 px | spring UP from below the status bar (y 44, scale .82, 16 % overshoot); the stack makes room with a glide FLIP; exit drops 14 px in 160 ms | bouncy · glide | `.transition(.move(edge: .bottom).combined(with: .scale(0.82)))` | status line message |
| 14 | Palette | settle .96 | springs open from just above (scale .9, y −18); results ripple (first 10, 18 ms); the selection is a stretching slab | bouncy · glide | sheet `.scaleEffect(anchor: .top)`; `matchedGeometryEffect` selection | instant |
| 15 | Hover card | scale .92 | GROWS out of the link (scale .3 × .2 from the link's x) with a tether nub that pops first | bouncy | popover `.scaleEffect(anchor: linkPoint)` | box draws |
| 16 | Kanban | lift 1.035, flip 180 ms | lift 1.06 with lean up to 9° by velocity; neighbours make room on glide (overshoot); the drop BOUNCES into the slot | bouncy (drop) · glide (room) | `.scaleEffect(1.06)` + `.rotationEffect` from velocity; `.spring` reorder | reverse video at the cursor |
| 17 | Canvas | nodes pop .92 at once | on open, nodes pop from .6 + 14 px, 28 ms apart (≤ 14 steps); edges draw after (620 ms) and arrow heads pop at the end; pan coasts (τ 260 ms); keyboard zoom eases (gentle) | bouncy · soft | `.scaleEffect` with `.delay(i*0.028)`; `trim(from:to:)` | instant |
| 18 | Home | 2 blooms, 48 s drift, static to the pointer | 3 brighter blooms on a 26 s drift that lean AWAY from the pointer (parallax per depth); title, meta, actions and recents rise in turn | gentle · glide · soft | `TimelineView` + `MeshGradient`; `.onContinuousHover` parallax | none |
| 19 | Book switch | — | the new book OPENS like a cover: the page swings in from its spine (rotateY −24° → 3° → 0, perspective) | 560 ms ease-out | `.rotation3DEffect(.degrees(-24), axis: (0,1,0), anchor: .leading)` | instant |
| 20 | Books ⇄ files (Livros) | — | scope change: content arrives from another depth (1.1 / .88), the panel takes the hit (.94 / 1.03, bouncy) | gentle · bouncy | `.scaleEffect` + opacity | instant |

---

## 7. Performance and accessibility rules (hard)

1. **Compositor only** for anything that repeats or runs over 140 ms: `transform`, `opacity`, `filter` on small elements. Exceptions are allowed one element at a time for ≤ 320 ms: `clip-path` (zoom reveal), box-shadow or colour transitions on hover/focus, and SVG `stroke-dashoffset` (edge draw).
2. **60/120 fps.** No layout reads inside animation loops; FLIP reads once, before the change. Large animated layers get `will-change: transform` and nothing else does.
3. **Input never waits.** No `await` on an animation in a key handler. The DOM is final at once and the animation only decorates it. Keys typed mid-animation apply immediately. The typing path (editor) gets no new animation beyond the 80 ms caret glide and the breathing blink.
4. **Reduce Motion** (the OS setting plus the app setting Movimento: reduzido → `data-motion="reduce"`): every animation and transition is off (a global kill-switch in `app.css`), JS helpers return early (`motionAllowed()`), View Transitions are skipped, and canvas inertia is off. State changes are instant and identical in meaning.
5. **Hidden tab:** all CSS animations are paused.
6. **Contrast:** §2.4 floors; focus is never shown by colour alone (ring and edge).
7. **Budgets** (measured in headless Chromium, 1440 px, `scripts/perf/motion-perf.mjs`): explorer open, cursor, folder travel, note open, pane focus/zoom, palette, kanban drag and canvas pan hold p95 frame time ≤ 17.5 ms; CLS < 0.01; no new long task over 50 ms caused by motion; typing latency (keydown → next frame) unchanged.
8. **Motion level** (§6.1): `reduzido` is the old Reduce Motion; `suave` must keep every choreography readable with less travel.

---

## 8. Event → motion token map (porting table)

v1 table. Where a row overlaps §6.10, §6.10 (motion v2) wins; the token names are the same, their numbers are §6.1's.

| # | UI event | web implementation | token | Mac (SwiftUI) | TUI |
|---|---|---|---|---|---|
| 1 | Explorer opens | scrim fade + panel `fx-in` (y 10, scale .965 → 1) | bouncy · fade 140 | `.spring(0.25, 0.60)` scale + opacity | draw immediately; rows ripple in 1 frame apart (≤ 8 frames) |
| 2 | Explorer rows enter | `stagger()` 18 ms, ≤ 14 rows | gentle | `.spring(0.31, 0.88).delay(i*0.018)` | row i revealed at frame ⌈i/2⌉ |
| 3 | Explorer cursor moves | `.glide` slab translateY | glide 260 | matchedGeometryEffect / offset on glide | highlight moves at once; previous row keeps a dim trail for 1 frame |
| 4 | Folder in/out (l/h) | columns translateX ±14 → 0, preview fades 40 ms later | gentle | `.transition(.move(edge:))` on gentle | columns redraw; current column shifts 1 cell over 2 frames |
| 5 | Note opens from explorer | row name → title shared element; old page exits 90 ms, new arrives at +70 ms | glide / exit | `matchedGeometryEffect` on title | instant; title line flashes accent once (2 frames) |
| 6 | Pane split opens | new pane slides from its side (18 px) + scale .985 | gentle | `.transition(.move(edge:).combined(with:.opacity))` | border draws in from the split side (3 frames) |
| 7 | Pane focus moves | ring blooms in, veil crossfades | gentle (in) · exit 120 (out) · soft (veil) | overlay stroke + opacity on gentle | border colour → accent; inactive borders → ink-4 |
| 8 | Pane zoom | slide from old box + clip widens | glide | `matchedGeometryEffect` frame grow | instant; "zoomed" badge |
| 9 | Pane unzoom | other panes fade/scale .97 → 1, 30 ms stagger | gentle | opacity + scale | instant |
| 10 | Hover card | fade + scale from the link (shared `hover-card`) | bouncy | popover with spring | box draws at once |
| 11 | Palette / sheets | `settle-in` | bouncy | `.spring(0.25, 0.60)` | instant |
| 12 | Toast | slide up 16 px + tone-dot pop; stack FLIP | bouncy · gentle | `.move(edge:.bottom)` | status line message, 2.4 s |
| 13 | Error | shake 360 ms | shake curve | `.modifier(Shake)` | message in error colour, 1 frame invert |
| 14 | Button / row press | scale .96 / .985 | tap 90 → snappy | `ButtonStyle` scale on snappy | reverse video for 1 frame |
| 15 | Hover | lift 0.5–2 px + tint; `.glow` light | snappy · tap | `.onHover` + spring | none (no pointer) |
| 16 | Vim mode change | edge + bar line colour, pill width | fade 140 · glide | edge `Rectangle` colour animation | gutter/edge colour = mode colour |
| 17 | Caret move / blink | 80 ms glide; eased blink | ease | TextKit caret glide | terminal caret |
| 18 | Remote cursor | caret + name tag (pop, hold 2.2 s, fade); idle dim 12 s | bouncy · linear | overlay label with timer | coloured block cursor + name in status line |
| 19 | Remote edit | range flash 700 ms; sync ping | flash | background highlight fade | changed lines get an accent gutter mark for 700 ms |
| 20 | Presence arrives / leaves | avatar pop / fade; live dot breathes | bouncy · breath | scale/opacity; repeatForever breathing | "● Maya" in status line, dot pulses ● ◦ every 1.2 s |
| 21 | Sync in progress | dot breathes 1.6 s | breath | repeatForever opacity | spinner frames ⠋⠙⠹ at 80 ms |
| 22 | Kanban drag lift | ghost scale 1.035, elev-4, lean by velocity | bouncy · glide | `.scaleEffect` + `.rotationEffect` from drag velocity | card shown in reverse video at the cursor |
| 23 | Kanban drop target | column glow; slot breathes | gentle | stroke + shadow | column border accent |
| 24 | Kanban land | from ghost to slot | glide | matchedGeometryEffect | instant |
| 25 | Canvas pan release | inertial coast τ 260 ms | friction | `DragGesture` predicted end + decay | none |
| 26 | Canvas node spawn / edge add | pop .92 → 1; edge draws | bouncy · soft | scale + `trim(from:to:)` | instant |
| 27 | Loading → content | skeleton sweep → resolve-in | gentle | `.redacted` → opacity on gentle | "…" then content |
| 28 | Home ambient | 2 blooms drifting 48/61 s | ambient | `TimelineView` + `MeshGradient`, paused when inactive | none (calm) |
| 29 | Explorer drag-out to split | overlay steps aside (opacity .22, scale .94), edge ghost glides between sides | gentle · glide | drop delegate + highlight | n/a (keys: ⌃v / ⌃s) |
| 30 | Reduce Motion | everything instant | — | `accessibilityReduceMotion` | `--reduce-motion` / setting: no frame stepping |

---

## 9. Porting notes

**Mac (SwiftUI, `Sources/LumiUI/Shell/Motion.swift`).**

1. Add a `LumiSpring` enum with the five springs as `.interpolatingSpring(mass: 1, stiffness:, damping:)`, plus `LumiDuration` for the durations in §6.1. Replace the ad-hoc `.spring(response: 0.24, dampingFraction: 0.9)` calls.
2. Derive the ramps in `LumiTheme` with `Color.mix(_:by:in: .perceptual)` (macOS 15), or by interpolating in OKLab (a helper of about 20 lines).
3. Use glass as `.background(.ultraThinMaterial)` with a theme tint overlay on floating panels only.
4. Draw the pane focus ring as an overlay `RoundedRectangle().strokeBorder(accentLine)` plus a top `Rectangle` whose opacity breathes with `repeatForever(autoreverses:)` on 2.6 s halves.
5. Honour `@Environment(\.accessibilityReduceMotion)` and the app setting in one `motionAllowed` helper.

**TUI (Bubbletea, `tui-client/ui`).** Motion becomes frame stepping on a 60 Hz `tea.Tick`, only while something is animating. The idle TUI must not tick.

- The mode edge maps to the gutter or border colour.
- Focus maps to border colour, accent for the focused pane and ink-4 for the rest.
- Presence and sync map to dot glyphs that alternate every 1.2 s, ticking only while a peer or a sync is live.
- Remote edits map to an accent gutter mark for 700 ms.
- Ambient effects and hover do not port.
- `--reduce-motion` or the setting disables every tick.
