# Local Agent

Go Local Agent for onCode (ADR-002). Phase 1 core: tool dispatcher, workspace boundary, system tools, `workspace.read_file`.

```text
local-agent/
├─ cmd/oncode-agent/
└─ internal/
   ├─ agent/
   ├─ audit/
   ├─ config/
   ├─ tools/
   │  ├─ system/
   │  └─ workspace/
   ├─ transport/
   └─ workspace/
```

## Build / test

From repository root:

```text
go test ./local-agent/...
go build -o oncode-agent.exe ./local-agent/cmd/oncode-agent
```

Optional startup registration:

```text
oncode-agent -workspace C:\path\to\project
```

## Tools (today)

| Tool | Status |
|------|--------|
| `system.ping` | implemented |
| `system.get_capabilities` | implemented |
| `workspace.read_file` | implemented (UTF-8, SHA-256) |
| `workspace.read_files` | implemented (per-path SUCCESS/FAILED, PARTIAL) |
| `workspace.search` | implemented (name / path / content literal) |
| `workspace.propose_changes` | implemented (no mutate; actual unified diff + hash) |
| `workspace.apply_changes` | implemented (approval boundary, atomic apply/rollback) |
| `build.run` | Maven (`mvnw` preferred, else `mvn -B -DskipTests compile`) |
| `test.run` | Maven (`mvn -B test`, Surefire summary parse) |
| `git.status` | local branch/head/changed/staged/untracked |
| `git.diff` | local unstaged or staged diff (optional path) |

Sample project: `local-agent/testdata/sample-java-maven`.

Not yet: git commit/push, live gRPC transport.
