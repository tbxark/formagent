package testcases

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/tbxark/formagent"
)

type agentOptions struct {
	commandParser formagent.CommandParser
}

type AgentOption func(*agentOptions)

func WithCommandParser(parser formagent.CommandParser) AgentOption {
	return func(o *agentOptions) {
		o.commandParser = parser
	}
}

type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

func loadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Config
	err = json.Unmarshal(file, &conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

func InitChatModel(t *testing.T) *openai.ChatModel {
	if os.Getenv("FORMAGENT_RUN_LIVE_TESTS") != "1" {
		t.Skip("set FORMAGENT_RUN_LIVE_TESTS=1 to run live LLM tests")
		return nil
	}

	ctx := context.Background()
	conf, err := loadConfig("../config.json")
	if err != nil {
		t.Skipf("failed to load config: %v", err)
		return nil
	}
	if conf.APIKey == "" {
		t.Skip("config.json api_key is empty")
		return nil
	}
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  conf.APIKey,
		Model:   conf.Model,
		BaseURL: conf.BaseURL,
	})
	if err != nil {
		t.Fatalf("failed to init chat model: %v", err)
		return nil
	}
	return chatModel
}

func NewTestAgent(t *testing.T, opts ...AgentOption) *formagent.FormAgent[UserRegistrationForm] {
	chatModel := InitChatModel(t)
	if chatModel == nil {
		return nil
	}

	o := &agentOptions{}
	for _, opt := range opts {
		opt(o)
	}

	if o.commandParser == nil {
		agent, err := formagent.NewToolBasedFormAgent[UserRegistrationForm](context.Background(), &FormSpec{}, chatModel)
		if err != nil {
			t.Fatalf("创建 agent 失败: %v", err)
		}
		return agent
	}

	patchGen, err := formagent.NewToolBasedPatchGenerator[UserRegistrationForm](context.Background(), chatModel)
	if err != nil {
		t.Fatalf("创建 patch generator 失败: %v", err)
	}
	dialogueGen, err := formagent.NewToolBasedDialogueGenerator[UserRegistrationForm](context.Background(), chatModel)
	if err != nil {
		t.Fatalf("创建 dialogue generator 失败: %v", err)
	}
	agent, err := formagent.NewFormAgent(&FormSpec{}, patchGen, dialogueGen, o.commandParser)
	if err != nil {
		t.Fatalf("创建 agent 失败: %v", err)
	}
	return agent
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{BaseURL:%q, Model:%q}", c.BaseURL, c.Model)
}
