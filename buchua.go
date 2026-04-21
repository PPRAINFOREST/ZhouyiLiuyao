package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// Yao 爻的结构体定义
type Yao struct {
	IsChange bool // 是否为老爻（变爻）
	IsYang   bool // 是否为阳爻
}

// String 返回爻的字符串表示
func (y Yao) String() string {
	if y.IsYang {
		if y.IsChange {
			return "老阳 (九)"
		}
		return "少阳 (七)"
	}
	if y.IsChange {
		return "老阴 (六)"
	}
	return "少阴 (八)"
}

// numberToYao 将数字结果转换为爻
func numberToYao(num int) Yao {
	switch num {
	case 9:
		return Yao{IsChange: true, IsYang: true} // 老阳
	case 7:
		return Yao{IsChange: false, IsYang: true} // 少阳
	case 8:
		return Yao{IsChange: false, IsYang: false} // 少阴
	case 6:
		return Yao{IsChange: true, IsYang: false} // 老阴
	default:
		return Yao{} // 默认情况
	}
}

type StreamCallback func(message string)

// randNormInt 生成正态分布的随机整数，范围在 [min, max] 之间
// 使用 Box-Muller 变换生成标准正态分布，然后映射到目标范围
// 使用拒绝采样（Rejection Sampling）确保值在范围内
func randNormInt(r *rand.Rand, min, max int) int {
	// 计算均值和标准差
	mean := float64(min+max) / 2.0
	stdDev := float64(max-min) / 3.0

	for {
		norm := r.NormFloat64()
		scaled := mean + norm*stdDev
		result := int(math.Round(scaled))
		if result >= min && result <= max {
			return result
		}
	}
}

// DiceBuchguaStream 揲蓍布卦主函数（流式输出版本）
func DiceBuchguaStream(callback StreamCallback, question string) []Yao {
	var gua []Yao

	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 输出占卜祈词
	callback("\n下面我将为您占卜，请您面向北方，并将屏幕置于面前……\n")
	time.Sleep(1500 * time.Millisecond)
	callback("假尔泰筮有常，假尔泰筮有常，")
	time.Sleep(500 * time.Millisecond)
	if question != "" {
		callback(fmt.Sprintf("今以%s未知。", question))
		time.Sleep(500 * time.Millisecond)
	}
	callback("爰质所疑，于神与灵，吉凶得失，悔吝忧虞，惟尔有神，尚明告之。\n")
	time.Sleep(500 * time.Millisecond)
	callback(strings.Repeat("=", 50) + "\n\n")

	for i := range 6 {
		yaoName := GetYaoName(i)
		callback(fmt.Sprintf("【%s起卦】\n", yaoName))

		rest := 49
		var bian int

		// 三变
		for bianCount := 1; bianCount <= 3; bianCount++ {
			callback(fmt.Sprintf("  第%d揲：分二、挂一、揲四、归奇...\n", bianCount))
			x := randNormInt(localRand, 1, rest-1)

			if bianCount == 1 {
				bian = (x-1)%4 + (47-x)%4 + 3
				rest = 49 - bian
			} else {
				bian = (x-1)%4 + (rest-2-x)%4 + 3
				rest = rest - bian
			}

			callback(fmt.Sprintf("    挂揲数: %d，剩余: %d\n\n", bian, rest))
		}

		// 确定爻的数值 (9, 8, 7, 6)
		resultNum := rest / 4
		yao := numberToYao(resultNum)
		symbol := "--  --"
		if yao.IsYang {
			symbol = "------"
		}

		callback(fmt.Sprintf("  %s成：%s %s\n\n", yaoName, symbol, yao.String()))
		gua = append(gua, yao)
		time.Sleep(500 * time.Millisecond)
	}

	callback(strings.Repeat("=", 50) + "\n")
	callback("卦象已完成！\n\n")

	return gua
}

// GetFormattedYaoName 获取格式化的爻名（如"初七"、"六二"、"九三"、"六四"、"八五"、"上八"）
// 参数：index是数组索引（0-5），yao是爻
func GetFormattedYaoName(index int, yao Yao) string {
	positionNames := []string{"初", "二", "三", "四", "五", "上"}
	position := positionNames[index]

	// 获取爻的数字
	num := ""
	if yao.IsChange && yao.IsYang {
		num = "九" // 老阳
	} else if yao.IsYang {
		num = "七" // 少阳
	} else if yao.IsChange && !yao.IsYang {
		num = "六" // 老阴
	} else {
		num = "八" // 少阴
	}

	if index == 5 || index == 0 {
		return fmt.Sprintf("%s%s", position, num)
	}
	return fmt.Sprintf("%s%s", num, position)
}

// FormatGua 将爻数组格式化为可视化的卦象
func FormatGua(gua []Yao) string {
	var result strings.Builder
	result.WriteString("卦象如下:\n")

	for i := 5; i >= 0; i-- {
		yao := gua[i]
		yaoName := GetFormattedYaoName(i, yao)
		symbol := "--  --"
		if yao.IsYang {
			symbol = "------"
		}

		fmt.Fprintf(&result, "  %s: %s\n", yaoName, symbol)
	}

	return result.String()
}

// CalculateBianGua 计算变卦，变卦是将所有变爻的阴阳逆转后的卦
func CalculateBianGua(gua []Yao) []Yao {
	bianGua := make([]Yao, len(gua))
	for i, yao := range gua {
		if yao.IsChange {
			bianGua[i] = Yao{IsChange: false, IsYang: !yao.IsYang}
		} else {
			bianGua[i] = Yao{IsChange: false, IsYang: yao.IsYang}
		}
	}
	return bianGua
}

// GetUnchangingYaoPositions 获取不变爻的位置
func GetUnchangingYaoPositions(gua []Yao) []int {
	var positions []int
	for i, yao := range gua {
		if !yao.IsChange {
			positions = append(positions, i)
		}
	}
	return positions
}

// GetGuaBinary 获取卦象的二进制表示（0为阴，1为阳）
func GetGuaBinary(gua []Yao) string {
	var binary strings.Builder
	for _, yao := range gua {
		if yao.IsYang {
			binary.WriteString("1")
		} else {
			binary.WriteString("0")
		}
	}
	return binary.String()
}

// HasChangingYao 检查卦象是否有变爻
func HasChangingYao(gua []Yao) bool {
	for _, yao := range gua {
		if yao.IsChange {
			return true
		}
	}
	return false
}

// GetChangingYaoPositions 获取变爻的位置
func GetChangingYaoPositions(gua []Yao) []int {
	var positions []int
	for i, yao := range gua {
		if yao.IsChange {
			positions = append(positions, i)
		}
	}
	return positions
}

// GetFormattedChangingPositions 获取格式化的变爻位置（将索引0-5转换为1-6）
func GetFormattedChangingPositions(positions []int) string {
	var result strings.Builder
	for i, pos := range positions {
		if i > 0 {
			result.WriteString(" ")
		}
		fmt.Fprintf(&result, "%d", pos+1)
	}
	return result.String()
}

// GetYaoName 根据索引获取爻的名称
func GetYaoName(index int) string {
	yaoNames := []string{"初爻", "二爻", "三爻", "四爻", "五爻", "上爻"}
	return yaoNames[index]
}

// 八卦映射
var trigramMap = map[string]string{
	"111": "乾", "110": "兑", "101": "离", "100": "震",
	"011": "巽", "010": "坎", "001": "艮", "000": "坤",
}

// GetTrigramName 获取八卦名称（根据三爻的二进制表示）
func GetTrigramName(binary string) string {
	if name, exists := trigramMap[binary]; exists {
		return name
	}
	return "未知"
}

// getTrigramBinary 从卦象中提取指定位置的三爻二进制
func getTrigramBinary(gua []Yao, start, end int) string {
	var binary strings.Builder
	for i := start; i < end; i++ {
		if gua[i].IsYang {
			binary.WriteString("1")
		} else {
			binary.WriteString("0")
		}
	}
	return binary.String()
}

// GetLowerTrigram 获取下卦
func GetLowerTrigram(gua []Yao) string {
	return GetTrigramName(getTrigramBinary(gua, 0, 3))
}

// GetUpperTrigram 获取上卦
func GetUpperTrigram(gua []Yao) string {
	return GetTrigramName(getTrigramBinary(gua, 3, 6))
}

// GetHuGua 获取互卦
func GetHuGua(gua []Yao) (string, string) {
	lowerBinary := getTrigramBinary(gua, 1, 4)
	upperBinary := getTrigramBinary(gua, 2, 5)
	return GetTrigramName(lowerBinary), GetTrigramName(upperBinary)
}

// AnalyzeBianGuaImpact 分析变爻对卦象的影响
func AnalyzeBianGuaImpact(gua []Yao) string {
	changingPositions := GetChangingYaoPositions(gua)
	if len(changingPositions) == 0 {
		return ""
	}
	// 获取最下方的变爻
	lowestChangingYao := changingPositions[0]
	modifiedGua := make([]Yao, len(gua))
	copy(modifiedGua, gua)
	modifiedGua[lowestChangingYao].IsYang = !gua[lowestChangingYao].IsYang

	var binary string
	if lowestChangingYao < 3 {
		binary = getTrigramBinary(modifiedGua, 0, 3)
	} else {
		binary = getTrigramBinary(modifiedGua, 3, 6)
	}
	return GetTrigramName(binary)
}

// WuXing 五行
type WuXing string

const (
	WuXingJin  WuXing = "金"
	WuXingMu   WuXing = "木"
	WuXingShui WuXing = "水"
	WuXingHuo  WuXing = "火"
	WuXingTu   WuXing = "土"
)

// 八卦到五行的映射
var wuXingMap = map[string]WuXing{
	"乾": WuXingJin, "兑": WuXingJin,
	"坤": WuXingTu, "艮": WuXingTu,
	"震": WuXingMu, "巽": WuXingMu,
	"坎": WuXingShui, "离": WuXingHuo,
}

// GetWuXingFromTrigram 根据八卦获取五行
func GetWuXingFromTrigram(trigramName string) WuXing {
	if wuXing, exists := wuXingMap[trigramName]; exists {
		return wuXing
	}
	return ""
}

// TrigramStats 八卦统计（分别统计每个八卦的数量）
type TrigramStats struct {
	QianCount int // 乾金
	DuiCount  int // 兑金
	ZhenCount int // 震木
	XunCount  int // 巽木
	KanCount  int // 坎水
	LiCount   int // 离火
	KunCount  int // 坤土
	GenCount  int // 艮土
}

// AnalyzeWuXing 分析卦象的五行（分别统计每个八卦）
func AnalyzeWuXing(gua []Yao, hasChangingYao bool) TrigramStats {
	var stats TrigramStats

	// 本卦
	stats.addTrigram(GetUpperTrigram(gua))
	stats.addTrigram(GetLowerTrigram(gua))

	// 互卦
	huLower, huUpper := GetHuGua(gua)
	stats.addTrigram(huLower)
	stats.addTrigram(huUpper)

	// 变爻影响
	if hasChangingYao {
		if bianImpact := AnalyzeBianGuaImpact(gua); bianImpact != "" {
			stats.addTrigram(bianImpact)
		}
	}

	return stats
}

// addTrigram 添加八卦到统计
func (s *TrigramStats) addTrigram(trigramName string) {
	switch trigramName {
	case "乾":
		s.QianCount++
	case "兑":
		s.DuiCount++
	case "震":
		s.ZhenCount++
	case "巽":
		s.XunCount++
	case "坎":
		s.KanCount++
	case "离":
		s.LiCount++
	case "坤":
		s.KunCount++
	case "艮":
		s.GenCount++
	}
}

// GetWuXingCount 获取某一五行的总数量
func (s *TrigramStats) GetWuXingCount(wuXing WuXing) int {
	switch wuXing {
	case WuXingJin:
		return s.QianCount + s.DuiCount
	case WuXingMu:
		return s.ZhenCount + s.XunCount
	case WuXingShui:
		return s.KanCount
	case WuXingHuo:
		return s.LiCount
	case WuXingTu:
		return s.KunCount + s.GenCount
	}
	return 0
}

// GetBestLunarMonth 根据五行统计推荐最佳农历月份
// 木1月2月，火4月5月，金7月8月，水10月11月，土3月6月9月12月
func (s *TrigramStats) GetBestLunarMonth() string {
	jinCount := s.GetWuXingCount(WuXingJin)
	muCount := s.GetWuXingCount(WuXingMu)
	shuiCount := s.GetWuXingCount(WuXingShui)
	huoCount := s.GetWuXingCount(WuXingHuo)
	tuCount := s.GetWuXingCount(WuXingTu)

	maxCount := 0
	var bestWuXing WuXing

	if jinCount > maxCount {
		maxCount, bestWuXing = jinCount, WuXingJin
	}
	if muCount > maxCount {
		maxCount, bestWuXing = muCount, WuXingMu
	}
	if shuiCount > maxCount {
		maxCount, bestWuXing = shuiCount, WuXingShui
	}
	if huoCount > maxCount {
		maxCount, bestWuXing = huoCount, WuXingHuo
	}
	if tuCount > maxCount {
		maxCount, bestWuXing = tuCount, WuXingTu
	}

	monthMap := map[WuXing]string{
		WuXingMu:   "1月、2月（木旺之月）",
		WuXingHuo:  "4月、5月（火旺之月）",
		WuXingJin:  "7月、8月（金旺之月）",
		WuXingShui: "10月、11月（水旺之月）",
		WuXingTu:   "3月、6月、9月、12月（土旺之月）",
	}

	if month, exists := monthMap[bestWuXing]; exists {
		return month
	}
	return "无明显优势五行，诸事皆宜"
}

// String 格式化八卦统计信息
func (s *TrigramStats) String() string {
	var result strings.Builder
	result.WriteString("五行分析：\n")

	items := []struct {
		count int
		name  string
	}{
		{s.QianCount, "乾金"},
		{s.DuiCount, "兑金"},
		{s.ZhenCount, "震木"},
		{s.XunCount, "巽木"},
		{s.KanCount, "坎水"},
		{s.LiCount, "离火"},
		{s.KunCount, "坤土"},
		{s.GenCount, "艮土"},
	}

	for _, item := range items {
		if item.count > 0 {
			fmt.Fprintf(&result, "  %s：%d个\n", item.name, item.count)
		}
	}

	return result.String()
}

// GetTrigramAnalysis 获取完整的卦象分析
func GetTrigramAnalysis(gua []Yao) string {
	var result strings.Builder
	result.WriteString("卦象结构分析：\n")

	// 本卦
	result.WriteString("【本卦】\n")
	fmt.Fprintf(&result, "  上卦：%s卦\n", GetUpperTrigram(gua))
	fmt.Fprintf(&result, "  下卦：%s卦\n", GetLowerTrigram(gua))

	// 互卦
	huLower, huUpper := GetHuGua(gua)
	result.WriteString("【互卦】（潜在发展趋势）\n")
	fmt.Fprintf(&result, "  互卦上卦：%s卦\n", huUpper)
	fmt.Fprintf(&result, "  互卦下卦：%s卦\n", huLower)

	// 变爻影响
	if hasChangingYao := HasChangingYao(gua); hasChangingYao {
		changingPositions := GetChangingYaoPositions(gua)
		bianImpact := AnalyzeBianGuaImpact(gua)
		result.WriteString("【变爻影响】\n")
		fmt.Fprintf(&result, "  原初变爻：%s\n", GetYaoName(changingPositions[0]))
		fmt.Fprintf(&result, "  变化后：%s卦\n", bianImpact)
	}

	return result.String()
}
