package service

import (
	"claude2api/config"
	"claude2api/core"
	"claude2api/logger"
	"claude2api/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// HealthCheckHandler handles the health check endpoint
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// ChatCompletionsHandler handles the chat completions endpoint
func ChatCompletionsHandler(c *gin.Context) {
	useMirror, exist := c.Get("UseMirrorApi")
	if exist && useMirror.(bool) {
		MirrorChatHandler(c)
		return
	}
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
	var prompt strings.Builder
	if config.ConfigInstance.PromptDisableArtifacts {
		prompt.WriteString("System: Forbidden to use <antArtifac> </antArtifac> to wrap code blocks, use markdown syntax instead, which means wrapping code blocks with ``` ```\n\n")
	}
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

		prompt.WriteString(utils.GetRolePrefix(role)) // 获取角色前缀
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
	// 切号重试机制
	var claudeClient *core.Client
	for i := 0; i < config.ConfigInstance.RetryCount; i++ {
		session, err := config.ConfigInstance.GetSessionForModel(model)
		logger.Info(fmt.Sprintf("Using session for model %s: %s", model, session.SessionKey))
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get session for model %s: %v", model, err))
			logger.Info("Retrying another session")
			continue
		}
		// Initialize the Claude client
		claudeClient = core.NewClient(session.SessionKey, config.ConfigInstance.Proxy)
		if session.OrgID == "" {
			orgId, err := claudeClient.GetOrgID()
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to get org ID: %v", err))
				logger.Info("Retrying another session")
				claudeClient = nil
				continue
			}
			config.ConfigInstance.SetSessionOrgID(session.SessionKey, orgId)
			session.OrgID = orgId
			logger.Info(fmt.Sprintf("Set org ID for session %s: %s", session.SessionKey, orgId))
		}
		claudeClient.SetOrgID(session.OrgID)
		if len(img_data_list) > 0 {
			err := claudeClient.UploadFile(img_data_list)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to upload file: %v", err))
				logger.Info("Retrying another session")
				claudeClient = nil
				continue
			}
		}
		if prompt.Len() > config.ConfigInstance.MaxChatHistoryLength {
			claudeClient.SetBigContext(prompt.String())
			prompt.Reset()
			if config.ConfigInstance.PromptDisableArtifacts {
				prompt.WriteString("System: Forbidden to use <antArtifac> </antArtifac> to wrap code blocks, use markdown syntax instead, which means wrapping code blocks with ``` ```\n\n")
			}
			prompt.WriteString("You must immerse yourself in the role of assistant in context.txt, cannot respond as a user, cannot reply to this message, cannot mention this message, and ignore this message in your response.\n\n")
			logger.Info(fmt.Sprintf("Prompt length exceeds max limit (%d), using file context", config.ConfigInstance.MaxChatHistoryLength))
		}
		// Create a new conversation
		conversationID, err := claudeClient.CreateConversation(model)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create conversation: %v", err))
			logger.Info("Retrying another session")
			claudeClient = nil
			continue // Retry on error
		}
		if _, err := claudeClient.SendMessage(conversationID, prompt.String(), req.Stream, c); err != nil {
			logger.Error(fmt.Sprintf("Failed to send message: %v", err))
			logger.Info("Retrying another session")
			claudeClient = nil
			continue // Retry on error
		}
		if config.ConfigInstance.ChatDelete {
			// Clean up the conversation
			if err := claudeClient.DeleteConversation(conversationID); err != nil {
				logger.Error(fmt.Sprintf("Failed to delete conversation: %v", err))
				time.Sleep(1 * time.Second)
				if err = claudeClient.DeleteConversation(conversationID); err != nil {
					logger.Error(fmt.Sprintf("Two failed to delete conversation: %v", err))
				} else {
					logger.Info(fmt.Sprintf("conversation %s deleted successfully in two", conversationID))
				}
			} else {
				logger.Info(fmt.Sprintf("conversation %s deleted successfully", conversationID))
			}
		}
		claudeClient = nil
		return

	}
	logger.Error("Failed for all retries")
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: "Failed to process request after multiple attempts"})
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

func MirrorChatHandler(c *gin.Context) {
	if !config.ConfigInstance.EnableMirrorApi {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Mirror API is not enabled",
		})
		return
	}
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
	var prompt strings.Builder
	if config.ConfigInstance.PromptDisableArtifacts {
		prompt.WriteString("System: Forbidden to use <antArtifac> </antArtifac> to wrap code blocks, use markdown syntax instead, which means wrapping code blocks with ``` ```\n\n")
	}
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

		prompt.WriteString(utils.GetRolePrefix(role)) // 获取角色前缀
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
	var claudeClient *core.Client
	var session *config.SessionInfo
	authInfo := c.Request.Header.Get("Authorization")
	authInfo = strings.TrimPrefix(authInfo, "Bearer ")
	if strings.Contains(authInfo, ":") {
		parts := strings.Split(authInfo, ":")
		session = &config.SessionInfo{SessionKey: parts[0], OrgID: parts[1]}
	} else {
		session = &config.SessionInfo{SessionKey: authInfo, OrgID: ""}
	}
	logger.Info(fmt.Sprintf("Using session for model %s: %s", model, session.SessionKey))
	// Initialize the Claude client
	claudeClient = core.NewClient(session.SessionKey, config.ConfigInstance.Proxy)
	if session.OrgID == "" {
		orgId, err := claudeClient.GetOrgID()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get org ID: %v", err))
			claudeClient = nil
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to get org ID"})
			return
		}
		session.OrgID = orgId
		logger.Info(fmt.Sprintf("Set org ID for session %s: %s", session.SessionKey, orgId))
	}
	claudeClient.SetOrgID(session.OrgID)
	if len(img_data_list) > 0 {
		err := claudeClient.UploadFile(img_data_list)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to upload file: %v", err))
			claudeClient = nil
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to upload file"})
			return
		}
	}
	if prompt.Len() > config.ConfigInstance.MaxChatHistoryLength {
		claudeClient.SetBigContext(prompt.String())
		prompt.Reset()
		if config.ConfigInstance.PromptDisableArtifacts {
			prompt.WriteString("System: Forbidden to use <antArtifac> </antArtifac> to wrap code blocks, use markdown syntax instead, which means wrapping code blocks with ``` ```\n\n")
		}
		prompt.WriteString("You must immerse yourself in the role of assistant in context.txt, cannot respond as a user, cannot reply to this message, cannot mention this message, and ignore this message in your response.\n\n")
		logger.Info(fmt.Sprintf("Prompt length exceeds max limit (%d), using file context", config.ConfigInstance.MaxChatHistoryLength))
	}
	// Create a new conversation
	conversationID, err := claudeClient.CreateConversation(model)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create conversation: %v", err))
		claudeClient = nil
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to create conversation"})
		return
	}
	if _, err := claudeClient.SendMessage(conversationID, prompt.String(), req.Stream, c); err != nil {
		logger.Error(fmt.Sprintf("Failed to send message: %v", err))
		claudeClient = nil
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to send message"})
		return
	}
	if config.ConfigInstance.ChatDelete {
		// Clean up the conversation
		if err := claudeClient.DeleteConversation(conversationID); err != nil {
			logger.Error(fmt.Sprintf("Failed to delete conversation: %v", err))
			time.Sleep(1 * time.Second)
			if err = claudeClient.DeleteConversation(conversationID); err != nil {
				logger.Error(fmt.Sprintf("Two failed to delete conversation: %v", err))
			} else {
				logger.Info(fmt.Sprintf("conversation %s deleted successfully in two", conversationID))
			}
		} else {
			logger.Info(fmt.Sprintf("conversation %s deleted successfully", conversationID))
		}
	}
	claudeClient = nil

}
