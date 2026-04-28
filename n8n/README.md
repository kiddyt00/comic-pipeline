# n8n AI Comic Pipeline 搭建指南

本 workflow 是 AI 漫剧自动生产流水线的编排核心，负责串联 DeepSeek（剧本）、即梦（图片）、可灵（视频）、Edge-TTS（配音）和 FFmpeg（合成）6 个阶段，并通过回调 URL 驱动 Go 后端的事件状态机。

---

## 1. 快速启动 n8n

### Docker 一键启动

```bash
docker run -d --name n8n \
  -p 5678:5678 \
  -v n8n_data:/home/node/.n8n \
  -e N8N_SECURE_COOKIE=false \
  -e DEEPSEEK_API_KEY=sk-your-deepseek-key \
  -e JIMENG_API_KEY=your-jimeng-key \
  -e KLING_API_KEY=your-kling-key \
  n8nio/n8n
```

启动后访问：http://localhost:5678

> 如需持久化 workflow 文件，可以挂载本地目录：
> ```bash
> -v $(pwd)/n8n/workflows:/home/node/.n8n/workflows
> ```

### 环境变量说明

| 变量名 | 用途 | 必需 |
|---|---|---|
| `DEEPSEEK_API_KEY` | DeepSeek 剧本生成 | 是 |
| `JIMENG_API_KEY` | 即梦图片生成 | 否（可选 provider） |
| `KLING_API_KEY` | 可灵视频生成 | 否（可选 provider） |

---

## 2. 导入 Workflow

1. 打开 n8n 管理界面 http://localhost:5678
2. 点击右上角 **Import from File**
3. 选择 `n8n/workflow-template.json`
4. 导入后点击 **Save** 保存

或者手动创建，按下方 6 个步骤逐个添加节点。

---

## 3. Workflow 节点链（共 6 步）

### 整体数据流

```
Go Backend                         n8n Workflow
    |                                    |
    |  POST /webhook/episode-pipeline    |
    |  {project_id, world_setting,       |
    |   story_text, callback_url, ...}   |
    |----------------------------------->|
    |                                    | [Step 1] Webhook Trigger
    |                                    | [Step 2] DeepSeek 生成剧本+分镜
    |                                    | [Step 3] Code: 解析剧本 → POST callback_url
    |  POST /api/callback                |
    |  event=script_done                 |
    |<-----------------------------------|
    |                                    | [Step 4] Loop: 即梦图片生成
    |                                    |       每完成一个 scene → callback
    |  event=scene_image_done            |
    |<-----------------------------------|
    |                                    | [Step 5] Loop: 可灵视频生成
    |                                    |       每完成一个 scene → callback
    |  event=scene_video_done            |
    |<-----------------------------------|
    |                                    | [Step 6] Edge-TTS 配音 + FFmpeg 合成
    |                                    |       → callback event=episode_complete
    |  event=episode_complete            |
    |<-----------------------------------|
```

---

### Step 1: Webhook Trigger — 接收 Go 后端的流水线触发

- **节点类型**: Webhook
- **HTTP Method**: POST
- **Path**: `episode-pipeline`
- **Response Mode**: Last Node（或 Response Node）

Go 后端通过 `n8n.Client.TriggerEpisode()` 调用：
```
POST http://localhost:5678/webhook/episode-pipeline
```

入参 JSON 格式（来自 Go 的 `n8n.TriggerPayload`）：

```json
{
  "project_id": "proj-001",
  "episode_id": "ep-001",
  "episode_num": 1,
  "title": "第一章",
  "world_setting": "仙侠世界观...",
  "story_text": "故事正文...",
  "characters": [
    {"name": "主角", "description": "...", "ref_image_url": "...", "feature_tags": ["..."]}
  ],
  "callback_url": "http://host.docker.internal:8080/api/callback",
  "image_provider": "jimeng",
  "video_provider": "kling",
  "tts_provider": "edge-tts",
  "output_format": "mp4"
}
```

> **注意**: 若 Go 后端和 n8n 都在 Docker 中运行，callback_url 需使用 `host.docker.internal` 或 Docker 网络别名。

---

### Step 2: DeepSeek 剧本生成

- **节点类型**: HTTP Request
- **Method**: POST
- **URL**: `https://api.deepseek.com/v1/chat/completions`
- **Authentication**: Generic Credential Type（或直接 Header 注入）
- **Headers**:
  - `Authorization`: `Bearer {{ $env.DEEPSEEK_API_KEY }}`
  - `Content-Type`: `application/json`

**System Prompt**（在 Body JSON 中配置）:

```
你是专业的漫剧编剧。根据给定的世界观设定和故事文本，
生成适合制作漫剧的中文剧本和分镜脚本。

输出要求：
- 严格输出 JSON 格式，不要包含 markdown 代码块标记
- scene 数量控制在 5-10 个
- 每个 scene 包含：num（分镜号）、description（画面描述，用于图片生成）、dialogue（台词/旁白）、duration_ms（时长毫秒）

输出 JSON 格式：
{
  "script_text": "完整剧本文字",
  "scenes": [
    {"num": 1, "description": "...", "dialogue": "...", "duration_ms": 5000},
    ...
  ]
}
```

**User Message**（从 Webhook 入参构建）:

```
世界观设定：
{{ $json.world_setting }}

故事内容：
{{ $json.story_text }}

角色列表：
{{ JSON.stringify($json.characters) }}
```

**Body JSON 模板**:

```json
{
  "model": "deepseek-chat",
  "messages": [
    {"role": "system", "content": "你是专业的漫剧编剧..."},
    {"role": "user", "content": "世界观设定：\n{{ $json.world_setting }}\n\n故事内容：\n{{ $json.story_text }}"}
  ],
  "temperature": 0.8,
  "max_tokens": 4096,
  "response_format": {"type": "json_object"}
}
```

---

### Step 3: Code Node — 解析剧本并回调 script_done

- **节点类型**: Code (JavaScript)
- **Purpose**: 将 DeepSeek 返回的 JSON 解析，提取 script_text 和 scenes，POST 到 Go 后端的 callback_url

```javascript
// 获取 DeepSeek 返回的原始响应
const items = $input.all();
for (const item of items) {
  const body = JSON.parse(item.json.body || '{}');
  const content = body.choices[0].message.content;

  // 解析 DeepSeek 返回的 JSON（可能被包裹在 markdown 代码块中）
  let parsed;
  try {
    parsed = JSON.parse(content);
  } catch (e) {
    // 尝试提取 JSON 块
    const match = content.match(/\{[\s\S]*\}/);
    if (match) {
      parsed = JSON.parse(match[0]);
    } else {
      throw new Error('无法解析 DeepSeek 返回的 JSON');
    }
  }

  // 构建回调 payload
  return {
    event: 'script_done',
    project_id: $json.project_id,
    episode_id: $json.episode_id,
    data: {
      script_text: parsed.script_text,
      scene_count: parsed.scenes.length,
      scenes: parsed.scenes
    }
  };
}
```

> **注意**: 该节点使用 `$json` 引用 Webhook 原始输入（因为是在同一执行链中，数据会被传递）。

---

### Step 4: Loop Scenes — 即梦 API 图片生成

- **节点类型**: Loop Over Items (Split In Batches)
- **输入**: Step 3 输出中的 `data.scenes` 数组

循环内节点：

1. **HTTP Request** → 即梦图片生成 API
   - URL: `https://api.jimeng.ai/v1/image/generate`（示例，以即梦实际 API 为准）
   - 用 `{{ $json.description }}` 作为 prompt
   - 结合 `{{ $json.character_refs }}` 确保角色一致性

2. **Code Node** — 回调 `scene_image_done`
   ```javascript
   return {
     event: 'scene_image_done',
     project_id: $json.project_id,
     episode_id: $json.episode_id,
     scene_id: $json.scene_id,  // 从 Go 后端创建 scene 后返回的 ID
     data: {
       scene_num: $json.num,
       image_url: $json.image_url  // 即梦返回的图片 URL
     }
   };
   ```

3. **HTTP Request** → POST 到 `callback_url`

---

### Step 5: Loop Scenes — 可灵 API 视频生成

- **节点类型**: Loop Over Items (Split In Batches)

循环内节点：

1. **HTTP Request** → 可灵视频生成 API
   - URL: `https://api.kling.ai/v1/video/generate`（示例）
   - 以 Step 4 生成的图片作为输入帧

2. **Code Node** — 回调 `scene_video_done`

3. **HTTP Request** → POST 到 `callback_url`

---

### Step 6: Edge-TTS 配音 + FFmpeg 合成

- 所有 scene 视频生成完毕后执行
- 对每个 scene 的 dialogue 调用 Edge-TTS 生成音频
- 使用 FFmpeg 将视频 + 音频合成最终 MP4
- 回调 `event=episode_complete`，传入 `final_video_url`

```javascript
// 回调 episode_complete
return {
  event: 'episode_complete',
  project_id: $json.project_id,
  episode_id: $json.episode_id,
  data: {
    final_video_url: 'https://storage.example.com/episodes/ep-001.mp4',
    duration_ms: totalDuration
  }
};
```

---

## 4. API Key 配置

n8n 支持通过环境变量注入密钥，在 HTTP Request 节点的 Header 中使用 `{{ $env.VARIABLE_NAME }}` 引用。

### 方式一：Docker 环境变量（推荐）

```bash
docker run -d --name n8n \
  -e DEEPSEEK_API_KEY=sk-xxx \
  -e JIMENG_API_KEY=xxx \
  -e KLING_API_KEY=xxx \
  ... \
  n8nio/n8n
```

### 方式二：n8n Credentials 管理

1. 在 n8n 界面左侧点击 **Credentials**
2. 添加对应服务的 API Key credential
3. 在 HTTP Request 节点中选择该 credential

---

## 5. Callback URL 说明

Go 后端提供的回调端点：

| 事件 | HTTP Method | 说明 |
|---|---|---|
| `POST /api/callback` | POST | 统一回调入口 |

Body 格式：

```json
{
  "event": "script_done",
  "project_id": "...",
  "episode_id": "...",
  "scene_id": "...",
  "data": { ... }
}
```

支持的事件类型（来自 `model.SceneStatus`）：

- `script_done` — 剧本生成完成
- `scene_image_done` — 场景图片生成完成
- `scene_video_done` — 场景视频生成完成
- `episode_complete` — 整集合成完成

---

## 6. 本地开发调试

### 启动完整环境

```bash
# 1. 启动 n8n
docker run -d --name n8n -p 5678:5678 \
  --add-host host.docker.internal:host-gateway \
  -e DEEPSEEK_API_KEY=sk-xxx \
  n8nio/n8n

# 2. 启动 Go 后端
cd /home/kiddyt00/claude-projects/comic-pipeline/.worktrees/phase1-go-n8n
go run . serve --addr :8080 --n8n-url http://localhost:5678

# 3. 打开 Web 面板
open http://localhost:8080
```

### 手动触发测试

```bash
curl -X POST http://localhost:5678/webhook/episode-pipeline \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "test-001",
    "episode_id": "test-ep-001",
    "episode_num": 1,
    "world_setting": "测试世界观",
    "story_text": "测试故事内容",
    "characters": [],
    "callback_url": "http://host.docker.internal:8080/api/callback",
    "image_provider": "jimeng",
    "video_provider": "kling",
    "tts_provider": "edge-tts",
    "output_format": "mp4"
  }'
```

---

## 7. 故障排查

| 问题 | 排查方向 |
|---|---|
| n8n webhook 收不到请求 | 检查 Go 后端 `--n8n-url` 参数是否正确；确认 n8n webhook 节点的 path 是否匹配 `/webhook/episode-pipeline` |
| DeepSeek 返回非 JSON | 检查 `response_format: {type: "json_object"}` 是否配置；在 Code 节点添加 try-catch |
| 回调失败 | 确认 callback_url 可被 n8n 容器访问；Docker 内用 `host.docker.internal` |
| 图片/视频 API 超时 | 即梦/可灵 API 可能是异步模式，需要轮询查询结果后再回调 |
