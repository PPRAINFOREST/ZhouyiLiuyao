package main

import (
	"context"
	"fmt"
	"os"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

var MilvusCli *milvusclient.Client

// func InitClient() {
// 	// 初始化客户端
// 	ctx := context.Background()
// 	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
// 		Address: "localhost:19530",
// 		DBName:  "GuaCi",
// 	})
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	MilvusCli = cli
// }

func InitClient() {
	ctx := context.Background()

	address := os.Getenv("MILVUS_ADDRESS")
	apiKey := os.Getenv("MILVUS_API_KEY")

	var err error
	MilvusCli, err = milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: address,
		APIKey:  apiKey, // 使用 API Key
	})

	if err != nil {
		panic(err)
	}
	fmt.Println("✅ Milvus 连接成功（Zilliz Cloud）")
}
