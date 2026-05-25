package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/Furthen64/lltop/internal/config"
)

func SaveRunRecord(runsDir string, record *RunRecord) (string, error) {
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return "", err
	}
	name := record.StartedAt.Format("2006-01-02_150405") + "_" + config.SlugifyName(record.ProfileName) + ".json"
	path := filepath.Join(runsDir, name)
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func LoadRunRecords(runsDir string) ([]*RunRecord, error) {
	entries, err := filepath.Glob(filepath.Join(runsDir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	records := make([]*RunRecord, 0, len(entries))
	for _, entry := range entries {
		data, err := os.ReadFile(entry)
		if err != nil {
			return nil, err
		}
		var record RunRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, err
		}
		records = append(records, &record)
	}
	return records, nil
}
