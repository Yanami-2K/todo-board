# todo-board

自己部署的待办看板。管三件事：你自己的 todo、你的 Agent 的 todo、别人登记给你的事。

Agent 自己建任务、写进度、标完成。需要你拍板时，它提一个问题，出现在首页「待确认」。同事和外部 Agent 从公开登记页或 API 提交，先进收件箱，你接受（并指定自己或某个 Agent 来做）或拒绝之后才会上看板。没有单独的验收步骤。

Agent 和项目都在运行时创建，不写死任何具体名字。一条 todo 可以不属于项目，也可以属于一个项目。

## 跑起来

需要 Docker。

```bash
docker compose up --build
```

打开 <http://localhost:8080> ，密码是 `docker-compose.yml` 里的 `OWNER_PASSWORD`（示例是 `changeme`）。数据在卷 `todo-data` 里的 SQLite 文件 `/data/todo.db`。

不用 Docker 时：

```bash
go build -o todo-board ./cmd/todo-board
OWNER_PASSWORD=changeme ./todo-board
```

二进制默认听 `:8080`，数据库在 `./data/todo.db`。

没设 `OWNER_PASSWORD`、库里也还没有密码时，进程会在日志里打印一次随机密码。之后每次带上 `OWNER_PASSWORD` 启动都会把密码改成这个值。

### 环境变量

| 变量 | 默认 | 作用 |
| --- | --- | --- |
| `ADDR` | `:8080` | 监听地址 |
| `DATA_DIR` | `./data`（容器里是 `/data`） | SQLite 所在目录 |
| `OWNER_PASSWORD` | 空 | 主人登录密码。空则首次生成并只打印一次 |
| `OWNER_NAME` | `我` | 看板和登记页上的名字 |
| `TZ` | `Asia/Shanghai` | 「几分钟前」和页眉日期 |
| `INTAKE_MODE` | `open` | `open` 匿名可登记；`token` 必须带登记 token |
| `INTAKE_TOKEN` | 空 | `INTAKE_MODE=token` 时必填 |
| `INTAKE_INTRO` | 按名字生成的一句介绍 | 登记页介绍，`**这样**` 会加粗 |
| `INTAKE_HOMEPAGE` | 空 | 登记页主页链接 |

备份就是复制 `todo.db`（以及旁边的 `-wal` / `-shm`，或先停掉进程再拷）。

### 可选示例数据

```bash
DATA_DIR=./data ./todo-board seed
```

库里已经有 Agent、项目或 todo 时会拒绝。`seed --force` 会清掉这些业务数据（登录会话和密码留着），再写入一套虚构示例，并在日志里打印每个示例 Agent 的 token。产品本身不依赖这些名字。

## 三种身份

- **主人**：浏览器登录。Cookie `tb_session`，14 天。界面上的写操作要求同源。
- **Agent**：每个 Agent 一把 token，以 `tb_` 开头，只在创建或轮换时显示一次。请求头 `Authorization: Bearer tb_…`。只能看和改指派给自己的 todo。吊销后立即失效，可以再发一把。动态里记的是这个 Agent 的名字。
- **登记**：公开页是 `/intake`。默认匿名。改成 `INTAKE_MODE=token` 后，网页要填 token，API 用 `Authorization: Bearer <INTAKE_TOKEN>` 或 JSON 字段 `token`。Agent token 在两种模式下都可以调用登记接口。

登记页上「有结果会发邮件告诉你」只是界面文案，这一版不发邮件。

## REST

规范在服务上的 [`/openapi.yaml`](/openapi.yaml)，仓库里是 [`web/openapi.yaml`](web/openapi.yaml)。

主人登录：

```bash
curl -c cj -X POST localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"password":"changeme"}'
```

建一个 Agent（token 只返回这一次）：

```bash
curl -b cj -X POST localhost:8080/api/agents \
  -H 'Content-Type: application/json' \
  -d '{"name":"builder","role":"写代码"}'
```

用 Agent token 建 todo、提问、自己标完成：

```bash
curl -X POST localhost:8080/api/todos \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"补上集成测试","status":"in_progress"}'

curl -X POST localhost:8080/api/questions \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"todo_id":"T-1","question":"失败的两例现在修吗？","options":["现在修","先记下来"]}'

curl -X POST localhost:8080/api/todos/T-1/complete \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"note":"用例全绿"}'
```

别人登记（默认开放，不需要登录）：

```bash
curl -X POST localhost:8080/api/intake \
  -H 'Content-Type: application/json' \
  -d '{"title":"看一下配额","name":"同事","suggest":"builder"}'
```

`suggest` 是已有 Agent 的名字，对得上就作为接受时的建议人选，对不上就忽略。接受时 `assignee` 填 `"owner"`、Agent id 或 Agent 名字。

状态是 `todo`、`in_progress`、`waiting`、`done`。Agent 不能用 PATCH 直接改成 `done`，要走 `POST /api/todos/{id}/complete`。一条 todo 同时只有一个未回答的问题。

## MCP

同一个进程，`POST /mcp`，JSON-RPC 2.0，协议版本 `2025-03-26`。无状态：会回显或签发 `Mcp-Session-Id`，不提供 SSE（`GET /mcp` 返回 405）。

Agent token 可以使用全部工具：

- `list_todos`
- `create_todo`
- `update_todo`
- `complete_todo`
- `ask_owner`
- `submit_to_intake`

`INTAKE_MODE=open` 时，不带 token 只能调用 `submit_to_intake`。`token` 模式下匿名调用返回 401。

Cursor 里可以这样接（把地址和 token 换成你的）：

```json
{
  "mcpServers": {
    "todo-board": {
      "url": "http://127.0.0.1:8080/mcp",
      "headers": { "Authorization": "Bearer tb_替换成Agent的token" }
    }
  }
}
```

`update_todo` 的 `note` 会追加一条进度。REST 里短备注是 todo 的 `note` 字段，进度正文用 `POST /api/todos/{id}/comments`。

## Skill

仓库根目录的 [`SKILL.md`](SKILL.md) 写给会写代码的 Agent：怎么认领、汇报进度、需要决定时问主人、自己标完成，以及什么时候改走登记。把它交给 Agent，并设置 `TODO_BOARD_URL` 和 `TODO_BOARD_TOKEN`。

## 开发

```bash
go test ./...
```

GitHub Actions 在推送和拉取请求上跑同一组测试并编译二进制。

## English

todo-board is a single-user, self-hosted todo board for three lanes: your own todos, one lane per AI agent you create, and todos other people register for you.

Agents create, update, and complete their own todos. A question that needs you shows up under 待确认 on the home page. Public registrations (`/intake` and `POST /api/intake`) sit in the inbox until you accept them (choosing yourself or an agent) or decline them. There is no separate review step.

```bash
docker compose up --build
```

Sign in at <http://localhost:8080> with `OWNER_PASSWORD`. SQLite lives in the `todo-data` volume. Set `INTAKE_MODE=token` and `INTAKE_TOKEN` if the public form should not be anonymous. Each agent has its own bearer token (`tb_…`), shown once, revocable from the UI.

- REST: `GET /openapi.yaml`
- MCP: `POST /mcp` (streamable HTTP, JSON responses, no SSE)
- Skill: [`SKILL.md`](SKILL.md)

The intake page mentions email; this version does not send any.
