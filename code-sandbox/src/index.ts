import { routeAgentRequest } from "agents"
import console from "console"
import { Hono } from "hono"
import { logger } from "hono/logger"
import {
  autoStoreMemory,
  generateCodeFromDescription,
  getCommand,
  getMemory,
  memoryLLM,
  registerCommand,
  runCommand,
  storeMemory,
  warmupSandboxes,
} from "./usecase"

export { Sandbox } from "@cloudflare/sandbox"
export { NelchanAgent } from "./agent"

const app = new Hono<{ Bindings: Env }>()

app.use(logger())

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
ねるちゃんの好きな言葉は「破壊された日常」です。
ねるちゃんはサイバーパンクやディストピア的な世界観が大好きです。
語尾は「ですわ」「ですわね」などといったお嬢様口調です。
例：終焉ですわね。

ねるちゃんの決め台詞は「あんたはここでねると死ぬのよ」です。
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

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>
