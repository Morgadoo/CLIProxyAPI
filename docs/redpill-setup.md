# RedPill Provider Setup for CLIProxyAPI

Use your RedPill Pro subscription ($35/month unlimited) through CLIProxyAPI alongside Claude, Codex, and other providers.

## How It Works

RedPill's web chat at `chat.redpill.ai` accepts standard OpenAI-format requests and returns standard OpenAI SSE responses. The only difference from a normal API is authentication: it uses a JWT cookie instead of a Bearer token.

```
AI Clients (Claude Code, Cursor, Kilo Code)
         |
CLIProxyAPI (localhost:8317)
         |  Cookie: token=<JWT>
         v
chat.redpill.ai/api/chat/completions
```

CLIProxyAPI injects the JWT cookie on every request. No format translation needed.

## Prerequisites

- RedPill Pro subscription (https://www.redpill.ai/pricing)
- CLIProxyAPI built from the fork with RedPill support (`localhost/cliproxyapi-redpill` image)

## Building the Image

```bash
cd /path/to/CLIProxyAPI-fork
docker build -t cliproxyapi-redpill .
```

## Compose Configuration

Update `compose.yml` to use the custom image:

```yaml
services:
  cli-proxy-api:
    image: localhost/cliproxyapi-redpill:latest
    container_name: cli-proxy-api
    ports:
      - "8317:8317"
      - "8085:8085"
      - "1455:1455"
      - "54545:54545"
      - "51121:51121"
      - "11451:11451"
    security_opt:
      - label=disable
    volumes:
      - ./config.yaml:/CLIProxyAPI/config.yaml
      - ./auths:/root/.cli-proxy-api
      - ./logs:/CLIProxyAPI/logs
    restart: unless-stopped
```

## Login

### Getting your JWT token

1. Open https://chat.redpill.ai in your browser
2. Log in with your Pro account
3. Open DevTools (F12) > Application > Cookies > `chat.redpill.ai`
4. Copy the value of the `token` cookie (starts with `eyJ...`)

### CLI Login (inside container)

```bash
# From host (Fedora Silverblue)
echo "<YOUR_JWT_TOKEN>" | flatpak-spawn --host podman exec -i cli-proxy-api \
  /CLIProxyAPI/CLIProxyAPI -config /CLIProxyAPI/config.yaml -redpill-cookie

# Or from a regular system with Docker
echo "<YOUR_JWT_TOKEN>" | docker exec -i cli-proxy-api \
  /CLIProxyAPI/CLIProxyAPI -config /CLIProxyAPI/config.yaml -redpill-cookie
```

Expected output:

```
RedPill Cookie Login
====================
Enter RedPill Cookie (JWT token from browser): Saving credentials to /root/.cli-proxy-api/redpill-user@email.com-1234567890.json
Authentication successful! Email: user@email.com
Expires at: 2026-04-29T09:48:01+08:00
Authentication saved to: /root/.cli-proxy-api/redpill-user@email.com-1234567890.json
```

### Management UI Login

```bash
curl -X POST http://localhost:8317/v0/management/redpill-auth-url \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <MANAGEMENT_PASSWORD>" \
  -d '{"cookie":"<YOUR_JWT_TOKEN>"}'
```

### After Login

Restart the container to pick up the new auth file:

```bash
# Fedora Silverblue
flatpak-spawn --host podman restart cli-proxy-api

# Regular Docker
docker restart cli-proxy-api
```

## Available Models

18 models from the RedPill Pro plan:

| Model ID | Description |
|----------|-------------|
| `deepseek/deepseek-r1-0528` | DeepSeek R1 (reasoning, 163K context) |
| `deepseek/deepseek-v3.2` | DeepSeek V3.2 (163K context) |
| `deepseek/deepseek-chat-v3.1` | DeepSeek V3.1 (163K context) |
| `qwen/qwen3-coder-480b-a35b-instruct` | Qwen3 Coder 480B MoE (262K context) |
| `qwen/qwen3.5-397b-a17b` | Qwen3.5 397B MoE (262K context) |
| `qwen/qwen3.5-27b` | Qwen3.5 27B dense (262K context) |
| `qwen/qwen3-30b-a3b-instruct-2507` | Qwen3 30B MoE (262K context) |
| `qwen/qwen-2.5-7b-instruct` | Qwen2.5 7B (32K context) |
| `qwen/qwen3-vl-30b-a3b-instruct` | Qwen3 VL 30B multimodal (128K context) |
| `openai/gpt-oss-120b` | GPT OSS 120B MoE (131K context) |
| `google/gemma-3-27b-it` | Gemma 3 27B (53K context) |
| `meta-llama/llama-3.3-70b-instruct` | Llama 3.3 70B (131K context) |
| `moonshotai/kimi-k2-thinking` | Kimi K2 Thinking (262K context) |
| `moonshotai/kimi-k2.5` | Kimi K2.5 multimodal (262K context) |
| `z-ai/glm-5` | GLM 5 (202K context) |
| `z-ai/glm-4.7` | GLM 4.7 (131K context) |
| `z-ai/glm-4.7-flash` | GLM 4.7 Flash (202K context) |
| `phala/uncensored-24b` | Venice Uncensored 24B (32K context) |

New models added to RedPill's platform will need to be added to the model definitions in the code.

## Testing

```bash
# List models
curl http://localhost:8317/v1/models | python3 -m json.tool

# Non-streaming completion
curl http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen/qwen-2.5-7b-instruct","messages":[{"role":"user","content":"Hello"}]}'

# Streaming completion
curl -N http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"z-ai/glm-5","messages":[{"role":"user","content":"Count 1 to 5"}],"stream":true}'
```

## Token Expiry and Renewal

The JWT token from RedPill is valid for approximately **30 days**. It cannot be refreshed programmatically — you must get a new one from the browser.

CLIProxyAPI will log a warning when the token is within 48 hours of expiry.

To renew:

1. Open https://chat.redpill.ai and log in
2. Copy the new `token` cookie value
3. Re-run the cookie login command (see Login section above)
4. Restart the container

## Architecture

### Files Added to CLIProxyAPI

| File | Purpose |
|------|---------|
| `internal/auth/redpill/redpill_auth.go` | JWT decode, token verification, cookie storage |
| `internal/auth/redpill/redpill_token.go` | Auth file persistence (`redpill-*.json`) |
| `internal/auth/redpill/cookie_helpers.go` | Cookie normalization, duplicate checking |
| `internal/runtime/executor/redpill_executor.go` | Streaming + non-streaming executor with cookie auth |
| `sdk/auth/redpill.go` | SDK authenticator and refresh lead |
| `internal/cmd/redpill_cookie.go` | CLI `--redpill-cookie` command |

### Auth File Format

Saved at `auths/redpill-<email>-<timestamp>.json`:

```json
{
  "email": "user@email.com",
  "cookie": "token=eyJ...;",
  "expired": "2026-04-29T09:48:01+08:00",
  "last_refresh": "2026-03-30T11:57:18+08:00",
  "type": "redpill"
}
```

### Key Discovery

RedPill's `chat.redpill.ai/api/chat/completions` endpoint:
- Accepts standard OpenAI request format (just `model`, `messages`, `stream`)
- Returns standard OpenAI response format (both streaming SSE and non-streaming JSON)
- Authenticates via `Cookie: token=<JWT>` header (not Bearer token)
- The extra fields sent by the web UI (session_id, chat_id, model_item, etc.) are **not required**

This means the proxy is a simple cookie injection — no request or response translation needed.

## Troubleshooting

### `redpill executor: missing cookie`

No valid RedPill auth file found. Run the cookie login and restart the container.

### `redpill cookie authentication: token has expired`

JWT token expired. Get a fresh one from the browser and re-run login.

### `redpill cookie authentication: verification failed`

The token was rejected by RedPill's API. Make sure you're copying the correct `token` cookie and that your Pro subscription is active.

### Models not showing up

Restart the container after adding the auth file. CLIProxyAPI loads auth files at startup.
