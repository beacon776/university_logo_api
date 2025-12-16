package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 预定义的颜色名称映射（可以根据您的需求扩展）
var colorNames = map[string]string{
	"white":   "FFFFFF",
	"black":   "000000",
	"red":     "FF0000",
	"green":   "008000", // 注意：Web Green 是 008000
	"blue":    "0000FF",
	"yellow":  "FFFF00",
	"cyan":    "00FFFF",
	"magenta": "FF00FF",
	// 更多颜色...
}

// 用于匹配 rgb(r, g, b) 格式的正则表达式
var rgbRegex = regexp.MustCompile(`rgb\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*\)`)

// NormalizeColor 统一将各种颜色格式转换为 6 位大写十六进制字符串 (RRGGBB)
func NormalizeColor(bgColor string) string {
	// 1. 统一转为小写并去除空格，方便匹配
	cleanBgColor := strings.ToLower(strings.TrimSpace(bgColor))

	// A. 尝试处理颜色名称 (e.g., "blue")
	if hex, ok := colorNames[cleanBgColor]; ok {
		return hex
	}

	// B. 尝试处理 RGB 格式 (e.g., "rgb(255, 0, 128)")
	matches := rgbRegex.FindStringSubmatch(cleanBgColor)
	if len(matches) == 4 {
		// 提取 R, G, B 分量
		r, _ := strconv.Atoi(matches[1])
		g, _ := strconv.Atoi(matches[2])
		b, _ := strconv.Atoi(matches[3])

		// 确保值在 0-255 范围内
		if r >= 0 && r <= 255 && g >= 0 && g <= 255 && b >= 0 && b <= 255 {
			// 转换为 RRGGBB 大写十六进制
			return fmt.Sprintf("%02X%02X%02X", r, g, b)
		}
	}

	// C. 尝试处理简化的 RGB 格式 (e.g., "255,0,128") - 备用/可选
	parts := strings.Split(cleanBgColor, ",")
	if len(parts) == 3 {
		// ... (可以添加逻辑来解析逗号分隔的 RGB，逻辑与上面类似)
	}

	// D. 处理十六进制格式 (e.g., "#FF0000", "F00", "ff0000")
	hexColor := strings.TrimPrefix(cleanBgColor, "#")

	// 统一为大写
	hexColor = strings.ToUpper(hexColor)

	// 6 位十六进制 (RRGGBB)
	if len(hexColor) == 6 {
		return hexColor
	}

	// 3 位十六进制 (RGB -> RRGGBB 扩展)
	if len(hexColor) == 3 {
		// e.g. "F00" -> "FF0000"
		r := string(hexColor[0])
		g := string(hexColor[1])
		b := string(hexColor[2])
		return r + r + g + g + b + b
	}

	// 4. 如果所有格式都不符合，返回一个默认值或空字符串，确保哈希稳定
	return "" // 返回空字符串意味着该参数将不参与哈希或使用一个固定的默认值
}
