package metrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub/protocol"
)

const (
	SchemaConfigV1       = "mfh.metrics.config.v1"
	SchemaConfigUpdateV1 = "mfh.metrics.config-update.v1"

	ResourceConfig       = "metrics/config"
	ResourceConfigUpdate = "metrics/config/update"
)

type Name string

const (
	BatteryPercent    Name = "battery_percent"
	BatteryCharging   Name = "battery_charging"
	BatteryOnAC       Name = "battery_on_ac"
	NetworkOnline     Name = "network_online"
	NetworkType       Name = "network_type"
	CPUPercent        Name = "cpu_percent"
	MemoryPercent     Name = "memory_percent"
	VolumePercent     Name = "volume_percent"
	VolumeMuted       Name = "volume_muted"
	BrightnessPercent Name = "brightness_percent"
)

type Definition struct {
	Name         Name
	Unit         string
	Controllable bool
	Platforms    map[string]bool
	IntervalMS   int64
}

var definitions = []Definition{
	{Name: BatteryPercent, Unit: "percent", Platforms: platforms("windows"), IntervalMS: 30_000},
	{Name: BatteryCharging, Unit: "boolean", Platforms: platforms("windows"), IntervalMS: 30_000},
	{Name: BatteryOnAC, Unit: "boolean", Platforms: platforms("windows"), IntervalMS: 30_000},
	{Name: NetworkOnline, Unit: "boolean", Platforms: platforms("windows"), IntervalMS: 5_000},
	{Name: NetworkType, Unit: "label", Platforms: platforms("windows"), IntervalMS: 5_000},
	{Name: CPUPercent, Unit: "percent", Platforms: platforms("windows"), IntervalMS: 2_000},
	{Name: MemoryPercent, Unit: "percent", Platforms: platforms("windows"), IntervalMS: 2_000},
	{Name: VolumePercent, Unit: "percent", Controllable: true, Platforms: platforms("windows"), IntervalMS: 1_000},
	{Name: VolumeMuted, Unit: "boolean", Controllable: true, Platforms: platforms("windows"), IntervalMS: 1_000},
	{Name: BrightnessPercent, Unit: "percent", Controllable: true, Platforms: platforms("windows"), IntervalMS: 2_000},
}

type SettingV1 struct {
	Metric     string `json:"metric"`
	Enabled    bool   `json:"enabled"`
	Writable   bool   `json:"writable"`
	IntervalMS int64  `json:"interval_ms"`
}

func (s SettingV1) Validate() error {
	definition, ok := DefinitionFor(Name(s.Metric))
	if !ok {
		return fmt.Errorf("unsupported metric %q", s.Metric)
	}
	if s.Writable && !definition.Controllable {
		return fmt.Errorf("metric %q is not controllable", s.Metric)
	}
	if s.IntervalMS < 250 || s.IntervalMS > 86_400_000 {
		return fmt.Errorf("metric %q interval must be between 250 and 86400000ms", s.Metric)
	}
	return nil
}

type ConfigV1 struct {
	Version              int         `json:"version"`
	Revision             uint64      `json:"revision"`
	Platform             string      `json:"platform"`
	Settings             []SettingV1 `json:"settings"`
	NotificationChannels []string    `json:"notification_channels"`
}

func (c ConfigV1) Validate() error {
	if c.Version != 1 || c.Revision == 0 || !validPlatform(c.Platform) {
		return errors.New("metrics config version, revision, or platform is invalid")
	}
	if len(c.Settings) == 0 || len(c.Settings) > len(definitions) {
		return errors.New("metrics config settings count is invalid")
	}
	seen := make(map[string]struct{}, len(c.Settings))
	for index, setting := range c.Settings {
		if err := setting.Validate(); err != nil {
			return fmt.Errorf("settings[%d]: %w", index, err)
		}
		definition, _ := DefinitionFor(Name(setting.Metric))
		if !definition.Platforms[c.Platform] {
			return fmt.Errorf("metric %q is not available on %s", setting.Metric, c.Platform)
		}
		if _, exists := seen[setting.Metric]; exists {
			return fmt.Errorf("duplicate metric setting %q", setting.Metric)
		}
		seen[setting.Metric] = struct{}{}
		if index > 0 && c.Settings[index-1].Metric >= setting.Metric {
			return errors.New("metrics settings must be strictly sorted by metric")
		}
	}
	return validateNotificationChannels(c.NotificationChannels)
}

type ConfigUpdateV1 struct {
	Version              int         `json:"version"`
	ExpectedRevision     uint64      `json:"expected_revision"`
	Settings             []SettingV1 `json:"settings"`
	NotificationChannels []string    `json:"notification_channels"`
}

func (u ConfigUpdateV1) Validate() error {
	if u.Version != 1 || u.ExpectedRevision == 0 || len(u.Settings) == 0 || len(u.Settings) > len(definitions) {
		return errors.New("metrics config update version, revision, or settings count is invalid")
	}
	copySettings := append([]SettingV1(nil), u.Settings...)
	sort.Slice(copySettings, func(i, j int) bool { return copySettings[i].Metric < copySettings[j].Metric })
	for index, setting := range copySettings {
		if err := setting.Validate(); err != nil {
			return fmt.Errorf("settings[%d]: %w", index, err)
		}
		if index > 0 && copySettings[index-1].Metric == setting.Metric {
			return fmt.Errorf("duplicate metric setting %q", setting.Metric)
		}
	}
	channels := append([]string(nil), u.NotificationChannels...)
	sort.Strings(channels)
	return validateNotificationChannels(channels)
}

func Definitions(platform string) []Definition {
	result := make([]Definition, 0, len(definitions))
	for _, definition := range definitions {
		if definition.Platforms[platform] {
			definition.Platforms = clonePlatforms(definition.Platforms)
			result = append(result, definition)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func DefinitionFor(name Name) (Definition, bool) {
	for _, definition := range definitions {
		if definition.Name == name {
			definition.Platforms = clonePlatforms(definition.Platforms)
			return definition, true
		}
	}
	return Definition{}, false
}

func ResourceName(name Name) string { return "metrics/" + string(name) }

func DefaultConfig(platform string) (ConfigV1, error) {
	if !validPlatform(platform) {
		return ConfigV1{}, errors.New("metrics platform is invalid")
	}
	available := Definitions(platform)
	settings := make([]SettingV1, 0, len(available))
	for _, definition := range available {
		settings = append(settings, SettingV1{
			Metric: string(definition.Name), Enabled: true, Writable: definition.Controllable, IntervalMS: definition.IntervalMS,
		})
	}
	value := ConfigV1{Version: 1, Revision: 1, Platform: platform, Settings: settings, NotificationChannels: []string{}}
	return value, value.Validate()
}

// ValueDefinition declares the actual JSON value; resource schemas remain platform-owned.
func ValueDefinition(name Name) (protocol.DataSchemaDefinition, error) {
	definition, ok := DefinitionFor(name)
	if !ok {
		return protocol.DataSchemaDefinition{}, ErrUnsupported
	}
	value := protocol.DataSchemaDefinition{Title: string(name)}
	switch definition.Unit {
	case "percent":
		minimum, maximum := 0.0, 100.0
		value.Type, value.Unit, value.Minimum, value.Maximum = "number", "%", &minimum, &maximum
		if definition.Controllable {
			value.Type = "integer"
		}
	case "boolean":
		value.Type = "boolean"
	case "label":
		value.Type = "string"
		value.Enum = []string{"cellular", "ethernet", "none", "unknown", "wifi"}
	default:
		return protocol.DataSchemaDefinition{}, ErrUnsupported
	}
	return value, nil
}

func ValidateValue(name Name, value any) error {
	definition, err := ValueDefinition(name)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode metric value: %w", err)
	}
	return protocol.ValidateVariableValue(definition, payload)
}

func ConfigDefinition() protocol.DataSchemaDefinition {
	integer := func(minimum, maximum float64) *protocol.DataSchemaDefinition {
		return &protocol.DataSchemaDefinition{Type: "integer", Minimum: &minimum, Maximum: &maximum}
	}
	boolean := &protocol.DataSchemaDefinition{Type: "boolean"}
	names := make([]string, 0, len(definitions))
	for _, definition := range Definitions("windows") {
		names = append(names, string(definition.Name))
	}
	count, channels, label := len(names), 64, protocol.MaxIdentifierBytes
	setting := &protocol.DataSchemaDefinition{Type: "object", Properties: []protocol.DataSchemaProperty{
		{Name: "metric", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "string", Enum: names}},
		{Name: "enabled", Required: true, Schema: boolean},
		{Name: "writable", Required: true, Schema: boolean},
		{Name: "interval_ms", Required: true, Schema: integer(250, 86_400_000)},
	}}
	return protocol.DataSchemaDefinition{ID: SchemaConfigV1, Type: "object", Title: "Metrics configuration", Properties: []protocol.DataSchemaProperty{
		{Name: "version", Required: true, Schema: integer(1, 1)},
		{Name: "revision", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "string", Format: "uint64"}},
		{Name: "platform", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "string", Enum: []string{"windows"}}},
		{Name: "settings", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "array", Items: setting, MinItems: &count, MaxItems: &count}},
		{Name: "notification_channels", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "array", MaxItems: &channels, Items: &protocol.DataSchemaDefinition{Type: "string", MaxLength: &label}}},
	}}
}

func validPlatform(value string) bool { return value == "windows" }

func platforms(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func clonePlatforms(source map[string]bool) map[string]bool {
	result := make(map[string]bool, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func validateNotificationChannels(channels []string) error {
	if len(channels) > 64 {
		return errors.New("metrics notification channels exceed 64 entries")
	}
	for index, channel := range channels {
		if channel == "" || strings.TrimSpace(channel) != channel || !utf8.ValidString(channel) || len(channel) > protocol.MaxIdentifierBytes {
			return fmt.Errorf("notification channel %d is invalid", index)
		}
		if index > 0 && channels[index-1] >= channel {
			return errors.New("metrics notification channels must be strictly sorted")
		}
	}
	return nil
}
