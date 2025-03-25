package utils

// **获取角色前缀**
func GetRolePrefix(role string) string {
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
