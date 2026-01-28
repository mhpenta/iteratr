# Config Management

Centralized configuration using Viper with `iteratr setup` command for first-time initialization.

## Overview

Replace scattered config handling with Viper-managed layered configuration. New `iteratr setup` TUI command creates initial config. Config required before first `iteratr build`.

## User Story

**As a** developer using iteratr  
**I want** a single config file with sensible defaults  
**So that** I don't have to pass CLI flags repeatedly and can share project settings with my team

## Requirements

### Functional

1. **Config locations**: `<PROJECT>/iteratr.yml` (project) and `$XDG_CONFIG_HOME/iteratr/iteratr.yml` (global, defaults to `~/.config`)
2. **Precedence**: CLI flags > ENV vars > project config > XDG global config > defaults
3. **Setup command**: `iteratr setup` creates config via TUI wizard
4. **Setup --project flag**: Creates config in current directory instead of XDG
5. **Required config**: `iteratr build` errors if no config exists AND no `ITERATR_MODEL` env var
6. **Required field**: `model` must be non-empty (from config or env var)
7. **Deprecation**: Remove `.iteratr.template` auto-detection, use `template` key in config instead

### Non-Functional

1. Setup runs as BubbleTea program (consistent with build wizard)
2. Viper handles all config loading and merging
3. ENV vars prefixed with `ITERATR_`

## Config Schema

```yaml
# iteratr.yml
model: ""              # required, no default
auto_commit: true      # auto-commit after iterations
data_dir: .iteratr     # NATS/session storage
log_level: info        # debug, info, warn, error
log_file: ""           # empty = no file logging
iterations: 0          # 0 = infinite
headless: false        # run without TUI
template: ""           # path to template file, empty = embedded default
```

## ENV Var Mapping

| Config Key | ENV Var | Type |
|------------|---------|------|
| `model` | `ITERATR_MODEL` | string |
| `auto_commit` | `ITERATR_AUTO_COMMIT` | bool |
| `data_dir` | `ITERATR_DATA_DIR` | string |
| `log_level` | `ITERATR_LOG_LEVEL` | string |
| `log_file` | `ITERATR_LOG_FILE` | string |
| `iterations` | `ITERATR_ITERATIONS` | int |
| `headless` | `ITERATR_HEADLESS` | bool |
| `template` | `ITERATR_TEMPLATE` | string |

## btca Resources

Query these for implementation guidance:

| Resource | Usage |
|----------|-------|
| `viper` | Config loading, env binding, precedence |
| `bubbleteaV2` | Setup TUI program structure |
| `bubbles` | Textinput for model search, list for selections |

```bash
btca ask -r viper -q "How do I set up layered config with file + env vars + defaults?"
btca ask -r viper -q "How do I write config back to a file?"
```

## Technical Implementation

### Package: `internal/config/`

**config.go** - Viper setup and loading:

```go
type Config struct {
    Model      string `mapstructure:"model"`
    AutoCommit bool   `mapstructure:"auto_commit"`
    DataDir    string `mapstructure:"data_dir"`
    LogLevel   string `mapstructure:"log_level"`
    LogFile    string `mapstructure:"log_file"`
    Iterations int    `mapstructure:"iterations"`
    Headless   bool   `mapstructure:"headless"`
    Template   string `mapstructure:"template"`
}

func Load() (*Config, error)           // Load with full precedence
func Exists() bool                     // Check if any config exists
func GlobalPath() string               // ~/.config/iteratr/iteratr.yml
func ProjectPath() string              // ./iteratr.yml
func WriteGlobal(cfg *Config) error    // Write to XDG location
func WriteProject(cfg *Config) error   // Write to project location
```

**Viper setup:**
```go
func Load() (*Config, error) {
    v := viper.New()
    v.SetConfigType("yaml")
    
    // Defaults (except model - required)
    v.SetDefault("auto_commit", true)
    v.SetDefault("data_dir", ".iteratr")
    v.SetDefault("log_level", "info")
    v.SetDefault("log_file", "")
    v.SetDefault("iterations", 0)
    v.SetDefault("headless", false)
    v.SetDefault("template", "")
    
    // ENV binding
    v.SetEnvPrefix("ITERATR")
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    // Load global config first (if exists)
    globalPath := GlobalPath()
    if fileExists(globalPath) {
        v.SetConfigFile(globalPath)
        if err := v.ReadInConfig(); err != nil {
            return nil, fmt.Errorf("reading global config: %w", err)
        }
    }
    
    // Merge project config on top (if exists)
    projectPath := ProjectPath()
    if fileExists(projectPath) {
        v.SetConfigFile(projectPath)
        if err := v.MergeInConfig(); err != nil {
            return nil, fmt.Errorf("merging project config: %w", err)
        }
    }
    
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshaling config: %w", err)
    }
    return &cfg, nil
}

func GlobalPath() string {
    if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
        return filepath.Join(xdg, "iteratr", "iteratr.yml")
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "iteratr", "iteratr.yml")
}
```

### Command: `iteratr setup`

**Flags:**
- `--project` / `-p`: Create config in current directory instead of XDG
- `--force` / `-f`: Overwrite existing config

**Behavior:**
1. Check if config already exists at target location
2. If exists and no `--force`, error with message
3. Launch TUI wizard
4. Write config to target location
5. Print success message and exit

### Setup TUI Wizard

Two steps, reuses patterns from build wizard.

**Step 1: Model Selection**
- Fetch models via `opencode models`
- Fuzzy filter list
- Allow custom model entry

**Step 2: Auto-Commit**
- Yes/No selection
- "Yes (recommended)" as default

### Build Command Changes

**Validation on `iteratr build`:**
```go
func runBuild(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    // Model required - can come from config file or ITERATR_MODEL env var
    if cfg.Model == "" {
        if !config.Exists() {
            return fmt.Errorf("no configuration found - run 'iteratr setup' first")
        }
        return fmt.Errorf("model not configured - run 'iteratr setup' or set ITERATR_MODEL")
    }
    
    // Merge CLI flags over config values
    // ...
}
```

**CLI flag precedence:**
- CLI flags override config values when explicitly set
- Use `cmd.Flags().Changed("flag")` to detect explicit CLI usage

### Migration: Deprecate .iteratr.template Auto-Detection

Only the auto-detection is deprecated. Users can still use `.iteratr.template` by setting `template: .iteratr.template` in config.

1. Remove fallback logic in `cmd/iteratr/build.go` that auto-detects `.iteratr.template`
2. Update `internal/template/template.go` to only use config `template` path
3. Add deprecation warning if `.iteratr.template` exists but `template` config is empty
4. `gen-template` command unchanged - still outputs to `.iteratr.template` by default

### Build Wizard Integration

Build wizard (launched when `iteratr build` has no `--spec`) keeps all 4 steps. Model step pre-fills from config:

```go
// In model_selector.go Init()
func (m *ModelSelectorStep) Init() tea.Cmd {
    // Pre-select config model if set
    if cfg, err := config.Load(); err == nil && cfg.Model != "" {
        m.preselectedModel = cfg.Model
    }
    return m.fetchModels()
}
```

User can override model for this session; doesn't modify config file.

### Tool Commands Integration

Tool subcommands (`task-add`, `note-add`, etc.) should use `config.Load()` for `data_dir`:

```go
// In tool.go
cfg, _ := config.Load()  // Ignore error, fall back to flag/env
dataDir := cfg.DataDir
if flagDataDir != "" {
    dataDir = flagDataDir  // CLI flag still overrides
}
```

## Tasks

### Tracer Bullet: Minimal E2E Config Flow

Goal: Prove Viper integration works with setup command writing config and build command reading it.

- [ ] Create `internal/config/config.go` with `Config` struct and `Load()` function
- [ ] Add hardcoded model default in Load() (temporary, for tracer bullet only)
- [ ] Add `Exists()`, `GlobalPath()`, `WriteGlobal()`
- [ ] Create `cmd/iteratr/setup.go` - writes hardcoded config (no TUI yet)
- [ ] Modify `cmd/iteratr/build.go` - load config via Viper, error if missing
- [ ] Test: `iteratr setup` creates file, `iteratr build` reads model from it

### 1. Config Package Foundation

- [ ] Remove hardcoded model default, make `model` required
- [ ] Add `ProjectPath()` and `WriteProject()`
- [ ] Add validation: `Validate()` checks model non-empty
- [ ] Add config merge logic for CLI flag precedence
- [ ] Write unit tests for Load/Exists/Write functions

### 2. Setup Command Scaffold

- [ ] Create `cmd/iteratr/setup.go` with cobra command
- [ ] Add `--project` flag to write to current directory
- [ ] Add `--force` flag to overwrite existing
- [ ] Check existing config, error without `--force`
- [ ] Wire command into root

### 3. Setup TUI - Model Step

- [ ] Create `internal/tui/setup/setup.go` with `SetupModel` struct
- [ ] Create `internal/tui/setup/model_step.go`
- [ ] Reuse model fetching logic from build wizard (`opencode models`)
- [ ] Add fuzzy filter with textinput
- [ ] Add custom model entry option
- [ ] Handle loading/error states

### 4. Setup TUI - Auto-Commit Step

- [ ] Create `internal/tui/setup/autocommit_step.go`
- [ ] Simple Yes/No selection list
- [ ] "Yes (recommended)" styling

### 5. Setup TUI - Navigation and Completion

- [ ] Add step navigation (Back/Next)
- [ ] Add completion screen with file path
- [ ] Wire `RunSetup()` into setup command
- [ ] Pass `--project` flag to determine write location

### 6. Build Command Integration

- [ ] Replace direct env/flag reads with `config.Load()`
- [ ] Add config existence check with setup prompt
- [ ] Add model validation check
- [ ] Implement CLI flag override logic using `cmd.Flags().Changed()`
- [ ] Remove `DefaultModel` constant

### 7. Template Deprecation

- [ ] Remove `.iteratr.template` fallback in build.go
- [ ] Update template loading to use only config `template` path
- [ ] Add warning if `.iteratr.template` file exists in project
- [ ] Update documentation

### 8. Tool Commands Integration

- [ ] Update `cmd/iteratr/tool.go` to use `config.Load()` for data_dir
- [ ] Maintain CLI flag override for backward compatibility
- [ ] Test tool commands with config file present

### 9. Build Wizard Update

- [ ] Update `internal/tui/wizard/model_selector.go` to pre-fill from config
- [ ] Ensure wizard selection overrides config for session only
- [ ] Test wizard flow with existing config

### 10. Cleanup and Polish

- [ ] Remove old env var reads scattered in codebase
- [ ] Update `--help` text to reference config file
- [ ] Add `iteratr config` command to print current config (nice-to-have)
- [ ] Update AGENTS.md with new config info

## UI Mockups

### Setup Step 1: Model Selection

```
+------------------------------------------+
|                                          |
|  iteratr setup                           |
|                                          |
|  Select your preferred model:            |
|                                          |
|  Search: claude█                         |
|                                          |
|  > anthropic/claude-sonnet-4-5           |
|    anthropic/claude-opus-4               |
|    openrouter/anthropic/claude-3.5       |
|    [ Enter custom model ]                |
|                                          |
|  up/down navigate | enter select         |
|                                          |
+------------------------------------------+
```

### Setup Step 2: Auto-Commit

```
+------------------------------------------+
|                                          |
|  iteratr setup                           |
|                                          |
|  Auto-commit changes after iteration?    |
|                                          |
|  > Yes (recommended)                     |
|    No                                    |
|                                          |
|  up/down navigate | enter select         |
|                                          |
+------------------------------------------+
```

### Setup Complete

```
+------------------------------------------+
|                                          |
|  iteratr setup                           |
|                                          |
|  Config written to:                      |
|  /home/user/.config/iteratr/iteratr.yml  |
|                                          |
|  Run 'iteratr build' to get started.     |
|                                          |
|  press any key to exit                   |
|                                          |
+------------------------------------------+
```

### Build Error (No Config, No ENV)

```
$ iteratr build -s ./specs/feature.md

Error: no configuration found

Run 'iteratr setup' to create a config file, or set ITERATR_MODEL env var.
```

## Gotchas

### 1. Viper Global State

Use `viper.New()` instead of package-level functions to avoid global state issues in tests and concurrent usage.

### 2. Directory Creation

`WriteGlobal()` must create parent directory before writing:
```go
func WriteGlobal(cfg *Config) error {
    path := GlobalPath()
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return err
    }
    // ... write file
}
```

### 3. Config Existence vs Validity

`Exists()` returns true if any config file exists. `Load()` may still fail if file is malformed. `Validate()` checks required fields after successful load.

**Solution:** Always call in order: `Load()` then `Validate()`. Don't rely on `Exists()` alone.
```go
cfg, err := config.Load()
if err != nil {
    return err  // File malformed or read error
}
if err := cfg.Validate(); err != nil {
    return err  // Missing required fields
}
```

### 4. ENV Vars Override Both Files

ENV vars take precedence over both project and global config. User sets `ITERATR_MODEL=x` and it wins regardless of config files.

### 5. Bool ENV Vars

Viper parses bool env vars as strings. Need explicit binding for reliable parsing.

**Solution:** Use `BindEnv` with explicit key mapping:
```go
v.BindEnv("auto_commit", "ITERATR_AUTO_COMMIT")
v.SetDefault("auto_commit", true)
```
Viper handles `true`, `false`, `1`, `0`, `yes`, `no` automatically when bound this way. Document accepted values in help text.

### 6. ENV-Only Mode (CI/CD)

Config file not required if `ITERATR_MODEL` env var is set. Useful for CI/CD where config comes entirely from environment. All other values use defaults.

## Out of Scope

- Config profiles (`--profile=fast`)
- Config validation command
- Interactive config editor
- Config export/import
- Encrypted secrets in config
- Remote config sources
- Schema versioning/migrations

## Open Questions

None - all resolved in discussion.
