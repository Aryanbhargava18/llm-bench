#!/bin/bash
set -e

# Store the old branch
git branch old_main

# Get all commits in chronological order
git log --reverse --format="%H" old_main > commits.txt

# Array of dates to use
declare -a DATES=(
  "2026-08-12T10:15:00+05:30"
  "2026-08-12T11:42:00+05:30"
  "2026-08-12T13:20:00+05:30"
  "2026-08-12T14:45:00+05:30"
  "2026-08-12T15:30:00+05:30"
  "2026-08-12T16:50:00+05:30"
  "2026-08-12T18:10:00+05:30"
  
  "2026-08-13T10:05:00+05:30"
  "2026-08-13T11:30:00+05:30"
  "2026-08-13T12:45:00+05:30"
  "2026-08-13T14:10:00+05:30"
  "2026-08-13T15:20:00+05:30"
  "2026-08-13T16:40:00+05:30"
  "2026-08-13T17:55:00+05:30"
  "2026-08-13T19:15:00+05:30"
  
  "2026-08-14T00:10:00+05:30"
  "2026-08-14T00:50:00+05:30"
  "2026-08-14T01:30:00+05:30"
  "2026-08-14T02:10:00+05:30"
  "2026-08-14T02:45:00+05:30"
  "2026-08-14T03:05:00+05:30"
  "2026-08-14T03:15:00+05:30"
)

# Start detached at the first commit
FIRST_COMMIT=$(head -n 1 commits.txt)
git checkout $FIRST_COMMIT

# Amend the first commit's date
GIT_COMMITTER_DATE="${DATES[0]}" GIT_AUTHOR_DATE="${DATES[0]}" git commit --amend --no-edit

i=1
# Read from the second commit onwards
tail -n +2 commits.txt | while read commit; do
  git cherry-pick $commit
  GIT_COMMITTER_DATE="${DATES[$i]}" GIT_AUTHOR_DATE="${DATES[$i]}" git commit --amend --no-edit
  i=$((i+1))
done

# Force move main to current HEAD
git branch -f main HEAD
git checkout main
git branch -D old_main
rm commits.txt

# Force push to GitHub
git push -f origin main
