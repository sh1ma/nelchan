import { Hono } from "hono"
import { logger } from "hono/logger"
import { getCommand, registerCommand, runCommand } from "./usecase"

export { Sandbox } from "@cloudflare/sandbox"

const app = new Hono<{ Bindings: Env }>()

app.use(logger())

app.get("/", (c) => {
  return c.json({
    message: "Hello, world!",
  })
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
}

app.post("/run_command", async (c) => {
  const request = await c.req.json<RunCommandRequest>()
  const command = await runCommand(
    c.executionCtx,
    c.env,
    request.command_name,
    request.is_code,
    request.vars
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
