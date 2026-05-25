package history

import (
	"strings"
	"time"

	"github.com/Furthen64/lltop/internal/config"
)

type ScenarioKey struct {
	ProfileName string
	Model       string
	Ctx         int
	NGL         int
	CacheK      string
	CacheV      string
	Batch       int
	UBatch      int
	Parallel    int
	ExtraArgs   string
}

func BuildScenarioKey(p *config.Profile) ScenarioKey {
	return ScenarioKey{
		ProfileName: p.Name,
		Model:       p.Model,
		Ctx:         p.Ctx,
		NGL:         p.NGL,
		CacheK:      p.CacheK,
		CacheV:      p.CacheV,
		Batch:       p.Batch,
		UBatch:      p.UBatch,
		Parallel:    p.Parallel,
		ExtraArgs:   strings.Join(p.ExtraArgs, "\x00"),
	}
}

func FindRecentFailure(runsDir string, key ScenarioKey, windowSeconds int, startupFailureSecs int) (*RunRecord, error) {
	records, err := LoadRunRecords(runsDir)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return nil, nil
		}
		return nil, err
	}
	cutoff := time.Now().Add(-time.Duration(windowSeconds) * time.Second)
	var newest *RunRecord
	for _, record := range records {
		if record.StartedAt.Before(cutoff) {
			continue
		}
		if record.ExitCode == 0 || record.DurationSeconds >= float64(startupFailureSecs) {
			continue
		}
		if !matchesScenario(record, key) {
			continue
		}
		if newest == nil || record.StartedAt.After(newest.StartedAt) {
			newest = record
		}
	}
	return newest, nil
}

func matchesScenario(record *RunRecord, key ScenarioKey) bool {
	return record.ProfileName == key.ProfileName &&
		record.Model == key.Model &&
		record.Ctx == key.Ctx &&
		record.NGL == key.NGL &&
		record.CacheK == key.CacheK &&
		record.CacheV == key.CacheV &&
		record.Batch == key.Batch &&
		record.UBatch == key.UBatch &&
		record.Parallel == key.Parallel &&
		strings.Join(record.ExtraArgs, "\x00") == key.ExtraArgs
}
