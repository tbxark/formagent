package patch

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	einoSchema "github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/structured"
	"github.com/tbxark/formagent/types"
)

const (
	updateFormToolName        = "update_form"
	updateFormToolDescription = "Generate RFC6902 JSON Patch operations to update form fields based on user input. Only include operations for information explicitly provided by the user."
)

type ToolBasedPatchGenerator[T any] struct {
	chain *structured.Chain[*types.ToolRequest[T], UpdateFormArgs]
}

func NewToolBasedPatchGenerator[T any](chatModel model.ToolCallingChatModel) (*ToolBasedPatchGenerator[T], error) {
	chain, err := structured.NewChain[*types.ToolRequest[T], UpdateFormArgs](
		chatModel,
		buildPatchPrompt[T],
		updateFormToolName,
		updateFormToolDescription,
	)
	if err != nil {
		return nil, err
	}
	return &ToolBasedPatchGenerator[T]{chain: chain}, nil
}

func (g *ToolBasedPatchGenerator[T]) GeneratePatch(ctx context.Context, req *types.ToolRequest[T]) (*UpdateFormArgs, error) {
	result, err := g.chain.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	return result, nil
}

func buildPatchPrompt[T any](ctx context.Context, req *types.ToolRequest[T]) ([]*einoSchema.Message, error) {
	message, err := types.FormatToolRequest(req)
	if err != nil {
		return nil, fmt.Errorf("convert to prompt message failed: %w", err)
	}

	systemPrompt := fmt.Sprintf(`You are a form assistant. Analyze user input and call '%s' to generate RFC6902 JSON Patch operations.

General rules:
- Only use information explicitly provided by the user in this turn. Do not infer or guess.
- Output only valid RFC6902 JSON Patch operations. If there is nothing to update, return an empty operations list.
- Do NOT include unchanged fields. Do NOT include operations with empty/unknown values unless the user explicitly says so.

RFC6902 / JSON Patch rules (MUST follow):
1) Operation types:
   - Use "add" to create a missing field/path or to set a field that is not currently present in the JSON document.
   - Use "replace" only when the target path already exists in the current JSON document.
     IMPORTANT: If a field may be absent due to "omitempty" or never set before, you MUST use "add" instead of "replace".
   - Use "remove" only if the user explicitly asks to delete/clear a field (e.g., "delete", "remove", "clear").
   - Use "test" only if absolutely necessary (usually not needed). Prefer not to use "move" or "copy".

2) Path / JSON Pointer (RFC6901):
   - "path" MUST be a JSON Pointer starting with "/" (e.g., "/title", "/address/city").
   - Escape special characters in keys: "~" => "~0", "/" => "~1".
   - Do not invent paths not present in the provided form schema.

3) Add vs Replace semantics:
   - For objects: "add" at "/a/b" creates key "b" under object "a" if it does not exist.
   - "replace" requires the member to already exist; otherwise it will fail ("doc is missing key").
   - Therefore: when in doubt between "add" and "replace", choose "add" to avoid failure.

4) Arrays:
   - To append to an array, use "add" with path "/array/-".
   - To set by index, use "replace" or "add" at "/array/0" only if that index exists (replace) or is valid per RFC6902 (add can insert).
   - Avoid complex array edits unless the user clearly specifies them.

5) Value typing:
   - The "value" MUST be a valid JSON value matching the schema type (string/number/boolean/object/array/null).
   - Do not stringify numbers/booleans.
   - Use null only if the user explicitly wants null/empty AND the schema allows it.

6) Minimization & determinism:
   - Produce the minimal set of operations needed.
   - If the user provides multiple updates, output them in a stable order (e.g., top-to-bottom by path).

Context:
- The form schema is provided below.
- The form is currently being edited; ignore "required" constraints for now.
`, updateFormToolName)

	return []*einoSchema.Message{
		einoSchema.SystemMessage(systemPrompt),
		einoSchema.UserMessage(message),
	}, nil
}
