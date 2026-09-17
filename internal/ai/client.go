package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client AI API客户端（OpenAI协议）
type Client struct {
	baseURL     string
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *resty.Client
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient 创建AI客户端
func NewClient(baseURL, apiKey, model string, maxTokens int, temperature float64, timeout int) *Client {
	if timeout <= 0 {
		timeout = 120
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	if temperature <= 0 {
		temperature = 0.3
	}

	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetTimeout(time.Duration(timeout) * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(5 * time.Second)

	return &Client{
		baseURL:     baseURL,
		apiKey:      apiKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		httpClient:  httpClient,
	}
}

// ChatCompletion 发送聊天请求
func (c *Client) ChatCompletion(messages []Message) (*ChatResponse, error) {
	req := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	}

	var resp ChatResponse
	httpResp, err := c.httpClient.R().
		SetBody(req).
		SetResult(&resp).
		Post("/chat/completions")

	if err != nil {
		return nil, fmt.Errorf("AI API请求失败: %w", err)
	}

	if httpResp.StatusCode() != 200 {
		return nil, fmt.Errorf("AI API返回错误状态: %d, body: %s", httpResp.StatusCode(), httpResp.String())
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("AI API返回空结果")
	}

	return &resp, nil
}

// ExtractJSON 从AI响应中提取JSON内容
func ExtractJSON(content string) (string, error) {
	// 尝试直接解析
	content = strings.TrimSpace(content)

	// 移除markdown代码块标记
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	// 尝试找到JSON对象
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		jsonStr := content[start : end+1]
		// 验证是否是合法JSON
		var js json.RawMessage
		if err := json.Unmarshal([]byte(jsonStr), &js); err == nil {
			return jsonStr, nil
		}
	}

	return "", fmt.Errorf("无法从AI响应中提取JSON")
}
