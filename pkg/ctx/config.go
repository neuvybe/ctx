package ctx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	currentSchemaVersion = 2
)

// InitOptions controls scaffold creation.
type InitOptions struct {
	Folder string
	Mode   Mode
	// Addons selects the exact add-on set. A nil slice uses the catalog
	// defaults; a non-nil empty slice creates a core-only scaffold.
	Addons []string
}

// Config is the persisted scaffold configuration.
type Config struct {
	SchemaVersion    int      `json:"schemaVersion"`
	LayoutVersion    int      `json:"layoutVersion,omitempty"`
	TemplateRevision string   `json:"templateRevision,omitempty"`
	Project          string   `json:"project,omitempty"`
	Mode             Mode     `json:"mode"`
	Addons           []string `json:"addons,omitempty"`
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
	if opts.Addons == nil {
		opts.Addons = DefaultAddonIDs()
	}
	addons, err := normalizeAddonNames(opts.Addons)
	if err != nil {
		return InitOptions{}, err
	}
	opts.Addons = addons
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
	native := filepath.Clean(filepath.FromSlash(folder))
	clean := filepath.ToSlash(native)
	input := filepath.ToSlash(folder)
	if folder == "" || filepath.IsAbs(native) || filepath.VolumeName(native) != "" || clean == "." || clean == ".." || clean != input || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("invalid folder %q: use a relative path contained in the target repository", folder)
	}
	return nil
}

func parseConfig(data []byte) (Config, error) {
	var header struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return Config{}, err
	}

	switch header.SchemaVersion {
	case 1:
		var persisted struct {
			SchemaVersion int  `json:"schemaVersion"`
			Mode          Mode `json:"mode"`
		}
		if err := decodeStrictJSON(data, &persisted); err != nil {
			return Config{}, err
		}
		if _, err := ParseMode(string(persisted.Mode)); err != nil {
			return Config{}, err
		}
		return Config{
			SchemaVersion:    1,
			LayoutVersion:    LegacyLayoutVersion,
			TemplateRevision: layouts[LegacyLayoutVersion].TemplateRevision,
			Mode:             persisted.Mode,
		}, nil
	case currentSchemaVersion:
		var persisted Config
		if err := decodeStrictJSON(data, &persisted); err != nil {
			return Config{}, err
		}
		if persisted.LayoutVersion != CurrentLayoutVersion {
			return Config{}, fmt.Errorf("unsupported layoutVersion %d for schemaVersion %d", persisted.LayoutVersion, persisted.SchemaVersion)
		}
		if strings.TrimSpace(persisted.TemplateRevision) == "" {
			return Config{}, fmt.Errorf("templateRevision must not be empty")
		}
		if strings.TrimSpace(persisted.Project) == "" {
			return Config{}, fmt.Errorf("project must not be empty")
		}
		if _, err := ParseMode(string(persisted.Mode)); err != nil {
			return Config{}, err
		}
		addons, err := normalizeAddonNames(persisted.Addons)
		if err != nil {
			return Config{}, err
		}
		persisted.Addons = addons
		return persisted, nil
	default:
		return Config{}, fmt.Errorf("unsupported schemaVersion %d (this CLI supports 1 and %d)", header.SchemaVersion, currentSchemaVersion)
	}
}

func marshalConfig(cfg Config) ([]byte, error) {
	var value any
	switch cfg.SchemaVersion {
	case 1:
		if _, err := ParseMode(string(cfg.Mode)); err != nil {
			return nil, err
		}
		value = struct {
			SchemaVersion int  `json:"schemaVersion"`
			Mode          Mode `json:"mode"`
		}{SchemaVersion: 1, Mode: cfg.Mode}
	case currentSchemaVersion:
		if cfg.LayoutVersion != CurrentLayoutVersion {
			return nil, fmt.Errorf("unsupported layoutVersion %d for schemaVersion %d", cfg.LayoutVersion, cfg.SchemaVersion)
		}
		if strings.TrimSpace(cfg.TemplateRevision) == "" {
			return nil, fmt.Errorf("templateRevision must not be empty")
		}
		if strings.TrimSpace(cfg.Project) == "" {
			return nil, fmt.Errorf("project must not be empty")
		}
		if _, err := ParseMode(string(cfg.Mode)); err != nil {
			return nil, err
		}
		addons, err := normalizeAddonNames(cfg.Addons)
		if err != nil {
			return nil, err
		}
		cfg.Addons = addons
		value = cfg
	default:
		return nil, fmt.Errorf("unsupported schemaVersion %d (this CLI supports 1 and %d)", cfg.SchemaVersion, currentSchemaVersion)
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decodeStrictJSON(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func writeScaffoldConfig(dest string, cfg Config) error {
	data, err := marshalConfig(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, configFileName), data, 0o644)
}

// writeConfig preserves the schema-1 writer while the v1 layout remains
// supported. New layout creation uses writeScaffoldConfig with an explicit
// schema-2 descriptor.
func writeConfig(dest string, mode Mode) error {
	return writeScaffoldConfig(dest, Config{SchemaVersion: 1, Mode: mode})
}

func loadScaffoldState(dest string) (scaffoldState, error) {
	configPath := filepath.Join(dest, configFileName)
	if _, err := os.Lstat(configPath); os.IsNotExist(err) {
		if _, localErr := os.Lstat(filepath.Join(dest, "local", "CONTINUE.md")); localErr == nil {
			return scaffoldState{}, fmt.Errorf("missing %s in a new-layout scaffold", configFileName)
		} else if !os.IsNotExist(localErr) {
			return scaffoldState{}, fmt.Errorf("inspect local continuation: %w", localErr)
		}
		if _, legacyErr := inspectDoctorFile(dest, "CONTINUE.md"); legacyErr != nil {
			if os.IsNotExist(legacyErr) {
				return scaffoldState{}, fmt.Errorf("missing %s and no legacy CONTINUE.md; cannot determine scaffold mode", configFileName)
			}
			return scaffoldState{}, fmt.Errorf("inspect legacy continuation: %w", legacyErr)
		}
		// Scaffolds produced before schema 1 were always local and kept the living
		// continuation file at the scaffold root.
		return scaffoldState{Config: Config{
			LayoutVersion:    LegacyLayoutVersion,
			TemplateRevision: layouts[LegacyLayoutVersion].TemplateRevision,
			Mode:             ModeLocal,
		}, Legacy: true}, nil
	} else if err != nil {
		return scaffoldState{}, fmt.Errorf("read %s: %w", configFileName, err)
	}
	data, err := inspectDoctorFile(dest, configFileName)
	if err != nil {
		return scaffoldState{}, fmt.Errorf("read %s: %w", configFileName, err)
	}
	cfg, err := parseConfig(data)
	if err != nil {
		return scaffoldState{}, fmt.Errorf("parse %s: %w", configFileName, err)
	}
	return scaffoldState{Config: cfg}, nil
}

func (s scaffoldState) layoutVersion() int {
	if s.Legacy || s.Config.LayoutVersion == 0 {
		return LegacyLayoutVersion
	}
	return s.Config.LayoutVersion
}

func (s scaffoldState) templateRevision() string {
	if s.Config.TemplateRevision != "" {
		return s.Config.TemplateRevision
	}
	if layout, ok := LayoutForVersion(s.layoutVersion()); ok {
		return layout.TemplateRevision
	}
	return ""
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
