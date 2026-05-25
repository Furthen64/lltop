package runner

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Furthen64/lltop/internal/config"
)

type CommandSpec struct {
	Path    string
	Args    []string
	Display string
}

func BuildCommand(cfg *config.GlobalConfig, profile *config.Profile) (CommandSpec, error) {
	p := *profile
	p.ApplyDefaults(cfg)

	cmdPath := config.EffectiveLlamaServer(cfg, &p)
	if cmdPath == "" {
		return CommandSpec{}, fmt.Errorf("llama_server path is required")
	}
	if !filepath.IsAbs(cmdPath) {
		expanded, err := config.ExpandPath(cmdPath)
		if err != nil {
			return CommandSpec{}, err
		}
		cmdPath = expanded
	}

	args := []string{
		"-m", p.Model,
		"--host", p.Host,
		"--port", strconv.Itoa(p.Port),
	}
	if p.Alias != "" {
		args = append(args, "-a", p.Alias)
	}
	args = append(args,
		"-c", strconv.Itoa(p.Ctx),
		"-ngl", strconv.Itoa(p.NGL),
		"--cache-type-k", p.CacheK,
		"--cache-type-v", p.CacheV,
		"--temp", formatFloat(p.Temp),
		"--top-p", formatFloat(p.TopP),
		"--top-k", strconv.Itoa(p.TopK),
		"--min-p", formatFloat(p.MinP),
		"-b", strconv.Itoa(p.Batch),
		"-ub", strconv.Itoa(p.UBatch),
		"--parallel", strconv.Itoa(p.Parallel),
	)
	if p.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(p.Threads))
	}
	if p.Metrics {
		args = append(args, "--metrics")
	}
	if p.Jinja {
		args = append(args, "--jinja")
	}
	args = append(args, p.ExtraArgs...)

	var b strings.Builder
	b.WriteString(shellQuote(cmdPath))
	for _, arg := range args {
		b.WriteByte(' ')
		b.WriteString(shellQuote(arg))
	}

	return CommandSpec{Path: cmdPath, Args: args, Display: b.String()}, nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'`$&|;()<>{}[]*?!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
