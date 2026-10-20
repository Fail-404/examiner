# Agent Instructions

See `CLAUDE.md` for build, deployment, and project conventions.

## Owner workflow for Fail-404/examiner

- Read `docs/development/user-guide-ru.md` and `docs/development/work-log-ru.md`
  before continuing the owner's work. Update the relevant user instructions and
  the factual work log with each change; keep work items in GitHub Issues.
- Continue the U (user request), A (assistant message), and D (change record)
  identifiers from the current conversation and work log. Include the related
  U identifier in assistant progress updates and final reports.
- Before publishing, verify that GitHub API authentication is `Fail-404` and
  that Git author and committer match the owner's identity documented in the guide.
  Git metadata and GitHub authentication are separate checks.
- For first-run Windows instructions, link `docs/development/run-windows-ru.md`.
  Distinguish documented steps from operations actually verified on the owner's PC.

## Issue tracking

- **Work items:** GitHub Issues (`gh issue list`, `gh issue view <n>`).
  Do not create markdown TODO lists.
- **Decisions and rationale:** Hindsight memory.

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

Shell commands like `cp`, `mv`, and `rm` may be aliased to include `-i`
(interactive) mode on some systems, causing the agent to hang
indefinitely waiting for y/n input.

**Use these forms instead:**

```bash
# Force overwrite without prompting
cp -f source dest           # NOT: cp source dest
mv -f source dest           # NOT: mv source dest
rm -f file                  # NOT: rm file

# For recursive operations
rm -rf directory            # NOT: rm -r directory
cp -rf source dest          # NOT: cp -r source dest
```

**Other commands that may prompt:**

- `scp` - use `-o BatchMode=yes` for non-interactive
- `ssh` - use `-o BatchMode=yes` to fail instead of prompting
- `apt-get` - use `-y` flag
- `brew` - use `HOMEBREW_NO_AUTO_UPDATE=1` env var
