/**
 * ユーザーサービス
 * Discord ユーザー情報のCRUD操作を行う
 */

import type { DiscordUser } from "./types/discord"

/**
 * ユーザー情報を取得
 */
export async function getUser(
  env: Env,
  userId: string
): Promise<DiscordUser | null> {
  const result = await env.nelchan_db
    .prepare(
      `SELECT id, username, display_name, avatar_url, updated_at
       FROM discord_users
       WHERE id = ?`
    )
    .bind(userId)
    .first<DiscordUser>()

  return result ?? null
}

/**
 * 複数のユーザー情報を取得
 */
export async function getUsers(
  env: Env,
  userIds: string[]
): Promise<DiscordUser[]> {
  if (userIds.length === 0) {
    return []
  }

  // SQLインジェクション対策としてプレースホルダを使用
  const placeholders = userIds.map(() => "?").join(", ")
  const query = `SELECT id, username, display_name, avatar_url, updated_at
                 FROM discord_users
                 WHERE id IN (${placeholders})`

  const result = await env.nelchan_db
    .prepare(query)
    .bind(...userIds)
    .all<DiscordUser>()

  return result.results ?? []
}

/**
 * すべてのユーザー情報を取得
 */
export async function getAllUsers(env: Env): Promise<DiscordUser[]> {
  const result = await env.nelchan_db
    .prepare(
      `SELECT id, username, display_name, avatar_url, updated_at
       FROM discord_users`
    )
    .all<DiscordUser>()

  return result.results ?? []
}

/**
 * ユーザー情報を作成または更新（upsert）
 */
export async function upsertUser(
  env: Env,
  user: {
    id: string
    username: string
    display_name?: string | null
    avatar_url?: string | null
  }
): Promise<void> {
  await env.nelchan_db
    .prepare(
      `INSERT INTO discord_users (id, username, display_name, avatar_url, updated_at)
       VALUES (?, ?, ?, ?, datetime('now'))
       ON CONFLICT(id) DO UPDATE SET
         username = excluded.username,
         display_name = excluded.display_name,
         avatar_url = excluded.avatar_url,
         updated_at = datetime('now')`
    )
    .bind(
      user.id,
      user.username,
      user.display_name ?? null,
      user.avatar_url ?? null
    )
    .run()

  console.log(`[userService] upserted user: ${user.id}`)
}

/**
 * 複数のユーザー情報を一括作成または更新
 */
export async function upsertUsers(
  env: Env,
  users: {
    id: string
    username: string
    display_name?: string | null
    avatar_url?: string | null
  }[]
): Promise<void> {
  if (users.length === 0) {
    return
  }

  // バッチ処理で複数のユーザーを一括更新
  const statements = users.map((user) =>
    env.nelchan_db
      .prepare(
        `INSERT INTO discord_users (id, username, display_name, avatar_url, updated_at)
         VALUES (?, ?, ?, ?, datetime('now'))
         ON CONFLICT(id) DO UPDATE SET
           username = excluded.username,
           display_name = excluded.display_name,
           avatar_url = excluded.avatar_url,
           updated_at = datetime('now')`
      )
      .bind(
        user.id,
        user.username,
        user.display_name ?? null,
        user.avatar_url ?? null
      )
  )

  await env.nelchan_db.batch(statements)

  console.log(`[userService] upserted ${users.length} users`)
}

/**
 * ユーザー情報を削除
 */
export async function deleteUser(env: Env, userId: string): Promise<void> {
  await env.nelchan_db
    .prepare(`DELETE FROM discord_users WHERE id = ?`)
    .bind(userId)
    .run()

  console.log(`[userService] deleted user: ${userId}`)
}

/**
 * ユーザー情報を更新
 */
export async function updateUser(
  env: Env,
  userId: string,
  updates: {
    username?: string
    display_name?: string | null
    avatar_url?: string | null
  }
): Promise<void> {
  const setClauses: string[] = []
  const values: (string | null)[] = []

  if (updates.username !== undefined) {
    setClauses.push("username = ?")
    values.push(updates.username)
  }

  if (updates.display_name !== undefined) {
    setClauses.push("display_name = ?")
    values.push(updates.display_name)
  }

  if (updates.avatar_url !== undefined) {
    setClauses.push("avatar_url = ?")
    values.push(updates.avatar_url)
  }

  if (setClauses.length === 0) {
    return
  }

  setClauses.push("updated_at = datetime('now')")
  values.push(userId)

  const query = `UPDATE discord_users SET ${setClauses.join(", ")} WHERE id = ?`

  await env.nelchan_db
    .prepare(query)
    .bind(...values)
    .run()

  console.log(`[userService] updated user: ${userId}`)
}
