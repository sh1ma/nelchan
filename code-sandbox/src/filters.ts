/**
 * ノイズフィルタリング関数
 * メッセージの保存・ベクトル化の判定を行う
 */

// URL のみかどうかを判定する正規表現
const URL_ONLY_PATTERN = /^\s*https?:\/\/\S+\s*$/

// 絵文字のみかどうかを判定する正規表現
// Discord カスタム絵文字: <:name:id> or <a:name:id>
const DISCORD_EMOJI_PATTERN = /<a?:\w+:\d+>/g

// Unicode 絵文字の範囲
const UNICODE_EMOJI_PATTERN =
  /[\u{1F300}-\u{1F9FF}]|[\u{2600}-\u{26FF}]|[\u{2700}-\u{27BF}]|[\u{FE00}-\u{FE0F}]|[\u{1F000}-\u{1FFFF}]|[\u{200D}]|[\u{20E3}]|[\u{1F1E0}-\u{1F1FF}]/gu

/**
 * URL のみのメッセージかどうかを判定
 */
export function isUrlOnly(content: string): boolean {
  return URL_ONLY_PATTERN.test(content)
}

/**
 * 絵文字のみのメッセージかどうかを判定
 */
export function isEmojiOnly(content: string): boolean {
  // Discord カスタム絵文字を除去
  let remaining = content.replace(DISCORD_EMOJI_PATTERN, '')

  // Unicode 絵文字を除去
  remaining = remaining.replace(UNICODE_EMOJI_PATTERN, '')

  // 空白を除去して残りがなければ絵文字のみ
  remaining = remaining.trim()

  return remaining.length === 0 && content.trim().length > 0
}

/**
 * Bot コマンドかどうかを判定
 */
export function isBotCommand(content: string): boolean {
  return content.startsWith('!')
}

/**
 * メッセージをDBに保存すべきかどうかを判定
 *
 * 保存しない条件:
 * - 5文字未満
 *
 * 保存する条件:
 * - 5文字以上
 * - 添付ファイルのみ（テキストなし）でも保存
 */
export function shouldStore(content: string, hasAttachments: boolean): boolean {
  // 添付ファイルのみでもDBには保存
  if (hasAttachments && content.length === 0) {
    return true
  }

  // 5文字未満は除外
  if (content.length < 5) {
    return false
  }

  return true
}

/**
 * メッセージをベクトル化すべきかどうかを判定
 *
 * ベクトル化しない条件:
 * - 5文字未満
 * - Bot コマンド（! プレフィックス）
 * - URL のみ
 * - 絵文字のみ
 * - 添付ファイルのみ（テキストなし）
 *
 * ベクトル化する条件:
 * - 上記以外の5文字以上のテキスト
 */
export function shouldVectorize(
  content: string,
  hasAttachments: boolean
): boolean {
  // 添付ファイルのみはベクトル化しない
  if (hasAttachments && content.length === 0) {
    return false
  }

  // 5文字未満は除外
  if (content.length < 5) {
    return false
  }

  // Bot コマンドは除外
  if (isBotCommand(content)) {
    return false
  }

  // URL のみは除外
  if (isUrlOnly(content)) {
    return false
  }

  // 絵文字のみは除外
  if (isEmojiOnly(content)) {
    return false
  }

  return true
}

/**
 * フィルタリング結果の型
 */
export interface FilterResult {
  shouldStore: boolean
  shouldVectorize: boolean
  reason?: string
}

/**
 * メッセージのフィルタリング結果を取得
 */
export function filterMessage(
  content: string,
  hasAttachments: boolean
): FilterResult {
  // 添付ファイルのみの場合
  if (hasAttachments && content.length === 0) {
    return {
      shouldStore: true,
      shouldVectorize: false,
      reason: 'attachment_only',
    }
  }

  // 5文字未満
  if (content.length < 5) {
    return {
      shouldStore: false,
      shouldVectorize: false,
      reason: 'too_short',
    }
  }

  // Bot コマンド
  if (isBotCommand(content)) {
    return {
      shouldStore: true,
      shouldVectorize: false,
      reason: 'bot_command',
    }
  }

  // URL のみ
  if (isUrlOnly(content)) {
    return {
      shouldStore: true,
      shouldVectorize: false,
      reason: 'url_only',
    }
  }

  // 絵文字のみ
  if (isEmojiOnly(content)) {
    return {
      shouldStore: true,
      shouldVectorize: false,
      reason: 'emoji_only',
    }
  }

  // 通常のメッセージ
  return {
    shouldStore: true,
    shouldVectorize: true,
  }
}
