package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type Profile struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	LlamaServer string   `toml:"llama_server"`
	Model       string   `toml:"model"`
	Host        string   `toml:"host"`
	Port        int      `toml:"port"`
	Alias       string   `toml:"alias"`
	Ctx         int      `toml:"ctx"`
	NGL         int      `toml:"ngl"`
	CacheK      string   `toml:"cache_k"`
	CacheV      string   `toml:"cache_v"`
	Temp        float64  `toml:"temp"`
	TopP        float64  `toml:"top_p"`
	TopK        int      `toml:"top_k"`
	MinP        float64  `toml:"min_p"`
	Batch       int      `toml:"batch"`
	UBatch      int      `toml:"ubatch"`
	Parallel    int      `toml:"parallel"`
	Threads     int      `toml:"threads"`
	Jinja       bool     `toml:"jinja"`
	Metrics     bool     `toml:"metrics"`
	ExtraArgs   []string `toml:"extra_args"`
}

func DefaultProfile(cfg *GlobalConfig, name string) *Profile {
	host := "0.0.0.0"
	port := 8080
	llamaServer := ""
	if cfg != nil {
		if cfg.DefaultHost != "" {
			host = cfg.DefaultHost
		}
		if cfg.DefaultPort != 0 {
			port = cfg.DefaultPort
		}
		llamaServer = cfg.LlamaServer
	}
	return &Profile{
		Name:        name,
		Description: "",
		LlamaServer: llamaServer,
		Host:        host,
		Port:        port,
		Alias:       "",
		Ctx:         65536,
		NGL:         0,
		CacheK:      "q4_0",
		CacheV:      "q4_0",
		Temp:        0.1,
		TopP:        0.95,
		TopK:        40,
		MinP:        0.05,
		Batch:       512,
		UBatch:      256,
		Parallel:    1,
		Threads:     0,
		Jinja:       true,
		Metrics:     true,
		ExtraArgs:   []string{},
	}
}

func (p *Profile) ApplyDefaults(cfg *GlobalConfig) {
	defaults := DefaultProfile(cfg, p.Name)
	if p.Host == "" {
		p.Host = defaults.Host
	}
	if p.Port == 0 {
		p.Port = defaults.Port
	}
	if p.Ctx == 0 {
		p.Ctx = defaults.Ctx
	}
	if p.CacheK == "" {
		p.CacheK = defaults.CacheK
	}
	if p.CacheV == "" {
		p.CacheV = defaults.CacheV
	}
	if p.Temp == 0 {
		p.Temp = defaults.Temp
	}
	if p.TopP == 0 {
		p.TopP = defaults.TopP
	}
	if p.TopK == 0 {
		p.TopK = defaults.TopK
	}
	if p.MinP == 0 {
		p.MinP = defaults.MinP
	}
	if p.Batch == 0 {
		p.Batch = defaults.Batch
	}
	if p.UBatch == 0 {
		p.UBatch = defaults.UBatch
	}
	if p.Parallel == 0 {
		p.Parallel = defaults.Parallel
	}
	if p.LlamaServer == "" && cfg != nil {
		p.LlamaServer = cfg.LlamaServer
	}
}

func LoadProfile(path string) (*Profile, error) {
	p := DefaultProfile(nil, "")
	if _, err := toml.DecodeFile(path, p); err != nil {
		return nil, err
	}
	if p.LlamaServer != "" {
		expanded, err := ExpandPath(p.LlamaServer)
		if err != nil {
			return nil, err
		}
		p.LlamaServer = expanded
	}
	if p.Model != "" {
		expanded, err := ExpandPath(p.Model)
		if err != nil {
			return nil, err
		}
		p.Model = expanded
	}
	if err := ValidateProfileConfig(p); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return p, nil
}

func LoadProfiles(dir string) ([]*Profile, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.toml"))
	if err != nil {
		return nil, err
	}
	profiles := make([]*Profile, 0, len(entries))
	for _, entry := range entries {
		p, err := LoadProfile(entry)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})
	return profiles, nil
}

func SaveProfile(path string, p *Profile) error {
	var b strings.Builder
	fmt.Fprintf(&b, "name = %q\n", p.Name)
	fmt.Fprintf(&b, "description = %q\n", p.Description)
	if p.LlamaServer != "" {
		fmt.Fprintf(&b, "llama_server = %q\n", p.LlamaServer)
	}
	fmt.Fprintf(&b, "model = %q\n", p.Model)
	fmt.Fprintf(&b, "host = %q\n", p.Host)
	fmt.Fprintf(&b, "port = %d\n", p.Port)
	fmt.Fprintf(&b, "alias = %q\n", p.Alias)
	fmt.Fprintf(&b, "ctx = %d\n", p.Ctx)
	fmt.Fprintf(&b, "ngl = %d\n", p.NGL)
	fmt.Fprintf(&b, "cache_k = %q\n", p.CacheK)
	fmt.Fprintf(&b, "cache_v = %q\n", p.CacheV)
	fmt.Fprintf(&b, "temp = %.3f\n", p.Temp)
	fmt.Fprintf(&b, "top_p = %.3f\n", p.TopP)
	fmt.Fprintf(&b, "top_k = %d\n", p.TopK)
	fmt.Fprintf(&b, "min_p = %.3f\n", p.MinP)
	fmt.Fprintf(&b, "batch = %d\n", p.Batch)
	fmt.Fprintf(&b, "ubatch = %d\n", p.UBatch)
	fmt.Fprintf(&b, "parallel = %d\n", p.Parallel)
	fmt.Fprintf(&b, "threads = %d\n", p.Threads)
	fmt.Fprintf(&b, "jinja = %t\n", p.Jinja)
	fmt.Fprintf(&b, "metrics = %t\n", p.Metrics)
	b.WriteString("extra_args = [")
	for i, arg := range p.ExtraArgs {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q", arg)
	}
	b.WriteString("]\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func FindProfile(profiles []*Profile, name string) *Profile {
	for _, p := range profiles {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func SlugifyName(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "profile"
	}
	return slug
}
