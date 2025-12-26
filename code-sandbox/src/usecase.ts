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

export const runCommand = async (env: Env, commandName: string) => {
  const result = await env.nelchan_db
    .prepare(
      `SELECT 
    c.id, 
    c.name, 
    c.author_id, 
    d.id as dictionary_id, 
    d.text, 
    co.id as code_id, 
    co.code 
    FROM commands c  
    LEFT JOIN dictionaries d ON c.id = d.command_id 
    LEFT JOIN codes co ON c.id = co.command_id 
    WHERE c.name = ? LIMIT 1`
    )
    .bind(commandName)
    .first<NelchanGetCommandResult>()

  if (!result) {
    console.log(`Command ${commandName} not found`)
    return undefined
  }

  const { type, content } = getCommandContent(result)

  if (type === "code") {
    const randomString = crypto.randomUUID()
    const sandbox = getSandbox(env.Sandbox, randomString)
    const ctx = await sandbox.createCodeContext({ language: "python" })
    const executionResult = await sandbox.runCode(content, { context: ctx })
    const codeResultOutput = executionResult.logs.stdout.join("\n")
    return {
      id: result.id,
      name: result.name,
      content: codeResultOutput,
    }
  }

  if (type === "text") {
    return {
      id: result.id,
      name: result.name,
      content: getCommandContent(result),
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
      `SELECT c.id, co.id as code_id FROM commands c LEFT JOIN codes co ON c.id = co.command_id WHERE c.name = ?`
    )
    .bind(commandName)
    .first<{ id: string; code_id: string | null }>()

  if (existingCommand) {
    console.log(
      "[registerCodeCommand] UPDATE existing command",
      existingCommand.id
    )
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
      `SELECT c.id, d.id as dictionary_id FROM commands c LEFT JOIN dictionaries d ON c.id = d.command_id WHERE c.name = ?`
    )
    .bind(commandName)
    .first<{ id: string; dictionary_id: string | null }>()

  if (existingCommand) {
    console.log(
      "[registerTextCommand] UPDATE existing command",
      existingCommand.id
    )
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
