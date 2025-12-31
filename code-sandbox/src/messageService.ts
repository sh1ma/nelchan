/**
 * メッセージサービス
 * Discord メッセージのCRUD操作とベクトル化連携を行う
 */

import type {
  DiscordMessage,
  StoreMessageRequest,
  StoreMessageResponse,
  RecentMessage,
  SimilarMessage,
  VectorizedMessageMetadata,
} from './types/discord'
import { filterMessage } from './filters'
import {
  upsertMessageVector,
  deleteMessageVector,
  searchSimilarMessages,
} from './vectorService'
import { upsertUser } from './userService'

/**
 * メッセージを取得
 */
export async function getMessage(
  env: Env,
  messageId: string
): Promise<DiscordMessage | null> {
  const result = await env.nelchan_db
    .prepare(
      `SELECT id, channel_id, user_id, content, timestamp, edited_timestamp,
              reference_message_id, mention_user_ids, mention_role_ids,
              has_attachments, is_vectorized, created_at
       FROM discord_messages
       WHERE id = ?`
    )
    .bind(messageId)
    .first<DiscordMessage>()

  return result ?? null
}

/**
 * 直近N件のメッセージを取得（チャンネル指定）
 */
export async function getRecentMessages(
  env: Env,
  channelId: string,
  limit: number = 10
): Promise<RecentMessage[]> {
  const result = await env.nelchan_db
    .prepare(
      `SELECT m.user_id, u.username, m.content, m.timestamp
       FROM discord_messages m
       LEFT JOIN discord_users u ON m.user_id = u.id
       WHERE m.channel_id = ?
       ORDER BY m.timestamp DESC
       LIMIT ?`
    )
    .bind(channelId, limit)
    .all<{
      user_id: string
      username: string | null
      content: string
      timestamp: string
    }>()

  // 時系列順に並び替え（古い順）
  const messages = (result.results ?? []).reverse()

  return messages.map((m) => ({
    user_id: m.user_id,
    username: m.username ?? 'Unknown',
    content: m.content,
    timestamp: m.timestamp,
  }))
}

/**
 * 類似メッセージを検索
 */
export async function getSimilarMessages(
  env: Env,
  query: string,
  options?: {
    topK?: number
    channelId?: string
  }
): Promise<SimilarMessage[]> {
  const results = await searchSimilarMessages(env, query, {
    topK: options?.topK ?? 5,
    channelId: options?.channelId,
  })

  return results.map((r) => ({
    content: r.metadata.content,
    username: r.metadata.username,
    score: r.score,
  }))
}

/**
 * メッセージを保存（フィルタリング + ベクトル化連携）
 */
export async function storeMessage(
  env: Env,
  request: StoreMessageRequest
): Promise<StoreMessageResponse> {
  // フィルタリング判定
  const hasAttachments = request.has_attachments ?? false
  const filterResult = filterMessage(request.content, hasAttachments)

  if (!filterResult.shouldStore) {
    console.log(
      `[messageService] skipped storing message: ${request.id} (${filterResult.reason})`
    )
    return {
      error: null,
      stored: false,
      vectorized: false,
    }
  }

  // ユーザー情報も一緒に保存
  await upsertUser(env, {
    id: request.user_id,
    username: request.username,
    display_name: request.display_name,
  })

  // mention_user_ids と mention_role_ids を JSON 文字列に変換
  const mentionUserIds = request.mention_user_ids
    ? JSON.stringify(request.mention_user_ids)
    : null
  const mentionRoleIds = request.mention_role_ids
    ? JSON.stringify(request.mention_role_ids)
    : null

  // メッセージを DB に保存
  await env.nelchan_db
    .prepare(
      `INSERT INTO discord_messages
       (id, channel_id, user_id, content, timestamp, edited_timestamp,
        reference_message_id, mention_user_ids, mention_role_ids,
        has_attachments, is_vectorized, created_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
       ON CONFLICT(id) DO UPDATE SET
         content = excluded.content,
         edited_timestamp = excluded.edited_timestamp,
         mention_user_ids = excluded.mention_user_ids,
         mention_role_ids = excluded.mention_role_ids,
         has_attachments = excluded.has_attachments,
         is_vectorized = excluded.is_vectorized`
    )
    .bind(
      request.id,
      request.channel_id,
      request.user_id,
      request.content,
      request.timestamp,
      request.edited_timestamp ?? null,
      request.reference_message_id ?? null,
      mentionUserIds,
      mentionRoleIds,
      hasAttachments ? 1 : 0,
      filterResult.shouldVectorize ? 1 : 0
    )
    .run()

  console.log(`[messageService] stored message: ${request.id}`)

  // ベクトル化が必要な場合
  if (filterResult.shouldVectorize) {
    const metadata: VectorizedMessageMetadata = {
      content: request.content,
      channel_id: request.channel_id,
      user_id: request.user_id,
      username: request.username,
      timestamp: request.timestamp,
    }

    await upsertMessageVector(env, request.id, request.content, metadata)

    return {
      error: null,
      stored: true,
      vectorized: true,
    }
  }

  return {
    error: null,
    stored: true,
    vectorized: false,
  }
}

/**
 * メッセージを更新（再ベクトル化含む）
 */
export async function updateMessage(
  env: Env,
  messageId: string,
  content: string,
  editedTimestamp: string
): Promise<StoreMessageResponse> {
  // 既存メッセージを取得
  const existingMessage = await getMessage(env, messageId)
  if (!existingMessage) {
    return {
      error: 'Message not found',
      stored: false,
      vectorized: false,
    }
  }

  // フィルタリング判定
  const hasAttachments = existingMessage.has_attachments === 1
  const filterResult = filterMessage(content, hasAttachments)

  // メッセージを更新
  await env.nelchan_db
    .prepare(
      `UPDATE discord_messages
       SET content = ?, edited_timestamp = ?, is_vectorized = ?
       WHERE id = ?`
    )
    .bind(
      content,
      editedTimestamp,
      filterResult.shouldVectorize ? 1 : 0,
      messageId
    )
    .run()

  console.log(`[messageService] updated message: ${messageId}`)

  // ベクトル化の処理
  if (filterResult.shouldVectorize) {
    // ユーザー情報を取得
    const userResult = await env.nelchan_db
      .prepare(`SELECT username FROM discord_users WHERE id = ?`)
      .bind(existingMessage.user_id)
      .first<{ username: string }>()

    const username = userResult?.username ?? 'Unknown'

    const metadata: VectorizedMessageMetadata = {
      content,
      channel_id: existingMessage.channel_id,
      user_id: existingMessage.user_id,
      username,
      timestamp: existingMessage.timestamp,
    }

    await upsertMessageVector(env, messageId, content, metadata)

    return {
      error: null,
      stored: true,
      vectorized: true,
    }
  } else if (existingMessage.is_vectorized === 1) {
    // 以前はベクトル化されていたが、今回はベクトル化対象外になった場合
    await deleteMessageVector(env, messageId)
  }

  return {
    error: null,
    stored: true,
    vectorized: false,
  }
}

/**
 * メッセージを削除（ベクトル削除含む）
 */
export async function deleteMessage(
  env: Env,
  messageId: string
): Promise<{ error: string | null; success: boolean }> {
  // 既存メッセージを取得
  const existingMessage = await getMessage(env, messageId)
  if (!existingMessage) {
    return {
      error: 'Message not found',
      success: false,
    }
  }

  // ベクトルを削除（ベクトル化されていた場合）
  if (existingMessage.is_vectorized === 1) {
    await deleteMessageVector(env, messageId)
  }

  // メッセージを削除
  await env.nelchan_db
    .prepare(`DELETE FROM discord_messages WHERE id = ?`)
    .bind(messageId)
    .run()

  console.log(`[messageService] deleted message: ${messageId}`)

  return {
    error: null,
    success: true,
  }
}

/**
 * 複数のメッセージを一括保存
 */
export async function storeMessages(
  env: Env,
  messages: StoreMessageRequest[]
): Promise<{
  stored_count: number
  vectorized_count: number
}> {
  let storedCount = 0
  let vectorizedCount = 0

  for (const message of messages) {
    const result = await storeMessage(env, message)
    if (result.stored) {
      storedCount++
    }
    if (result.vectorized) {
      vectorizedCount++
    }
  }

  return {
    stored_count: storedCount,
    vectorized_count: vectorizedCount,
  }
}
