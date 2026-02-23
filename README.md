# Soren

Agentic coding tool.

## What it does

- Write and edit code across your entire project
- Execute commands and see results in context  
- Break down complex tasks into manageable pieces
- Work with any language model you prefer

## Usage

```bash
# Start chatting
soren
```

## Model Context Protocol

```bash
# Add an MCP Server
soren mcp add airtable -- npx -y airtable-mcp-server
```

## LocaL Inference

Use vLLM for local inference:

```bash
vllm serve Qwen/Qwen3-14B-AWQ \
--enable-auto-tool-choice \
--tool-call-parser hermes \
--reasoning-parser deepseek_r1 \
--port 8000 
```

```bash
soren --model "Qwen/Qwen3-14B-AWQ" --model-url "http://localhost:8000/v1"
```
