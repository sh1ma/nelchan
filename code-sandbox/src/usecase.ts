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
  envVars: Record<string, string>,
  args: string[]
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
    const sandboxId = "nelchan-sandbox"
    console.log(`[runCommand] using sandbox: ${sandboxId}`)
    const sandbox = getSandbox(env.Sandbox, sandboxId)
    try {
      const apiKey = env.NELCHAN_API_KEY
      const envEmbededCode = `
import requests
${Object.entries(envVars)
  .map(([key, value]) => `${key}="${value}"`)
  .join("\n")}
args = ${JSON.stringify(args)}

def cs(s: str):
    return f"\`\`\`{s}\`\`\`"

def llm(prompt: str):
  return requests.post("https://my-sandbox.sh1ma.workers.dev/llm", json={"prompt": prompt}, headers={"Authorization": "Bearer ${apiKey}"}).json()["output"]

def mget(query: str, top_k: int = 6):
  return requests.post("https://my-sandbox.sh1ma.workers.dev/mget", json={"query": query, "topK": top_k}, headers={"Authorization": "Bearer ${apiKey}"}).json()["results"]

def automemory(text: str):
  return requests.post("https://my-sandbox.sh1ma.workers.dev/automemory", json={"text": text}, headers={"Authorization": "Bearer ${apiKey}"}).json()["count"]

def mllm(prompt: str, top_k: int = 6):
  return requests.post("https://my-sandbox.sh1ma.workers.dev/mllm", json={"prompt": prompt, "topK": top_k}, headers={"Authorization": "Bearer ${apiKey}"}).json()["output"]

def llmWithAgent(prompt: str):
  return requests.post("https://my-sandbox.sh1ma.workers.dev/llmWithAgent", json={"prompt": prompt}, headers={"Authorization": "Bearer ${apiKey}"}).json()["output"]

${result.code}
`
      console.log(envEmbededCode)
      const executionResult = await sandbox.runCode(envEmbededCode)
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
    }
    // destroy()を呼ばないことでコンテナを再利用する
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

/**
 * Store a memory with vector embeddings for RAG
 * @param env - The environment
 * @param id - The unique ID for the memory (UUID)
 * @param content - The content to store
 */
export const storeMemory = async (env: Env, id: string, content: string) => {
  console.log("[storeMemory] id: ", id)
  console.log("[storeMemory] content: ", content)

  // 1. Generate embedding using Workers AI
  const embeddingResponse = await env.AI.run("@cf/baai/bge-base-en-v1.5", {
    text: [content],
  })

  if (!("data" in embeddingResponse) || !embeddingResponse.data) {
    throw new Error("Failed to generate embedding")
  }

  const embedding = embeddingResponse.data[0]

  // 2. Upsert vector into Vectorize
  await env.VECTORIZE.upsert([
    {
      id,
      values: embedding,
      metadata: { content },
    },
  ])

  console.log("[storeMemory] memory stored successfully")
}

/**
 * Get memories by semantic search
 * @param env - The environment
 * @param query - The query to search for
 * @param topK - The number of results to return (default: 3)
 * @returns Array of matching memories with content and score
 */
export const getMemory = async (
  env: Env,
  query: string,
  topK: number = 6
): Promise<{ id: string; content: string; score: number }[]> => {
  console.log("[getMemory] query: ", query)

  // 1. Generate embedding for query
  const embeddingResponse = await env.AI.run("@cf/baai/bge-base-en-v1.5", {
    text: [query],
  })

  if (!("data" in embeddingResponse) || !embeddingResponse.data) {
    throw new Error("Failed to generate embedding")
  }

  const embedding = embeddingResponse.data[0]

  // 2. Query Vectorize for similar vectors
  const results = await env.VECTORIZE.query(embedding, {
    topK,
    returnMetadata: "all",
  })

  console.log("[getMemory] results: ", results)

  return results.matches.map((m) => ({
    id: m.id,
    content: (m.metadata?.content as string) ?? "",
    score: m.score,
  }))
}

/**
 * Extract output text from Workers AI response
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const extractLLMOutput = (response: any): string | null => {
  if (response.status !== "completed") {
    return null
  }
  const output = response.output as
    | { type: string; content?: { type: string; text?: string }[] }[]
    | undefined
  const messageOutput = output?.find((item) => item.type === "message")
  return (
    messageOutput?.content?.find((item) => item.type === "output_text")?.text ??
    null
  )
}

/**
 * Auto-extract and store memories from text using LLM
 * @param env - The environment
 * @param text - The text to extract memories from
 * @returns Number of memories stored
 */
export const autoStoreMemory = async (
  env: Env,
  text: string
): Promise<number> => {
  console.log("[autoStoreMemory] text: ", text)

  // 1. Ask LLM to extract memorable information
  const extractionPrompt = `以下のテキストから、意味のある情報を抽出してください。

## ルール

- 情報はできるだけ多く抽出してください。
- ユーザーと情報を紐づけてください。
  - 例: ユーザー「A」がいた場合
    - Aは果物が好き
    - Aはxxxと言っていた
    - AはBにxxxを教えた
- 重要なキーワード、イベント、事実、関係性などを含めてください。
- 抽出する情報は具体的であればあるほど良いです。
- 抽出した情報は後で検索や参照ができるようにしてください。
- 抽出した情報は簡潔にまとめてください。
  

各情報について、内容を表すcontentを生成してください。
[{"content": "..."}, ...]
記憶すべき情報がない場合は空配列[]を返してください。


テキスト:
${text}`

  const response = await env.AI.run(
    "@cf/deepseek-ai/deepseek-r1-distill-qwen-32b",
    {
      messages: [
        {
          role: "system",
          content: extractionPrompt,
        },
        {
          role: "user",
          content: extractionPrompt,
        },
      ],
      response_format: {
        type: "json_schema",
        json_schema: {
          name: "memories",
          schema: {
            type: "object",
            properties: {
              memories: {
                type: "array",
                items: {
                  type: "object",
                  properties: {
                    content: {
                      type: "string",
                      description: "内容テキスト",
                    },
                  },
                  required: ["content"],
                },
              },
            },
            required: ["memories"],
          },
        },
      },
    }
  )

  const outputText = response.response ?? null
  console.log("[autoStoreMemory] LLM output: ", outputText)

  if (!outputText) {
    console.log("[autoStoreMemory] No output from LLM")
    return 0
  }

  // 2. Parse JSON and store each memory
  try {
    // 変更しないこと
    const { memories } = outputText as unknown as {
      memories: { content: string }[]
    }

    if (!Array.isArray(memories) || memories.length === 0) {
      console.log("[autoStoreMemory] No memories to store")
      return 0
    }

    // Store each memory with a unique key
    let storedCount = 0
    for (const memory of memories) {
      if (memory.content) {
        const key = crypto.randomUUID()
        await storeMemory(env, key, memory.content)
        storedCount++
      }
    }

    console.log(`[autoStoreMemory] Stored ${storedCount} memories`)
    return storedCount
  } catch (error) {
    console.error("[autoStoreMemory] Failed to parse JSON: ", error)
    return 0
  }
}

/**
 * LLM with memory context (RAG)
 * @param env - The environment
 * @param prompt - The user prompt
 * @param topK - Number of memories to retrieve (default: 3)
 * @returns LLM response with memory context
 */
export const memoryLLM = async (
  env: Env,
  prompt: string,
  topK: number = 6
): Promise<string | null> => {
  console.log("[memoryLLM] prompt: ", prompt)

  // 1. Get relevant memories
  const memories = await getMemory(env, prompt, topK)
  console.log("[memoryLLM] retrieved memories: ", memories)

  // 2. Build context from memories
  let context = ""
  if (memories.length > 0) {
    context = "以下は関連する記憶です:\n"
    context += memories.map((m) => `- ${m.content}`).join("\n")
    context += "\n\n"
  }

  // 3. Call LLM with context
  const response = await env.AI.run("@cf/openai/gpt-oss-20b", {
    input: prompt,
    max_output_tokens: 1000,
    instructions: `あなたはDiscord上で動くBotのねるちゃんです。
ねるちゃんの好きな言葉は「破壊された日常」です。
ねるちゃんはサイバーパンクやディストピア的な世界観が大好きです。
語尾は「ですわ」「ですわね」などといったお嬢様口調です。
例：終焉ですわね。

ねるちゃんの決め台詞は「あんたはここでねると死ぬのよ」です。

${context}上記の記憶を参考にして回答してください。`,
  })

  const outputText = extractLLMOutput(response)
  console.log("[memoryLLM] output: ", outputText)

  return outputText
}

/**
 * Generate Python code from natural language description using LLM
 * @param env - The environment
 * @param description - Natural language description of the command
 * @returns Generated Python code
 */
export type GeneratedCommand = {
  code: string
  usage: string
}

/**
 * Get the mention command name
 * @param env - The environment
 * @returns The mention command name or null if not set
 */
export const getMentionCommand = async (
  env: Env
): Promise<string | null> => {
  const result = await env.nelchan_db
    .prepare(`SELECT mention_command FROM settings LIMIT 1`)
    .first<{ mention_command: string | null }>()

  return result?.mention_command ?? null
}

/**
 * Set the mention command name
 * @param env - The environment
 * @param commandName - The command name to set (null to clear)
 */
export const setMentionCommand = async (
  env: Env,
  commandName: string | null
): Promise<void> => {
  await env.nelchan_db
    .prepare(`UPDATE settings SET mention_command = ?`)
    .bind(commandName)
    .run()

  console.log(`[setMentionCommand] set to: ${commandName}`)
}

export const generateCodeFromDescription = async (
  env: Env,
  commandName: string,
  description: string
): Promise<GeneratedCommand> => {
  console.log("[generateCodeFromDescription] commandName: ", commandName)
  console.log("[generateCodeFromDescription] description: ", description)

  const systemPrompt = `あなたはPythonコード生成の専門家です。ユーザーの説明に基づいて、Pythonコードを生成してください。

## コマンドの呼び出し形式
このコードは「!${commandName}」というDiscordコマンドとして登録されます。
ユーザーはDiscordで「!${commandName} 引数1 引数2 ...」のように呼び出します。
引数はスペース区切りで、args変数にリストとして格納されます。
コマンドを実行するとPythonコードが実行されます。

## 利用可能な変数（グローバルで定義済み）
- username: str - コマンドを実行したユーザーの表示名
- user_id: str - コマンドを実行したユーザーのID
- user_avatar: str - コマンドを実行したユーザーのアバターURL
- args: list[str] - コマンドの引数リスト（例: 「!${commandName} foo bar」なら args = ["foo", "bar"]）

## 利用可能な関数（グローバルで定義済み）
- llm(prompt: str) -> str: LLMを使ってテキストを生成する
- mget(query: str, top_k: int = 6) -> list[dict]: メモリから関連する情報を検索する（RAG）
- mllm(prompt: str, top_k: int = 6) -> str: メモリを参照してLLMで回答を生成する
- automemory(text: str) -> int: テキストから情報を抽出してメモリに保存する
- cs(s: str) -> str: 文字列をコードブロック（\`\`\`）で囲む

## 出力形式
以下の2つを生成してください：
1. code: Pythonコード（print()で結果を出力）
2. usage: コマンドの使い方説明（「!${commandName} <引数>」の形式で、引数の説明とどんな結果が返るかを記載）

## 注意事項
- 出力はprint()で行ってください
- 関数定義のみではなく、スタンドアロンで実行できるようにしてください。
- Pythonコードを実行したとき、結果がprint()で出力されるようにしてください。
- import文は必要に応じて追加可能です（requestsは既にインポート済み）
- コードのみを出力し、説明やマークダウンは含めないでください`

  const response = await env.AI.run(
    "@cf/deepseek-ai/deepseek-r1-distill-qwen-32b",
    {
      messages: [
        { role: "system", content: systemPrompt },
        {
          role: "user",
          content: `以下の説明に基づいて「!${commandName}」コマンドのPythonコードを生成してください:\n\n${description}`,
        },
      ],
      response_format: {
        type: "json_schema",
        json_schema: {
          name: "python_code",
          schema: {
            type: "object",
            properties: {
              code: {
                type: "string",
                description: "生成されたPythonコード",
              },
              usage: {
                type: "string",
                description:
                  "コマンドの使い方説明（引数の説明、どんな結果が返るか）",
              },
            },
            required: ["code", "usage"],
          },
        },
      },
    }
  )

  const outputText = response.response ?? null
  console.log("[generateCodeFromDescription] LLM output: ", outputText)

  if (!outputText) {
    throw new Error("LLMからの応答がありませんでした")
  }

  const { code, usage } = outputText as unknown as {
    code: string
    usage: string
  }

  if (!code || code.trim() === "") {
    throw new Error("有効なコードが生成されませんでした")
  }

  return {
    code: code.trim(),
    usage: usage?.trim() ?? "",
  }
}
