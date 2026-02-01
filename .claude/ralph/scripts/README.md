# Ralph Scripts

Utility scripts for managing the Ralph project.

## PRD Manager

Tool for managing tasks in `prd.json`.

## Quick Status Tool

Fast status updates for tasks without editing JSON manually.

### Installation

```bash
cd /home/nn/workspace/erp/.claude/ralph/scripts
chmod +x prd_manager.py prd_status.py
```

---

## Quick Status Tool

### Usage

#### Mark Task as Passed

```bash
./prd_status.py pass P0-SEED-001
```

Sets `"passes": true` in prd.json.

#### Mark Task as Failed

```bash
./prd_status.py fail P0-SEED-001
```

Sets `"passes": false` in prd.json.

#### Mark Task as Pending

```bash
./prd_status.py pending P0-SEED-001
```

Removes the `passes` field (status becomes N/A).

### Examples

```bash
# Complete a task
./prd_status.py pass P1-BE-005

# Mark test failure
./prd_status.py fail P3-INT-001

# Reset to pending
./prd_status.py pending P2-FE-003
```

---

## PRD Manager (Full Tool)

### Usage

#### Show Statistics

```bash
./prd_manager.py stats
```

Shows task counts by priority and status.

#### List Tasks

```bash
# List all tasks
./prd_manager.py list

# List first 10 tasks
./prd_manager.py list --limit 10
```

#### Search Task by ID

```bash
./prd_manager.py search P0-SEED-001
```

Shows full task details including requirements and acceptance criteria.

#### Add New Task

```bash
./prd_manager.py add
```

Interactive prompt to create a new task:
- Task ID (e.g., P1-FEATURE-001)
- Story/Description
- Priority (critical/high/medium/low)
- Requirements (multi-line)
- Acceptance Criteria (multi-line)

#### Batch Add Tasks

Add multiple tasks at once from JSON file or stdin:

```bash
# From file
./prd_manager.py add-batch --file tasks.json

# From stdin (pipe)
echo '[{"id":"bug-001","story":"Fix login issue","priority":"high"}]' | ./prd_manager.py add-batch

# From heredoc
./prd_manager.py add-batch <<EOF
[
  {
    "id": "bug-001",
    "story": "Fix login timeout issue",
    "priority": "high",
    "requirements": ["Reproduce issue", "Fix timeout logic"]
  },
  {
    "id": "bug-002",
    "story": "Fix inventory calculation",
    "priority": "critical"
  }
]
EOF
```

**JSON Format:**
```json
[
  {
    "id": "task-id",
    "story": "Task description",
    "priority": "critical|high|medium|low",
    "requirements": ["req1", "req2"],
    "acceptance_criteria": ["criterion1"],
    "status": "pending"
  }
]
```

Required fields: `id`, `story`
Optional fields: `priority` (default: medium), `requirements`, `acceptance_criteria`, `status` (default: pending), `passes`

#### Update Task

```bash
./prd_manager.py update P0-SEED-001
```

Interactive prompt to update task fields (story, priority, status).

#### Delete Task

```bash
./prd_manager.py delete P0-SEED-001
```

Deletes a task after confirmation.

### Examples

```bash
# Check project statistics
./prd_manager.py stats

# Find a specific task
./prd_manager.py search P0-BE-001

# List first 5 tasks
./prd_manager.py list --limit 5

# Add a new feature task
./prd_manager.py add
# Then follow the prompts

# Update task status
./prd_manager.py update P1-FEATURE-001

# Remove completed task
./prd_manager.py delete P0-SEED-001
```

### Task Structure

Each task in `prd.json` has:

```json
{
  "id": "P0-SEED-001",
  "story": "Task description",
  "priority": "critical|high|medium|low",
  "status": "pending|in_progress|completed",
  "requirements": ["req1", "req2"],
  "acceptance_criteria": ["criterion1", "criterion2"]
}
```

### Current Statistics

- **Total Tasks**: 542
- **By Priority**:
  - Critical: 49 (9.0%)
  - High: 262 (48.3%)
  - Medium: 186 (34.3%)
  - Low: 45 (8.3%)

## Workflow Recommendations

### For Quick Status Updates
Use `prd_status.py` when you just need to mark pass/fail:
```bash
./prd_status.py pass P1-BE-005
```

### For Full Task Management
Use `prd_manager.py` when you need to:
- Add new tasks/bugs
- Search and view details
- Update task fields
- Delete tasks
- View statistics

### Example Workflow

```bash
# 1. Check what needs to be done
./prd_manager.py stats
./prd_manager.py list --limit 10

# 2. View task details
./prd_manager.py search P1-BE-005

# 3. Work on the task...

# 4. Mark as complete
./prd_status.py pass P1-BE-005

# 5. Add bugs found during testing (single)
./prd_manager.py add

# 6. Or batch add multiple bugs
./prd_manager.py add-batch --file bugs.json
```

### Batch Operations Example

```bash
# Generate bug list from test results and add to PRD
cat > bugs.json <<EOF
[
  {"id":"bug-fix-101","story":"Fix inventory lock race condition","priority":"critical"},
  {"id":"bug-fix-102","story":"Fix order amount calculation","priority":"high"},
  {"id":"bug-fix-103","story":"Fix customer balance display","priority":"medium"}
]
EOF

./prd_manager.py add-batch --file bugs.json

# Or use pipe for dynamic generation
jq -r '.failures[] | {id: .bug_id, story: .description, priority: "high"}' test-results.json | \
  jq -s '.' | \
  ./prd_manager.py add-batch
```

## Log Viewer

See `log_viewer.py` for viewing application logs.
