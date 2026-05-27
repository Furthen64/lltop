package config

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const wizardModelSearchDepth = 3
const llamaServerSearchDepth = 4

func RunFirstStartWizard(cfg *GlobalConfig) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is required")
	}
	if !isInteractiveTerminal() {
		return "Created config at ~/.config/lltop/. Run again in an interactive terminal to finish first-run setup.", nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintln(os.Stdout, "Welcome to lltop first-run setup")
	fmt.Fprintln(os.Stdout, "Press Enter to accept defaults.")
	fmt.Fprintln(os.Stdout)

	defaultLlama := firstNonEmpty(cfg.LlamaServer, detectLlamaServerPath())
	llamaPath, err := promptLlamaPath(reader, os.Stdout, "Path to llama-server binary or directory", defaultLlama)
	if err != nil {
		return "", err
	}
	cfg.LlamaServer = llamaPath

	modelsDir, err := promptPath(reader, os.Stdout, "Path to models directory (optional)", cfg.ModelsDir, optionalDirectory)
	if err != nil {
		return "", err
	}
	cfg.ModelsDir = modelsDir

	configPath, err := ConfigPath()
	if err != nil {
		return "", err
	}
	if err := WriteConfig(configPath, cfg); err != nil {
		return "", err
	}

	if modelsDir == "" {
		return "Saved first-run setup. Add models_dir later to auto-generate profiles.", nil
	}

	models, err := DiscoverModelFiles(modelsDir, wizardModelSearchDepth)
	if err != nil {
		return "", err
	}
	if len(models) == 0 {
		return fmt.Sprintf("Saved first-run setup. No models found under %s.", modelsDir), nil
	}

	created, err := GenerateProfilesForModels(cfg, models)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Saved first-run setup. Found %d model(s), created %d profile(s).", len(models), created), nil
}

func DiscoverModelFiles(root string, maxDepth int) ([]string, error) {
	if root == "" {
		return nil, nil
	}
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("models path must be a directory")
	}

	var models []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path != root {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			depth := strings.Count(rel, string(filepath.Separator)) + 1
			if depth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == ".gguf" || ext == ".bin" {
			models = append(models, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(models)
	return models, nil
}

func GenerateProfilesForModels(cfg *GlobalConfig, models []string) (int, error) {
	if cfg == nil {
		return 0, fmt.Errorf("config is required")
	}

	existing := map[string]struct{}{}
	entries, err := filepath.Glob(filepath.Join(cfg.ProfilesDir, "*.toml"))
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		name := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))
		existing[name] = struct{}{}
	}

	created := 0
	for _, modelPath := range models {
		baseName := strings.TrimSuffix(filepath.Base(modelPath), filepath.Ext(modelPath))
		slug := uniqueProfileSlug(SlugifyName(baseName), existing)
		existing[slug] = struct{}{}
		profilePath := filepath.Join(cfg.ProfilesDir, slug+".toml")
		if _, err := os.Stat(profilePath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return created, err
		}

		profile := DefaultProfile(cfg, slug)
		profile.Description = "Auto-generated from first-run setup"
		profile.Model = modelPath
		if err := SaveProfile(profilePath, profile); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

func uniqueProfileSlug(base string, existing map[string]struct{}) string {
	if _, ok := existing[base]; !ok {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}
}

func promptPath(reader *bufio.Reader, out io.Writer, label, defaultValue string, validator func(string) error) (string, error) {
	for {
		if defaultValue != "" {
			fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
		} else {
			fmt.Fprintf(out, "%s: ", label)
		}
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(input)
		if value == "" {
			value = defaultValue
		}
		expanded, err := ExpandPath(value)
		if err != nil {
			fmt.Fprintf(out, "Invalid path: %v\n", err)
			continue
		}
		if err := validator(expanded); err != nil {
			fmt.Fprintf(out, "Invalid path: %v\n", err)
			continue
		}
		return expanded, nil
	}
}

func promptLlamaPath(reader *bufio.Reader, out io.Writer, label, defaultValue string) (string, error) {
	for {
		if defaultValue != "" {
			fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
		} else {
			fmt.Fprintf(out, "%s: ", label)
		}
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(input)
		if value == "" {
			value = defaultValue
		}
		expanded, err := ExpandPath(value)
		if err != nil {
			fmt.Fprintf(out, "Invalid path: %v\n", err)
			continue
		}
		resolved, err := resolveLlamaServerPath(expanded)
		if err != nil {
			fmt.Fprintf(out, "Invalid path: %v\n", err)
			continue
		}
		return resolved, nil
	}
}

func requireExecutableFile(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("path must point to a file")
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("file is not executable")
	}
	return nil
}

func resolveLlamaServerPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		if err := requireExecutableFile(path); err != nil {
			return "", err
		}
		return path, nil
	}
	matches, err := findNamedExecutables(path, "llama-server", llamaServerSearchDepth)
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no executable llama-server found under %s", path)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple executable llama-server binaries found, use full path: %s", strings.Join(matches, ", "))
	}
}

func findNamedExecutables(root, filename string, maxDepth int) ([]string, error) {
	root = filepath.Clean(root)
	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path != root {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			depth := strings.Count(rel, string(filepath.Separator)) + 1
			if depth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != filename {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Mode()&0o111 == 0 {
			return nil
		}
		matches = append(matches, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func optionalDirectory(path string) error {
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path must point to a directory")
	}
	return nil
}

func detectLlamaServerPath() string {
	path, err := exec.LookPath("llama-server")
	if err != nil {
		return ""
	}
	return path
}

func isInteractiveTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
