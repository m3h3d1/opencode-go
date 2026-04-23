package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type ReadArgs struct {
	FilePath string `json:"file_path"`
}

type WriteArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func main() {
	var prompt string = "Hello"
	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.Parse()

	if prompt == "" {
		panic("Prompt must not be empty")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")
	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}

	if apiKey == "" {
		panic("Env variable OPENROUTER_API_KEY not found")
	}

	model := os.Getenv("LOCAL_MODEL")
	if model == "" {
		model = "anthropic/claude-haiku-4.5"
	}

	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseUrl))

	messages := []openai.ChatCompletionMessageParamUnion{
		{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfString: openai.String(prompt),
				},
			},
		},
	}

	for {
		resp, err := client.Chat.Completions.New(context.Background(),
			openai.ChatCompletionNewParams{
				Model: model,
				Messages: append(messages,
					openai.ChatCompletionMessageParamUnion{
						OfAssistant: &openai.ChatCompletionAssistantMessageParam{},
					},
				),
				Tools: []openai.ChatCompletionToolUnionParam{
					openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
						Name:        "Read",
						Description: openai.String("Read and return the contents of a file"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"file_path": map[string]any{
									"type":        "string",
									"description": "The path to the file to read",
								},
							},
							"required": []string{"file_path"},
						},
					}),
					openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
						Name:        "Write",
						Description: openai.String("Write content to a file"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"file_path": map[string]any{
									"type":        "string",
									"description": "The path of the file to write to",
								},
								"content": map[string]any{
									"type":        "string",
									"description": "The content to write to the file",
								},
							},
							"required": []string{"file_path", "content"},
						},
					}),
					openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
						Name:        "Bash",
						Description: openai.String("Execute a shell command"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"command": map[string]any{
									"type":        "string",
									"description": "The command to execute",
								},
							},
							"required": []string{"command"},
						},
					}),
				},
			},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(resp.Choices) == 0 {
			panic("No choices in response")
		}

		assistantMsg := resp.Choices[0].Message
		contentStr := assistantMsg.Content
		var toolCallParam []openai.ChatCompletionMessageToolCallUnionParam
		for _, tc := range assistantMsg.ToolCalls {
			toolCallParam = append(toolCallParam, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID:   tc.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				},
			})
		}

		messages = append(messages, openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(contentStr),
				},
				ToolCalls: toolCallParam,
			},
		})

		if len(assistantMsg.ToolCalls) == 0 {
			fmt.Print(contentStr)
			return
		}

		for _, tc := range assistantMsg.ToolCalls {
			if tc.Function.Name == "Read" {
				var args ReadArgs
				err := json.Unmarshal([]byte(tc.Function.Arguments), &args)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error parsing tool arguments: %v\n", err)
					os.Exit(1)
				}

				fileContent, err := os.ReadFile(args.FilePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error reading file %s: %v\n", args.FilePath, err)
					os.Exit(1)
				}

				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfTool: &openai.ChatCompletionToolMessageParam{
						ToolCallID: tc.ID,
						Content: openai.ChatCompletionToolMessageParamContentUnion{
							OfString: openai.String(string(fileContent)),
						},
					},
				})
			} else if tc.Function.Name == "Write" {
				var args WriteArgs
				err := json.Unmarshal([]byte(tc.Function.Arguments), &args)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error parsing tool arguments: %v\n", err)
					os.Exit(1)
				}

				err = os.WriteFile(args.FilePath, []byte(args.Content), 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error writing file %s: %v\n", args.FilePath, err)
					os.Exit(1)
				}

				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfTool: &openai.ChatCompletionToolMessageParam{
						ToolCallID: tc.ID,
						Content: openai.ChatCompletionToolMessageParamContentUnion{
							OfString: openai.String("File written successfully"),
						},
					},
				})
			} else if tc.Function.Name == "Bash" {
				var args struct {
					Command string `json:"command"`
				}
				err := json.Unmarshal([]byte(tc.Function.Arguments), &args)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error parsing tool arguments: %v\n", err)
					os.Exit(1)
				}

				cmd := exec.Command("sh", "-c", args.Command)
				output, err := cmd.Output()

				result := string(output)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfTool: &openai.ChatCompletionToolMessageParam{
						ToolCallID: tc.ID,
						Content: openai.ChatCompletionToolMessageParamContentUnion{
							OfString: openai.String(result),
						},
					},
				})
			}
		}
	}
}