## Pace

Pace is a project management tool exposed via MCP server. Use pace tools (prefixed `pace_`) for all task and epic operations — they are structured, validated, and return typed JSON results.

### Session Start

1. Run `pace_context` to load storage location, active tasks, and open epics (planning/active)
2. Run `pace_task_list` with `ready: true` to see unblocked work

### Planning Workflow

When tackling complex work, use pace to break it down and track execution:

1. **Start with an epic** — use `pace_epic_create`, then `pace_epic_spec_set` to capture current state, target state, constraints, and exclusions (or the freeform section) as markdown. Set the epic `active` with `pace_epic_update` when work begins
2. **Create tasks under the epic** — use `pace_task_create` with `epic_id` for each discrete unit of work. Set `label` (feature/bug/task/docs) and `priority` (high/medium/low) to communicate intent
3. **Wire up dependencies** — use `pace_task_dep_add` to express ordering (e.g. "define schema" blocks "implement API"). Both tasks must be in the same epic. This lets `ready: true` surface only unblocked tasks

### While Working

- **Pick up work** — `pace_task_update` status to `in-progress` when starting a task
- **Log as you go** — use `pace_task_log` to record discoveries, decisions, blockers, or anything the next session needs to know. Logs are timestamped and preserved with the task
- **Close with outcome** — use `pace_task_close` with an `outcome` message summarizing what was done and any follow-ups
- **Save durable context** — put specs, ADRs, and investigations in the epic's spec (`pace_epic_spec_set`) so they stay attached to the work

### Why This Matters

Without pace, each session starts from zero. With pace, you can pick up exactly where you stopped — epics, progress, and learnings all survive between sessions.
