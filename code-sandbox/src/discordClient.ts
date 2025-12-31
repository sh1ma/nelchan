/**
 * Discord APIクライアント
 * チャンネルからメッセージ履歴を取得する
 */

import type { StoreMessageRequest } from "./types/discord"

/**
 * Discord API経由でメッセージを取得するためのレスポンス型
 */
interface DiscordAPIMessage {
  id: string
  channel_id: string
  author: {
    id: string
    username: string
    global_name?: string | null
    avatar?: string | null
  }
  content: string
  timestamp: string
  edited_timestamp?: string | null
  message_reference?: {
    message_id?: string
  } | null
  mentions: Array<{
    id: string
  }>
  mention_roles: string[]
  attachments: Array<unknown>
}

/**
 * チャンネルからメッセージ履歴を取得
 */
export async function fetchChannelMessages(
  env: Env,
  channelId: string,
  limit: number = 100
): Promise<StoreMessageRequest[]> {
  const botToken = env.DISCORD_BOT_TOKEN

  if (!botToken) {
    throw new Error("DISCORD_BOT_TOKEN is not configured")
  }

  const messages: StoreMessageRequest[] = []
  let beforeId: string | undefined = undefined
  let remaining = limit

  // Discord APIは1リクエストで最大100件
  while (remaining > 0) {
    const fetchCount = Math.min(remaining, 100)
    const url = new URL(
      `https://discord.com/api/v10/channels/${channelId}/messages`
    )
    url.searchParams.set("limit", fetchCount.toString())
    if (beforeId) {
      url.searchParams.set("before", beforeId)
    }

    const response = await fetch(url.toString(), {
      headers: {
        Authorization: `Bot ${botToken}`,
        "Content-Type": "application/json",
      },
    })

    if (!response.ok) {
      const errorText = await response.text()
      throw new Error(
        `Discord API error: ${response.status} ${response.statusText} - ${errorText}`
      )
    }

    const apiMessages: DiscordAPIMessage[] = await response.json()

    if (apiMessages.length === 0) {
      break
    }

    for (const msg of apiMessages) {
      // Bot自身のメッセージはスキップ
      if (msg.author.id === env.DISCORD_BOT_ID) {
        continue
      }

      const storeRequest: StoreMessageRequest = {
        id: msg.id,
        channel_id: msg.channel_id,
        user_id: msg.author.id,
        content: msg.content,
        timestamp: msg.timestamp,
        edited_timestamp: msg.edited_timestamp ?? undefined,
        reference_message_id: msg.message_reference?.message_id ?? undefined,
        mention_user_ids: msg.mentions.map((m) => m.id),
        mention_role_ids: msg.mention_roles,
        has_attachments: msg.attachments.length > 0,
        username: msg.author.username,
        display_name: msg.author.global_name ?? undefined,
      }

      messages.push(storeRequest)
    }

    // 次のページのための準備
    beforeId = apiMessages[apiMessages.length - 1].id
    remaining -= apiMessages.length

    // レート制限を考慮して少し待つ
    if (remaining > 0) {
      await new Promise((resolve) => setTimeout(resolve, 200))
    }
  }

  console.log(
    `[discordClient] fetched ${messages.length} messages from channel ${channelId}`
  )

  return messages
}
