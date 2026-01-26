# FormAgent

A conversational form-filling agent built with [CloudWeGo Eino](https://www.cloudwego.io/docs/eino/), implementing intelligent form orchestration through natural language interactions.

## Overview

FormAgent is a Go library that enables building conversational form-filling experiences powered by Large Language Models (LLMs). It leverages Eino's component-based architecture and orchestration capabilities to manage complex form workflows through natural dialogue.

### Key Features

- **Conversational Interface**: Users fill forms through natural language instead of traditional form fields
- **Intelligent Field Extraction**: Automatically extracts and validates form data from user input
- **Multi-Phase Workflow**: Supports collecting, confirming, and submitting phases
- **State Management**: Built-in checkpoint/restore capabilities for resuming conversations
- **Type-Safe**: Leverages Go generics for compile-time type safety
- **Extensible**: Customizable parsers, validators, and submission handlers

## Architecture

FormAgent is built on Eino's orchestration framework, treating components as first-class citizens:

- **PatchGenerator**: Extracts structured data from user input using RFC 6902 JSON Patch
- **DialogueGenerator**: Generates contextual responses based on form state
- **CommandParser**: Recognizes user commands (confirm, cancel, back)
- **FormSpec**: Defines form schema, validation rules, and submission logic

## Installation

```bash
go get github.com/TBXark/formagent
```

### Dependencies

```bash
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai
```

## Quick Start

### 1. Define Your Form Structure

```go
type UserRegistrationForm struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Age      int    `json:"age"`
    Password string `json:"password"`
}
```

### 2. Implement FormSpec Interface

```go
type RegistrationSpec struct{}

func (s *RegistrationSpec) AllowedJSONPointers() []string {
    return []string{"/name", "/email", "/age", "/password"}
}

func (s *RegistrationSpec) MissingFacts(current UserRegistrationForm) []formagent.FieldInfo {
    var missing []formagent.FieldInfo
    if current.Name == "" {
        missing = append(missing, formagent.FieldInfo{
            JSONPointer: "/name",
            DisplayName: "Name",
            Required:    true,
        })
    }
    if current.Email == "" {
        missing = append(missing, formagent.FieldInfo{
            JSONPointer: "/email",
            DisplayName: "Email",
            Required:    true,
        })
    }
    return missing
}

func (s *RegistrationSpec) ValidateFacts(current UserRegistrationForm) []formagent.ValidationError {
    var errors []formagent.ValidationError
    if current.Age < 18 || current.Age > 100 {
        errors = append(errors, formagent.ValidationError{
            JSONPointer: "/age",
            Message:     "Age must be between 18 and 100",
        })
    }
    return errors
}

func (s *RegistrationSpec) FieldGuide(fieldPath string) string {
    guides := map[string]string{
        "/email": "Please provide a valid email address",
        "/age":   "Age must be between 18 and 100",
    }
    return guides[fieldPath]
}

func (s *RegistrationSpec) Summary(current UserRegistrationForm) string {
    return fmt.Sprintf("Name: %s, Email: %s, Age: %d", 
        current.Name, current.Email, current.Age)
}

func (s *RegistrationSpec) Submit(ctx context.Context, final UserRegistrationForm) error {
    // Implement your submission logic here
    fmt.Printf("Submitting form: %+v\n", final)
    return nil
}
```

### 3. Configure LLM Model

Create `config.json`:

```json
{
  "api_key": "your-api-key",
  "base_url": "https://api.openai.com/v1",
  "model": "gpt-4o-mini"
}
```

### 4. Initialize and Use the Agent

```go
package main

import (
    "context"
    "fmt"
    "formagent"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

func main() {
    ctx := context.Background()
    
    // Initialize chat model
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        APIKey:  "your-api-key",
        Model:   "gpt-4o-mini",
        BaseURL: "https://api.openai.com/v1",
    })
    if err != nil {
        panic(err)
    }
    
    // Create agent
    agent, err := formagent.NewToolBasedFormAgent[UserRegistrationForm](
        &RegistrationSpec{},
        chatModel,
    )
    if err != nil {
        panic(err)
    }
    
    // Conversation flow
    resp, err := agent.Invoke(ctx, "My name is John and email is john@example.com")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Agent: %s\n", resp.Message)
    fmt.Printf("Phase: %s\n", resp.Phase)
    fmt.Printf("State: %+v\n", resp.FormState)
    
    // Continue conversation
    resp, err = agent.Invoke(ctx, "I'm 25 years old")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Agent: %s\n", resp.Message)
    
    // Confirm submission
    resp, err = agent.Invoke(ctx, "confirm")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Agent: %s\n", resp.Message)
    fmt.Printf("Completed: %v\n", resp.Completed)
}
```

## Usage Patterns

### Basic Conversation Flow

```go
// Round 1: Provide partial information
resp, _ := agent.Invoke(ctx, "My name is Alice, email alice@example.com")
// Phase: collecting

// Round 2: Complete missing fields
resp, _ = agent.Invoke(ctx, "I'm 28 years old")
// Phase: confirming (auto-transitions when all required fields are filled)

// Round 3: Confirm submission
resp, _ = agent.Invoke(ctx, "confirm")
// Phase: submitted, Completed: true
```

### Commands

Users can issue special commands during the conversation:

- **confirm**: Submit the form (only in confirming phase)
- **cancel**: Cancel the form filling process
- **back**: Return to editing mode from confirming phase

### Checkpoint & Restore

Save and restore conversation state:

```go
// Create checkpoint
checkpointData, err := agent.CreateCheckpoint()

// Later, restore and continue
resp, err := agent.InvokeWithCheckpoint(ctx, checkpointData, "update my age to 30")
```

### Initial State

Pre-populate form fields:

```go
initial := UserRegistrationForm{
    Name: "Bob",
    Email: "bob@example.com",
}

resp, err := agent.InvokeWithInit(ctx, initial, "I'm 35 years old")
```

### Query Current State

```go
currentState := agent.GetCurrentState()
currentPhase := agent.GetPhase()
```

## Workflow Phases

1. **Collecting**: Agent collects required information through conversation
2. **Confirming**: All required fields filled, waiting for user confirmation
3. **Submitted**: Form successfully submitted
4. **Cancelled**: User cancelled the form filling process

## Advanced Usage

### Custom Command Parser

Implement your own command recognition logic:

```go
type CustomParser struct{}

func (p *CustomParser) ParseCommand(ctx context.Context, input string) formagent.Command {
    // Custom command parsing logic
    return formagent.CommandNone
}

agent, err := formagent.NewFormAgent(
    spec,
    patchGen,
    dialogGen,
    &CustomParser{}, // Use custom parser
)
```

### Custom Components

For advanced scenarios, create custom implementations:

```go
agent, err := formagent.NewFormAgent(
    spec,
    customPatchGenerator,    // Custom PatchGenerator
    customDialogueGenerator, // Custom DialogueGenerator
    customCommandParser,     // Custom CommandParser
)
```

## Testing

Run tests with live LLM:

```bash
export FORMAGENT_RUN_LIVE_TESTS=1
go test ./testcases/...
```

## Design Philosophy

FormAgent follows Eino's orchestration principles:

- **Component-First**: Business logic encapsulated in reusable components
- **Clear Separation**: Orchestration layer separate from business logic
- **Type Safety**: Leverages Go's type system for compile-time guarantees
- **Extensibility**: Easy to customize and extend for specific use cases

## References

- [Eino Documentation](https://www.cloudwego.io/docs/eino/)
- [Eino Chain & Graph Orchestration](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/)
- [CloudWeGo Eino GitHub](https://github.com/cloudwego/eino)

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
