# go-bot

A lightweight Go application to connect to OpenAI's API, featuring three simple yet powerful endpoints: `status/`, `stream/`, and `chat/`. Built with a focus on clarity, performance, and ease of use.

---

## Features

- **Status Endpoint**: Quickly verify the service health.
- **Stream Endpoint**: Stream OpenAI responses for real-time interaction.
- **Chat Endpoint**: Send messages and receive responses from OpenAI.
- **Minimalistic and Efficient**: Built with the Gin framework for fast API handling.
- **Environment Friendly**: Easy configuration with `.env` support.
- **Ready for Scale**: Clean structure for future extensibility.

1. Clone the repository:
   ```bash
   git clone https://github.com/your-repo/go-bot.git
   cd go-bot
   ```
2. Install Depenencies

   ```bash
    go mod tidy
   ```

3. Set your.env

   ```bash
   OPENAI_API_KEY=your_openai_api_key
   PORT=8080
   ```

   Run the Application

### Start the server:

```bash
 go run ./cmd/server
```

Access the API at http://localhost:8080.

### API Endpoints

```bash
GET /status: Health check endpoint to verify API connectivity and service status
POST /chat: Standard chat endpoint for single request-response interactions, returning complete responses
POST /stream:  Real-time streaming endpoint for receiving continuous AI responses
Example Usage
Chat
curl -X POST http://localhost:8080/chat \
-H "Content-Type: application/json" \
-d '{"message": "Hello, how are you?"}'
```

### API Documentation

- [openAPI](./openapi3_0.json)
