---
name: Овертайм
description: Night-match broadcast scoreboard and printed match programme for a simulated 1X2 football line.
colors:
  sodium: "#ff6a2a"
  sodium-hover: "#ff7c45"
  ink: "#140a06"
  paper: "#0a0a0b"
  surface: "#101012"
  surface-raised: "#141416"
  surface-toast: "#17171a"
  surface-well: "#0c0c0e"
  surface-control: "#151517"
  foreground: "#f2eee7"
  muted-foreground: "#938d83"
  faint: "#8b857b"
  line: "#232326"
  border: "#26262a"
  border-strong: "#34343a"
  sodium-muted-surface: "#241a14"
  sodium-muted-ink: "#a4836f"
  destructive: "#f26d6d"
typography:
  display:
    fontFamily: "Oswald Variable, Arial Narrow, sans-serif"
    fontSize: "4.125rem"
    fontWeight: 500
    lineHeight: 0.9
    letterSpacing: "0.01em"
  headline:
    fontFamily: "Oswald Variable, Arial Narrow, sans-serif"
    fontSize: "1.4375rem"
    fontWeight: 500
    lineHeight: 1
    letterSpacing: "0.02em"
  title:
    fontFamily: "Oswald Variable, Arial Narrow, sans-serif"
    fontSize: "1.1875rem"
    fontWeight: 500
    lineHeight: 1.2
    letterSpacing: "0.02em"
  odds:
    fontFamily: "Oswald Variable, Arial Narrow, sans-serif"
    fontSize: "1.6875rem"
    fontWeight: 500
    lineHeight: 1
    letterSpacing: "0.01em"
    fontFeature: "'tnum' 1"
  body:
    fontFamily: "Golos Text Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: "Golos Text Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.5625rem"
    fontWeight: 500
    lineHeight: 1.2
    letterSpacing: "0.16em"
rounded:
  tag: "3px"
  badge: "4px"
  chip: "6px"
  control: "8px"
  card: "10px"
  dialog: "11px"
spacing:
  "2xs": "2px"
  xs: "4px"
  sm: "6px"
  md: "8px"
  lg: "12px"
  xl: "16px"
  "2xl": "24px"
  "3xl": "32px"
components:
  button-primary:
    backgroundColor: "{colors.sodium}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    height: "44px"
    padding: "0 16px"
  button-outline:
    backgroundColor: "transparent"
    textColor: "{colors.muted-foreground}"
    rounded: "{rounded.chip}"
    height: "28px"
    padding: "0 10px"
  odds-selected:
    backgroundColor: "{colors.sodium}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    height: "62px"
    padding: "8px 12px"
  card-panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.card}"
    padding: "16px"
  input-stake:
    backgroundColor: "{colors.surface-well}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    height: "48px"
    padding: "0 40px 0 12px"
  badge-outline:
    backgroundColor: "transparent"
    textColor: "{colors.muted-foreground}"
    rounded: "{rounded.tag}"
    height: "18px"
    padding: "0 6px"
  dialog-surface:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.dialog}"
    width: "440px"
---

# Design System: Овертайм

## Overview

**Creative North Star: "The Night-Match Programme"**

A floodlit football fixture list printed on near-black paper. The screen should feel like the
scoreboard graphic on a night broadcast and the match programme bought at the turnstile: condensed
uppercase names, oversized tabular prices, hairlines instead of boxes, and one sodium-orange signal
that means the line is alive. It is a working betting surface first; the atmosphere comes from
typography and the grid, never from chrome.

The palette is a warm near-black paper (#0a0a0b) with a single accent (#ff6a2a). Everything that is
not a decision is quiet: muted warm grey labels, 1px rules, flat tonal panels. The orange is reserved
for the selected outcome, the primary action, live connection, and the current price. Errors and a
dead feed are the only red. Amounts and prices are tabular and use Russian decimal commas; money is
always shown from integer kopecks.

Motion is a broadcast cue, not decoration: a number ticks to its new value, a changed price gets one
short opacity flash, a selected slip gets a slow shine border, and rows enter once with a 320ms
ease-out. Nothing loops except the connection dot and the active-slip shine. All of it collapses
under `prefers-reduced-motion`.

**Key Characteristics:**
- Near-black paper, one sodium accent, warm off-white ink.
- Condensed uppercase display against a humanist Russian text face.
- Hairline rules, flat tonal surfaces, sharp 8–11px corners.
- Oversized tabular odds; every price can disclose its two sources and the final price.
- Calm default, precise flash on change; honest demo labels.

## Colors

A two-family palette: warm near-black neutrals and one hot sodium orange. There is no secondary hue.

### Primary
- **Sodium Floodlight** (#ff6a2a): The only accent. Fills the selected odds button, primary submit, live dot, brand mark and active-slip shine. Text on sodium is always **Press Ink** (#140a06), never white.
- **Sodium Hover** (#ff7c45): Hover for filled orange actions only.

### Neutral
- **Near-Black Paper** (#0a0a0b): Page ground. Chosen for a night-match scene, not as a generic dark default.
- **Programme Surface** (#101012): Match list, bet slip, history panel, header chips.
- **Raised Paper** (#141416): Dialogs and league tags.
- **Toast Ground** (#17171a): Ding toast surface; one step above the panel so it reads as a floating layer.
- **Well** (#0c0c0e): Stake input and source table; sits one step below its panel.
- **Control Surface** (#151517): Unselected odds buttons.
- **Floodlit Ink** (#f2eee7): Primary text and rule labels. Warm, never pure white.
- **Programme Grey** (#938d83): Secondary text and labels; meets 4.5:1 on the panel.
- **Faint Grey** (#8b857b): Metadata, footnotes, placeholder text.
- **Hairline** (#232326): The dominant separator, always 1px.
- **Border** (#26262a): Panel and control outline.
- **Border Strong** (#34343a): Input outline and hover edges.
- **Signal Red** (#f26d6d): Errors, lost connection, destructive status only.

### Named Rules
**The One Sodium Rule.** Orange marks action, selection, live state or change. It may fill a control; it never becomes a page field, a gradient wash, or a decorative border.

**The Hairline Rule.** Structure is drawn with 1px rules and one-step tonal shifts. Cards do not get a shadow to separate them from the page.

**The Honest Red Rule.** Red is diagnostic, never decorative. If the line is stale, the status dot, the stale strip and the paused slip all carry it together.

## Typography

**Display Font:** Oswald Variable (fallback Arial Narrow, sans-serif)
**Body Font:** Golos Text Variable (fallback ui-sans-serif, system-ui, sans-serif)
**Label Font:** Golos Text Variable, uppercase micro-labels

**Character:** Oswald is the stadium announcer: narrow, upright, built for names and numbers at a
distance. Golos is the printed programme's reading voice: a contemporary Russian face with generous
Cyrillic, used for sentences, metadata and controls. The pairing is deliberately not a single
do-everything grotesque.

### Hierarchy
- **Display** (500, 66px desktop / 58px tablet / 46px mobile, 0.9): The single page title «Линия дня» and the brand voice. Uppercase, tight leading, never below 46px.
- **Headline** (500, 23px, 1.0): Dialog titles and the highest-priority panel heading. Uppercase.
- **Title** (500, 19px, 1.2): Fixture names, match names in the slip, section headings. Uppercase.
- **Odds** (500, 27px desktop / 25px mobile, 1.0, tabular): Prices in buttons and the slip. Two decimals, comma separator. Price changes tick, never jump.
- **Body** (400, 13px, 1.6): Explanations and empty-state copy, kept under ~65ch. The body measure on the page is capped by the layout, not by a long paragraph.
- **Label** (500, 9–10px, 0.14–0.18em, uppercase): Column headers, kickoff metadata, status pills, source-table headers.

### Named Rules
**The Condensed Upper Rule.** Display type is uppercase and condensed. Body copy is sentence case. Never set a paragraph in Oswald or an uppercase micro-label in Golos at body size.

**The Two-Decimals Rule.** Odds and money always render with two decimals and a Russian comma (`1,95`, `955,00 ₽`). Prices use tabular figures; money never comes from a float.

## Layout

A 12-column desktop grid inside a 1440px container, with 32px outer padding at `lg` and 16px at
mobile. The line list takes columns 1–8; the bet slip takes columns 9–12 at a 400px maximum and is
sticky at `top: 84px`. The header is a 64px sticky band (56px on mobile). The page masthead sits
above the grid; its title is left-aligned and its date/update note is a right-hand programme block.

Match rows are a three-column grid on desktop: 64px kickoff, a flexible fixture column, and a 330px
odds block (288px between 1024 and 1279px). Rows are separated by hairlines inside one 10px panel;
there are no per-row desktop cards. At mobile the same data becomes individual 10px cards with a
16px gap, and the odds stay three equal columns.

Spacing rhythm is 4 / 8 / 12 / 16 / 24 / 32. Dense groups (odds buttons) use 8px gaps; sections use
16px padding; major blocks use 24–32px. More space sits above a heading than below it. The mobile
bet slip lives in a bottom sheet capped at `88svh`; the fixed bottom bar is roughly 60px plus the
safe-area inset.

## Elevation & Depth

Flat by default. Depth is tonal: page → surface → raised, each a small step in near-black, separated
by a 1px hairline. Shadows appear only where a layer genuinely floats: the selected odds button, the
dialog, and the toast.

The only atmosphere is a fixed 64px grid at 5% white, masked to fade out below the masthead, plus a
single 640×480px sodium wash at 7% behind the top-right corner. No glass, no blur panels, no
gradient fields. The dialog scrim is flat 70% black.

### Shadow Vocabulary
- **Selected odds** (`0 0 0 1px rgba(255,106,42,0.25), 0 8px 28px -12px rgba(255,106,42,0.55)`): The one orange lift in the line.
- **Floating toast** (`0 24px 60px -18px rgba(0,0,0,0.85), 0 0 0 1px rgba(255,255,255,0.08)`): Used by Ding's viewport.
- **Dialog** (shadcn ring at 10% white plus the raised surface): No large drop shadow; the 70% scrim does the separation.

## Shapes

Sharp, printed forms: 8px for buttons, the stake field and odds buttons; 10px for cards and the bet
slip; 11px for dialogs and the top of the mobile sheet; 6px for quick-stake chips; 3–4px for league
and status tags. Nothing is pill-shaped except the tiny circular live dot and the scrollbar thumb.

Borders are full 1px outlines on controls and panels. There are no one-sided accent bars, no clipped
corners, no split borders. Selected odds and the active slip are the only places where the border
itself turns sodium. Icons are 12–16px Lucide strokes; the brand mark is a 24px sodium square with a
4px corner holding a condensed «О».

## Components

### Buttons
- **Shape:** 8px radius, height 44px for the primary action, 28px for chips and small controls.
- **Primary:** Sodium fill, ink text, Oswald uppercase 15px with 0.1em tracking. Label names the action and, when a stake is valid, the amount («Поставить 500 ₽»).
- **Hover / Focus:** Sodium hover darkens toward #ff7c45. Focus is a sodium border plus a 2px 30% sodium ring; keyboard use is always visible.
- **Secondary / Ghost:** Transparent with a hairline border, muted text, hover raises to the control surface. The "Повторить отправку" state is still the primary button: a transport failure reuses the same idempotency key and changes the label, not the action.
- **Disabled:** Sodium Muted Surface (#241a14) with Sodium Muted Ink (#a4836f), never a faded orange fill with unreadable text.

### Chips
- **Stake chips:** 28px high, 6px radius, transparent with a hairline. Selected is a sodium-tinted outline, not a filled pill.
- **League / outcome tags:** 3px radius, 18px high, uppercase 9px; outcome tags in the slip are sodium with ink text, league tags stay transparent with grey text.

### Cards / Containers
- **Corner Style:** 10px for the bet slip, match cards and history panel; 11px for dialogs.
- **Background:** Programme Surface for working panels, Raised Paper for dialogs and the toast; the stake input and source table drop to the Well.
- **Shadow Strategy:** None at rest. See Elevation.
- **Border:** 1px Hairline; selected slip gets the animated ShineBorder overlay at 60% opacity.
- **Internal Padding:** 14–16px for cards, 16px vertical / 20px horizontal for desktop match rows, 16px for slip sections.

### Inputs / Fields
- **Stake field:** 48px high, 8px radius, Well fill, strong 1px outline, Oswald 23px tabular value, fixed `₽` suffix. Quick chips 100 / 500 / 1 000 / Макс sit directly below.
- **Focus:** Sodium border and a 2px 25% sodium ring; caret is sodium.
- **Error:** Signal-red-tinted border and a 11px message under the field; the submit button states the same problem in words and stays disabled while the value is invalid.
- **Disabled:** The field dims to 50% and drops pointer events while a request is in flight or an unresolved retry owns the key. Odds buttons dim to 40% when the line is stale or the attempt is locked.

### Navigation
The header is a sticky 64px band with the brand mark at left, connection pill, demo balance and a
single player tag at right. It is not a navigation menu: a thin 1px rule and generous negative space
are enough. On mobile the player tag and tagline disappear; the balance label collapses below 360px.
The footer is a hairline with the simulation disclaimer and the brand signature.

### Odds Button (signature)
Three equal buttons per fixture, 58–62px high, label above and a 25–27px tabular price below, with a
small trend arrow in the top-right. Unselected is Control Surface with a hairline; hover raises the
surface and the border. Selected is full sodium with ink text, an orange ring/glow and no ambiguity.
A price change starts one 160ms opacity flash through a Vue `<Transition>`, while a NumberTicker
rolls the value. A short green/red wash at the bottom of the card marks the direction: green for a
higher price, red for a lower one. On the selected sodium fill the direction also appears as a dark
chip around the arrow and a 3px bottom edge, so it never disappears into orange. Stale line dims
every button to 40% and removes the pointer; the bet slip explains why in text.

### Source Breakdown (signature)
An expandable hairline table under a fixture: outcome, PulseBet price, SharpLine price, and our
price. The header reads «Цены источников»; no de-vig, margin or commission language appears on the
customer surface. Source prices are muted; our price is ink. The disclosure animates in with a 200ms
fade/translate and remains legible at 320px.

### Bet Slip (signature)
A sticky 10px panel whose sections are separated by the shadcn Separator, never nested cards. Empty:
a ticket icon, «Купон пуст», and a one-line instruction. Selected: league/kickoff, fixture, outcome
tag + price, price-change notice (Vue Transition), stake field, potential payout ticker, remaining
balance, inline error and the primary action. The 409 dialog is a separate 440px surface showing the
old price, the new price, the new payout, and two explicit actions; it never auto-retries.

### Bet History (demo)
A hairline list below the line. A pending row shows «В игре», the potential payout and a small
«Демо-исход» action; resolving it changes the badge to «Выиграла» / «Проиграла», shows the actual
payout (zero on a loss) and updates the balance. A one-line note states that automatic settlement
after a match is out of scope.

### Status and Toasts
Connection is a 24px hairline pill with a 6px dot: sodium + pulse for "В линии", sodium at 70% for
"Переподключение…", red for "Нет связи". Toasts are dark raised cards, 10px radius, orange for
success, red for error, with a 1px border and the floating shadow.

## Do's and Don'ts

### Do:
- **Do** keep orange for selection, primary action, live state and price change. If a region is orange, it is a decision.
- **Do** use green and red only for price direction and resolved bet outcomes; they are signals, not decoration.
- **Do** show both source prices and the final price in the disclosure; keep margin out of the customer surface.
- **Do** use 1px hairlines and one-step tonal surfaces before reaching for a shadow.
- **Do** keep odds and money tabular, comma-decimal, and floored from integer kopecks.
- **Do** label the simulation plainly (balance chip, slip badge, footer) without turning the page into a disclaimer.
- **Do** keep the 8–12px radius range, 9–10px uppercase labels and stacked team names that can wrap on narrow desktop.

### Don't:
- **Don't** introduce a second accent, purple, blue gradients or a colored page field.
- **Don't** use glassmorphism, backdrop-blur panels or gradient text.
- **Don't** nest cards inside cards or turn every surface into a rounded pill.
- **Don't** use emoji, stock photography, decorative sparklines or icon-plus-heading-plus-text card grids.
- **Don't** animate every price with a large movement; the flash is neutral opacity, the direction lives in the trend mark.
- **Don't** let a stale or unconfirmed price look actionable: dim the odds and replace the primary label with the recovery action.
