package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"io"
	"net/http"
	"os"
	"time"
)

func HandleRequest(ctx context.Context, event map[string]interface{}) (string, error) {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		return "", fmt.Errorf("TARGET_URL not set")
	}

	// 设置 http client，带超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 构造请求
	reqBody := []byte("{}")
	resp, err := client.Post(targetURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("http post failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}

	// 返回结果（SCF 会打印日志）
	return fmt.Sprintf("status: %d, body: %s", resp.StatusCode, string(body)), nil
}

func main() {
	// 启动函数入口
	cloudfunction.Start(HandleRequest)
}
