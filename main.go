package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	milvus2_retriever "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

// 全局变量
var (
	globalRetriever *milvus2_retriever.Retriever
	milvusEnabled   bool = false
)

// DivinationResult 后台执行流的结果
type DivinationResult struct {
	guaDoc *schema.Document
	err    error
}

// GuaData 卦数据结构
type GuaData struct {
	Binary         string
	Name           string
	GuaText        string
	GuaExplanation string
}

func initMilvusGlobals(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("⚠️  Milvus 初始化失败: %v\n", r)
			fmt.Printf("   将使用默认方式解卦\n")
			milvusEnabled = false
		}
	}()

	InitClient()
	var err error
	globalRetriever, err = milvus2_retriever.NewRetriever(ctx, &milvus2_retriever.RetrieverConfig{
		Client:       MilvusCli,
		Collection:   "Gua",
		TopK:         1,
		VectorField:  "vector",
		OutputFields: []string{"id", "metadata"},
		SearchMode:   search_mode.NewScalar(),
	})
	if err != nil {
		panic(fmt.Sprintf("创建 retriever 失败: %v", err))
	}

	milvusEnabled = true
	fmt.Println("✅ Milvus 已启用（标量搜索模式）")
}

func searchGuaByBinary(ctx context.Context, binary string) (*schema.Document, error) {
	targetID := "gua_" + binary
	filterExpr := fmt.Sprintf(`id == "%s"`, targetID)

	documents, err := globalRetriever.Retrieve(ctx, filterExpr)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	if len(documents) == 0 {
		return nil, fmt.Errorf("未找到 id 为 %s 的卦", targetID)
	}

	doc := documents[0]

	if metadataStr, ok := doc.MetaData["metadata"].(string); ok {
		var meta GuaData
		err := json.Unmarshal([]byte(metadataStr), &meta)
		if err == nil {
			doc.MetaData["name"] = meta.Name
			doc.MetaData["gua_text"] = meta.GuaText
			doc.MetaData["gua_explanation"] = meta.GuaExplanation
			doc.MetaData["binary"] = meta.Binary
		}
	}

	return doc, nil
}

func createSlowStreamCallback() func(message string) {
	return func(message string) {
		for _, char := range message {
			fmt.Printf("%c", char)
			os.Stdout.Sync()

			switch {
			case char == '\n':
				time.Sleep(30 * time.Millisecond)
			case char >= '0' && char <= '9':
				time.Sleep(10 * time.Millisecond)
			case (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z'):
				time.Sleep(10 * time.Millisecond)
			case char >= 0x4e00 && char <= 0x9fff:
				time.Sleep(25 * time.Millisecond)
			default:
				time.Sleep(5 * time.Millisecond)
			}
		}
	}
}

func printWelcome() {
	fmt.Println("========================================")
	fmt.Println("         周易算卦 Agent (揲蓍布卦法)")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("使用说明：")
	fmt.Println("  1. 输入您的问题或事项")
	fmt.Println("  2. 输入 '算卦'、'起卦'、'占卜' 等关键词进行算卦")
	fmt.Println("  3. 输入 'exit' 或 'quit' 退出程序")
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	chatModel := NewArkModel(ctx)

	// 初始化 Milvus（如果配置了）
	fmt.Println("正在初始化 Milvus...")
	initMilvusGlobals(ctx)

	printWelcome()
	reader := bufio.NewReader(os.Stdin)

	// 主对话循环
	for {
		fmt.Print("\n请输入您的问题或指令: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入失败: %v\n", err)
			continue
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			fmt.Println("感谢使用，再见！")
			break
		}

		// ⭐ 开始计时
		startTime := time.Now()

		if containsDivinationKeywords(input) {
			handleDivination(ctx, chatModel, input)
		} else {
			handleChat(ctx, chatModel, input)
		}

		// ⭐ 输出耗时
		duration := time.Since(startTime)
		seconds := duration.Seconds()

		fmt.Printf("\n⏱️  本次对话耗时: %.2f秒\n", seconds)
	}
}

// GuaAnalysisContext 卦象分析上下文，用于复用计算结果
type GuaAnalysisContext struct {
	binary            string
	upperTrigram      string
	lowerTrigram      string
	huLower           string
	huUpper           string
	hasChangingYao    bool
	wuXingStats       TrigramStats
	bestMonth         string
	changingPositions []int
	changeCount       int
}

// buildGuaAnalysisContext 构建卦象分析上下文（一次性计算所有需要的值）
func buildGuaAnalysisContext(gua []Yao) GuaAnalysisContext {
	binary := GetGuaBinary(gua)
	upperTrigram := GetUpperTrigram(gua)
	lowerTrigram := GetLowerTrigram(gua)
	huLower, huUpper := GetHuGua(gua)
	hasChangingYao := HasChangingYao(gua)
	wuXingStats := AnalyzeWuXing(gua, hasChangingYao)
	bestMonth := wuXingStats.GetBestLunarMonth()
	changingPositions := GetChangingYaoPositions(gua)
	changeCount := len(changingPositions)

	return GuaAnalysisContext{
		binary:            binary,
		upperTrigram:      upperTrigram,
		lowerTrigram:      lowerTrigram,
		huLower:           huLower,
		huUpper:           huUpper,
		hasChangingYao:    hasChangingYao,
		wuXingStats:       wuXingStats,
		bestMonth:         bestMonth,
		changingPositions: changingPositions,
		changeCount:       changeCount,
	}
}

// handleDivination 处理算卦（修复版本）
func handleDivination(ctx context.Context, chatModel *ark.ChatModel, input string) {
	question := extractQuestion(input)

	// 第1步：大模型总结问题（较慢）
	var questionSummary string
	var err error
	if question != "" {
		fmt.Print("\n--- 正在理解您的问题 ---\n\n")
		questionSummary, err = summarizeQuestion(ctx, chatModel, question)
		if err != nil {
			fmt.Printf("理解问题时出错: %v，将使用默认短语\n", err)
			questionSummary = "诸事吉凶"
		}
	}

	// 第2步：随机生成六爻（很快）
	gua := DiceBuchgua(questionSummary)

	// 第3步：一次性计算所有卦象分析（避免重复计算）⭐
	analysisContext := buildGuaAnalysisContext(gua)

	// 第4步：并行执行两个执行流 ⭐
	// 创建 channel 用于同步
	resultChan := make(chan DivinationResult, 1)
	divinationReader, divinationWriter := io.Pipe()
	divinationReadyChan := make(chan bool, 1)

	// 使用 WaitGroup 等待后台解卦完成 ⭐
	var wg sync.WaitGroup

	// 【后台执行流】查询卦辞 -> 调用大模型解卦（流式输出到 pipe）
	wg.Add(1) // 标记后台任务开始
	go func() {
		defer wg.Done() // 标记后台任务完成
		defer divinationWriter.Close()

		// 查询卦辞（较快）
		var guaDoc *schema.Document
		var err error
		if milvusEnabled {
			guaDoc, err = searchGuaByBinary(ctx, analysisContext.binary)
			if err != nil {
				guaDoc = nil
				err = nil // 不影响后续流程
			}
		}

		// 发送卦辞结果到主线程（使用 DivinationResult 结构）⭐
		resultChan <- DivinationResult{guaDoc, err}

		// 等待前台准备好接收流式输出
		<-divinationReadyChan

		// 构建 prompt（使用预先计算的上下文，避免重复计算）⭐
		prompt := buildDivinationPrompt(question, gua, guaDoc, analysisContext)

		// 调用大模型解卦（流式输出到 pipe）⭐
		messages := []*schema.Message{
			schema.SystemMessage("你是一位精通周易、卜卦、传统文化和哲学的算卦大师。你具备深厚的周易知识，能够准确解读卦象，为人们提供有价值的指导。你的回答应该专业、客观、有启发性，同时通俗易懂。并要在解卦后提醒用户占卜并不具有科学依据，仅供娱乐用。"),
			schema.UserMessage(prompt),
		}

		reader, err := chatModel.Stream(ctx, messages)
		if err != nil {
			divinationWriter.Write([]byte(fmt.Sprintf("解卦失败: %v\n", err)))
		} else {
			for {
				chunk, err := reader.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					divinationWriter.Write([]byte(fmt.Sprintf("读取流式输出失败: %v\n", err)))
					break
				}
				if chunk.Content != "" {
					divinationWriter.Write([]byte(chunk.Content))
				}
			}
			reader.Close()
		}
	}()

	// 【前台执行流】输出算卦过程 -> 输出卦象结构分析
	streamCallback := createSlowStreamCallback()

	// 输出算卦过程
	PrintBuchguaProcess(streamCallback, questionSummary, gua)

	// 输出卦象结构分析（使用预先计算的上下文）⭐
	printGuaInfoWithContext(streamCallback, gua, analysisContext)

	// 第5步：等待后台准备好卦辞
	result := <-resultChan

	// 流式输出卦辞
	if milvusEnabled && result.guaDoc != nil {
		fmt.Print("\n--- 正在查询卦辞 ---\n\n")
		slowStreamCallback := createSlowStreamCallback()
		if name, ok := result.guaDoc.MetaData["name"]; ok {
			slowStreamCallback(fmt.Sprintf("   卦名: %s卦\n", name))
		}
		if guaText, ok := result.guaDoc.MetaData["gua_text"]; ok {
			slowStreamCallback(fmt.Sprintf("   卦辞: %s\n", guaText))
		}
		if guaExplanation, ok := result.guaDoc.MetaData["gua_explanation"]; ok {
			slowStreamCallback(fmt.Sprintf("   象/彖: %s\n", guaExplanation))
		}
	}

	// 第6步：流式输出解卦结果
	fmt.Print("\n--- 正在为您解卦 ---\n\n")

	// 通知后台可以开始流式输出了
	divinationReadyChan <- true

	// 从 pipe 中流式读取并输出解卦结果
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := divinationReader.Read(buf)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("\n读取解卦结果失败: %v\n", err)
				}
				break
			}
			if n > 0 {
				fmt.Print(string(buf[:n]))
				os.Stdout.Sync()
			}
		}
		divinationReader.Close()
	}()

	// 等待后台解卦完成 ⭐
	wg.Wait()

	// 确保所有解卦内容都已输出
	time.Sleep(100 * time.Millisecond)
	fmt.Println()
}

// printGuaInfoWithContext 输出卦象信息（使用预先计算的上下文，避免重复计算）⭐
func printGuaInfoWithContext(callback func(string), gua []Yao, ctx GuaAnalysisContext) {
	callback("\n" + strings.Repeat("=", 50) + "\n")
	callback("完整卦象\n")
	callback(strings.Repeat("=", 50) + "\n")
	callback(FormatGua(gua))

	if ctx.hasChangingYao {
		callback(fmt.Sprintf("变爻位置: %v\n", GetFormattedChangingPositions(ctx.changingPositions)))
	} else {
		callback("本卦无变爻\n")
	}

	callback(strings.Repeat("=", 50) + "\n")
	callback("\n" + GetTrigramAnalysis(gua))

	callback(ctx.wuXingStats.String())
	callback(fmt.Sprintf("\n最佳农历月份：%s\n", ctx.bestMonth))
	callback("\n" + strings.Repeat("=", 50) + "\n")
}

func handleChat(ctx context.Context, chatModel *ark.ChatModel, input string) {
	fmt.Print("\n--- 回复 ---\n")
	if err := callChatModelStream(ctx, chatModel, input); err != nil {
		fmt.Printf("对话失败: %v\n", err)
	}
	fmt.Print("\n")
}

func extractQuestion(input string) string {
	keywords := []string{"算卦", "起卦", "占卜", "卜卦", "揲蓍", "布卦", "算一算", "算一下"}
	question := input
	for _, keyword := range keywords {
		question = strings.ReplaceAll(question, keyword, "")
		question = strings.ReplaceAll(question, "帮我", "")
		question = strings.ReplaceAll(question, "请", "")
		question = strings.ReplaceAll(question, "你", "")
	}
	question = strings.TrimSpace(question)
	return question
}

func summarizeQuestion(ctx context.Context, model *ark.ChatModel, question string) (string, error) {
	if question == "" {
		return "", nil
	}

	prompt := fmt.Sprintf(`请将以下问题总结为2-6个汉字的简洁短语，用于周易占卜的祈词。

要求：
1. 只返回2-6个汉字
2. 准确概括问题主题
3. 符合周易占卜的传统语境
4. 不要包含任何其他内容或标点

问题：%s

请直接返回短语：`, question)

	messages := []*schema.Message{
		schema.SystemMessage("你是一个专业的周易占卜助手，擅长将问题总结为简洁的短语。"),
		schema.UserMessage(prompt),
	}

	resp, err := model.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("AI总结失败: %w", err)
	}

	summary := strings.TrimSpace(resp.Content)

	// 如果返回太长，截取前4个字符
	if len([]rune(summary)) > 4 {
		summary = string([]rune(summary)[:4])
	}

	// 如果返回太短，添加默认
	if len([]rune(summary)) < 2 {
		summary = "诸事吉凶"
	}

	return summary, nil
}

func callChatModelStream(ctx context.Context, model *ark.ChatModel, prompt string) error {
	messages := []*schema.Message{
		schema.SystemMessage("你是一位精通周易、卜卦、传统文化和哲学的算卦大师。你具备深厚的周易知识，能够准确解读卦象，为人们提供有价值的指导。你的回答应该专业、客观、有启发性，同时通俗易懂。并要在解卦后提醒用户占卜并不具有科学依据，仅供娱乐用。"),
		schema.UserMessage(prompt),
	}

	reader, err := model.Stream(ctx, messages)
	if err != nil {
		return fmt.Errorf("模型调用失败: %w", err)
	}
	defer reader.Close()

	for {
		chunk, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取流式输出失败: %w", err)
		}

		if chunk.Content != "" {
			fmt.Print(chunk.Content)
			os.Stdout.Sync()
		}
	}
	return nil
}

func containsDivinationKeywords(input string) bool {
	keywords := []string{"算卦", "起卦", "占卜", "卜卦", "揲蓍", "布卦", "算一下", "算一算"}
	input = strings.ToLower(input)
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

// buildDivinationPrompt 构建解卦提示词（使用预先计算的上下文，避免重复计算）⭐
func buildDivinationPrompt(question string, gua []Yao, guaDoc *schema.Document, ctx GuaAnalysisContext) string {
	yaoDetails := buildYaoDetails(gua)
	unchangingPositions := GetUnchangingYaoPositions(gua)
	bianGua := CalculateBianGua(gua)
	bianBinary := GetGuaBinary(bianGua)

	questionText := "无具体问题"
	if question != "" {
		questionText = question
	}

	additionalInfo := buildAdditionalInfo(ctx.huLower, ctx.huUpper, ctx.wuXingStats, ctx.bestMonth)
	ruleText := buildRuleText(ctx.changeCount, ctx.changingPositions, unchangingPositions, bianBinary)

	// 构建卦辞部分
	var guaCiSection string
	if guaDoc != nil {
		guaCiSection = "【卦辞原文】\n"
		if name, ok := guaDoc.MetaData["name"]; ok {
			guaCiSection += fmt.Sprintf("卦名：%s卦\n", name)
		}
		if guaText, ok := guaDoc.MetaData["gua_text"]; ok {
			guaCiSection += fmt.Sprintf("卦辞：%s\n", guaText)
		}
		if guaExplanation, ok := guaDoc.MetaData["gua_explanation"]; ok {
			guaCiSection += fmt.Sprintf("象/彖：%s\n", guaExplanation)
		}
		guaCiSection += "\n"
	} else {
		guaCiSection = ""
	}

	return fmt.Sprintf(`你是一位精通周易的算卦大师。请根据以下揲蓍布卦的结果，为用户的问题提供专业的解卦指导。

【用户问题】
%s

【本卦信息】
本卦二进制（从上至下，上爻到初爻）: %s
本卦上卦：%s卦
本卦下卦：%s卦

本卦各爻详情:
%s

%s
%s

%s

【解卦要求】
1. 根据本卦二进制代码，识别出具体的卦名（如乾卦、坤卦等）
2. 根据解卦规则，确定参考的卦辞和爻辞
3. 像用户列出参考的卦辞和爻辞的原文
4. 结合周易的卦辞、爻辞和象辞，深入解析卦象的含义
5. 结合互卦分析事物发展的潜在趋势
6. 结合五行分析，说明五行元素对卦象的影响
7. 将卦象的含义与用户的问题紧密结合，提供有针对性的指导和建议
8. 解释卦象的吉凶属性和注意事项
9. 给出行动建议和启示

请以专业、客观、有启发性的方式回答。并在回答结束后提醒用户占卜没有科学依据，仅供娱乐参考`, questionText, ctx.binary, ctx.upperTrigram, ctx.lowerTrigram, yaoDetails, guaCiSection, additionalInfo, ruleText)
}

func buildYaoDetails(gua []Yao) string {
	var yaoDetails strings.Builder
	for i := 5; i >= 0; i-- {
		fmt.Fprintf(&yaoDetails, "%s: %s\n", GetYaoName(i), gua[i].String())
	}
	return yaoDetails.String()
}

func buildAdditionalInfo(huLower, huUpper string, wuXingStats TrigramStats, bestMonth string) string {
	var additionalInfo strings.Builder
	additionalInfo.WriteString("【互卦信息】（事物发展的潜在趋势）\n")
	additionalInfo.WriteString(fmt.Sprintf("  互卦上卦：%s卦\n", huUpper))
	additionalInfo.WriteString(fmt.Sprintf("  互卦下卦：%s卦\n", huLower))
	additionalInfo.WriteString("  互卦说明：互卦代表事物发展的中间过程和潜在趋势，需要结合本卦综合考虑。\n\n")
	additionalInfo.WriteString("【五行分析】\n")
	additionalInfo.WriteString(wuXingStats.String())
	additionalInfo.WriteString(fmt.Sprintf("\n最佳农历月份：%s\n", bestMonth))
	return additionalInfo.String()
}

func buildRuleText(changeCount int, changingPositions, unchangingPositions []int, bianBinary string) string {
	switch changeCount {
	case 0:
		return `【解卦规则】
此卦无变爻。
解卦方法：仅参考本卦的卦辞进行解卦。

请重点解析本卦的卦辞含义，并结合用户的问题提供指导。`

	case 1:
		changeYaoPos := changingPositions[0]
		return fmt.Sprintf(`【解卦规则】
此卦有%d个变爻，变爻位置：%s。
解卦方法：参考本卦的卦辞以及%s的爻辞进行解卦。

请同时解析本卦卦辞和%s爻辞的含义，重点关注变爻爻辞的指示。`, changeCount, GetYaoName(changeYaoPos), GetYaoName(changeYaoPos), GetYaoName(changeYaoPos))

	case 2:
		changeYao1, changeYao2 := changingPositions[0], changingPositions[1]
		return fmt.Sprintf(`【解卦规则】
此卦有%d个变爻，变爻位置：%s、%s。
解卦方法：参考本卦的卦辞以及%s的爻辞作为主要参考，%s的爻辞作为次级参考。

请优先解析本卦卦辞和%s爻辞，同时将%s爻辞的内容作为补充和印证。`, changeCount, GetYaoName(changeYao1), GetYaoName(changeYao2), GetYaoName(changeYao1), GetYaoName(changeYao2), GetYaoName(changeYao1), GetYaoName(changeYao2))

	case 3:
		return fmt.Sprintf(`【解卦规则】
此卦有%d个变爻，变爻位置：%v。
解卦方法：参考本卦的卦辞和变卦（之卦）的卦辞进行解卦。

【变卦信息】
变卦二进制: %s
变卦说明：变卦是将所有变爻的阴阳逆转后得到的卦象。

请同时解析本卦和变卦的卦辞含义，理解从本卦到变卦的变化趋势。`, changeCount, changingPositions, bianBinary)

	case 4:
		unchangingYao1, unchangingYao2 := unchangingPositions[0], unchangingPositions[1]
		return fmt.Sprintf(`【解卦规则】
此卦有%d个变爻，变爻位置：%v。
解卦方法：参考变卦的卦辞作为主要依据，并参考两个不变爻中%s的爻辞作为次级参考，%s的爻辞作为补充。

【变卦信息】
变卦二进制: %s
变卦说明：变卦是将所有变爻的阴阳逆转后得到的卦象。

请重点解析变卦的卦辞，同时考虑%s爻辞和%s爻辞的指示。`, changeCount, changingPositions, GetYaoName(unchangingYao1), GetYaoName(unchangingYao2), bianBinary, GetYaoName(unchangingYao1), GetYaoName(unchangingYao2))

	case 5:
		unchangingYao := unchangingPositions[0]
		return fmt.Sprintf(`【解卦规则】
此卦有%d个变爻，变爻位置：%v。
解卦方法：参考变卦的卦辞以及变卦中%s的爻辞进行解卦。

【变卦信息】
变卦二进制: %s
变卦说明：变卦是将所有变爻的阴阳逆转后得到的卦象。

请同时解析变卦的卦辞和%s爻辞的含义，重点关注变卦的整体趋势。`, changeCount, changingPositions, GetYaoName(unchangingYao), bianBinary, GetYaoName(unchangingYao))

	default:
		return fmt.Sprintf(`【解卦规则】
此卦有%d个变爻（全变），变爻位置：%v。
解卦方法：仅参考变卦（之卦）的卦辞进行解卦。

【变卦信息】
变卦二进制: %s
变卦说明：变卦是将所有变爻的阴阳逆转后得到的卦象。此卦为全变卦，所有爻都发生了变化。

请重点解析变卦的卦辞含义，理解完全变化后的局势和趋势。`, changeCount, changingPositions, bianBinary)
	}
}
