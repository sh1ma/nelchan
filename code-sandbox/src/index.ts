import { routeAgentRequest } from "agents"
import console from "console"
import { Hono } from "hono"
import { bearerAuth } from "hono/bearer-auth"
import { logger } from "hono/logger"
import {
  autoStoreMemory,
  enhancedMemoryLLM,
  generateCodeFromDescription,
  getCommand,
  getMemory,
  getMentionCommand,
  memoryLLM,
  registerCommand,
  runCommand,
  setMentionCommand,
  storeMemory,
} from "./usecase"
import {
  storeMessage,
  storeMessages,
  updateMessage,
  deleteMessage,
} from "./messageService"
import type {
  StoreMessageRequest,
  UpdateMessageRequest,
  DeleteMessageRequest,
  EnhancedMllmRequest,
  FetchChannelRequest,
} from "./types/discord"
import { fetchChannelMessages } from "./discordClient"

export { Sandbox } from "@cloudflare/sandbox"
export { NelchanAgent } from "./agent"

const app = new Hono<{ Bindings: Env }>()

app.use(logger())

// API認証ミドルウェア（静的アセット以外のすべてのエンドポイントに適用）
app.use("/*", async (c, next) => {
  // 静的アセット（/）はスキップ
  if (c.req.path === "/" && c.req.method === "GET") {
    return next()
  }

  // Bearer token認証
  const authMiddleware = bearerAuth({ token: c.env.NELCHAN_API_KEY })
  return authMiddleware(c, next)
})

app.get("/", async (c) => {
  return c.html(await (await c.env.ASSETS.fetch("index.html")).text())
})

type RegisterCommandRequest = {
  command_name: string
  command_content: string
  isCode: boolean
  author_id: string
}

app.post("/register_command", async (c) => {
  const request = await c.req.json<RegisterCommandRequest>()
  console.log("[registerCommand] request: ", request)

  await registerCommand(
    c.env,
    request.command_name,
    request.command_content,
    request.isCode,
    request.author_id
  )

  return c.json({
    error: null,
  })
})

type RunCommandRequest = {
  command_name: string
  is_code: boolean
  vars: Record<string, string>
  args: string[]
}

app.post("/run_command", async (c) => {
  const request = await c.req.json<RunCommandRequest>()
  const command = await runCommand(
    c.executionCtx,
    c.env,
    request.command_name,
    request.is_code,
    request.vars,
    request.args ?? []
  )

  if (!command) {
    return c.json(
      {
        error: "Command not found",
      },
      404
    )
  }

  return c.json({
    error: null,
    command,
  })
})

type LLMRequest = {
  prompt: string
}

app.post("/llm", async (c) => {
  const request = await c.req.json<LLMRequest>()
  console.log("[llm] request: ", request)

  const response = await c.env.AI.run("@cf/openai/gpt-oss-20b", {
    input: request.prompt,
    max_output_tokens: 700,
    instructions: `あなたはDiscord上で動くBotのねるちゃんです。
ユーザーからの質問に対して、親切かつ簡潔かつ丁寧に答えてください。
口調は「ですわ」「ますわ」といったお嬢様っぽい話し方をしてください。
もしわからないことがあれば、正直に「わかりません」と答えてください。
`,
  })

  console.log("[llm] response: ", response)

  if (response.status !== "completed") {
    return c.json(
      {
        error: "Failed to generate text",
        output: null,
      },
      500
    )
  }

  // 出力テキストを抽出
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const output = response.output as any[] | undefined
  const messageOutput = output?.find((item) => item.type === "message")
  const outputText =
    messageOutput?.content?.find(
      (item: { type: string }) => item.type === "output_text"
    )?.text ?? null

  return c.json({
    error: null,
    output: outputText,
  })
})

type GetCommandRequest = {
  command_name: string
}

app.post("/get_command", async (c) => {
  const request = await c.req.json<GetCommandRequest>()
  console.log("[getCommand] request: ", request)

  const command = await getCommand(c.env, request.command_name)

  if (!command) {
    return c.json(
      {
        error: "Command not found",
        command: null,
      },
      404
    )
  }

  return c.json({
    error: null,
    command,
  })
})

type MemoryRequest = {
  key: string
  content: string
}

app.post("/memory", async (c) => {
  const request = await c.req.json<MemoryRequest>()
  console.log("[memory] request: ", request)

  try {
    await storeMemory(c.env, request.key, request.content)
    return c.json({
      error: null,
      success: true,
    })
  } catch (error) {
    console.error("[memory] error: ", error)
    return c.json(
      {
        error: "Failed to store memory",
        success: false,
      },
      500
    )
  }
})

type MgetRequest = {
  query: string
  topK?: number
}

app.post("/mget", async (c) => {
  const request = await c.req.json<MgetRequest>()
  console.log("[mget] request: ", request)

  try {
    const results = await getMemory(c.env, request.query, request.topK ?? 3)
    return c.json({
      error: null,
      results,
    })
  } catch (error) {
    console.error("[mget] error: ", error)
    return c.json(
      {
        error: "Failed to get memory",
        results: [],
      },
      500
    )
  }
})

type AutoMemoryRequest = {
  text: string
}

app.post("/automemory", async (c) => {
  const request = await c.req.json<AutoMemoryRequest>()
  console.log("[automemory] request: ", request)

  try {
    const count = await autoStoreMemory(c.env, request.text)
    return c.json({
      error: null,
      count,
    })
  } catch (error) {
    console.error("[automemory] error: ", error)
    return c.json(
      {
        error: "Failed to auto-store memory",
        count: 0,
      },
      500
    )
  }
})

type MLLMRequest = {
  prompt: string
  topK?: number
}

app.post("/mllm", async (c) => {
  const request = await c.req.json<MLLMRequest>()
  console.log("[mllm] request: ", request)

  try {
    const output = await memoryLLM(c.env, request.prompt, request.topK ?? 3)
    return c.json({
      error: null,
      output,
    })
  } catch (error) {
    console.error("[mllm] error: ", error)
    return c.json(
      {
        error: "Failed to generate response with memory",
        output: null,
      },
      500
    )
  }
})

type SmartRegisterRequest = {
  command_name: string
  description: string
  author_id: string
}

app.post("/smart_register", async (c) => {
  const request = await c.req.json<SmartRegisterRequest>()
  console.log("[smartRegister] request: ", request)

  try {
    const generated = await generateCodeFromDescription(
      c.env,
      request.command_name,
      request.description
    )

    await registerCommand(
      c.env,
      request.command_name,
      generated.code,
      true, // isCode
      request.author_id
    )

    return c.json({
      error: null,
      command_name: request.command_name,
      generated_code: generated.code,
      usage: generated.usage,
    })
  } catch (error) {
    console.error("[smartRegister] error: ", error)
    return c.json(
      {
        error:
          error instanceof Error ? error.message : "コード生成に失敗しました",
        command_name: null,
        generated_code: null,
        usage: null,
      },
      500
    )
  }
})

type LLMWithAgentRequest = {
  prompt: string
  path: string
}

// ==================
// Message Endpoints
// ==================

// POST /message - Store a Discord message
app.post("/message", async (c) => {
  const request = await c.req.json<StoreMessageRequest>()
  console.log("[message] POST request: ", request.id)

  try {
    const result = await storeMessage(c.env, request)
    return c.json(result)
  } catch (error) {
    console.error("[message] POST error: ", error)
    return c.json(
      {
        error:
          error instanceof Error ? error.message : "Failed to store message",
        stored: false,
        vectorized: false,
      },
      500
    )
  }
})

// PUT /message - Update a Discord message
app.put("/message", async (c) => {
  const request = await c.req.json<UpdateMessageRequest>()
  console.log("[message] PUT request: ", request.id)

  try {
    const result = await updateMessage(
      c.env,
      request.id,
      request.content,
      request.edited_timestamp
    )

    if (result.error === "Message not found") {
      return c.json(result, 404)
    }

    return c.json(result)
  } catch (error) {
    console.error("[message] PUT error: ", error)
    return c.json(
      {
        error:
          error instanceof Error ? error.message : "Failed to update message",
        stored: false,
        vectorized: false,
      },
      500
    )
  }
})

// DELETE /message - Delete a Discord message
app.delete("/message", async (c) => {
  const request = await c.req.json<DeleteMessageRequest>()
  console.log("[message] DELETE request: ", request.id)

  try {
    const result = await deleteMessage(c.env, request.id)

    if (result.error === "Message not found") {
      return c.json(result, 404)
    }

    return c.json(result)
  } catch (error) {
    console.error("[message] DELETE error: ", error)
    return c.json(
      {
        error:
          error instanceof Error ? error.message : "Failed to delete message",
        success: false,
      },
      500
    )
  }
})

// ==================
// Enhanced mllm Endpoint
// ==================

// POST /mllm/v2 - Enhanced mllm with 3-layer context
app.post("/mllm/v2", async (c) => {
  const request = await c.req.json<EnhancedMllmRequest>()
  console.log("[mllm/v2] request: ", request)

  try {
    const result = await enhancedMemoryLLM(
      c.env,
      request.prompt,
      request.channel_id,
      request.user_id,
      request.recent_count,
      request.similar_count
    )

    return c.json({
      error: null,
      output: result.output,
      context: result.context,
    })
  } catch (error) {
    console.error("[mllm/v2] error: ", error)
    return c.json(
      {
        error:
          error instanceof Error
            ? error.message
            : "Failed to generate response",
        output: null,
        context: null,
      },
      500
    )
  }
})

// ==================
// Admin Endpoints
// ==================

// POST /admin/fetch_channel - Fetch channel messages and store them
app.post("/admin/fetch_channel", async (c) => {
  const request = await c.req.json<FetchChannelRequest>()
  console.log("[admin/fetch_channel] request: ", request)

  try {
    // Discord APIからメッセージを取得
    const messages = await fetchChannelMessages(
      c.env,
      request.channel_id,
      request.limit ?? 100
    )

    // メッセージを保存
    const result = await storeMessages(c.env, messages)

    return c.json({
      error: null,
      fetched_count: messages.length,
      stored_count: result.stored_count,
      vectorized_count: result.vectorized_count,
    })
  } catch (error) {
    console.error("[admin/fetch_channel] error: ", error)
    return c.json(
      {
        error:
          error instanceof Error
            ? error.message
            : "Failed to fetch channel messages",
        fetched_count: 0,
        stored_count: 0,
        vectorized_count: 0,
      },
      500
    )
  }
})

// Get mention command
app.get("/mention_command", async (c) => {
  try {
    const commandName = await getMentionCommand(c.env)
    return c.json({
      error: null,
      command_name: commandName,
    })
  } catch (error) {
    console.error("[getMentionCommand] error: ", error)
    return c.json(
      {
        error: "Failed to get mention command",
        command_name: null,
      },
      500
    )
  }
})

type SetMentionCommandRequest = {
  command_name: string | null
}

// Set mention command
app.post("/mention_command", async (c) => {
  const request = await c.req.json<SetMentionCommandRequest>()
  console.log("[setMentionCommand] request: ", request)

  try {
    await setMentionCommand(c.env, request.command_name)
    return c.json({
      error: null,
      command_name: request.command_name,
    })
  } catch (error) {
    console.error("[setMentionCommand] error: ", error)
    return c.json(
      {
        error: "Failed to set mention command",
        command_name: null,
      },
      500
    )
  }
})

// /llmWithAgent は内部でAgentにプロキシする
app.post("/llmWithAgent/:path", async (c) => {
  const requestBody = await c.req.json<LLMWithAgentRequest>()
  // URLを書き換えてAgentにルーティング
  const originalUrl = new URL(c.req.url)
  originalUrl.pathname = `/agents/nelchan-agent/default/${c.req.param("path")}`

  const request = new Request(originalUrl.toString(), {
    body: JSON.stringify(requestBody),
    method: "POST",
  })

  const response = await routeAgentRequest(request, c.env)
  if (response) {
    return response
  }
  return c.json(
    {
      error: "Agent not found",
      output: null,
    },
    500
  )
})

import { handleScheduled } from "./scheduler"

export default {
  fetch: app.fetch,
  scheduled: handleScheduled,
} satisfies ExportedHandler<Env>
