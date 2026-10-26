# Giode feature requests from Pocket Agent

Pocket Agent is built entirely on Giode (`github.com/mehran9675/giode`). This document lists
what the app needed that Giode doesn't provide yet, why, how the app works around it today,
and a proposed API. Each entry names the Pocket Agent file holding the workaround, so it can be
deleted once Giode ships the feature.

Priorities: **P0** blocks a polished mobile app, **P1** is needed for parity with desktop AI
apps, **P2** is polish.

| # | Request | Priority | Workaround in Pocket Agent |
| --- | --- | --- | --- |
| 1 | [`MaxWidth` on expanding children](#1-maxwidth-should-cap-expanding-children) | P0 | `internal/ui/widgets/centered.go` |
| 2 | [Rich text element](#2-richtext-element) | P0 | `internal/ui/markdown/inline.go` (gioui.org/x/richtext via `Raw`) |
| 3 | [Selectable text](#3-selectable-text) | P0 | Copy buttons only |
| 4 | [Live setters on stateful components](#4-live-setters-on-stateful-components) | P0 | Theme tokens; composer draws its own placeholder |
| 5 | [Picked file metadata](#5-picked-file-metadata-and-a-photo-picker) | P0 | `fileName()` in `internal/ui/attachments.go` |
| 6 | [Open URL](#6-openurl) | P0 | `internal/platform/openurl_*.go` |
| 7 | [Secure storage](#7-secure-storage) | P0 | Keys in the app-private settings file |
| 8 | [Scroll: anchoring, fit-content, horizontal](#8-scroll-improvements) | P0 | `chat_view.go`, `fit_scroll.go` |
| 9 | [Sheet component](#9-sheet-component) | P1 | `internal/ui/sheet.go`, `scrim.go` |
| 10 | [Multiline input behaviour](#10-multiline-input-behaviour) | P1 | Send button only |
| 11 | [Switch, SegmentedControl, Chip, Badge](#11-switch-segmentedcontrol-chip-badge) | P1 | `internal/ui/widgets/` |
| 12 | [Background execution](#12-background-execution) | P1 | None - runs stop when the OS suspends the app |
| 13 | [iOS parity](#13-ios-parity) | P1 | Share falls back to the clipboard on iOS |
| 14 | [Long-press and swipe gestures](#14-long-press-and-swipe-gestures) | P1 | Overflow (⋯) buttons |
| 15 | [Keyboard and safe-area insets](#15-keyboard-and-safe-area-insets) | P1 | Relies on `adjustResize` (Android) |
| 16 | [Color compositing / `Mix`](#16-color-compositing) | P1 | `internal/ui/theme/mix.go` |
| 17 | [System theme events](#17-system-theme-and-device-info) | P2 | Polls `x/pref/theme` every frame |
| 18 | [Device info](#17-system-theme-and-device-info) | P2 | `internal/local/memory_*.go` |
| 19 | [Haptics and notifications](#18-haptics-and-local-notifications) | P2 | None |
| 20 | [Text details: SemiBold, tabs](#19-text-details) | P2 | `widgets/codetext.go`; SemiBold registered as Bold |
| 21 | [Drawer layout](#20-drawer-should-not-take-layout-space) | P2 | Absolute wrapper in `shell.go` |
| 22 | [Voice input](#21-voice-input) | P2 | None |

---

## 1. `MaxWidth` should cap expanding children

**Problem.** A `Width: -1` (or `FlexGrow`) child of a row is laid out by Gio's `Flexed`, which
sets `Min.X = Max.X` to the allotted space. `applyMinMax` clamps `Max.X` to `MaxWidth` but then
raises it back to `Min.X`, so `MaxWidth` has no effect. The common "readable column" layout -
CSS `width: 100%; max-width: 760px; margin: 0 auto` - is impossible with Styles alone.

**Proposal.** When `MaxWidth` is set, clamp `Min.X` as well, and honour the parent's `Justify` to
place the narrower child. Same for `MaxHeight` on the vertical axis.

```go
giode.Box(giode.Styles{Width: -1, Justify: giode.JustifyCenter},
	giode.Stack(giode.Styles{Width: -1, MaxWidth: 760}, messages...)) // 760dp, centered
```

## 2. `RichText` element

**Problem.** `Text` renders one style per string. Chat apps need mixed inline styles - bold,
italic, inline code in a monospace font, strikethrough, links - in one wrapping paragraph.

**Proposal.** A stateless element over `gioui.org/x/richtext`, with tappable links:

```go
giode.RichText(giode.Styles{FontSize: 15, LineHeight: 22},
	giode.Span("Run "),
	giode.Span("go test", giode.Styles{FontFamily: "JetBrains Mono", Color: "theme.accent"}),
	giode.Span(" or read the "),
	giode.Link("docs", "https://…"),
).OnLink(func(url string) { … })
```

A `giode.Markdown(src, opts)` element built on it (headings, lists, task lists, code blocks with a
copy button, tables, quotes) would cover most of `internal/ui/markdown`.

## 3. Selectable text

**Problem.** Users expect to long-press a reply and select part of it. Giode's `Text` isn't
selectable; `Input.ReadOnly` is a field with a border and doesn't size to its content.

**Proposal.** `giode.Text(s).Selectable(true)` and `RichText(...).Selectable(true)`, using
`widget.Selectable`, with the platform selection handles and a Copy action on touch screens.

## 4. Live setters on stateful components

**Problem.** `Input`, `Scroll`, `Drawer` and `Pressable` keep the `Styles` they were created
with. A theme switch or a mode change (a different placeholder for "Chat" vs "Deep research")
can't be applied without recreating the component and losing its state. Pocket Agent uses
`"theme.*"` tokens where it can, but app tokens outside the Palette (a sidebar color) can't
follow, and the composer draws its own placeholder in an absolutely positioned `Text`.

**Proposal.** Chainable setters that apply on the next frame: `Input.Styles(st)`,
`Input.Placeholder(s)`, `Scroll.Styles(st)`, `Drawer.Styles(st)`, `Pressable.Styles(st)`.

## 5. Picked file metadata, and a photo picker

**Problem.** `OpenFile`/`OpenFiles` return bare `io.ReadCloser`s. On Android and iOS there is no
way to learn the file name, MIME type or size - an attachment chip can only say "Document 1",
and type detection falls back to sniffing bytes. iPhone photos arrive as HEIC, which Go can't
decode.

**Proposal.**

```go
type PickedFile struct {
	io.ReadCloser
	Name string // display name ("Q3 report.pdf")
	MIME string // "application/pdf"
	Size int64  // -1 if unknown
}
files, err := app.PickFiles(ctx, giode.PickOptions{Types: []string{"image/*", "application/pdf"}, Multiple: true})
img, err := app.PickPhoto(ctx, giode.PhotoOptions{Source: giode.PhotoLibrary /* or Camera */, MaxSide: 2048, Format: "jpeg"})
```

`PickPhoto` should use PHPicker (iOS) and the Android photo picker, and return JPEG so HEIC
never reaches the app. A `ReadClipboardImage` would round this out (paste a screenshot).

## 6. `OpenURL`

**Problem.** Links in replies and source citations must open in the browser. Gio has no API
for it. Pocket Agent ships JNI (`ACTION_VIEW` intent), Objective-C (`UIApplication openURL`) and
`xdg-open`/`open`/`rundll32` implementations in `internal/platform`.

**Proposal.** `app.OpenURL(ctx, url) error` on every platform, optionally
`OpenURLInApp` (SFSafariViewController / Custom Tabs) for reading sources without leaving the app.

## 7. Secure storage

**Problem.** API keys belong in the iOS Keychain and the Android Keystore. Giode has no
access to either, so keys live in the app's private data directory (sandboxed, but included in
unencrypted backups).

**Proposal.**

```go
giode.SecureStore.Set("anthropic.key", []byte(key))
key, err := giode.SecureStore.Get("anthropic.key")
giode.SecureStore.Delete("anthropic.key")
```

Keychain with `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly` on iOS/macOS,
EncryptedSharedPreferences/Keystore on Android, libsecret on Linux, DPAPI/Credential Manager on
Windows.

## 8. Scroll improvements

**Problems.**

- **Bottom anchoring.** A chat must stay pinned to the bottom while a reply streams, and stop
  following once the user scrolls up. `chat_view.go` emulates it by comparing `ContentHeight`
  between frames and calling `ScrollTo(1e9)`.
- **Fit content.** `Scroll` always fills its maximum constraints, so a bottom sheet with three
  rows would be as tall as the screen. `fit_scroll.go` sizes the viewport from last frame's
  `ContentHeight` (one frame late).
- **Horizontal scrolling.** Code blocks and wide tables need horizontal scroll on phones; today
  code wraps and tables squeeze.

**Proposal.** `Scroll(...).Anchor(giode.AnchorBottom)`, `Scroll(...).FitContent(true)`,
`Scroll(...).Axis(giode.Horizontal | giode.Vertical)`, plus `OnScroll(func(offset, max float32))`.

## 9. Sheet component

**Problem.** Phones use bottom sheets for pickers and menus; wide screens use centered
dialogs. `Select` has a sheet internally but it isn't reusable, and `Dialog` is centered only.
Pocket Agent builds its own with `RegisterOverlay` (`sheet.go`), and had to learn that an
absolutely positioned scrim paints over flow content (both layers must be absolute).

**Proposal.** `giode.Sheet(open *bool, content func() Element, st ...Styles)` with `Title`,
`Footer` (pinned below the scrolling body), `Detents` (medium/large), drag-to-dismiss, slide
animation, `Back`/Escape handling, and `.Adaptive(true)` to become a centered dialog above a
breakpoint.

## 10. Multiline input behaviour

**Problem.** Desktop and iPad-with-keyboard users expect Enter to send and Shift+Enter to
insert a newline in a chat composer. A `Multiline` input inserts a newline on Enter and never
calls `Submit`. There is also no auto-growing field with min/max lines.

**Proposal.** `Input.SubmitOn(giode.EnterUnlessShift)`, `Input.Lines(min, max int)` (grows with
content, scrolls past max), `Input.OnFocus/OnBlur`, `Input.SelectAll()`, `Input.InsertText(s)`.

## 11. Switch, SegmentedControl, Chip, Badge

**Problem.** These are standard on iOS and Android and every settings screen needs them.
Pocket Agent implements them in `internal/ui/widgets` (`toggle.go`, `segmented.go`, `chip.go`,
`badge.go`, `avatar.go`).

**Proposal.** `Input.Switch(label)` with an animated knob and `OnChange`,
`Input.Segmented(options...)` (icons optional), and stateless `Chip`, `Badge`, `Avatar`.

## 12. Background execution

**Problem.** Agent runs, deep research and model downloads take minutes. When the user locks
the phone, Android freezes the app and iOS suspends it, and the work stops.

**Proposal.**

- `app.KeepAlive(ctx, reason string) (release func())`: an Android foreground service with an
  ongoing notification ("Researching…") and a partial wake lock; on iOS
  `beginBackgroundTask` (≈30 s grace) and, for downloads, a background `URLSession`.
- `app.KeepScreenOn(bool)` for on-device generation.

## 13. iOS parity

Giode's docs note these are Android-only today; Pocket Agent needs them on iPhone and iPad:

- `Share` (UIActivityViewController with a popover anchor on iPad) - falls back to the
  clipboard now.
- `OnBackground`/`OnForeground` (UIApplication notifications) - the app saves on these.
- Status bar style following the theme (light/dark content).
- `OnFileDrop` for iPad drag and drop from Files and other apps.

## 14. Long-press and swipe gestures

**Problem.** Touch apps open context actions with a long press and expose delete/pin with
swipe actions. `Pressable.OnSecondaryClick` only fires for a mouse right-click.

**Proposal.** `Pressable.OnLongPress(func(ClickEvent))` (with haptic feedback), and
`giode.Swipeable(content, leading, trailing []SwipeAction)` for list rows.

## 15. Keyboard and safe-area insets

**Problem.** The composer must sit right above the on-screen keyboard, and edge-to-edge
layouts need the safe-area insets (notch, home indicator) to pad content rather than the
whole window.

**Proposal.** `giode.Insets() (safeArea, keyboard Inset)` readable during layout, a
`Config.EdgeToEdge` option, and `app.HideKeyboard()`.

## 16. Color compositing

**Problem.** Gio blends translucent colors in linear light. A 5% white hover overlay
(`#ffffff0d`), which is subtle in CSS, comes out several times brighter: over `#101115` it
renders as `#434344` instead of about `#1c1d21`, so web-style tokens look wrong. Pocket Agent precomputes opaque tints (`theme.Mix`).

**Proposal.** Document the behaviour, and add `giode.Mix(fg, bg string, t float64) string`
(sRGB-space blend) and/or a `Palette` with derived `Hover`/`Pressed`/`Soft` shades so components
don't need alpha overlays.

## 17. System theme and device info

**Problems.** "Follow the system" theme needs a change notification; Pocket Agent polls
`gioui.org/x/pref/theme` every frame. Choosing which on-device model to recommend needs the
device's RAM and form factor; Pocket Agent reads `/proc/meminfo` and `sysctl hw.memsize`.

**Proposal.** `Config.OnSystemThemeChange(func(dark bool))` or a `ModeSystem` that follows the
platform automatically, and `giode.Device() DeviceInfo{TotalRAM, Class (phone/tablet/desktop),
OS, OSVersion, Model}`.

## 18. Haptics and local notifications

**Proposal.** `giode.Haptic(giode.HapticLight | HapticSuccess | HapticWarning)` (wrapping
`gioui.org/x/haptic`, which needs the view) for send, approve and long-press; and
`app.Notify(title, body string)` (wrapping `gioui.org/x/notify`) to tell the user an agent run
or a model download finished while the app was in the background.

## 19. Text details

- `FontWeightSemiBold` (600) - Inter SemiBold is the natural weight for UI headings; the app
  registers it as Bold.
- Tab characters render as missing-glyph boxes; code needs tab stops (`TabWidth` style).
  Pocket Agent replaces tabs with spaces (`widgets/codetext.go`).

## 20. Drawer should not take layout space

**Problem.** Laid out as a child of a column (as the docs suggest, "last so it renders above"),
`Drawer` takes the column's space even while closed, squashing the page. Pocket Agent wraps it
in an absolutely positioned box. The drawer should register itself as an overlay
(`RegisterOverlay`) and take no space, like `Toaster`. Swipe from the edge to open would be
welcome too.

## 21. Voice input

**Proposal.** `app.Dictate(ctx, lang) (<-chan string, error)` using `SFSpeechRecognizer` and
Android `SpeechRecognizer` for a microphone button in the composer, and access to raw audio
recording (with the microphone permission prompt) for on-device Whisper later.
