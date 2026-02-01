#!/usr/bin/env python3
"""
PRD Manager - Utility tool for managing prd.json
Usage:
    ./prd_manager.py list [--limit N]
    ./prd_manager.py search <id>
    echo '[
  {"id":"bug-fix-001","story":"Fix login issue","priority":"high"},
  {"id":"improvement-001","story":"Add search feature","priority":"medium"}
]' | .claude/ralph/scripts/prd_manager.py add
    ./prd_manager.py delete <id>
    ./prd_manager.py update <id>
    ./prd_manager.py stats
"""

import json
import sys
import os
from pathlib import Path
from typing import List, Dict, Any, Optional

# Path to prd.json
PRD_PATH = Path(__file__).parent.parent / "plans" / "prd.json"


class PRDManager:
    def __init__(self, prd_path: Path = PRD_PATH):
        self.prd_path = prd_path
        self.tasks: List[Dict[str, Any]] = []
        self.load()

    def load(self):
        """Load PRD data from JSON file"""
        try:
            with open(self.prd_path, "r", encoding="utf-8") as f:
                self.tasks = json.load(f)
            print(f"✓ Loaded {len(self.tasks)} tasks from {self.prd_path}")
        except FileNotFoundError:
            print(f"✗ File not found: {self.prd_path}")
            sys.exit(1)
        except json.JSONDecodeError as e:
            print(f"✗ Invalid JSON: {e}")
            sys.exit(1)

    def save(self):
        """Save PRD data to JSON file"""
        try:
            with open(self.prd_path, "w", encoding="utf-8") as f:
                json.dump(self.tasks, f, ensure_ascii=False, indent=2)
            print(f"✓ Saved {len(self.tasks)} tasks to {self.prd_path}")
        except Exception as e:
            print(f"✗ Failed to save: {e}")
            sys.exit(1)

    def list_tasks(self, limit: Optional[int] = None):
        """List all tasks with optional limit"""
        tasks_to_show = self.tasks[:limit] if limit else self.tasks

        print(f"\n{'=' * 80}")
        print(f"Total Tasks: {len(self.tasks)}")
        if limit:
            print(f"Showing: {len(tasks_to_show)}")
        print(f"{'=' * 80}\n")

        for task in tasks_to_show:
            self._print_task_summary(task)
            print()

    def search_task(self, task_id: str) -> Optional[Dict[str, Any]]:
        """Search for a task by ID"""
        for task in self.tasks:
            if task.get("id") == task_id:
                return task
        return None

    def display_task(self, task_id: str):
        """Display full task details"""
        task = self.search_task(task_id)
        if not task:
            print(f"✗ Task not found: {task_id}")
            return

        print(f"\n{'=' * 80}")
        self._print_task_full(task)
        print(f"{'=' * 80}\n")

    def add_batch(self, tasks_data: List[Dict[str, Any]]):
        """Batch add tasks from JSON data"""
        if not tasks_data:
            print("✗ No tasks provided")
            return

        added = []
        skipped = []
        errors = []

        for task in tasks_data:
            # Validate required fields
            if "id" not in task:
                errors.append(f"Missing 'id' field in task: {task}")
                continue

            if "story" not in task:
                errors.append(
                    f"Missing 'story' field in task: {task.get('id', 'unknown')}"
                )
                continue

            task_id = task["id"]

            # Check if ID already exists
            if self.search_task(task_id):
                skipped.append(task_id)
                continue

            # Set defaults
            new_task = {
                "id": task_id,
                "story": task["story"],
                "priority": task.get("priority", "medium"),
                "requirements": task.get("requirements", []),
                "acceptance_criteria": task.get("acceptance_criteria", []),
                "status": task.get("status", "pending"),
            }

            # Add optional fields if present
            if "passes" in task:
                new_task["passes"] = task["passes"]

            self.tasks.append(new_task)
            added.append(task_id)

        # Save if any tasks were added
        if added:
            self.save()

        # Print summary
        print(f"\n{'=' * 80}")
        print("Batch Add Summary")
        print(f"{'=' * 80}")
        print(f"✓ Added: {len(added)}")
        if added:
            for task_id in added:
                print(f"  - {task_id}")

        if skipped:
            print(f"\n⊘ Skipped (already exists): {len(skipped)}")
            for task_id in skipped:
                print(f"  - {task_id}")

        if errors:
            print(f"\n✗ Errors: {len(errors)}")
            for error in errors:
                print(f"  - {error}")

        print(f"{'=' * 80}\n")

    def delete_task(self, task_id: str):
        """Delete a task by ID"""
        task = self.search_task(task_id)
        if not task:
            print(f"✗ Task not found: {task_id}")
            return

        print("\n=== Task to Delete ===")
        self._print_task_summary(task)

        confirm = input(f"\nDelete task {task_id}? (y/n): ").strip().lower()
        if confirm == "y":
            self.tasks = [t for t in self.tasks if t.get("id") != task_id]
            self.save()
            print(f"✓ Task deleted: {task_id}")
        else:
            print("✗ Cancelled")

    def update_task(self, task_id: str):
        """Update task fields"""
        task = self.search_task(task_id)
        if not task:
            print(f"✗ Task not found: {task_id}")
            return

        print("\n=== Current Task ===")
        self._print_task_full(task)

        print("\n=== Update Task ===")
        print("Leave blank to keep current value\n")

        # Update story
        new_story = input(f"Story [{task.get('story', '')}]: ").strip()
        if new_story:
            task["story"] = new_story

        # Update priority
        new_priority = input(f"Priority [{task.get('priority', 'medium')}]: ").strip()
        if new_priority:
            task["priority"] = new_priority

        # Update status
        new_status = input(f"Status [{task.get('status', 'pending')}]: ").strip()
        if new_status:
            task["status"] = new_status

        # Save changes
        self.save()
        print(f"✓ Task updated: {task_id}")

    def show_stats(self):
        """Show statistics about tasks"""
        total = len(self.tasks)

        # Count by priority
        priorities = {}
        for task in self.tasks:
            priority = task.get("priority", "unknown")
            priorities[priority] = priorities.get(priority, 0) + 1

        # Count by status
        statuses = {}
        for task in self.tasks:
            status = task.get("status", "unknown")
            statuses[status] = statuses.get(status, 0) + 1

        print(f"\n{'=' * 80}")
        print("PRD Statistics")
        print(f"{'=' * 80}")
        print(f"\nTotal Tasks: {total}")

        print("\nBy Priority:")
        for priority, count in sorted(priorities.items()):
            print(f"  {priority:15} {count:5} ({count / total * 100:.1f}%)")

        print("\nBy Status:")
        for status, count in sorted(statuses.items()):
            print(f"  {status:15} {count:5} ({count / total * 100:.1f}%)")

        print(f"\n{'=' * 80}\n")

    def _print_task_summary(self, task: Dict[str, Any]):
        """Print task summary (one-liner)"""
        task_id = task.get("id", "N/A")
        story = task.get("story", "N/A")
        priority = task.get("priority", "N/A")
        status = task.get("status", "N/A")

        # Truncate story if too long
        if len(story) > 60:
            story = story[:57] + "..."

        print(f"[{task_id}] {story}")
        print(f"  Priority: {priority} | Status: {status}")

    def _print_task_full(self, task: Dict[str, Any]):
        """Print full task details"""
        print(f"ID: {task.get('id', 'N/A')}")
        print(f"Story: {task.get('story', 'N/A')}")
        print(f"Priority: {task.get('priority', 'N/A')}")
        print(f"Status: {task.get('status', 'N/A')}")

        requirements = task.get("requirements", [])
        if requirements:
            print(f"\nRequirements ({len(requirements)}):")
            for req in requirements:
                print(f"  - {req}")

        acceptance_criteria = task.get("acceptance_criteria", [])
        if acceptance_criteria:
            print(f"\nAcceptance Criteria ({len(acceptance_criteria)}):")
            for criterion in acceptance_criteria:
                print(f"  - {criterion}")


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    command = sys.argv[1]
    manager = PRDManager()

    if command == "list":
        limit = None
        if len(sys.argv) > 2 and sys.argv[2] == "--limit":
            limit = int(sys.argv[3]) if len(sys.argv) > 3 else 10
        manager.list_tasks(limit)

    elif command == "search":
        if len(sys.argv) < 3:
            print("Usage: prd_manager.py search <id>")
            sys.exit(1)
        task_id = sys.argv[2]
        manager.display_task(task_id)

    elif command == "add":
        # Check if reading from file or stdin
        if len(sys.argv) > 2 and sys.argv[2] == "--file":
            if len(sys.argv) < 4:
                print("Usage: prd_manager.py add-batch --file <path>")
                sys.exit(1)
            file_path = sys.argv[3]
            try:
                with open(file_path, "r", encoding="utf-8") as f:
                    tasks_data = json.load(f)
            except FileNotFoundError:
                print(f"✗ File not found: {file_path}")
                sys.exit(1)
            except json.JSONDecodeError as e:
                print(f"✗ Invalid JSON in file: {e}")
                sys.exit(1)
        else:
            # Read from stdin
            try:
                tasks_data = json.load(sys.stdin)
            except json.JSONDecodeError as e:
                print(f"✗ Invalid JSON from stdin: {e}")
                sys.exit(1)

        # Ensure it's a list
        if not isinstance(tasks_data, list):
            print("✗ Input must be a JSON array of tasks")
            sys.exit(1)

        manager.add_batch(tasks_data)

    elif command == "delete":
        if len(sys.argv) < 3:
            print("Usage: prd_manager.py delete <id>")
            sys.exit(1)
        task_id = sys.argv[2]
        manager.delete_task(task_id)

    elif command == "update":
        if len(sys.argv) < 3:
            print("Usage: prd_manager.py update <id>")
            sys.exit(1)
        task_id = sys.argv[2]
        manager.update_task(task_id)

    elif command == "stats":
        manager.show_stats()

    else:
        print(f"Unknown command: {command}")
        print(__doc__)
        sys.exit(1)


if __name__ == "__main__":
    main()
