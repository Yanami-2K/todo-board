---
name: todo-board
description: 在自托管的 todo-board 上登记、推进并完成当前 Agent 自己的待办；需要主人拍板时向主人提问；该交给主人的事走公开登记。在开始干活、汇报进度、需要决定、做完的时候使用。
---

# todo-board

你在帮主人干活。待办记在 todo-board 上。你只能动自己泳道里的 todo。做完由你自己标完成，没有单独的验收。

## 连接

环境变量：

- `TODO_BOARD_URL`：例如 `http://127.0.0.1:8080`
- `TODO_BOARD_TOKEN`：主人发给你的 Agent token，以 `tb_` 开头

优先用 MCP。端点是 `POST $TODO_BOARD_URL/mcp`，请求头 `Authorization: Bearer $TODO_BOARD_TOKEN`。工具有：

- `list_todos`
- `create_todo`
- `update_todo`
- `complete_todo`
- `ask_owner`
- `submit_to_intake`

没有 MCP 时用下面的 REST，同一个 token。

## 规矩

1. 开始前先 `list_todos`。有 `status=todo` 的就挑一条，用 `update_todo` 把 `status` 设为 `in_progress`。这就是认领。没有就 `create_todo` 给自己登记，再开始。
2. 只改 `delegate` 是你自己的 todo。不要去动主人的私人待办，也不要改别的 Agent 的。
3. 做到关键步骤时，用 `update_todo` 的 `note` 写一句进度。步骤清楚就带上 `steps`（字符串数组）和 `step`（从 1 开始的当前步）。
4. 需要主人做决定时调用 `ask_owner`，给出 2 到 4 个短选项。问过之后停下来等。不要自己猜。主人的回答会写进这条 todo 的动态，状态回到 `in_progress`。再 `list_todos` 就能看到。
5. 做完调用 `complete_todo`。不要等别人来点完成。
6. 这件事不该你做、应该记给主人时，用 `submit_to_intake`。它进收件箱，主人接受之后才会出现在看板上。给自己建任务用 `create_todo`，不要用登记接口。

## REST

下面的例子把 `$TODO_BOARD_URL` 和 `$TODO_BOARD_TOKEN` 换成实际值。

列出自己的：

```bash
curl -s -H "Authorization: Bearer $TODO_BOARD_TOKEN" \
  "$TODO_BOARD_URL/api/todos?status=todo"
```

新建并开始：

```bash
curl -s -X POST -H "Authorization: Bearer $TODO_BOARD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"跑一遍集成测试","status":"in_progress"}' \
  "$TODO_BOARD_URL/api/todos"
```

认领或写进度（`note` 在 REST 里是短备注；进度正文走评论接口）：

```bash
curl -s -X PATCH -H "Authorization: Bearer $TODO_BOARD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress","steps":["读需求","改代码","自测"],"step":2}' \
  "$TODO_BOARD_URL/api/todos/T-1"

curl -s -X POST -H "Authorization: Bearer $TODO_BOARD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"body":"自测过了 12 个用例"}' \
  "$TODO_BOARD_URL/api/todos/T-1/comments"
```

MCP 的 `update_todo.note` 会直接追加一条进度，不用再调评论接口。

向主人提问：

```bash
curl -s -X POST -H "Authorization: Bearer $TODO_BOARD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"todo_id":"T-1","question":"失败的 2 个要现在修吗？","options":["现在修","先记下来"]}' \
  "$TODO_BOARD_URL/api/questions"
```

标完成：

```bash
curl -s -X POST -H "Authorization: Bearer $TODO_BOARD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"note":"用例全绿"}' \
  "$TODO_BOARD_URL/api/todos/T-1/complete"
```

登记给主人（进收件箱，不上看板）：

```bash
curl -s -X POST -H "Authorization: Bearer $TODO_BOARD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"请主人看一下配额"}' \
  "$TODO_BOARD_URL/api/intake"
```

字段和错误说明见仓库里的 `web/openapi.yaml`，服务跑起来后也可以打开 `/openapi.yaml`。
