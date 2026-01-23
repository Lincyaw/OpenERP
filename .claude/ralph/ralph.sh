#!/bin/bash
set -e


if [ -z "$1" ]; then
  echo "Usage: $0 <max iterations>"
  echo "Example: $0 5"
  exit 1
fi

MAX_ITER=$1


echo "Starting Ralph loop, max iterations: $MAX_ITER"

for ((i=1; i<=MAX_ITER; i++)); do
  echo "========================================"
  echo "Iteration $i / $MAX_ITER"
  echo "========================================"

  result=$(claude --dangerously-skip-permissions -p "@.claude/ralph/plans/prd.json @.claude/ralph/progress.txt \
  1. Find the highest-priority feature to work on and work only on that feature. \
  This should be the one YOU decide has the highest priority - not necessarily the first item. \
  2. If the prd.json is not clear, you can reference .claude/ralph/docs/spec.md and .claude/ralph/docs/task.md, but be careful the files is long \
  3. Update the PRD with the work that was done. \
  4. Append your progress to the progress.txt file. \
  Use this to leave a note for the next person working in the codebase. \
  5. If there are some questions you have about the design, implementation, etc., stop your work and ask me. \
  6. Call proper subagent for specfic tasks. \
  7. Make a git commit of that feature. \
  ONLY WORK ON A SINGLE FEATURE. \
  8. Check CLAUDE.md to see if there are any special instructions for committing code, e.g., format code, lint, etc. \
  If, while implementing the feature, you notice the PRD is complete, output <promise>COMPLETE</promise> here.")

  echo "$result"

  if [[ "$result" == *"<promise>COMPLETE</promise>"* ]]; then
    echo "🎉 All tasks in the PRD are complete! Exiting loop."
    break
  fi

  echo "Iteration $i completed."
done
