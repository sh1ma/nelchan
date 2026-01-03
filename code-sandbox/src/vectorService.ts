/**
 * ベクトル化サービス
 * embedding生成とVectorizeへのupsert/delete操作を行う
 */

import type { VectorizedMessageMetadata } from "./types/discord"

/**
 * テキストのembeddingを生成
 */
export async function generateEmbedding(
  env: Env,
  text: string
): Promise<number[]> {
  const resp = await (
    await env.YUKI_SERVICE.fetch("http://10.99.99.1:8000/embeddings", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        text,
      }),
    })
  ).json<{ embedding: number[] }>()

  return resp.embedding
}

/**
 * 複数テキストのembeddingを一括生成
 */
export async function generateEmbeddings(
  env: Env,
  texts: string[]
): Promise<number[][]> {
  if (texts.length === 0) {
    return []
  }

  const response = await env.AI.run("@cf/baai/bge-base-en-v1.5", {
    text: texts,
  })

  if (!("data" in response) || !response.data) {
    throw new Error("Failed to generate embeddings")
  }

  return response.data
}

/**
 * メッセージをベクトル化してVectorizeに保存
 */
export async function upsertMessageVector(
  env: Env,
  messageId: string,
  content: string,
  metadata: VectorizedMessageMetadata
): Promise<void> {
  const embedding = await generateEmbedding(env, content)

  await env.VECTORIZE.upsert([
    {
      id: messageId,
      values: embedding,
      metadata: {
        content: metadata.content,
        channel_id: metadata.channel_id,
        user_id: metadata.user_id,
        username: metadata.username,
        timestamp: metadata.timestamp,
      },
    },
  ])

  console.log(`[vectorService] upserted vector for message: ${messageId}`)
}

/**
 * 複数のメッセージを一括ベクトル化してVectorizeに保存
 */
export async function upsertMessageVectors(
  env: Env,
  messages: {
    id: string
    content: string
    metadata: VectorizedMessageMetadata
  }[]
): Promise<void> {
  if (messages.length === 0) {
    return
  }

  // 一括でembedding生成
  const contents = messages.map((m) => m.content)
  const embeddings = await generateEmbeddings(env, contents)

  // ベクトルデータを作成
  const vectors = messages.map((msg, index) => ({
    id: msg.id,
    values: embeddings[index],
    metadata: {
      content: msg.metadata.content,
      channel_id: msg.metadata.channel_id,
      user_id: msg.metadata.user_id,
      username: msg.metadata.username,
      timestamp: msg.metadata.timestamp,
    },
  }))

  await env.VECTORIZE.upsert(vectors)

  console.log(`[vectorService] upserted ${messages.length} vectors`)
}

/**
 * メッセージのベクトルを削除
 */
export async function deleteMessageVector(
  env: Env,
  messageId: string
): Promise<void> {
  await env.VECTORIZE.deleteByIds([messageId])

  console.log(`[vectorService] deleted vector for message: ${messageId}`)
}

/**
 * 複数のメッセージのベクトルを一括削除
 */
export async function deleteMessageVectors(
  env: Env,
  messageIds: string[]
): Promise<void> {
  if (messageIds.length === 0) {
    return
  }

  await env.VECTORIZE.deleteByIds(messageIds)

  console.log(`[vectorService] deleted ${messageIds.length} vectors`)
}

/**
 * 類似メッセージを検索
 */
export async function searchSimilarMessages(
  env: Env,
  query: string,
  options?: {
    topK?: number
    channelId?: string
  }
): Promise<
  {
    id: string
    score: number
    metadata: VectorizedMessageMetadata
  }[]
> {
  const topK = options?.topK ?? 5
  const embedding = await generateEmbedding(env, query)

  const queryOptions: {
    topK: number
    returnMetadata: "all"
    filter?: { channel_id: string }
  } = {
    topK,
    returnMetadata: "all",
  }

  // チャンネルフィルタがあれば追加
  if (options?.channelId) {
    queryOptions.filter = { channel_id: options.channelId }
  }

  const results = await env.VECTORIZE.query(embedding, queryOptions)

  return results.matches.map((match) => ({
    id: match.id,
    score: match.score,
    metadata: {
      content: (match.metadata?.content as string) ?? "",
      channel_id: (match.metadata?.channel_id as string) ?? "",
      user_id: (match.metadata?.user_id as string) ?? "",
      username: (match.metadata?.username as string) ?? "",
      timestamp: (match.metadata?.timestamp as string) ?? "",
    },
  }))
}
