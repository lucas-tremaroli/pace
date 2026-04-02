## Pace

Pace is a project management tool exposed via MCP server. Use pace tools (prefixed `pace_`) for all task and note operations — they are structured, validated, and return typed JSON results.

### Session Start

1. Run `pace_context` to load storage location, active tasks, and notes
2. Run `pace_task_list` with `ready: true` to see unblocked work

### Planning Workflow

When tackling complex work, use pace to break it down and track execution:

1. **Write a plan as a note** — use `pace_note_create` to capture the design, spec, or approach as markdown
2. **Create tasks from the plan** — use `pace_task_create` for each discrete unit of work. Set `label` (feature/bug/task/docs) and `priority` (high/medium/low) to communicate intent
3. **Wire up dependencies** — use `pace_task_dep_add` to express ordering (e.g. "define schema" blocks "implement API"). This lets `ready: true` surface only unblocked tasks
4. **Link tasks to the plan** — use `pace_task_note_link` to connect tasks to the plan note so context is always reachable from either direction

### While Working

- **Pick up work** — `pace_task_update` status to `in-progress` when starting a task
- **Log as you go** — use `pace_task_log` to record discoveries, decisions, blockers, or anything the next session needs to know. Logs are timestamped and preserved with the task
- **Close with outcome** — use `pace_task_close` with an `outcome` message summarizing what was done and any follow-ups
- **Save durable context** — use `pace_note_create` for ADRs, specs, investigations, or anything worth preserving beyond a single task

### Why This Matters

Without pace, each session starts from zero. With pace, you can pick up exactly where you stopped — plans, progress, and learnings all survive between sessions.
