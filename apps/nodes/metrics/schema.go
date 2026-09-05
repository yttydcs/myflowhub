package metrics

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub/protocol"
)

const (
	SchemaSampleV1        = "mfh.metrics.sample.v1"
	SchemaControlV1       = "mfh.metrics.control.v1"
	SchemaControlResultV1 = "mfh.metrics.control-result.v1"
	SchemaConfigV1        = "mfh.metrics.config.v1"
	SchemaConfigUpdateV1  = "mfh.metrics.config-update.v1"

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

type SampleV1 struct {
	Version          int    `json:"version"`
	Metric           string `json:"metric"`
	Value            string `json:"value,omitempty"`
	Unit             string `json:"unit"`
	Status           string `json:"status"`
	SampledAtUnixMS  int64  `json:"sampled_at_unix_ms,omitempty"`
	ObservedAtUnixMS int64  `json:"observed_at_unix_ms"`
	Error            string `json:"error,omitempty"`
}

func (s SampleV1) Validate() error {
	if s.Version != 1 {
		return errors.New("metrics sample version must be 1")
	}
	definition, ok := DefinitionFor(Name(s.Metric))
	if !ok || s.Unit != definition.Unit {
		return errors.New("metrics sample metric or unit is invalid")
	}
	if s.Status != "fresh" && s.Status != "stale" && s.Status != "unavailable" {
		return errors.New("metrics sample status is invalid")
	}
	if s.ObservedAtUnixMS <= 0 || s.SampledAtUnixMS < 0 || s.SampledAtUnixMS > s.ObservedAtUnixMS {
		return errors.New("metrics sample timestamps are invalid")
	}
	if s.Status == "fresh" {
		if s.SampledAtUnixMS == 0 || s.Error != "" {
			return errors.New("fresh metrics sample must have a sample time and no error")
		}
		if err := ValidateValue(definition.Name, s.Value); err != nil {
			return err
		}
	} else {
		if s.Error == "" {
			return errors.New("non-fresh metrics sample requires an error summary")
		}
		if s.Status == "stale" && s.SampledAtUnixMS == 0 {
			return errors.New("stale metrics sample requires a previous sample")
		}
		if s.Status == "unavailable" && (s.SampledAtUnixMS != 0 || s.Value != "") {
			return errors.New("unavailable metrics sample must not contain a value")
		}
	}
	if len(s.Error) > 1024 {
		return errors.New("metrics sample error exceeds 1024 bytes")
	}
	return nil
}

type ControlV1 struct {
	Version int    `json:"version"`
	Value   string `json:"value"`
}

func (c ControlV1) Validate() error {
	if c.Version != 1 || c.Value == "" || len(c.Value) > 128 {
		return errors.New("metrics control version or value is invalid")
	}
	return nil
}

type ControlResultV1 struct {
	Version        int    `json:"version"`
	Metric         string `json:"metric"`
	RequestedValue string `json:"requested_value"`
	AppliedValue   string `json:"applied_value,omitempty"`
	Accepted       bool   `json:"accepted"`
	Applied        bool   `json:"applied"`
}

func (r ControlResultV1) Validate() error {
	definition, ok := DefinitionFor(Name(r.Metric))
	if r.Version != 1 || !ok || !definition.Controllable || !r.Accepted {
		return errors.New("metrics control result is invalid")
	}
	if err := ValidateValue(definition.Name, r.RequestedValue); err != nil {
		return err
	}
	if r.Applied {
		return ValidateValue(definition.Name, r.AppliedValue)
	}
	if r.AppliedValue != "" {
		return errors.New("pending metrics control result must not have an applied value")
	}
	return nil
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
func CommandName(name Name) string  { return ResourceName(name) + "/set" }

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

func ValidateValue(name Name, value string) error {
	definition, ok := DefinitionFor(name)
	if !ok {
		return errors.New("metrics value has an unknown metric")
	}
	value = strings.TrimSpace(value)
	switch definition.Unit {
	case "percent":
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || number < 0 || number > 100 {
			return errors.New("metrics percent value must be between 0 and 100")
		}
	case "boolean":
		if value != "true" && value != "false" {
			return errors.New("metrics boolean value must be true or false")
		}
	case "label":
		if value == "" || len(value) > protocol.MaxLabelBytes {
			return errors.New("metrics label value is invalid")
		}
	default:
		return errors.New("metrics unit is unsupported")
	}
	return nil
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
