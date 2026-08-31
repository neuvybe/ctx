package ctx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Mode controls which scaffold files are visible to Git.
type Mode string

const (
	// ModeTeam keeps durable context visible to Git while local session state is
	// ignored by the scaffold's own .gitignore.
	ModeTeam Mode = "team"
	// ModeLocal excludes the entire scaffold through the repository-local Git
	// exclude file.
	ModeLocal Mode = "local"

	configFileName       = "config.json"
	currentSchemaVersion = 1
)

// InitOptions controls scaffold creation.
type InitOptions struct {
	Folder string
	Mode   Mode
}

// Config is the persisted scaffold configuration.
type Config struct {
	SchemaVersion int  `json:"schemaVersion"`
	Mode          Mode `json:"mode"`
}

type scaffoldState struct {
	Config Config
	Legacy bool
}

var safeFolderName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ParseMode validates a user-facing mode value.
func ParseMode(value string) (Mode, error) {
	mode := Mode(value)
	switch mode {
	case ModeTeam, ModeLocal:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid mode %q (want %q or %q)", value, ModeTeam, ModeLocal)
	}
}

func normalizeInitOptions(opts InitOptions) (InitOptions, error) {
	if opts.Folder == "" {
		opts.Folder = ".ctx"
	}
	if opts.Mode == "" {
		opts.Mode = ModeTeam
	}
	if err := validateFolderName(opts.Folder); err != nil {
		return InitOptions{}, err
	}
	if _, err := ParseMode(string(opts.Mode)); err != nil {
		return InitOptions{}, err
	}
	return opts, nil
}

func validateFolderName(folder string) error {
	if folder == "" || folder == "." || folder == ".." || filepath.Base(folder) != folder || !safeFolderName.MatchString(folder) {
		return fmt.Errorf("invalid folder %q: use a single directory name containing only letters, digits, '.', '_', or '-'", folder)
	}
	return nil
}

// validateExistingFolderPath keeps Update and Doctor compatible with older
// custom folders while preventing absolute paths and traversal outside the
// repository. New initialization deliberately uses the stricter validator.
func validateExistingFolderPath(folder string) error {
	clean := filepath.Clean(folder)
	if folder == "" || filepath.IsAbs(folder) || clean == "." || clean == ".." || clean != folder || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid folder %q: use a relative path contained in the target repository", folder)
	}
	return nil
}

func writeConfig(dest string, mode Mode) error {
	data, err := json.MarshalIndent(Config{SchemaVersion: currentSchemaVersion, Mode: mode}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dest, configFileName), data, 0o644)
}

func loadScaffoldState(dest string) (scaffoldState, error) {
	data, err := os.ReadFile(filepath.Join(dest, configFileName))
	if os.IsNotExist(err) {
		if _, localErr := os.Lstat(filepath.Join(dest, "local", "CONTINUE.md")); localErr == nil {
			return scaffoldState{}, fmt.Errorf("missing %s in a new-layout scaffold", configFileName)
		} else if !os.IsNotExist(localErr) {
			return scaffoldState{}, fmt.Errorf("inspect local continuation: %w", localErr)
		}
		if info, legacyErr := os.Stat(filepath.Join(dest, "CONTINUE.md")); legacyErr != nil {
			if os.IsNotExist(legacyErr) {
				return scaffoldState{}, fmt.Errorf("missing %s and no legacy CONTINUE.md; cannot determine scaffold mode", configFileName)
			}
			return scaffoldState{}, fmt.Errorf("inspect legacy continuation: %w", legacyErr)
		} else if !info.Mode().IsRegular() {
			return scaffoldState{}, fmt.Errorf("missing %s and legacy CONTINUE.md is not a regular file", configFileName)
		}
		// Scaffolds produced before schema 1 were always local and kept the living
		// continuation file at the scaffold root.
		return scaffoldState{Config: Config{Mode: ModeLocal}, Legacy: true}, nil
	}
	if err != nil {
		return scaffoldState{}, fmt.Errorf("read %s: %w", configFileName, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return scaffoldState{}, fmt.Errorf("parse %s: %w", configFileName, err)
	}
	if cfg.SchemaVersion != currentSchemaVersion {
		return scaffoldState{}, fmt.Errorf("unsupported schemaVersion %d in %s (this CLI supports %d)", cfg.SchemaVersion, configFileName, currentSchemaVersion)
	}
	if _, err := ParseMode(string(cfg.Mode)); err != nil {
		return scaffoldState{}, fmt.Errorf("parse %s: %w", configFileName, err)
	}
	return scaffoldState{Config: cfg}, nil
}

func (s scaffoldState) continuePath() string {
	if s.Legacy {
		return "CONTINUE.md"
	}
	return filepath.ToSlash(filepath.Join("local", "CONTINUE.md"))
}

func (s scaffoldState) modeLabel() string {
	if s.Legacy {
		return "local (legacy)"
	}
	return string(s.Config.Mode)
}
