package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/agent"
	"github.com/tbxark/formagent/types"
)

func main() {
	conf := flag.String("config", "config.json", "path to config file")
	flag.Parse()
	config, err := loadConfig(*conf)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	err = startApp(context.Background(), config)
	if err != nil {
		log.Fatalf("start app: %v", err)
	}
}

type agentStateKey struct{}

func startApp(ctx context.Context, config *Config) error {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	key := agentStateKey{}
	ctx = context.WithValue(ctx, key, "invoice")
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  config.APIKey,
		Model:   config.Model,
		BaseURL: config.BaseURL,
	})
	if err != nil {
		return err
	}
	keygen := agent.KeyGen(func(ctx context.Context) (string, bool) {
		value := ctx.Value(key)
		if str, ok := value.(string); ok {
			return str, true
		}
		return "", false
	})
	historyStore := agent.NewStore[[]*schema.Message](
		agent.NewMemoryCore[[]*schema.Message](),
		"invoice:history:",
		keygen,
	)
	stateStore := agent.NewStore[*agent.State[*Invoice]](
		agent.NewMemoryCore[*agent.State[*Invoice]](),
		"invoice:state:",
		keygen,
	)
	historyManager := agent.NewHistoryStore(historyStore)
	stateManager := agent.NewStateStore[*Invoice](
		stateStore,
		func(ctx context.Context) *Invoice {
			return &Invoice{}
		},
	)
	spec := &InvoiceFormSpec{}
	specSchema, _ := spec.JsonSchema()
	flow, err := agent.NewToolBasedFormFlow[*Invoice](spec, cm)
	if err != nil {
		return err
	}
	formAgent := agent.NewAgent(
		"InvoiceFiller",
		"An agent that helps users fill and submit invoice forms via conversation",
		specSchema,
		flow,
		stateManager,
	)
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: formAgent,
	})
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("欢迎使用报销助手，请输入您的需求（如：我要报销差旅费）：")
	for {
		fmt.Print("用户: ")
		input, rErr := reader.ReadString('\n')
		if rErr != nil {
			fmt.Println("输入错误或已结束。退出。")
			break
		}
		input = strings.TrimSpace(input)
		history, rErr := historyManager.Append(ctx, schema.UserMessage(input))
		if rErr != nil {
			return rErr
		}
		iter := runner.Run(ctx, history)
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				return event.Err
			}
			msg, mErr := event.Output.MessageOutput.GetMessage()
			if mErr != nil {
				return mErr
			}
			if _, apErr := historyManager.Append(ctx, msg); apErr != nil {
				return apErr
			}
			state, mErr := stateManager.Load(ctx)
			if mErr != nil {
				return mErr
			}
			if state.Phase == types.PhaseConfirmed || state.Phase == types.PhaseCancelled {
				_ = historyManager.Clear(ctx)
				_ = stateManager.Clear(ctx)
			}
			fmt.Printf("\n助手: %v\n======\n", msg.Content)
		}
	}
	return nil
}
