package service

import (
	"claude2api/config"
	"claude2api/core"
	"claude2api/logger"
	"claude2api/middleware"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ChatCompletionRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Stream   bool                     `json:"stream"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// **获取角色前缀**
func getRolePrefix(role string) string {
	switch role {
	case "system":
		return "System: "
	case "user":
		return "Human: "
	case "assistant":
		return "Assistant: "
	default:
		return "Unknown: "
	}
}

// HealthCheckHandler handles the health check endpoint
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// ChatCompletionsHandler handles the chat completions endpoint
func ChatCompletionsHandler(c *gin.Context) {

	// Parse request body
	var req ChatCompletionRequest
	defaultStream := true
	req = ChatCompletionRequest{
		Stream: defaultStream,
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}
	// logger.Info(fmt.Sprintf("Received request: %v", req))
	// Validate request
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "No messages provided",
		})
		return
	}

	// Get model or use default
	model := req.Model
	if model == "" {
		model = "claude-3-7-sonnet-20250219"
	}
	// Get session for the model
	session, err := config.ConfigInstance.GetSessionForModel(model)
	logger.Info(fmt.Sprintf("Using session for model %s: %s", model, session.SessionKey))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get session for model %s: %v", model, err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Error: %v", err),
		})
		return
	}
	// Initialize the Claude client
	claudeClient := core.NewClient(session.SessionKey, config.ConfigInstance.Proxy)
	if session.OrgID == "" {
		orgId, err := claudeClient.GetOrgID()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get org ID: %v", err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: fmt.Sprintf("Failed to get org ID: %v", err),
			})
			return
		}
		config.ConfigInstance.SetSessionOrgID(session.SessionKey, orgId)
		session.OrgID = orgId
		logger.Info(fmt.Sprintf("Set org ID for session %s: %s", session.SessionKey, orgId))
	}
	claudeClient.SetOrgID(session.OrgID)
	// Create a new conversation
	conversationID, err := claudeClient.CreateConversation(model)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create conversation: %v", err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to create conversation: %v", err),
		})
		return
	}
	var prompt strings.Builder
	// 禁止使用<antArtifac> </antArtifac>包裹代码块，使用markdown语法，也就是``` ```包裹代码块
	prompt.WriteString("System: Forbidden to use <antArtifac> </antArtifac> to wrap code blocks, use markdown syntax instead, which means wrapping code blocks with ``` ```\n\n")
	img_data_list := []string{}
	// Format messages into a single prompt
	for _, msg := range req.Messages {
		role, roleOk := msg["role"].(string)
		if !roleOk {
			continue // 忽略无效格式
		}

		content, exists := msg["content"]
		if !exists {
			continue
		}

		prompt.WriteString(getRolePrefix(role)) // 获取角色前缀
		switch v := content.(type) {
		case string: // 如果 content 直接是 string
			prompt.WriteString(v + "\n\n")
		case []interface{}: // 如果 content 是 []interface{} 类型的数组
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if itemType, ok := itemMap["type"].(string); ok {
						if itemType == "text" {
							if text, ok := itemMap["text"].(string); ok {
								prompt.WriteString(text + "\n\n")
							}
						} else if itemType == "image_url" {
							if imageUrl, ok := itemMap["image_url"].(map[string]interface{}); ok {
								if url, ok := imageUrl["url"].(string); ok {
									img_data_list = append(img_data_list, url) // 收集图片数据
								}
							}
						}
					}
				}
			}
		}
	}
	fmt.Println(prompt.String())                 // 输出最终构造的内容
	fmt.Println("img_data_list:", img_data_list) // 输出图片数据
	if len(img_data_list) > 0 {
		err := claudeClient.UploadFile(img_data_list)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to upload file: %v", err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: fmt.Sprintf("Failed to upload file: %v", err),
			})
			return
		}
	}
	if prompt.Len() > config.ConfigInstance.MaxChatHistoryLength {
		claudeClient.SetBigContext(prompt.String())
		prompt.Reset()
		prompt.WriteString("System: Forbidden to use <antArtifac> </antArtifac> to wrap code blocks, use markdown syntax instead, which means wrapping code blocks with ``` ```\n\n")
		prompt.WriteString("You must immerse yourself in the role of assistant in context.txt, cannot respond as a user, cannot reply to this message, cannot mention this message, and ignore this message in your response.")
		logger.Info(fmt.Sprintf("Prompt length exceeds max limit (%d), using file context", config.ConfigInstance.MaxChatHistoryLength))
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()
	if err := claudeClient.SendMessage(conversationID, prompt.String(), req.Stream, c); err != nil {
		// Can't send JSON error as we've already started streaming
		logger.Error(fmt.Sprintf("Failed to send message: %v", err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("Failed to send message: %v", err),
		})
		return
	}
	if config.ConfigInstance.ChatDelete {
		// Clean up the conversation
		defer func() {
			if err := claudeClient.DeleteConversation(conversationID); err != nil {
				logger.Error(fmt.Sprintf("Failed to delete conversation: %v", err))
			}
		}()

	}
}

func MoudlesHandler(c *gin.Context) {
	models := []map[string]interface{}{
		{"id": "claude-3-7-sonnet-20250219"},
		{"id": "claude-3-7-sonnet-20250219-think"},
	}
	c.JSON(http.StatusOK, gin.H{
		"data": models,
	})
}

// SetupRoutes configures all the routes for the application
func SetupRoutes(r *gin.Engine) {
	// Apply middleware
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.AuthMiddleware())

	// Health check endpoint
	r.GET("/health", HealthCheckHandler)

	// Chat completions endpoint (OpenAI-compatible)
	r.POST("/v1/chat/completions", ChatCompletionsHandler)
	r.GET("/v1/models", MoudlesHandler)
	// HuggingFace compatible routes
	hfRouter := r.Group("/hf")
	{
		v1Router := hfRouter.Group("/v1")
		{
			v1Router.POST("/chat/completions", ChatCompletionsHandler)
			v1Router.GET("/models", MoudlesHandler)
		}
	}
}
