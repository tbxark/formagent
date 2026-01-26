package formagent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func getToolInfo(ctx context.Context, t tool.InvokableTool) (*schema.ToolInfo, error) {
	info, err := t.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool info: %w", err)
	}
	return info, nil
}
