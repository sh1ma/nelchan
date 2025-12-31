/**
 * コンテキストビルダー
 * mllm v2用の3層コンテキスト（直近会話 + 類似メッセージ + ユーザー情報）を構築する
 */

import type {
  MLLMContext,
  RecentMessage,
  SimilarMessage,
} from "./types/discord"
import { getRecentMessages, getSimilarMessages } from "./messageService"
import { getUser } from "./userService"

/**
 * mllm v2用のコンテキストを構築
 */
export async function buildMLLMContext(
  env: Env,
  options: {
    channelId: string
    userId: string
    prompt: string
    recentCount?: number
    similarCount?: number
  }
): Promise<MLLMContext> {
  const recentCount = options.recentCount ?? 10
  const similarCount = options.similarCount ?? 5

  // 並列でデータを取得
  const [recentMessages, similarMessages, userInfo] = await Promise.all([
    // 直近メッセージを取得
    getRecentMessages(env, options.channelId, recentCount),
    // 類似メッセージを検索
    getSimilarMessages(env, options.prompt, {
      topK: similarCount,
      channelId: undefined,
    }),
    // ユーザー情報を取得
    getUser(env, options.userId),
  ])

  return {
    recentMessages,
    similarMessages,
    userInfo: userInfo
      ? {
          username: userInfo.username,
          displayName: userInfo.display_name,
        }
      : null,
  }
}

/**
 * コンテキストからプロンプトを生成
 */
export function buildPromptFromContext(
  context: MLLMContext,
  userMessage: string
): string {
  const parts: string[] = []

  // ユーザー情報
  if (context.userInfo) {
    parts.push("## ユーザー情報")
    const displayName =
      context.userInfo.displayName ?? context.userInfo.username
    parts.push(`発話者: ${displayName}`)
  }

  // 直近の会話
  if (context.recentMessages.length > 0) {
    parts.push("")
    parts.push("## 直近の会話")
    for (const msg of context.recentMessages) {
      parts.push(`${msg.username}: ${msg.content}`)
    }
  }

  // 関連する過去の情報
  if (context.similarMessages.length > 0) {
    parts.push("")
    parts.push("## 関連する過去の情報")
    for (const msg of context.similarMessages) {
      parts.push(`- ${msg.username}: ${msg.content}`)
    }
  }

  // ユーザーメッセージ
  parts.push("")
  parts.push("上記の情報を踏まえて、以下のメッセージに応答してください:")
  parts.push(userMessage)

  return parts.join("\n")
}

/**
 * コンテキストのサマリー情報を生成
 */
export function getContextSummary(context: MLLMContext): {
  recent_count: number
  similar_count: number
  user_found: boolean
} {
  return {
    recent_count: context.recentMessages.length,
    similar_count: context.similarMessages.length,
    user_found: context.userInfo !== null,
  }
}
