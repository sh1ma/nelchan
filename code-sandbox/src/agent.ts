import { Agent } from "agents"
import { hasToolCall, streamText } from "ai"
import console from "console"
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

    if (url.pathname.endsWith("/connect") && request.method === "POST") {
      console.log("[NelchanAgent] llmWithAgent request")
      const { servers } = this.getMcpServers()
      if (Object.keys(servers).length === 0) {
        const result = await this.addMcpServer(
          "Playwright MCP",
          "http://10.99.99.1:8931/mcp"
        )
        if (result.state === "authenticating") {
          return new Response(JSON.stringify({ authUrl: result.authUrl }), {
            headers: { "Content-Type": "application/json" },
          })
        }
        console.log("[NelchanAgent] MCP server added: ", result)
      }

      return new Response(
        JSON.stringify({ error: null, output: "Connected" }),
        { status: 200 }
      )
    }

    if (url.pathname.endsWith("/chat") && request.method === "POST") {
      const { prompt } = (await request.json()) as { prompt: string }
      const result = await this.handleChat(prompt)
      return Response.json({ error: null, output: result })
    }

    return new Response("Not found", { status: 404 })
  }

  async handleChat(prompt: string): Promise<string> {
    // MCPツールをAI SDK互換形式で取得
    const tools = this.mcp.getAITools()
    console.log(tools)
    console.log("[NelchanAgent] Available tools:", Object.keys(tools))

    // Workers AIで推論（MCPツール付き）
    const workersAI = createWorkersAI({ binding: this.env.AI })
    const result = streamText({
      model: workersAI("@cf/meta/llama-4-scout-17b-16e-instruct"),
      prompt,
      tools,
      stopWhen: hasToolCall("tool_2QhcLAi4_browser_run_code"),
      toolChoice: "required",
      activeTools: Object.keys(tools),
      system: `常に日本語で応答してください。MCPを使用し与えられたプロンプトに対して、適切な回答を生成してください。`,
    })

    // ストリームを逐次読み出してログ出力
    let fullText = ""
    for await (const chunk of result.textStream) {
      // console.log("[NelchanAgent] chunk:", chunk)
      fullText += chunk
    }

    console.log((await result.steps).map((e) => e.content))

    return fullText
  }
}
