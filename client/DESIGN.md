## Overview

PandaAI IM follows Apple's design language — **near-invisible UI, single blue accent, generous whitespace**. The interface recedes so conversations take center stage. Every interactive element speaks in the same quiet blue (`#0066cc`); surfaces are white, pearl, or parchment; typography is SF Pro at 17px body with tight negative kerning at display sizes.

This is not a marketing site — it's a chat app — but the same visual grammar applies: no decorative gradients, no shadows on UI elements, pill-shaped action targets, and a single consistent rhythm of 8px-based spacing.

## Colors

### Brand & Accent
- **Action Blue** (`#0066cc`): The single interactive color. All primary buttons, unread badges, send button, incoming message link accents, focus indicators.
- **Sky Blue** (`#2997ff`): Read receipt checkmark, lighter accent for secondary states.

### Surface
- **White** (`#ffffff`): Text field backgrounds, own chat bubble background (on parchment canvas), profile card.
- **Parchment** (`#f5f5f7`): Default app canvas — conversation list background, chat view background, input bar background, connection status bar.
- **Pearl** (`#fafafc`): Other-user chat bubble background, secondary surface for non-interactive containers.

### Text
- **Ink** (`#1d1d1f`): Primary text — headlines, conversation names, own chat bubble text (white bubble on ink).
- **Ink Muted** (`#7a7a7a`): Secondary copy — timestamps, sender IDs, preview text, captions, placeholder text.

### Hairlines & Borders
- **Hairline** (`#e0e0e0`): Divider lines in profile sheet, picker separators, subtle structural lines.

## Typography

- **Font Family**: SF Pro (`system-ui, -apple-system, system` — resolves to SF Pro on macOS).
- **Body**: 17px / 400 — default paragraph, button labels, input text.
- **Caption**: 14px / 400 — timestamps, sender names, preview lines, error messages.
- **Fine Print**: 12px / 400 — unread counts, connection status, secondary badges.
- **Display Headline** (login/register): 40px / 600 / `-0.374px` kerning — the signature Apple tight tracking.
- **Semibold inline** (conversation names, profile name): 17px / 600.

## Layout & Spacing

- **Base unit**: 8px. All structural layout snaps to multiples of 8.
- **Spacing tokens**: `xs` 8px · `sm` 12px · `md` 17px · `lg` 24px · `xl` 32px · `xxl` 48px.
- **Content width**: Login/register form container is 320px wide.
- **Sheet sizes**: Profile 340×460px, New Chat 400×500px, Group Detail 360×450px.

## Window

- **Login/Register**: Title bar hidden (`titleVisibility: .hidden`, `titlebarAppearsTransparent: true`, no separator), window non-opaque with `NSVisualEffectView` for frosted glass background. Window is movable by dragging background.
- **Main App**: Standard titled window restored after login.

## Components

### Login & Register

Full-screen frosted glass canvas with centered form stack. No chrome — only the traffic light buttons persist.

- **Headline**: 40px / 600 / `-0.374px` kerning, ink color.
- **Text fields**: White capsule (`.clipShape(Capsule())`), 44px tall (`16px H + 12px V` padding × 2), no border, `.textFieldStyle(.plain)`.
- **Primary button**: Action Blue capsule, 44px tall, 22px horizontal padding, white text 17px/400. Active state: `scale(0.95)`.
- **Secondary button**: Action Blue outline capsule, 14px/400 text, 14×8px padding, `scale(0.95)` on press.
- **Spacing**: `xl` (32px) between form elements.

### Conversation List

Parchment canvas with plain list and hidden row separators.

- **Rows**: HStack with 48px avatar circle (`opacity(0.15)` tint) + text stack. 8px vertical padding.
- **Conversation name**: 17px / 600, ink color, single line.
- **Preview text**: 14px / 400, ink muted, single line.
- **Timestamp**: 12px / 400, ink muted.
- **Unread badge**: Action Blue capsule, 12px/600 white text, 8px horizontal padding.
- **Mention badge**: Orange capsule.

### Chat View

Parchment canvas. Messages are in a `LazyVStack` within a `ScrollView`.

- **Own bubble**: Action Blue background, white text 17px/400, `RoundedRectangle(cornerRadius: 18)`. Aligned right with `Spacer(minLength: 60)`.
- **Other bubble**: Pearl background, ink text 17px/400, `RoundedRectangle(cornerRadius: 18)`. Aligned left.
- **Sender label**: 12px / 400, ink muted, shown above other-user messages.
- **Timestamp**: 12px / 400, ink muted, below each bubble.

### Input Bar

Parchment bar at the bottom of chat view.

- **Text field**: Pearl capsule, 17px/400, no border, `.textFieldStyle(.plain)`.
- **Send button**: `arrow.up.circle.fill` SF Symbol, `title2` size, Action Blue when text non-empty, ink muted when empty/disbled.
- **Padding**: 12px around the HStack.

### Profile Sheet

White card presented as a `.sheet` at 340×460px.

- **Header**: 17px / 600 "Profile" left, Action Blue "Done" right.
- **Avatar**: 56px circle.
- **Name**: 20px / 600.
- **Account/ID**: 14px / 400, ink muted.
- **Settings**: Segmented pickers for theme and language.
- **Logout**: Red destructive label with icon, `.buttonStyle(.plain)`.

## Do's and Don'ts

### Do
- Use Action Blue (`#0066cc`) for every interactive element — buttons, unread badges, send icon. The single accent is non-negotiable.
- Use Parchment (`#f5f5f7`) as the default canvas color for content views.
- Reserve Pearl (`#fafafc`) for chat bubbles from others and secondary surfaces.
- Apply capsule shape to text inputs and all buttons.
- Use Ink Muted (`#7a7a7a`) for ALL secondary text — timestamps, captions, previews, placeholders.
- Use negative kerning (`-0.374px`) on display-size headlines (40px+).
- Apply `scale(0.95)` as the active/press state on interactive elements.
- Keep the input bar and chat view on Parchment, with bubbles in Pearl/Action Blue.

### Don't
- Don't introduce a second accent color; every interactive signal is Action Blue.
- Don't use shadows on UI elements — reserved for product photography (not applicable in chat).
- Don't use gradients as decorative backgrounds.
- Don't add borders to text inputs — capsule shapes replace border signaling.
- Don't use 16px body text — always 17px.
- Don't show row separators in the conversation list.
- Don't use weight 500 — the ladder is 400 / 600.

## Elevation & Depth

- **Flat** — most surfaces are flat with no shadow or border.
- **Window frosted glass** — Login/register uses `NSVisualEffectView` with `behindWindow` blending and `.hudWindow` material.
- **No UI shadows**. The only depth is the frosted glass backdrop on the login canvas.
