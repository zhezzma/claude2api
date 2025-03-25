# Claude2Api
将Claude 的网页服务转为 Api 服务，支持识图，文件上传，流式传输……
# Claude2API

[![Go Report Card](https://goreportcard.com/badge/github.com/yushangxiao/claude2api)](https://goreportcard.com/report/github.com/yushangxiao/claude2api)
[![License](https://img.shields.io/github/license/yushangxiao/claude2api)](LICENSE)
|![中文](https://github.com/yushangxiao/claude2api/blob/main/docs/chinses.md)

## ✨ Features

- 🖼️ **Image Recognition** - Send images to Claude for analysis
- 📝 **Automatic Conversation Management** -  Conversation can be automatically deleted after use
- 🌊 **Streaming Responses** - Get real-time streaming outputs from Claude
- 📁 **File Upload Support** - Upload long context
- 🧠 **Thinking Process** - Access Claude's step-by-step reasoning, support <think>
- 🔄 **Chat History Management** - Control the length of conversation context , exceeding will upload file
- 🌐 **Proxy Support** - Route requests through your preferred proxy
- 🔐 **API Key Authentication** - Secure your API endpoints

## 📋 Prerequisites

- Go 1.23+ (for building from source)
- Docker (for containerized deployment)

## 🚀 Deployment Options

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

Create a `docker-compose.yml` file:

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
      - PROXY=http://proxy:2080  # Optional
      - CHAT_DELETE=true
      - MAX_CHAT_HISTORY_LENGTH=10000
    restart: unless-stopped
    
```

Then run:

```bash
docker-compose up -d
```

### Hugging Face Spaces

You can deploy this project to Hugging Face Spaces with Docker:

1. Fork the Hugging Face Space at [https://huggingface.co/spaces/rclon/claude2api](https://huggingface.co/spaces/rclon/claude2api)
2. Configure your environment variables in the Settings tab
3. The Space will automatically  deploy the Docker image

notice: In Hugging Face, /v1 might be blocked, you can use /hf/v1 instead.
### Direct Deployment

```bash
# Clone the repository
git clone https://github.com/yushangxiao/claude2api.git
cd claude2api

# Build the binary
go build -o claude2api .

# Run the service
export SESSIONS=sk-ant-sid01-xxxx,sk-ant-sid01-yyyy
export ADDRESS=0.0.0.0:8080
export APIKEY=123
export CHAT_DELETE=true
export MAX_CHAT_HISTORY_LENGTH=10000
./claude2api
```

## ⚙️ Configuration

| Environment Variable | Description | Default |
|----------------------|-------------|---------|
| `SESSIONS` | Comma-separated list of Claude API session keys | Required |
| `ADDRESS` | Server address and port | `0.0.0.0:8080` |
| `APIKEY` | API key for authentication | Required |
| `PROXY` | HTTP proxy URL | Optional |
| `CHAT_DELETE` | Whether to delete chat sessions after use | `true` |
| `MAX_CHAT_HISTORY_LENGTH` | Exceeding will text to file | `10000` |

## 📝 API Usage

### Authentication

Include your API key in the request header:

```
Authorization: Bearer YOUR_API_KEY
```

### Chat Completion

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "claude-3-7-sonnet-20250219",
    "messages": [
      {
        "role": "user",
        "content": "Hello, Claude!"
      }
    ],
    "stream": true
  }'
```

### Image Analysis

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
            "text": "What\'s in this image?"
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


## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Anthropic](https://www.anthropic.com/) for creating Claude
- The Go community for the amazing ecosystem

---

Made with ❤️ by [yushangxiao](https://github.com/yushangxiao)
