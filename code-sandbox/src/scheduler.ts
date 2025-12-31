/**
 * Scheduled Trigger ハンドラ
 * 定期的にユーザー情報を更新する
 */

import { getAllUsers, upsertUsers } from "./userService"

/**
 * Scheduled Trigger のメインハンドラ
 * 毎日00:00 UTCに実行される
 */
export async function handleScheduled(
  controller: ScheduledController,
  env: Env,
  ctx: ExecutionContext
): Promise<void> {
  console.log("[scheduler] triggered at:", new Date().toISOString())
  console.log("[scheduler] cron:", controller.cron)

  try {
    await updateAllUsers(env)
    console.log("[scheduler] completed successfully")
  } catch (error) {
    console.error("[scheduler] error:", error)
    throw error
  }
}

/**
 * すべてのユーザー情報を更新
 * 現時点ではDBに保存されているユーザー情報をそのまま維持
 * 将来的にはDiscord APIから最新情報を取得して更新する
 */
async function updateAllUsers(env: Env): Promise<void> {
  // DBから全ユーザーを取得
  const users = await getAllUsers(env)
  console.log(`[scheduler] found ${users.length} users to check`)

  if (users.length === 0) {
    console.log("[scheduler] no users to update")
    return
  }

  // 現時点では、メッセージ保存時にユーザー情報が更新されるため
  // スケジューラーでは特別な処理は不要
  // 将来的にDiscord APIを使用して最新情報を取得する場合は
  // ここでAPIコールを行う

  console.log(
    `[scheduler] user update check completed for ${users.length} users`
  )
}
