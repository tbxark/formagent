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
