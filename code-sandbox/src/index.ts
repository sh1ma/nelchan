import { Hono } from "hono"
import { logger } from "hono/logger"
import { registerCommand, runCommand } from "./usecase"

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
  isCode: boolean
}

app.post("/run_command", async (c) => {
  const request = await c.req.json<RunCommandRequest>()
  const command = await runCommand(c.env, request.command_name, request.isCode)

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

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>
