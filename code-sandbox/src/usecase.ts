import { getSandbox } from "@cloudflare/sandbox"

type NelchanGetCommandResult = {
  id: string
  name: string
  author_id: string
  dictionary_id?: string
  text?: string
  code_id?: string
  code?: string
}

const getCommandContent = (result: NelchanGetCommandResult) => {
  if (result.text) {
    return { type: "text", content: result.text }
  }
  if (result.code) {
    return { type: "code", content: result.code }
  }
  throw new Error("No command content found")
}

export const runCommand = async (
  exCtx: ExecutionContext,
  env: Env,
  commandName: string,
  isCode: boolean,
  envVars: Record<string, string>
) => {
  // Query based on command type
  const query = isCode
    ? `SELECT 
        c.id, 
        c.name, 
        c.author_id, 
        co.id as code_id, 
        co.code 
        FROM commands c  
        INNER JOIN codes co ON c.id = co.command_id 
        WHERE c.name = ? LIMIT 1`
    : `SELECT 
        c.id, 
        c.name, 
        c.author_id, 
        d.id as dictionary_id, 
        d.text 
        FROM commands c  
        INNER JOIN dictionaries d ON c.id = d.command_id 
        WHERE c.name = ? LIMIT 1`

  const result = await env.nelchan_db
    .prepare(query)
    .bind(commandName)
    .first<NelchanGetCommandResult>()

  if (!result) {
    console.log(`Command ${commandName} not found (isCode: ${isCode})`)
    return undefined
  }

  if (isCode && result.code) {
    const randomString = crypto.randomUUID()
    const sandbox = getSandbox(env.Sandbox, randomString)
    const ctx = await sandbox.createCodeContext({ language: "python" })
    try {
      const envEmbededCode = `
${Object.entries(envVars)
  .map(([key, value]) => `${key}="${value}"`)
  .join("\n")}
def cs(s: str):
    return f"\`\`\`{s}\`\`\`"

${result.code}
`
      console.log(envEmbededCode)
      const executionResult = await sandbox.runCode(envEmbededCode, {
        context: ctx,
      })
      console.log("[runCommand] executionResult: ", executionResult)
      const codeResultOutput = executionResult.logs.stdout.join("\n")
      return {
        id: result.id,
        name: result.name,
        content: codeResultOutput,
      }
    } catch (error) {
      console.error("[runCommand] error: ", error)
      throw new Error("Error running code")
    } finally {
      exCtx.waitUntil(sandbox.destroy())
    }
  }

  if (!isCode && result.text) {
    return {
      id: result.id,
      name: result.name,
      content: result.text,
    }
  }

  throw new Error("Invalid command content type")
}

export const registerCommand = async (
  env: Env,
  commandName: string,
  commandContent: string,
  isCode: boolean,
  authorID: string
) => {
  if (isCode) {
    await registerCodeCommand(env, commandName, commandContent, authorID)
  } else {
    await registerTextCommand(env, commandName, commandContent, authorID)
  }
}

/*
 * Register a code command (upsert)
 * @param env - The environment
 * @param commandName - The name of the command
 * @param commandContent - The content of the command
 * @param authorID - The ID of the author
 */
export const registerCodeCommand = async (
  env: Env,
  commandName: string,
  commandContent: string,
  authorID: string
) => {
  console.log("[registerCodeCommand] commandName: ", commandName)
  console.log("[registerCodeCommand] commandContent: ", commandContent)

  // Check if command already exists
  const existingCommand = await env.nelchan_db
    .prepare(
      `SELECT c.id, co.id as code_id, d.id as dictionary_id FROM commands c LEFT JOIN codes co ON c.id = co.command_id LEFT JOIN dictionaries d ON c.id = d.command_id WHERE c.name = ?`
    )
    .bind(commandName)
    .first<{
      id: string
      code_id: string | null
      dictionary_id: string | null
    }>()

  if (existingCommand) {
    console.log(
      "[registerCodeCommand] UPDATE existing command",
      existingCommand.id
    )

    // Delete old dictionary record if exists (textCommand -> codeCommand migration)
    if (existingCommand.dictionary_id) {
      console.log(
        "[registerCodeCommand] DELETE old dictionary record",
        existingCommand.dictionary_id
      )
      await env.nelchan_db
        .prepare(`DELETE FROM dictionaries WHERE id = ?`)
        .bind(existingCommand.dictionary_id)
        .run()
    }

    if (existingCommand.code_id) {
      // Update existing code
      await env.nelchan_db
        .prepare(`UPDATE codes SET code = ? WHERE id = ?`)
        .bind(commandContent, existingCommand.code_id)
        .run()
    } else {
      // Insert new code for existing command
      const codeID = crypto.randomUUID()
      await env.nelchan_db
        .prepare(`INSERT INTO codes (id, command_id, code) VALUES (?, ?, ?)`)
        .bind(codeID, existingCommand.id, commandContent)
        .run()
    }
    console.log("[registerCodeCommand] command updated")
  } else {
    const commandID = crypto.randomUUID()
    const codeID = crypto.randomUUID()

    console.log("[registerCodeCommand] INSERT commands table", commandID)
    await env.nelchan_db
      .prepare(`INSERT INTO commands (id, name, author_id) VALUES (?, ?, ?)`)
      .bind(commandID, commandName, authorID)
      .run()

    console.log("[registerCodeCommand] INSERT codes table", codeID)
    await env.nelchan_db
      .prepare(`INSERT INTO codes (id, command_id, code) VALUES (?, ?, ?)`)
      .bind(codeID, commandID, commandContent)
      .run()

    console.log("[registerCodeCommand] command registered")
  }
}

/*
 * Register a text command (upsert)
 * @param env - The environment
 * @param commandName - The name of the command
 * @param commandContent - The content of the command
 * @param authorID - The ID of the author
 */
export type GetCommandResult = {
  name: string
  isCode: boolean
  content: string
}

/**
 * Get command content by name
 * @param env - The environment
 * @param commandName - The name of the command
 * @returns The command content or undefined if not found
 */
export const getCommand = async (
  env: Env,
  commandName: string
): Promise<GetCommandResult | undefined> => {
  // First try to find a code command
  const codeResult = await env.nelchan_db
    .prepare(
      `SELECT 
        c.id, 
        c.name, 
        co.code 
        FROM commands c  
        INNER JOIN codes co ON c.id = co.command_id 
        WHERE c.name = ? LIMIT 1`
    )
    .bind(commandName)
    .first<{ id: string; name: string; code: string }>()

  if (codeResult && codeResult.code) {
    return {
      name: codeResult.name,
      isCode: true,
      content: codeResult.code,
    }
  }

  // Then try to find a text command
  const textResult = await env.nelchan_db
    .prepare(
      `SELECT 
        c.id, 
        c.name, 
        d.text 
        FROM commands c  
        INNER JOIN dictionaries d ON c.id = d.command_id 
        WHERE c.name = ? LIMIT 1`
    )
    .bind(commandName)
    .first<{ id: string; name: string; text: string }>()

  if (textResult && textResult.text) {
    return {
      name: textResult.name,
      isCode: false,
      content: textResult.text,
    }
  }

  return undefined
}

export const registerTextCommand = async (
  env: Env,
  commandName: string,
  commandContent: string,
  authorID: string
) => {
  console.log("[registerTextCommand] commandName: ", commandName)
  console.log("[registerTextCommand] commandContent: ", commandContent)

  // Check if command already exists
  const existingCommand = await env.nelchan_db
    .prepare(
      `SELECT c.id, d.id as dictionary_id, co.id as code_id FROM commands c LEFT JOIN dictionaries d ON c.id = d.command_id LEFT JOIN codes co ON c.id = co.command_id WHERE c.name = ?`
    )
    .bind(commandName)
    .first<{
      id: string
      dictionary_id: string | null
      code_id: string | null
    }>()

  if (existingCommand) {
    console.log(
      "[registerTextCommand] UPDATE existing command",
      existingCommand.id
    )

    // Delete old code record if exists (codeCommand -> textCommand migration)
    if (existingCommand.code_id) {
      console.log(
        "[registerTextCommand] DELETE old code record",
        existingCommand.code_id
      )
      await env.nelchan_db
        .prepare(`DELETE FROM codes WHERE id = ?`)
        .bind(existingCommand.code_id)
        .run()
    }

    if (existingCommand.dictionary_id) {
      // Update existing dictionary
      await env.nelchan_db
        .prepare(`UPDATE dictionaries SET text = ? WHERE id = ?`)
        .bind(commandContent, existingCommand.dictionary_id)
        .run()
    } else {
      // Insert new dictionary for existing command
      const dictionaryID = crypto.randomUUID()
      await env.nelchan_db
        .prepare(
          `INSERT INTO dictionaries (id, command_id, text) VALUES (?, ?, ?)`
        )
        .bind(dictionaryID, existingCommand.id, commandContent)
        .run()
    }
    console.log("[registerTextCommand] command updated")
  } else {
    const commandID = crypto.randomUUID()
    const dictionaryID = crypto.randomUUID()

    console.log("[registerTextCommand] INSERT commands table", commandID)
    await env.nelchan_db
      .prepare(`INSERT INTO commands (id, name, author_id) VALUES (?, ?, ?)`)
      .bind(commandID, commandName, authorID)
      .run()

    console.log("[registerTextCommand] INSERT dictionaries table", dictionaryID)
    await env.nelchan_db
      .prepare(
        `INSERT INTO dictionaries (id, command_id, text) VALUES (?, ?, ?)`
      )
      .bind(dictionaryID, commandID, commandContent)
      .run()

    console.log("[registerTextCommand] command registered")
  }
}
