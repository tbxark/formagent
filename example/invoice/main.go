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
	"github.com/tbxark/formagent/agent"
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

func startApp(ctx context.Context, config *Config) error {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  config.APIKey,
		Model:   config.Model,
		BaseURL: config.BaseURL,
	})
	if err != nil {
		return err
	}
	store := agent.NewMemoryStateReadWriter[*Invoice](func(ctx context.Context) *Invoice {
		return &Invoice{}
	})
	manager := &InvoiceFormManager{}

	flow, err := agent.NewToolBasedFormFlow[*Invoice](
		&InvoiceFormSpec{},
		cm,
	)
	if err != nil {
		return err
	}
	formAgent := agent.NewAgent(
		"InvoiceFiller",
		"An agent that helps users fill and submit invoice forms via conversation",
		flow,
		store,
		manager,
	)
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: formAgent,
	})
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("欢迎使用报销助手，请输入您的需求（如：我要报销差旅费）：")
	for {
		fmt.Print("用户: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			// 处理 EOF 或输入错误
			fmt.Println("输入错误或已结束。退出。")
			break
		}
		input = strings.TrimSpace(input)
		iter := runner.Query(ctx, input)
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
			fmt.Printf("\n助手: %v\n======\n", msg.Content)
		}
	}
	return nil
}
