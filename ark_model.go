package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
)

func float32Ptr(f float32) *float32 {
	return &f
}

func NewArkModel(ctx context.Context) *ark.ChatModel {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		fmt.Println("请设置环境变量 ARK_API_KEY")
		fmt.Println("export ARK_API_KEY=你的_API_KEY")
		os.Exit(1)
	}
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:      apiKey,
		Model:       os.Getenv("MODEL"),
		Temperature: float32Ptr(0.7), // 控制输出随机性，范围 [0.0, 2.0]
		TopP:        float32Ptr(0.9), // 核采样参数，范围 [0.0, 1.0]
	})
	if err != nil {
		panic(err)
	}
	return chatModel
}
