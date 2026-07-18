package prometheus

import "github.com/pol-cova/observe/internal/config"

func MergePresets(custom []config.PrometheusPreset) []Preset {
	if len(custom) == 0 {
		return append([]Preset(nil), Presets...)
	}
	byName := make(map[string]Preset, len(Presets)+len(custom))
	order := make([]string, 0, len(Presets)+len(custom))
	for _, preset := range Presets {
		byName[preset.Name] = preset
		order = append(order, preset.Name)
	}
	for _, preset := range custom {
		if preset.Name == "" || preset.Query == "" {
			continue
		}
		if _, ok := byName[preset.Name]; !ok {
			order = append(order, preset.Name)
		}
		byName[preset.Name] = Preset{Name: preset.Name, Description: preset.Description, Query: preset.Query}
	}
	merged := make([]Preset, len(order))
	for i, name := range order {
		merged[i] = byName[name]
	}
	return merged
}
