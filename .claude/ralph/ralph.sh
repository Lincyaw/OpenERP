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
mkdir -p "$LOG_DIR"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
MAIN_LOG="$LOG_DIR/ralph_${TIMESTAMP}.log"

log() {
  echo "$1" | tee -a "$MAIN_LOG"
}

# Detect role from task ID pattern
detect_role() {
  local task_id=$1

  case "$task_id" in
    P*-PD-*)      echo "pm" ;;     # Product Design tasks
    P*-INT-*)     echo "qa" ;;     # Integration test tasks
    DDD-*)        echo "pm" ;;     # DDD validation tasks
    review-*)     echo "pm" ;;     # Review tasks
    acceptance-*) echo "pm" ;;     # Acceptance tasks
    e2e-test-*)   echo "qa" ;;     # E2E test implementation tasks
    *)            echo "dev" ;;    # Default: development tasks (BE, FE, API, bug-fix, etc.)
  esac
}

# Get next highest priority incomplete task from prd.json
get_next_task_id() {
  jq -r '
    .[]
    | select(.passes == false)
    | {priority: .priority, id: .id, sort_key: (if .priority == "high" then 1 elif .priority == "medium" then 2 else 3 end)}
    | [.sort_key, .id]
    | @tsv
  ' .claude/ralph/plans/prd.json \
  | sort -n \
  | head -1 \
  | awk '{print $2}'
}

log "Starting Ralph Multi-Role Workflow, max iterations: $MAX_ITER"
log "Logs will be saved to: $MAIN_LOG"

for ((i=1; i<=MAX_ITER; i++)); do
  log "========================================"
  log "Iteration $i / $MAX_ITER - $(date)"
  log "========================================"

  ITER_LOG="$LOG_DIR/iter_${TIMESTAMP}_${i}.log"
  ITER_JSON_LOG="$LOG_DIR/iter_${TIMESTAMP}_${i}.jsonl"

  # Get next task
  TASK_ID=$(get_next_task_id)

  if [ -z "$TASK_ID" ]; then
    log "No incomplete tasks found in prd.json"
    log "All tasks complete! 🎉"
    break
  fi

  log "Selected task: $TASK_ID"

  # Detect role
  ROLE=$(detect_role "$TASK_ID")
  AGENT_NAME="ralph-$ROLE"
  [ "$ROLE" = "dev" ] && AGENT_NAME="ralph-prd-developer"

  log "Assigned role: $ROLE (agent: $AGENT_NAME)"

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

  log "Executing agent: $AGENT_NAME"

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

  if [[ "$result" == *"<promise>COMPLETE</promise>"* ]]; then
    log "🎉 All tasks in the PRD are complete! Exiting loop."
    break
  fi

  log "Iteration $i completed."
  log ""
done

log "========================================"
log "Ralph loop finished at $(date)"
log "Full log saved to: $MAIN_LOG"
log "Individual iteration logs in: $LOG_DIR/"
log "JSON stream logs: $LOG_DIR/iter_*.jsonl"
log "========================================"
