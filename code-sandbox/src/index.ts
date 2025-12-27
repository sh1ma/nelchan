import { Hono } from "hono"
import { logger } from "hono/logger"
import { getCommand, registerCommand, runCommand } from "./usecase"

export { Sandbox } from "@cloudflare/sandbox"

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
    max_output_tokens: 1000,
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

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>
