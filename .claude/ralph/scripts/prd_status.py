#!/usr/bin/env python3
"""
PRD Status - Quick status update tool for prd.json
Usage:
    ./prd_status.py pass <id>      # Mark task as passed
    ./prd_status.py fail <id>      # Mark task as failed
    ./prd_status.py pending <id>   # Mark task as pending
"""

import json
import sys
from pathlib import Path
from typing import List, Dict, Any, Optional

# Path to prd.json
PRD_PATH = Path(__file__).parent.parent / "plans" / "prd.json"


class PRDStatus:
    def __init__(self, prd_path: Path = PRD_PATH):
        self.prd_path = prd_path
        self.tasks: List[Dict[str, Any]] = []
        self.load()

    def load(self):
        """Load PRD data from JSON file"""
        try:
            with open(self.prd_path, 'r', encoding='utf-8') as f:
                self.tasks = json.load(f)
        except FileNotFoundError:
            print(f"✗ File not found: {self.prd_path}")
            sys.exit(1)
        except json.JSONDecodeError as e:
            print(f"✗ Invalid JSON: {e}")
            sys.exit(1)

    def save(self):
        """Save PRD data to JSON file"""
        try:
            with open(self.prd_path, 'w', encoding='utf-8') as f:
                json.dump(self.tasks, f, ensure_ascii=False, indent=2)
        except Exception as e:
            print(f"✗ Failed to save: {e}")
            sys.exit(1)

    def find_task(self, task_id: str) -> Optional[Dict[str, Any]]:
        """Find task by ID"""
        for task in self.tasks:
            if task.get('id') == task_id:
                return task
        return None

    def update_status(self, task_id: str, status: str):
        """Update task status (pass/fail/pending)"""
        task = self.find_task(task_id)
        if not task:
            print(f"✗ Task not found: {task_id}")
            sys.exit(1)

        # Show current status
        current_passes = task.get('passes', 'N/A')
        print(f"\nTask: {task_id}")
        print(f"Story: {task.get('story', 'N/A')}")
        print(f"Current status: passes={current_passes}")

        # Update status
        if status == 'pass':
            task['passes'] = True
            new_status = "PASSED ✓"
        elif status == 'fail':
            task['passes'] = False
            new_status = "FAILED ✗"
        elif status == 'pending':
            # Remove passes field to indicate pending
            if 'passes' in task:
                del task['passes']
            new_status = "PENDING ⏳"
        else:
            print(f"✗ Invalid status: {status}")
            sys.exit(1)

        # Save changes
        self.save()
        print(f"New status: {new_status}")
        print(f"\n✓ Updated task {task_id} in {self.prd_path}")


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)

    command = sys.argv[1]
    task_id = sys.argv[2]

    if command not in ['pass', 'fail', 'pending']:
        print(f"✗ Invalid command: {command}")
        print(__doc__)
        sys.exit(1)

    status_manager = PRDStatus()
    status_manager.update_status(task_id, command)


if __name__ == "__main__":
    main()
