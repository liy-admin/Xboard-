package notify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TelegramClient 对应 TelegramService：所有调用均为 GET 查询参数，校验 ok 字段后返回 result。
type TelegramClient struct {
	Token   string
	BaseURL string // 默认 https://api.telegram.org，测试时可替换
	HTTP    *http.Client
}

// NewTelegram 创建客户端。token 为 telegram_bot_token。
func NewTelegram(token string) *TelegramClient {
	return &TelegramClient{Token: token, BaseURL: "https://api.telegram.org", HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// EscapeMarkdown 对应 sendMessage 中 markdown 模式下的下划线转义。
func EscapeMarkdown(text string) string { return strings.ReplaceAll(text, "_", "\\_") }

// SendMessage 发送文本消息，parseMode 为空、"markdown" 或 "HTML"。
func (c *TelegramClient) SendMessage(chatID int64, text, parseMode string) error {
	if parseMode == "markdown" {
		text = EscapeMarkdown(text)
	}
	params := url.Values{"chat_id": {fmt.Sprint(chatID)}, "text": {text}}
	if parseMode != "" {
		params.Set("parse_mode", parseMode)
	}
	_, err := c.Request("sendMessage", params)
	return err
}

// GetMe 返回机器人信息（/user/telegram/getBotInfo 使用）。
func (c *TelegramClient) GetMe() (json.RawMessage, error) { return c.Request("getMe", nil) }

// SetWebhook 设置 webhook 地址。
func (c *TelegramClient) SetWebhook(u string) (json.RawMessage, error) {
	return c.Request("setWebhook", url.Values{"url": {u}})
}

// Request 调用 Bot API，返回 result 字段。
func (c *TelegramClient) Request(method string, params url.Values) (json.RawMessage, error) {
	if c.Token == "" {
		return nil, errors.New("telegram bot token is empty")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	endpoint := fmt.Sprintf("%s/bot%s/%s", base, c.Token, method)
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Telegram 服务错误: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP 请求失败: %d", resp.StatusCode)
	}
	var env struct {
		OK          *bool           `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.OK == nil {
		return nil, errors.New("无效的 Telegram API 响应")
	}
	if !*env.OK {
		desc := env.Description
		if desc == "" {
			desc = "未知错误"
		}
		return nil, fmt.Errorf("Telegram API 错误: %s", desc)
	}
	return env.Result, nil
}

// ApproveJoinRequest 通过入群申请。
func (c *TelegramClient) ApproveJoinRequest(chatID, userID int64) error {
	_, err := c.Request("approveChatJoinRequest", url.Values{"chat_id": {fmt.Sprint(chatID)}, "user_id": {fmt.Sprint(userID)}})
	return err
}

// DeclineJoinRequest 拒绝入群申请。
func (c *TelegramClient) DeclineJoinRequest(chatID, userID int64) error {
	_, err := c.Request("declineChatJoinRequest", url.Values{"chat_id": {fmt.Sprint(chatID)}, "user_id": {fmt.Sprint(userID)}})
	return err
}

// SetMyCommands 注册机器人命令菜单（scope 为 default）。
func (c *TelegramClient) SetMyCommands(commands []map[string]string) error {
	b, _ := json.Marshal(commands)
	_, err := c.Request("setMyCommands", url.Values{"commands": {string(b)}, "scope": {`{"type":"default"}`}})
	return err
}
