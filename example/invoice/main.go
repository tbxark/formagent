package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/tbxark/formagent/agent"
)

func main() {
	conf := flag.String("config", "config.json", "path to config file")
	flag.Parse()
	config, err := loadConfig(*conf)
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}
	err = startApp(context.Background(), config)
	if err != nil {
		log.Fatalf("error starting app: %v", err)
	}
}

func startApp(ctx context.Context, config *Config) error {
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  config.APIKey,
		Model:   config.Model,
		BaseURL: config.BaseURL,
	})
	if err != nil {
		return err
	}
	store := agent.NewMemoryStateReadWriter[Invoice]()
	flow, err := agent.NewToolBasedFormFlow[Invoice](
		&InvoiceFormSpec{},
		cm,
		store,
	)
	if err != nil {
		return err
	}
	formTool, err := agent.NewFormFlowInvokableTool(
		"invoice_filler",
		"Fill and submit invoice forms based on user input",
		flow,
	)
	if err != nil {
		return err
	}
	formAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "InvoiceFiller",
		Description: "An agent that helps users fill and submit invoice forms via conversation",
		Instruction: `You are an expert invoice assistant. Guide the user to provide all required invoice information step by step, validate the data, and submit the form using the \"invoice_filler\" tool.`,
		Model:       cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{formTool},
			},
		},
	})
	if err != nil {
		return err
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: formAgent,
	})
	var input string
	fmt.Println("欢迎使用报销助手，请输入您的需求（如：我要报销差旅费）：")
	for {
		fmt.Print("用户: ")
		_, err := fmt.Scanln(&input)
		if err != nil {
			// 处理 EOF 或输入错误
			fmt.Println("输入错误或已结束。退出。")
			break
		}
		iter := runner.Query(ctx, input)
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				log.Fatal(event.Err)
			}
			msg, mErr := event.Output.MessageOutput.GetMessage()
			if mErr != nil {
				log.Fatal(err)
			}
			fmt.Printf("\n助手: %v\n======\n", msg)
		}
	}
	return nil
}
