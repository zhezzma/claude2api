package config

import (
	"claude2api/logger"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type SessionInfo struct {
	SessionKey string
	OrgID      string
}

type SessionRagen struct {
	index int
	mutex sync.Mutex
}

type Config struct {
	Sessions             []SessionInfo
	Address              string
	APIKey               string
	Proxy                string
	ChatDelete           bool
	MaxChatHistoryLength int
	RetryCount           int
}

// 初始化随机数生成器
func init() {
	rand.Seed(time.Now().UnixNano())
}

// 解析 SESSION 格式的环境变量
func parseSessionEnv(envValue string) (int, []SessionInfo) {
	if envValue == "" {
		return 0, []SessionInfo{}
	}
	var sessions []SessionInfo
	sessionPairs := strings.Split(envValue, ",")
	retryCount := len(sessionPairs) // 重试次数等于 session 数量
	for _, pair := range sessionPairs {
		parts := strings.Split(pair, ":")
		session := SessionInfo{
			SessionKey: parts[0],
		}

		if len(parts) > 1 {
			session.OrgID = parts[1]
		} else if len(parts) == 1 {
			session.OrgID = ""
		}

		sessions = append(sessions, session)
	}
	logger.Info(fmt.Sprintf("Sessions count: %d", retryCount))
	return retryCount, sessions
}

// 根据模型选择合适的 session
func (c *Config) GetSessionForModel(model string) (SessionInfo, error) {
	allSessions := c.Sessions

	// 如果没有可用的 session，返回空
	if len(allSessions) == 0 {
		return SessionInfo{}, fmt.Errorf("no sessions available for model %s", model)
	}

	// 如果只有一个 session，直接返回
	if len(allSessions) == 1 {
		return allSessions[0], nil
	}
	// 如果有多个 session，选择下一个
	Sr.mutex.Lock()
	defer Sr.mutex.Unlock()
	session := allSessions[Sr.index]
	Sr.index = (Sr.index + 1) % len(allSessions)
	return session, nil
}

func (c *Config) SetSessionOrgID(sessionKey, orgID string) {
	for i, session := range c.Sessions {
		if session.SessionKey == sessionKey {
			c.Sessions[i].OrgID = orgID
			return
		}
	}
}

// 从环境变量加载配置
func LoadConfig() *Config {
	maxChatHistoryLength, err := strconv.Atoi(os.Getenv("MAX_CHAT_HISTORY_LENGTH"))
	if err != nil {
		maxChatHistoryLength = 10000 // 默认值
	}
	retryCount, sessions := parseSessionEnv(os.Getenv("SESSIONS"))
	config := &Config{
		// 解析 SESSIONS 环境变量
		Sessions: sessions,
		// 设置服务地址，默认为 "0.0.0.0:8080"
		Address: os.Getenv("ADDRESS"),

		// 设置 API 认证密钥
		APIKey: os.Getenv("APIKEY"),
		// 设置代理地址
		Proxy: os.Getenv("PROXY"),
		//自动删除聊天
		ChatDelete: true,
		// 设置最大聊天历史长度
		MaxChatHistoryLength: maxChatHistoryLength,
		// 设置重试次数
		RetryCount: retryCount,
	}

	// 如果地址为空，使用默认值
	if config.Address == "" {
		config.Address = "0.0.0.0:8080"
	}
	if os.Getenv("ChAT_DELETE") == "false" {
		config.ChatDelete = false
	}

	return config
}

var ConfigInstance *Config
var Sr *SessionRagen

func init() {
	// 加载环境变量
	err := godotenv.Load()
	Sr = &SessionRagen{
		index: 0,
		mutex: sync.Mutex{},
	}
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	ConfigInstance = LoadConfig()
	fmt.Println("Loaded config:", ConfigInstance)
}
