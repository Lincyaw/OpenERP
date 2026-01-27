#!/bin/bash
set -e

# Ralph Multi-Role Workflow - Enhanced with role routing

if [ -z "$1" ]; then
  echo "Usage: $0 <max iterations>"
  echo "Example: $0 5"
  exit 1
fi

MAX_ITER=$1

LOG_DIR=".claude/ralph/logs"
DOING_DIR=".claude/ralph/doing"
mkdir -p "$LOG_DIR"
mkdir -p "$DOING_DIR"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
MAIN_LOG="$LOG_DIR/ralph_${TIMESTAMP}.log"
CURRENT_TASK_LOCK=""
SCRIPT_PID=$$

log() {
  echo "$1" | tee -a "$MAIN_LOG"
}

# Log to stderr (for use inside functions that return values)
log_stderr() {
  echo "$1" | tee -a "$MAIN_LOG" >&2
}

# Check if a task is locked by another process
is_task_locked() {
  local task_id=$1
  local lock_file="$DOING_DIR/${task_id}.lock"

  if [ ! -f "$lock_file" ]; then
    return 1  # Not locked
  fi

  # Check if the process that created the lock is still running
  local lock_pid=$(cat "$lock_file" 2>/dev/null | head -1)
  if [ -n "$lock_pid" ] && kill -0 "$lock_pid" 2>/dev/null; then
    return 0  # Locked by running process
  else
    # Stale lock, remove it
    log_stderr "[Lock] 🧹 Removing stale lock for task: $task_id (PID: $lock_pid)"
    rm -f "$lock_file"
    return 1  # Not locked anymore
  fi
}

# Lock a task
lock_task() {
  local task_id=$1
  local lock_file="$DOING_DIR/${task_id}.lock"

  # Double-check if already locked
  if is_task_locked "$task_id"; then
    log_stderr "[Lock] ⚠️  Task $task_id is already locked by another process"
    return 1
  fi

  # Create lock file with PID and timestamp
  {
    echo "$SCRIPT_PID"
    echo "$TIMESTAMP"
    date "+%Y-%m-%d %H:%M:%S"
  } > "$lock_file"

  CURRENT_TASK_LOCK="$lock_file"
  log_stderr "[Lock] 🔒 Locked task: $task_id (PID: $SCRIPT_PID)"
  return 0
}

# Unlock a task
unlock_task() {
  local task_id=$1
  local lock_file="$DOING_DIR/${task_id}.lock"

  if [ -f "$lock_file" ]; then
    rm -f "$lock_file"
    log_stderr "[Lock] 🔓 Unlocked task: $task_id"
  fi

  CURRENT_TASK_LOCK=""
}

# Cleanup function (called on exit)
cleanup() {
  if [ -n "$CURRENT_TASK_LOCK" ] && [ -f "$CURRENT_TASK_LOCK" ]; then
    local task_id=$(basename "$CURRENT_TASK_LOCK" .lock)
    log "[Cleanup] 🧹 Releasing lock on exit: $task_id"
    rm -f "$CURRENT_TASK_LOCK"
  fi
}

# Trap signals to ensure cleanup
trap cleanup EXIT INT TERM

# Detect role using Claude API for intelligent routing
detect_role() {
  local task_id=$1

  log_stderr "[Router] Analyzing task: $task_id"

  # Get task details from prd.json
  local task_json=$(jq -r --arg id "$task_id" '.[] | select(.id == $id)' .claude/ralph/plans/prd.json)

  if [ -z "$task_json" ]; then
    log_stderr "[Router] ⚠️  Task not found in prd.json, using default: dev"
    echo "dev"
    return
  fi

  # Extract task story and requirements
  local story=$(echo "$task_json" | jq -r '.story // ""')
  local requirements=$(echo "$task_json" | jq -r '.requirements // [] | join("; ")')
  local priority=$(echo "$task_json" | jq -r '.priority // "unknown"')

  log_stderr "[Router] 📋 Story: $story"
  log_stderr "[Router] 📌 Priority: $priority"
  log_stderr "[Router] 🔍 Requirements: ${requirements:0:100}..."

  # Use Claude API to intelligently determine the role
  local prompt="You are a task router for an ERP system development team.

Available agents:
- **ralph-pm** (Product Manager): Product design validation, DDD consistency check, requirement review, acceptance testing
- **ralph-qa** (QA Engineer): E2E test implementation with Playwright, integration test verification, test coverage analysis
- **ralph-prd-developer** (Developer): Backend/Frontend implementation, API development, bug fixes, refactoring

Task ID: $task_id
Task Story: $story
Requirements: $requirements

Based on the task ID, story, and requirements, determine which agent is MOST suitable for this task.

Response format (output ONLY the agent role, nothing else):
- If Product Manager tasks: output 'pm'
- If QA Engineer tasks: output 'qa'
- If Developer tasks: output 'dev'

Your answer (just the role, no explanation):"

  log_stderr "[Router] 🤖 Calling Claude API for intelligent routing..."
  local role=$(echo "$prompt" | claude -p 2>/dev/null | tail -1 | tr -d '[:space:]' | grep -Eo '(pm|qa|dev)')

  # Default to dev if API call fails
  if [ -z "$role" ]; then
    log_stderr "[Router] ⚠️  Claude API routing failed, using default: dev"
    role="dev"
  else
    log_stderr "[Router] ✅ Claude API routing result: $role (AI-based)"
  fi

  echo "$role"
}

# Get next highest priority incomplete task from prd.json (excluding locked tasks)
get_next_task_id() {
  # Get all incomplete tasks sorted by priority
  local tasks=$(jq -r '
    .[]
    | select(.passes == false)
    | {priority: .priority, id: .id, sort_key: (if .priority == "high" then 1 elif .priority == "medium" then 2 else 3 end)}
    | [.sort_key, .id]
    | @tsv
  ' .claude/ralph/plans/prd.json | sort -n)

  # Find first unlocked task
  while IFS=$'\t' read -r sort_key task_id; do
    if [ -n "$task_id" ] && ! is_task_locked "$task_id"; then
      echo "$task_id"
      return 0
    else
      log_stderr "[Scheduler] ⏭️  Skipping locked task: $task_id"
    fi
  done <<< "$tasks"

  # No unlocked tasks found
  echo ""
}

log "========================================"
log "🚀 Starting Ralph Multi-Role Workflow"
log "========================================"
log "Max iterations: $MAX_ITER"
log "Start time: $(date)"
log "Main log: $MAIN_LOG"
log "========================================"

for ((i=1; i<=MAX_ITER; i++)); do
  log ""
  log "========================================"
  log "🔄 Iteration $i / $MAX_ITER - $(date)"
  log "========================================"

  ITER_LOG="$LOG_DIR/iter_${TIMESTAMP}_${i}.log"
  ITER_JSON_LOG="$LOG_DIR/iter_${TIMESTAMP}_${i}.jsonl"

  # Get next task
  log "[Scheduler] 🔍 Fetching next highest priority incomplete task..."
  TASK_ID=$(get_next_task_id)

  if [ -z "$TASK_ID" ]; then
    log "[Scheduler] ✅ No incomplete tasks found in prd.json"
    log "[Scheduler] 🎉 All tasks complete! Workflow finished successfully."
    break
  fi

  log "[Scheduler] 🎯 Selected task: $TASK_ID"
  log ""

  # Try to lock the task
  if ! lock_task "$TASK_ID"; then
    log "[Scheduler] ⚠️  Failed to lock task $TASK_ID, skipping to next iteration"
    continue
  fi

  # Detect role
  ROLE=$(detect_role "$TASK_ID")
  AGENT_NAME="ralph-$ROLE"
  [ "$ROLE" = "dev" ] && AGENT_NAME="ralph-prd-developer"

  log ""
  log "[Dispatcher] 👤 Assigned role: $ROLE"
  log "[Dispatcher] 🤖 Agent: $AGENT_NAME"
  log "[Dispatcher] 📝 Logs: $ITER_LOG"
  log ""

  # Build prompt
  PROMPT="Work on task: $TASK_ID

Context:
- Read task details from .claude/ralph/plans/prd.json using jq
- Read spec.md (.claude/ralph/docs/spec.md) for design requirements (use grep for specific sections)
- Read progress.txt (tail -100 .claude/ralph/progress.txt) for recent work context
- Follow CLAUDE.md project rules for commits, linting, testing

Your mission:
1. Complete the assigned task according to your role
2. Update prd.json: Set passes: true when task is complete
3. Append detailed progress entry to progress.txt
4. Create git commit (if code changes were made)

Quality requirements:
- Follow self-verification checklist in your agent instructions
- Delegate to specialized agents when appropriate (tdd-guide, code-reviewer, security-reviewer)
- Document all decisions and changes

If ALL tasks in prd.json are complete (all passes: true), output:
<promise>COMPLETE</promise>"

  TEMP_RESULT=$(mktemp)

  log "[Executor] 🚀 Starting agent execution..."
  log "[Executor] ⚙️  Agent: $AGENT_NAME"
  log "[Executor] 📋 Task: $TASK_ID"

  claude --agent="$AGENT_NAME" \
         --dangerously-skip-permissions \
         -p \
         --verbose \
         --output-format stream-json \
         "$PROMPT" 2>&1 | \
    tee "$ITER_JSON_LOG" | \
    while IFS= read -r line; do
      echo "$line" >> "$ITER_LOG"
      if echo "$line" | jq -e 'select(.type == "assistant")' >/dev/null 2>&1; then
        echo "$line" | jq -r '.message.content[]? | select(.type == "text") | .text // empty' 2>/dev/null >> "$TEMP_RESULT"
      fi
    done

  result=$(cat "$TEMP_RESULT")
  rm -f "$TEMP_RESULT"

  echo "$result" >> "$MAIN_LOG"
  echo "$result"

  log ""
  log "[Executor] ✅ Agent execution completed"
  log "[Executor] 📊 Full stream log: $ITER_JSON_LOG"

  # Unlock the task
  unlock_task "$TASK_ID"

  log ""
  log "[Scheduler] ✅ Iteration $i completed."
  log ""
done

log "========================================"
log "🏁 Ralph Workflow Summary"
log "========================================"
log "Finished at: $(date)"
log "Total iterations: $i / $MAX_ITER"
log "Full log: $MAIN_LOG"
log "Iteration logs: $LOG_DIR/"
log "JSON stream logs: $LOG_DIR/iter_*.jsonl"
log "========================================"
