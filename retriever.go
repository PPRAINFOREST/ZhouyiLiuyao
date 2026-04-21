package main

// import (
// 	"context"
// 	"fmt"

// 	"github.com/cloudwego/eino-ext/components/embedding/ark"
// 	milvus2_retriever "github.com/cloudwego/eino-ext/components/retriever/milvus2"
// 	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
// )

// func NewArkRetriever(ctx context.Context, searchMode *search_mode.Approximate, embedder *ark.Embedder) *milvus2_retriever.Retriever {
// 	ret, err := milvus2_retriever.NewRetriever(ctx, &milvus2_retriever.RetrieverConfig{
// 		Client:       MilvusCli,
// 		Collection:   "Ci",
// 		Embedding:    embedder,
// 		VectorField:  "vector",
// 		TopK:         2,
// 		SearchMode:   searchMode,
// 		OutputFields: []string{"id", "content", "metadata"},
// 	})
// 	if err != nil {
// 		panic(fmt.Sprintf("failed to create retriever: %v", err))
// 	}
// 	return ret
// }
