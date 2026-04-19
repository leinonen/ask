# ask

CLI for querying LLMs from the terminal. Supports Anthropic (Claude) and OpenAI.

## Install

```sh
go install github.com/leinonen/ask@latest
```

Or build from source:

```sh
go build -o ask .
```

## Usage

```sh
ask "what is the capital of France?"
echo "func foo() {}" | ask "explain this"
cat file.txt | ask "summarize"
ask --shell "list files sorted by size"
ask --code "fibonacci in python"
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--shell` | `-s` | Generate a shell command (prompts to execute) |
| `--code` | `-c` | Output code only, no markdown fences |
| `--system TEXT` | `-S` | Custom system prompt |
| `--model TEXT` | `-m` | Model override |
| `--provider TEXT` | `-p` | Provider override (`anthropic` or `openai`) |

## Config

On first run, a setup wizard prompts for provider, API key, and default model. Config is saved to `~/.config/ask/config.toml`.

You can also set keys via environment variables:

```sh
export ANTHROPIC_API_KEY=sk-ant-...
export OPENAI_API_KEY=sk-...
```
