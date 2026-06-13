package genericcsv

import "sort"

// BuiltinConfigs maps config names to their Config values.
var BuiltinConfigs = map[string]Config{
	"dkb_v1": {
		HasHeader:          true,
		SkipLines:          4,
		DateCol:            0,
		DateFormat:         "02.01.06",
		PayeeCol:           4,
		AmountCol:          8,
		CurrencyCol:        -1,
		MemoCol:            5,
		Comma:              ';',
		DecimalSeparator:   ",",
		ThousandsSeparator: ".",
	},
}

// GetBuiltinConfig returns the config named name and true if it exists.
func GetBuiltinConfig(name string) (Config, bool) {
	cfg, ok := BuiltinConfigs[name]
	return cfg, ok
}

// AvailableConfigs returns a sorted list of available builtin config names.
func AvailableConfigs() []string {
	names := make([]string, 0, len(BuiltinConfigs))
	for k := range BuiltinConfigs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
