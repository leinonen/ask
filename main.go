package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Provider interface {
	Stream(ctx context.Context, system, prompt string, w io.Writer) error
}

func main() {
	var (
		shellMode    bool
		codeMode     bool
		systemPrompt string
		modelFlag    string
		providerFlag string
	)

	flag.BoolVar(&shellMode, "s", false, "generate a shell command")
	flag.BoolVar(&shellMode, "shell", false, "generate a shell command")
	flag.BoolVar(&codeMode, "c", false, "output code only")
	flag.BoolVar(&codeMode, "code", false, "output code only")
	flag.StringVar(&systemPrompt, "S", "", "custom system prompt")
	flag.StringVar(&systemPrompt, "system", "", "custom system prompt")
	flag.StringVar(&modelFlag, "m", "", "model override")
	flag.StringVar(&modelFlag, "model", "", "model override")
	flag.StringVar(&providerFlag, "p", "", "provider override (anthropic|openai)")
	flag.StringVar(&providerFlag, "provider", "", "provider override (anthropic|openai)")
	flag.Usage = usage
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	provider := cfg.Provider
	if providerFlag != "" {
		provider = providerFlag
	}

	model := cfg.defaultModel(provider)
	if modelFlag != "" {
		model = modelFlag
	}

	// Read piped stdin
	var stdinContent string
	stat, _ := os.Stdin.Stat()
	if stat.Mode()&os.ModeCharDevice == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
			os.Exit(1)
		}
		stdinContent = strings.TrimSpace(string(data))
	}

	prompt := strings.Join(flag.Args(), " ")
	if prompt == "" && stdinContent == "" {
		usage()
		os.Exit(1)
	}

	// Build final prompt
	if stdinContent != "" && prompt != "" {
		prompt = fmt.Sprintf("<stdin>\n%s\n</stdin>\n\n%s", stdinContent, prompt)
	} else if stdinContent != "" {
		prompt = stdinContent
	}

	// Build system prompt
	system := systemPrompt
	if shellMode {
		system = "Output only the shell command with no explanation, no markdown, no code fences."
	} else if codeMode {
		system = "Output only the code with no explanation, no markdown fences."
	}

	p := buildProvider(provider, cfg, model)

	// Collect output for shell mode execution prompt
	var buf strings.Builder
	var w io.Writer = os.Stdout
	if shellMode {
		w = io.MultiWriter(os.Stdout, &buf)
	}

	if err := p.Stream(context.Background(), system, prompt, w); err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()

	if shellMode && isTerminal() {
		cmd := strings.TrimSpace(buf.String())
		if cmd == "" {
			return
		}
		fmt.Fprintf(os.Stderr, "\nExecute? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) == "y" {
			c := exec.Command("sh", "-c", cmd)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin
			if err := c.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

func buildProvider(provider string, cfg Config, model string) Provider {
	apiKey := cfg.resolveAPIKey(provider)
	switch provider {
	case "openai":
		return newOpenAIProvider(apiKey, model)
	default:
		return newAnthropicProvider(apiKey, model)
	}
}

func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: ask [flags] [prompt]

Examples:
  ask "what is the capital of France?"
  echo "func foo() {}" | ask "explain this"
  cat file.txt | ask "summarize"
  ask --shell "list files sorted by size"
  ask --code "fibonacci in python"

Flags:
  -s, --shell          Generate a shell command (offers to execute)
  -c, --code           Output code only (no markdown fences)
  -S, --system TEXT    Custom system prompt
  -m, --model TEXT     Model override
  -p, --provider TEXT  Provider override (anthropic|openai)

Config: %s
`, configPath())
}
