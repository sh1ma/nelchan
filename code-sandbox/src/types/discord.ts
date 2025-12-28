/**
 * Discord関連の型定義
 */

// Discord User entity (D1に保存)
export interface DiscordUser {
  id: string
  username: string
  display_name: string | null
  avatar_url: string | null
  updated_at: string
}

// Discord Message entity (D1に保存)
export interface DiscordMessage {
  id: string
  channel_id: string
  user_id: string
  content: string
  timestamp: string
  edited_timestamp: string | null
  reference_message_id: string | null
  mention_user_ids: string | null // JSON array
  mention_role_ids: string | null // JSON array
  has_attachments: number // 0 or 1
  is_vectorized: number // 0 or 1
  created_at: string
}

// Vectorize metadata
export interface VectorizedMessageMetadata {
  content: string
  channel_id: string
  user_id: string
  username: string
  timestamp: string
}

// API Request types
export interface StoreMessageRequest {
  id: string
  channel_id: string
  user_id: string
  content: string
  timestamp: string
  edited_timestamp?: string | null
  reference_message_id?: string | null
  mention_user_ids?: string[]
  mention_role_ids?: string[]
  has_attachments?: boolean
  username: string
  display_name?: string | null
}

export interface UpdateMessageRequest {
  id: string
  content: string
  edited_timestamp: string
}

export interface DeleteMessageRequest {
  id: string
}

export interface EnhancedMllmRequest {
  prompt: string
  channel_id: string
  user_id: string
  recent_count?: number
  similar_count?: number
}

export interface FetchChannelRequest {
  channel_id: string
  limit?: number
}

// API Response types
export interface StoreMessageResponse {
  error: string | null
  stored: boolean
  vectorized: boolean
}

export interface EnhancedMllmResponse {
  error: string | null
  output: string | null
  context: {
    recent_count: number
    similar_count: number
    user_found: boolean
  }
}

export interface FetchChannelResponse {
  error: string | null
  fetched_count: number
  stored_count: number
  vectorized_count: number
}

// Context types for mllm
export interface RecentMessage {
  user_id: string
  username: string
  content: string
  timestamp: string
}

export interface SimilarMessage {
  content: string
  username: string
  score: number
}

export interface MLLMContext {
  recentMessages: RecentMessage[]
  similarMessages: SimilarMessage[]
  userInfo: {
    username: string
    displayName: string | null
  } | null
}
