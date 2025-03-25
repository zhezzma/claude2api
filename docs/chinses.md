# Claude2Api
将Claude的网页服务转为Api服务，支持识图，文件上传，流式传输……

# Claude2API
[![Go Report Card](https://goreportcard.com/badge/github.com/yushangxiao/claude2api)](https://goreportcard.com/report/github.com/yushangxiao/claude2api)
[![License](https://img.shields.io/github/license/yushangxiao/claude2api)](LICENSE)
|[英文](https://github.com/yushangxiao/claude2api/edit/main/README.md)

## ✨ 特性
- 🖼️ **图像识别** - 发送图像给Claude进行分析
- 📝 **自动对话管理** - 对话可在使用后自动删除
- 🌊 **流式响应** - 获取Claude实时流式输出
- 📁 **文件上传支持** - 上传长文本内容
- 🧠 **思考过程** - 访问Claude的逐步推理，支持

<details type="reasoning" done="false"
<summaryThinking…</summary
 标签
 - 🔄 **聊天历史管理** - 控制对话上下文长度，超出将上传为文件
 - 🌐 **代理支持** - 通过您首选的代理路由请求
 - 🔐 **API密钥认证** - 保护您的API端点
 
 ## 📋 前提条件
 - Go 1.23+（从源代码构建）
 - Docker（用于容器化部署）
 
 ## 🚀 部署选项
 ### Docker
 ```bash
 docker run -d \
   -p 8080:8080 \
   -e SESSIONS=sk-ant-sid01-xxxx,sk-ant-sid01-yyyy \
   -e APIKEY=123 \
   -e CHAT_DELETE=true \
   -e MAX_CHAT_HISTORY_LENGTH=10000 \
   --name claude2api \
   ghcr.io/yushangxiao/claude2api:latest
 ```
 
 ### Docker Compose
 创建一个`docker-compose.yml`文件：
 ```yaml
 version: '3'
 services:
   claude2api:
     image: ghcr.io/yushangxiao/claude2api:latest
     container_name: claude2api
     ports:
       - "8080:8080"
     environment:
       - SESSIONS=sk-ant-sid01-xxxx,sk-ant-sid01-yyyy
       - ADDRESS=0.0.0.0:8080
       - APIKEY=123
       - PROXY=http://proxy:2080  # 可选
       - CHAT_DELETE=true
       - MAX_CHAT_HISTORY_LENGTH=10000
     restart: unless-stopped
 ```
 然后运行：
 ```bash
 docker-compose up -d
 ```
 
 ### Hugging Face Spaces
 您可以使用Docker将此项目部署到Hugging Face Spaces：
 1. Fork Hugging Face Space：[https://huggingface.co/spaces/rclon/claude2api](https://huggingface.co/spaces/rclon/claude2api)
 2. 在设置选项卡中配置您的环境变量
 3. Space将自动部署Docker镜像
 
 注意：在Hugging Face中，/v1可能被屏蔽，您可以使用/hf/v1代替。
 
 ### 直接部署
 ```bash
 # 克隆仓库
 git clone https://github.com/yushangxiao/claude2api.git
 cd claude2api
 # 构建二进制文件
 go build -o claude2api .
 # 运行服务
 export SESSIONS=sk-ant-sid01-xxxx,sk-ant-sid01-yyyy
 export ADDRESS=0.0.0.0:8080
 export APIKEY=123
 export CHAT_DELETE=true
 export MAX_CHAT_HISTORY_LENGTH=10000
 ./claude2api
 ```
 
 ## ⚙️ 配置
 | 环境变量 | 描述 | 默认值 |
 |----------------------|-------------|---------|
 | `SESSIONS` | 逗号分隔的Claude API会话密钥列表 | 必填 |
 | `ADDRESS` | 服务器地址和端口 | `0.0.0.0:8080` |
 | `APIKEY` | 用于认证的API密钥 | 必填 |
 | `PROXY` | HTTP代理URL | 可选 |
 | `CHAT_DELETE` | 是否在使用后删除聊天会话 | `true` |
 | `MAX_CHAT_HISTORY_LENGTH` | 超出此长度将文本转为文件 | `10000` |
 
 ## 📝 API使用
 ### 认证
 在请求头中包含您的API密钥：
 ```
 Authorization: Bearer YOUR_API_KEY
 ```
 
 ### 聊天完成
 ```bash
 curl -X POST http://localhost:8080/v1/chat/completions \
   -H "Content-Type: application/json" \
   -H "Authorization: Bearer YOUR_API_KEY" \
   -d '{
     "model": "claude-3-7-sonnet-20250219",
     "messages": [
       {
         "role": "user",
         "content": "你好，Claude！"
       }
     ],
     "stream": true
   }'
 ```
 
 ### 图像分析
 ```bash
 curl -X POST http://localhost:8080/v1/chat/completions \
   -H "Content-Type: application/json" \
   -H "Authorization: Bearer YOUR_API_KEY" \
   -d '{
     "model": "claude-3-7-sonnet-20250219",
     "messages": [
       {
         "role": "user",
         "content": [
           {
             "type": "text",
             "text": "这张图片里有什么？"
           },
           {
             "type": "image_url",
             "image_url": {
               "url": "data:image/jpeg;base64,..."
             }
           }
         ]
       }
     ]
   }'
 ```
 
 ## 🤝 贡献
 欢迎贡献！请随时提交Pull Request。
 1. Fork仓库
 2. 创建特性分支（`git checkout -b feature/amazing-feature`）
 3. 提交您的更改（`git commit -m '添加一些惊人的特性'`）
 4. 推送到分支（`git push origin feature/amazing-feature`）
 5. 打开Pull Request
 
 ## 📄 许可证
 本项目采用MIT许可证 - 详见[LICENSE](LICENSE)文件。
 
 ## 🙏 致谢
 - 感谢[Anthropic](https://www.anthropic.com/)创建Claude
 - 感谢Go社区提供的优秀生态系统
 
 ---
 由[yushangxiao](https://github.com/yushangxiao)用❤️制作
</details
