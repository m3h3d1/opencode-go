This project implements an AI agent loop that can interact with the OpenRouter API to perform tasks using available tools.

## Features Implemented

### 1. Agent Loop
- Continuously sends messages to the LLM until no more tool calls are needed
- Maintains conversation history across iterations
- Handles multiple tool calls in a single response

### 2. Available Tools
- **Read**: Read and return the contents of a file
- **Write**: Write content to a file (creates or overwrites)
- **Bash**: Execute shell commands

## How It Works

The agent follows this loop:
1. Initialize conversation with user prompt
2. Send messages + available tools to LLM
3. Receive LLM response
4. If response contains tool calls:
   - Execute each tool
   - Append results to conversation
   - Repeat from step 2
5. If no tool calls: print final response and exit

## Usage

```bash
# Compile
go build -o /tmp/claude-code-go app/*.go

# Run with prompt
/tmp/claude-code-go -p "Your prompt here"
```

### Examples

**Read a file:**
```bash
/tmp/claude-code-go -p "Read README.md"
```

**Write to a file:**
```bash
/tmp/claude-code-go -p "Write 'Hello World' to greeting.txt"
```

**Execute shell commands:**
```bash
/tmp/claude-code-go -p "Bash: ls -la"
```

**Multi-step task:**
```bash
/tmp/claude-code-go -p "Read README.md and tell me the project structure"
```

## File Structure

```
.
├── app/
│   └── main.go          # Main implementation
├── test/
│   └── tools.sh         # Local test script
├── README.md            # This file
├── README_old.md        # Test file for deletion
├── your_program.sh      # Wrapper script
├── go.mod              # Go module definition
└── go.sum              # Go module checksums
```

## Implementation Details

- Language: Go
- LLM API: OpenRouter (compatible with OpenAI API)
- Model: anthropic/claude-haiku-4.5 (configurable via LOCAL_MODEL env var)
- Tools advertised: Read, Write, Bash

## Environment Variables

- `OPENROUTER_API_KEY`: API key for OpenRouter (required)
- `OPENROUTER_BASE_URL`: Base URL for API (defaults to https://openrouter.ai/api/v1)
- `LOCAL_MODEL`: Model to use (defaults to anthropic/claude-haiku-4.5)

## Local Testing

Run the test script:
```bash
./test/tools.sh
```

Tests verify:
- Read tool correctly retrieves file contents
- Write tool correctly creates/overwrites files
- Bash tool executes shell commands (rm, echo, etc.)
- All tools work in sequence within the agent loop