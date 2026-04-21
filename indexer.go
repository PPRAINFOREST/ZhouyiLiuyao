package main

// import (
// 	"context"
// 	"fmt"

// 	"github.com/cloudwego/eino-ext/components/embedding/ark"
// 	milvus2_indexer "github.com/cloudwego/eino-ext/components/indexer/milvus2"
// )

// func NewArkIndexer(ctx context.Context, embedder *ark.Embedder) *milvus2_indexer.Indexer {
// 	indexer, err := milvus2_indexer.NewIndexer(ctx, &milvus2_indexer.IndexerConfig{
// 		Client:              MilvusCli,
// 		Collection:          "Ci",
// 		EnableDynamicSchema: false,
// 		Embedding:           embedder,
// 		Vector: &milvus2_indexer.VectorConfig{
// 			VectorField: "vector",
// 		},
// 	})
// 	if err != nil {
// 		panic(fmt.Sprintf("failed to create indexer: %v", err))
// 	}
// 	if indexer == nil {
// 		panic("indexer is nil")
// 	}
// 	return indexer
// }
