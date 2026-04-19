package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Provider  string         `toml:"provider"`
	Model     string         `toml:"model"`
	Anthropic ProviderConfig `toml:"anthropic"`
	OpenAI    ProviderConfig `toml:"openai"`
}

type ProviderConfig struct {
	APIKey string `toml:"api_key"`
}

var anthropicModels = []string{
	"claude-opus-4-7",
	"claude-opus-4-6",
	"claude-sonnet-4-6",
	"claude-haiku-4-5",
}

var openaiModels = []string{
	"gpt-4o",
	"gpt-4o-mini",
	"gpt-4-turbo",
	"o1",
	"o3-mini",
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.Getenv("HOME")
	}
	return filepath.Join(dir, "ask", "config.toml")
}

func loadConfig() (Config, error) {
	path := configPath()
	cfg := Config{}

	exists := true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exists = false
	}

	if exists {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return cfg, fmt.Errorf("parsing config %s: %w", path, err)
		}
	}

	// Run setup wizard if config is missing or has no usable provider/key
	if needsSetup(cfg, exists) {
		if err := runSetupWizard(&cfg); err != nil {
			return cfg, err
		}
		if err := saveConfig(path, cfg); err != nil {
			return cfg, fmt.Errorf("saving config: %w", err)
		}
	}

	return cfg, nil
}

func needsSetup(cfg Config, exists bool) bool {
	if !exists {
		return true
	}
	// Has a provider set but no way to auth — prompt setup
	provider := cfg.Provider
	if provider == "" {
		return true
	}
	key := cfg.resolveAPIKey(provider)
	return key == ""
}

func runSetupWizard(cfg *Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("─────────────────────────────────────")
	fmt.Println("  ask — first-time setup")
	fmt.Println("─────────────────────────────────────")
	fmt.Println()

	// 1. Choose provider
	fmt.Println("Choose a provider:")
	fmt.Println("  1) Anthropic (Claude)")
	fmt.Println("  2) OpenAI (ChatGPT)")
	fmt.Print("\nProvider [1]: ")

	providerInput := strings.TrimSpace(readLine(reader))
	switch providerInput {
	case "2", "openai":
		cfg.Provider = "openai"
	default:
		cfg.Provider = "anthropic"
	}
	fmt.Printf("→ %s\n\n", cfg.Provider)

	// 2. API key
	envVar := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
	}[cfg.Provider]

	existingKey := os.Getenv(envVar)
	if existingKey != "" {
		fmt.Printf("API key found in $%s — using it.\n\n", envVar)
	} else {
		fmt.Printf("Enter your %s API key\n(or leave blank and set $%s later):\n> ", cfg.Provider, envVar)
		key := strings.TrimSpace(readLine(reader))
		if key != "" {
			switch cfg.Provider {
			case "openai":
				cfg.OpenAI.APIKey = key
			default:
				cfg.Anthropic.APIKey = key
			}
			fmt.Println()
		} else {
			fmt.Printf("\nNo key entered. Set $%s before running ask.\n\n", envVar)
		}
	}

	// 3. Pick model
	models := anthropicModels
	if cfg.Provider == "openai" {
		models = openaiModels
	}

	fmt.Println("Choose a default model:")
	for i, m := range models {
		marker := " "
		if i == 0 {
			marker = "*"
		}
		fmt.Printf("  %s %d) %s\n", marker, i+1, m)
	}
	fmt.Printf("\nModel [1]: ")

	modelInput := strings.TrimSpace(readLine(reader))
	chosen := models[0]
	if n, err := strconv.Atoi(modelInput); err == nil && n >= 1 && n <= len(models) {
		chosen = models[n-1]
	}
	cfg.Model = chosen
	fmt.Printf("→ %s\n\n", chosen)

	fmt.Printf("Config saved to %s\n", configPath())
	fmt.Println("─────────────────────────────────────")
	fmt.Println()
	return nil
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

func saveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Write TOML manually to keep comments readable
	var sb strings.Builder
	sb.WriteString("# ask configuration — edit freely\n\n")
	fmt.Fprintf(&sb, "provider = %q\n", cfg.Provider)
	fmt.Fprintf(&sb, "model    = %q\n\n", cfg.Model)
	sb.WriteString("[anthropic]\n")
	fmt.Fprintf(&sb, "api_key = %q\n\n", cfg.Anthropic.APIKey)
	sb.WriteString("[openai]\n")
	fmt.Fprintf(&sb, "api_key = %q\n", cfg.OpenAI.APIKey)

	return os.WriteFile(path, []byte(sb.String()), 0600)
}

func (c *Config) resolveAPIKey(provider string) string {
	switch provider {
	case "anthropic":
		if c.Anthropic.APIKey != "" {
			return c.Anthropic.APIKey
		}
		return os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		if c.OpenAI.APIKey != "" {
			return c.OpenAI.APIKey
		}
		return os.Getenv("OPENAI_API_KEY")
	}
	return ""
}

func (c *Config) defaultModel(provider string) string {
	if c.Model != "" {
		return c.Model
	}
	switch provider {
	case "openai":
		return "gpt-4o"
	default:
		return "claude-opus-4-7"
	}
}
