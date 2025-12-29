# Autoclaude PRD

## Overview

Autoclaude is a TUI app that monitors tmux panes running Claude Code and automatically sends "continue" commands when rate limits reset.

## User Requirements

1. **ASCII pane layout** - Show current tmux window layout as visual boxes
2. **Spatial navigation** - Arrow keys move to pane in that direction (left arrow = pane to left)
3. **Mode setting** - Tab cycles per-pane modes: `off` and `continue on rate limit`
4. **Mode display** - Label inside pane + color coding (green=auto, gray=off)
5. **Auto-detect Claude Code** - Look for characteristic prompt pattern in pane content
6. **Rate limit monitoring** - Poll every 10 seconds, detect `limit reached ∙ resets Xam/pm`
7. **Auto-continue** - When rate limit resets, send: `Enter` → `continue` → `Enter`

## Package Structure

```
internal/
├── tui/
│   ├── tui.go       # Bubbletea model, Update/View
│   ├── styles.go    # Lipgloss styles
│   └── layout.go    # ASCII pane rendering
├── tmux/
│   ├── tmux.go      # Tmux command wrapper
│   ├── pane.go      # Pane types, spatial navigation
│   └── layout.go    # Layout parsing
└── detection/
    ├── claude.go    # Claude Code detection
    └── ratelimit.go # Rate limit pattern matching
```

## Implementation Phases

### Phase 1: Tmux Infrastructure
**Files:** `internal/tmux/tmux.go`, `internal/tmux/pane.go`, `main.go`

- Validate TMUX environment variable at startup
- `tmux list-panes -F "#{pane_id} #{pane_left} #{pane_top} #{pane_width} #{pane_height}"`
- Parse output into `Layout` struct with `[]*Pane`
- Implement spatial navigation: `PaneInDirection(current, dir)` finds nearest pane in direction

### Phase 2: ASCII Layout Rendering
**Files:** `internal/tui/layout.go`, `internal/tui/tui.go`

- Scale tmux coordinates to fit TUI viewport
- Draw pane boxes using box-drawing characters
- Selected pane: double-line border `╔═╗║╚╝`
- Unselected: single-line `┌─┐│└┘`
- Show mode label centered in each pane box

### Phase 3: Pane Selection & Mode Cycling
**Files:** `internal/tui/tui.go`

- Arrow keys call `PaneInDirection()` and update `selectedPane`
- Tab cycles `selectedPane.Mode` between `ModeOff` and `ModeContinueOnRateLimit`
- Store mode per-pane, preserve across layout refreshes

### Phase 4: Claude Code Detection
**Files:** `internal/detection/claude.go`

- Pattern: box-drawing characters + `> ` prompt line
- `IsClaudeCode(content string) bool`
- Run on pane capture, set `pane.HasClaudeCode`

### Phase 5: Rate Limit Monitoring
**Files:** `internal/detection/ratelimit.go`

- Pattern: `limit reached.*resets\s+(\d{1,2}[ap]m)`
- Parse reset time, calculate duration until reset
- `CheckRateLimit(content) RateLimitStatus`

### Phase 6: Async Polling & Auto-Continue
**Files:** `internal/tui/tui.go`

- `tea.Tick` every 10 seconds triggers `PollTickMsg`
- For each pane with `ModeContinueOnRateLimit`: capture and check status
- Track `wasRateLimited` → `!isRateLimited` transition
- Send auto-continue: `tmux send-keys -t %ID Enter continue Enter`

### Phase 7: View & Styling
**Files:** `internal/tui/styles.go`, `internal/tui/tui.go`

- Color scheme: cyan accent, green for auto mode, gray for off, red for rate-limited
- Footer: selected pane info, mode, rate limit status, help text
- Show errors for 10 seconds then clear

## Key Types

```go
type PaneMode int
const (
    ModeOff PaneMode = iota
    ModeContinueOnRateLimit
)

type Pane struct {
    ID              string
    Left, Top       int
    Width, Height   int
    Mode            PaneMode
    HasClaudeCode   bool
    IsRateLimited   bool
    RateLimitResets string
}

type Layout struct {
    Panes        []*Pane
    WindowWidth  int
    WindowHeight int
}

// Bubbletea messages
type LayoutUpdateMsg { Layout *Layout; Err error }
type PaneStatusMsg { PaneID string; HasClaudeCode bool; RateLimit RateLimitStatus }
type PollTickMsg time.Time
```

## Error Handling

- Fail fast if not in tmux (check `$TMUX` env var)
- 5-second timeout on all tmux commands
- Display errors in footer for 10 seconds
- Continue polling even after errors
