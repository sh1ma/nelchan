import { Agent } from "agents"
import { generateText } from "ai"
import { createWorkersAI } from "workers-ai-provider"

// MCPサーバー設定（テスト用にハードコード）
const MCP_SERVERS = [
  { name: "Playwright MCP", url: "http://10.99.99.1:8931/mcp" },
]

export class NelchanAgent extends Agent<Env, never> {
  // onRequestを使うとリクエストコンテキスト内で処理できる
  async onRequest(request: Request): Promise<Response> {
    const url = new URL(request.url)

    console.log("[NelchanAgent] request: ", url.pathname)

    if (url.pathname.endsWith("/llmWithAgent") && request.method === "POST") {
      console.log("[NelchanAgent] llmWithAgent request")
      console.log(this.getMcpServers())
      const result = await this.addMcpServer(
        "Playwright MCP",
        "http://10.99.99.1:8931/mcp"
      )

      if (result.state === "authenticating") {
        return new Response(JSON.stringify({ authUrl: result.authUrl }), {
          headers: { "Content-Type": "application/json" },
        })
      }

      return new Response(`Connected: ${result.id}`, { status: 200 })
    }

    if (url.pathname.endsWith("/chat") && request.method === "POST") {
      const { prompt } = (await request.json()) as { prompt: string }
      const result = await this.chat(prompt)
      return Response.json({ error: null, output: result })
    }

    return new Response("Not found", { status: 404 })
  }

  async chat(prompt: string): Promise<string> {
    // MCPサーバーに接続（リクエストコンテキスト内なのでcallbackHost不要）
    for (const server of MCP_SERVERS) {
      const result = await this.addMcpServer(server.name, server.url)
      console.log(`[NelchanAgent] MCP server ${server.name}: ${result.state}`)
    }

    // MCPツールをAI SDK互換形式で取得
    const tools = this.mcp.getAITools()
    console.log("[NelchanAgent] Available tools:", Object.keys(tools))

    // Workers AIで推論（MCPツール付き）
    const workersAI = createWorkersAI({ binding: this.env.AI })
    const result = await generateText({
      model: workersAI("@cf/google/gemma-7b-it-lora"),
      prompt,
      tools,
      system: `常に日本語で応答してください。MCPを使用し与えられたプロンプトに対して、適切な回答を生成してください。`,
    })
    console.log(result)

    return result.text
  }
}
