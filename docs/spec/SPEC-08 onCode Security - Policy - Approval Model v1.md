# SPEC-08 onCode Security / Policy / Approval Model v1

## 1. 목적

본 명세는 onCode의 보안, 정책, 권한, Human-in-the-Loop 승인 모델을 정의한다.

onCode는 중앙 서버가 개발 작업을 판단하고 Local Agent가 실제 파일 수정, 빌드, 테스트, Git, Shell 작업을 수행하는 구조이므로, 중앙 서버의 판단과 로컬 실행 사이에 명확한 보안 경계가 필요하다.

핵심 원칙은 다음과 같다.

> 중앙 서버는 작업을 제안할 수 있지만, Local Agent의 보안 정책과 사용자 승인을 우회할 수 없다.

---

# 2. 보안 목표

onCode의 보안 목표는 다음과 같다.

1. 사용자별 프로젝트 접근 격리
2. 프로젝트 외 파일 접근 차단
3. 승인되지 않은 Source 변경 차단
4. 위험한 Git/Shell 작업 통제
5. Secret 및 Credential 노출 최소화
6. 취약 Dependency 사용 차단
7. 개발 가이드 및 조직 정책 강제
8. Agent별 Tool 최소 권한 적용
9. 모든 주요 행위 Audit 가능
10. 중앙 서버 침해 시에도 Local Agent가 최종 방어선 역할 수행

---

# 3. Trust Boundary

주요 Trust Boundary는 다음과 같다.

```text
Developer
   │
   ▼
IDE Extension
   │
   ▼
Local Agent
   │
   │  Trust Boundary
   ▼
onCode Server
   │
   ├─ Agent Runtime
   ├─ Orchestrator
   ├─ Context Storage
   ├─ Guide Repository
   ├─ Nexus
   └─ Security Services
```

Local Workspace는 중앙 서버보다 더 강하게 보호해야 하는 자산으로 본다.

---

# 4. Security Authority

보안 결정의 우선순위는 다음과 같이 정의한다.

```text
Local Security Policy

>

Organization Security Policy

>

Project Policy

>

Workflow Policy

>

User Preference

>

Agent Request
```

즉 Server Agent가 허용한 작업이라도 Local Policy가 금지하면 실행하지 않는다.

---

# 5. 정책 계층

정책은 크게 다음 네 계층으로 구성한다.

```text
1. Organization Policy
2. Project Policy
3. Local Agent Policy
4. Runtime Approval Policy
```

---

# 6. Organization Policy

조직 차원의 강제 정책이다.

예:

```text
외부 인터넷 접근 금지

Nexus 외 Package Registry 사용 금지

Git Push 승인 필수

Destructive Git 명령 금지

Secret 중앙 전송 금지

High/Critical Vulnerability Dependency 사용 금지
```

사용자가 변경할 수 없다.

---

# 7. Project Policy

프로젝트별 정책이다.

예:

```yaml
security:
  dependency:
    block_high: true
    block_critical: true

approval:
  code_change: required
  git_commit: required
  git_push: required

shell:
  mode: restricted
```

프로젝트 책임자 정책에 따라 설정한다.

---

# 8. Local Agent Policy

개발자 PC에서 강제하는 최종 정책이다.

예:

```text
Workspace 밖 파일 접근 금지

로컬 Credential 전송 금지

외부 Network 호출 금지

특정 Executable 실행 금지

Destructive Command 차단
```

Server 정책보다 느슨해질 수 없다.

---

# 9. User Preference

사용자 편의 설정이다.

예:

```text
Read Tool 자동 승인

Build 자동 승인

Test 자동 승인
```

단, 상위 보안 정책이 허용하는 범위 안에서만 적용된다.

---

# 10. Policy Decision

모든 위험 작업은 Policy Engine에서 다음 중 하나로 판단한다.

```text
ALLOW

APPROVAL_REQUIRED

DENY
```

예:

```json
{
  "decision": "APPROVAL_REQUIRED",
  "risk": "HIGH_RISK",
  "reason": "Git commit modifies repository state."
}
```

---

# 11. Tool Risk Level

공통 Risk Level은 다음과 같다.

```text
READ

WRITE

EXECUTE

HIGH_RISK

DESTRUCTIVE
```

---

# 12. READ

상태를 변경하지 않는 작업이다.

예:

```text
workspace.read_file
workspace.read_files
workspace.search

project.search

git.status
git.diff
git.log

guide.search
```

기본적으로 자동 실행 가능하다.

---

# 13. WRITE

Local Workspace를 변경한다.

예:

```text
workspace.apply_changes

workspace.create_file

workspace.rename
```

기본적으로 사용자 코드 승인 필요.

---

# 14. EXECUTE

프로세스를 실행하지만 직접 Source를 수정하는 것이 목적은 아니다.

예:

```text
build.run
test.run
```

프로젝트 정책에 따라 자동 또는 승인 필요.

---

# 15. HIGH_RISK

Repository 상태 또는 외부 시스템 상태를 변경할 수 있다.

예:

```text
git.commit
git.merge
git.rebase
git.push

dependency.install

shell.execute
```

기본적으로 명시적 승인 필요.

---

# 16. DESTRUCTIVE

되돌리기 어렵거나 데이터 손실 위험이 큰 작업이다.

예:

```text
git.reset --hard
git.clean -fd
git.push --force

workspace.delete large scope

rm -rf
del /s
```

기본 정책은 DENY 또는 Strong Approval이다.

---

# 17. Tool별 권장 기본 정책

```text
project.search
→ READ / AUTO

workspace.read_file
→ READ / AUTO

workspace.search
→ READ / AUTO

workspace.propose_changes
→ READ-like / AUTO

workspace.apply_changes
→ WRITE / CODE APPROVAL

build.run
→ EXECUTE / POLICY

test.run
→ EXECUTE / POLICY

git.status
→ READ / AUTO

git.diff
→ READ / AUTO

git.commit
→ HIGH_RISK / APPROVAL

git.push
→ HIGH_RISK / APPROVAL

git.rebase
→ HIGH_RISK / APPROVAL

git.reset --hard
→ DESTRUCTIVE / DENY or STRONG APPROVAL

shell.execute
→ HIGH_RISK / APPROVAL
```

---

# 18. Approval Model

Approval은 onCode Protocol의 1급 객체로 취급한다.

Approval Type:

```text
DESIGN

CODE_CHANGE

COMMAND

DEPENDENCY_INSTALL

GIT_COMMIT

GIT_PUSH

GIT_MERGE

GIT_REBASE

DESTRUCTIVE_OPERATION
```

---

# 19. Design Approval

코딩 전 설계 승인이다.

사용자에게 최소 다음을 보여준다.

```text
목표

변경 범위

대상 파일

주요 로직

Dependency

Security 고려

Test 계획
```

승인 전 Implementation 단계로 진입하지 않는다.

---

# 20. Code Change Approval

실제 Source 반영 전에 Local Agent가 현재 Workspace 기준으로 Diff를 생성한다.

```text
Server Proposed Change
        ↓
Local Diff Generation
        ↓
Developer Review
        ↓
Approval
        ↓
Apply
```

승인된 Change Set만 적용 가능하다.

---

# 21. Approval Resource Binding

Approval은 특정 리소스와 강하게 연결되어야 한다.

예:

```text
approval_id

project_id

workspace_id

work_item_id

change_set_id

resource_hash

action
```

다른 Change Set에 승인 ID를 재사용할 수 없다.

---

# 22. Approval Hash Binding

Code Change 승인 시:

```text
hash(
 change_set_id
 + file paths
 + proposed content hashes
 + base hashes
)
```

를 Approval Resource Hash로 사용할 수 있다.

승인 이후 변경 내용이 달라지면 승인 무효.

---

# 23. Command Approval Binding

Command 승인도 실제 실행 명령에 바인딩한다.

```text
command_hash =
SHA256(
 executable
 + arguments
 + working_directory
)
```

승인한 후 arguments가 바뀌면 실행하지 않는다.

---

# 24. Approval Expiration

Approval에는 만료시간을 둘 수 있다.

예:

```text
CODE_CHANGE
→ Workspace 변경 시 즉시 무효

GIT_PUSH
→ 일정 시간 후 만료

COMMAND
→ One-time approval
```

---

# 25. One-time Approval

위험 작업은 기본적으로 1회성 승인이다.

예:

```text
git push
```

승인 후 한 번 실행하면 해당 Approval은 consumed 상태가 된다.

---

# 26. Approval Status

```text
REQUESTED

APPROVED

REJECTED

CHANGE_REQUESTED

EXPIRED

CONSUMED

CANCELLED
```

---

# 27. Strong Approval

파괴적 작업은 일반 승인보다 강한 방식이 필요할 수 있다.

예:

```text
git reset --hard
```

사용자에게 명령과 영향 범위를 명확하게 표시한다.

필요하면 다음을 요구할 수 있다.

```text
Confirmation phrase
```

예:

```text
RESET 입력
```

MVP에서는 아예 DENY 정책도 가능하다.

---

# 28. Approval Chain

향후 조직 환경에서는 다단계 승인도 지원할 수 있다.

예:

```text
Developer
    ↓
Tech Lead
    ↓
Security Reviewer
```

하지만 일반 IDE coding workflow의 MVP에서는 단일 사용자 승인으로 시작할 수 있다.

---

# 29. Approval과 Workflow 분리

Approval Service가 Workflow 자체를 결정하지 않는다.

```text
Workflow
   ↓
Approval Request
   ↓
Approval Result
   ↓
Workflow Guard
```

예:

```text
WAITING_CODE_APPROVAL
   ↓
approved == true
   ↓
APPLYING_CHANGE
```

---

# 30. Agent Permission Model

Agent별로 Tool 권한을 제한한다.

원칙:

> Agent는 자신의 역할에 필요한 최소한의 Tool만 사용한다.

---

# 31. Requirement Agent 권한

허용:

```text
context.get
context.put

project.get_profile
```

금지:

```text
workspace.apply_changes
git.*
shell.*
```

---

# 32. Guide Agent 권한

허용:

```text
guide.search
guide.read
context.*
```

---

# 33. Code Analysis Agent 권한

허용:

```text
project.*
workspace.read_*
workspace.search
context.*
```

Read-only이다.

---

# 34. Design Agent 권한

허용:

```text
project.*
workspace.read_*
guide.*
context.*
```

금지:

```text
workspace.apply_changes
git.*
shell.*
```

---

# 35. Implementation Agent 권한

허용:

```text
workspace.read_*
workspace.search

project.*

context.*
```

결과는 Change Set만 생성한다.

직접 `workspace.apply_changes` 호출 금지.

---

# 36. Review Agent 권한

Read-only Tool 중심.

```text
workspace.read_*
project.*
guide.*
context.*
```

---

# 37. Security Agent 권한

```text
workspace.read_*
security.*
dependency.*
guide.*
context.*
```

---

# 38. Senior Developer Agent 권한

넓은 Read 권한은 가능하다.

하지만 다음 직접 실행은 금지한다.

```text
workspace.apply_changes

git.commit

git.push

shell.execute
```

Senior Developer도 Local Execution Policy를 우회할 수 없다.

---

# 39. Tool Gateway Authorization

Agent가 Tool을 호출하면 Tool Gateway가 다음을 검증한다.

```text
Agent Permission

User Scope

Project Scope

Workflow State

Tool Risk

Policy

Approval Requirement
```

---

# 40. Workflow State Authorization

Tool이 올바른 Workflow 단계에서만 호출되도록 한다.

예:

```text
workspace.apply_changes
```

허용 조건:

```text
workflow state == APPLYING_CHANGE

AND

code approval == APPROVED
```

---

# 41. Context Access Control

Context도 Agent별 접근 범위를 제한한다.

예:

```text
Implementation Agent
→ 현재 Work Item Requirement/Design만

Review Agent
→ 해당 Change Set 관련 데이터만
```

다른 사용자 또는 다른 Work Item의 Context 조회 차단.

---

# 42. Multi-user Isolation

모든 Tool Request는 최소 다음 Scope를 가진다.

```text
user_id

session_id

project_id

workspace_id

work_item_id
```

Tool Provider는 Scope를 검증한다.

---

# 43. Project Isolation

다른 프로젝트 파일 경로를 요청해도 Local Agent가 차단한다.

```text
requested path
   ↓
canonicalize
   ↓
workspace root 포함 여부
```

false이면 DENY.

---

# 44. Path Traversal 방지

다음 입력을 차단한다.

```text
../
../../
absolute path outside workspace
symlink escape
```

단순 문자열 검사보다 실제 canonical path 검증을 사용한다.

---

# 45. Symbolic Link

Workspace 내부 symlink가 외부 경로를 가리킬 수 있다.

따라서:

```text
resolved canonical target
```

기준으로 Workspace Boundary를 검증한다.

---

# 46. Secret Protection

중앙 서버로 전송될 수 있는 데이터에 대해 Secret Scanner를 적용한다.

대상:

```text
Source

Config

Build Log

Test Log

Diff

Shell Output
```

---

# 47. Secret 유형

예:

```text
Private Key

API Key

Access Token

Password

Connection Credential

Certificate Private Material
```

---

# 48. Secret Policy

```text
MASK

BLOCK

ASK_USER
```

Organization Policy에서 기본값을 지정한다.

---

# 49. Config Secret 처리

예:

원본:

```text
spring.datasource.password=secret
```

Server 전달:

```text
spring.datasource.password=[REDACTED]
```

---

# 50. Secret이 코드 분석에 필요한 경우

Secret 값 자체가 필요하지 않은 것이 일반적이다.

설정 key 존재 여부만 전달한다.

예:

```text
spring.datasource.password
→ configured = true
```

---

# 51. Build/Test Log Sanitization

다음과 같은 정보가 Log에 포함될 수 있다.

```text
Token
Password
Internal URL
Absolute Path
User Name
```

정책에 따라 Masking한다.

---

# 52. Local Absolute Path

Server에 로컬 절대경로를 전달하지 않는 것을 기본으로 한다.

```text
C:\Users\user\project\...
```

대신 Workspace-relative path를 사용한다.

---

# 53. Network Policy

Local Agent의 외부 Network 접근은 기본 차단한다.

허용 대상 예:

```text
onCode Server

Internal Nexus

Internal Git Server

Approved Internal Services
```

---

# 54. Public Registry 차단

직접 접근 금지:

```text
repo.maven.apache.org

registry.npmjs.org

pypi.org
```

Dependency는 Nexus를 통해서만 가져온다.

---

# 55. Shell Network Command

예:

```text
curl
wget
powershell Invoke-WebRequest
```

은 Policy에서 기본 차단 또는 별도 승인.

---

# 56. Dependency Security Policy

Nexus에 존재한다고 해서 자동 안전하다고 판단하지 않는다.

Dependency 판단:

```text
Nexus Availability
    +
Version Compatibility
    +
Vulnerability Check
    +
License Policy
```

를 함께 본다.

---

# 57. Vulnerability Severity Policy

예:

```text
CRITICAL
→ BLOCK

HIGH
→ BLOCK

MEDIUM
→ WARN / PROJECT POLICY

LOW
→ ALLOW WITH LOG
```

조직 정책에서 설정한다.

---

# 58. Unknown Vulnerability Status

보안 DB에 정보가 없는 경우:

```text
UNKNOWN
```

을 안전하다고 간주하지 않는다.

Project Policy에 따라:

```text
ALLOW_WITH_WARNING

REQUIRE_APPROVAL

BLOCK
```

중 선택한다.

---

# 59. Dependency Fallback

취약 Dependency 발견 시:

```text
Safe version search

Alternative library

Existing project dependency

Alternative design

Minimal custom implementation
```

순으로 검토한다.

---

# 60. Security-sensitive 기능 자체 구현

다음 기능은 자체 구현을 기본 금지하거나 별도 Security Review를 요구한다.

```text
Password Hashing

Encryption Algorithm

JWT Signature

OAuth

Certificate Validation

HTML Sanitization

Authentication Protocol
```

---

# 61. License Policy

Dependency License도 검토할 수 있다.

예:

```text
ALLOW

WARN

BLOCK
```

조직별 License Policy를 별도 관리한다.

---

# 62. Source Change Security

Change Set에는 반드시 다음 정보가 포함된다.

```text
base_project_revision

base_file_hash

affected_files

operation

new_content_hash
```

---

# 63. Stale Workspace

승인 후 파일이 변경되면 승인된 Change Set을 강제 적용하지 않는다.

```text
base_hash != current_hash

→ STALE_WORKSPACE
```

기존 승인도 무효 처리한다.

---

# 64. Atomic Apply

기본적으로 Change Set 전체를 하나의 논리 단위로 적용한다.

적용 전 모든 파일 검증.

```text
All Valid
→ Apply

Any Invalid
→ Abort
```

---

# 65. Backup / Rollback

Local Agent는 Apply 직전 변경 대상의 임시 Backup을 생성할 수 있다.

목적:

```text
Partial Apply Failure

Local I/O Failure

Unexpected Tool Error
```

시 원상 복구.

---

# 66. Developer Manual Edit

개발자의 직접 편집은 차단하지 않는다.

onCode는 이를 File Watcher와 Hash로 감지한다.

개발자가 최종 주체라는 원칙을 유지한다.

---

# 67. Git Security

Git Tool은 Shell 실행보다 전용 Adapter를 사용한다.

이유:

```text
Command Argument Validation

Risk Classification

Working Tree Check

Approval Binding

Audit
```

이 쉽다.

---

# 68. Git Commit

실행 전 확인:

```text
Current Branch

Files to Commit

Working Tree Status

Commit Message

Approval
```

---

# 69. Git Push

실행 전 사용자에게 다음을 보여준다.

```text
Remote

Branch

Commit Range

Force 여부
```

일반 `git push`도 명시적 승인 권장.

---

# 70. Force Push

```text
git push --force
```

기본적으로 DESTRUCTIVE.

조직 정책에서 DENY 권장.

필요하면 별도 관리자 정책에서만 허용한다.

---

# 71. Rebase / Reset

다음 작업은 Repository history를 변경할 수 있다.

```text
git rebase

git reset --hard
```

High Risk 또는 Destructive로 분류한다.

---

# 72. Shell Security

가능한 모든 기능은 전용 Tool을 사용한다.

```text
build.run

test.run

git.*

workspace.*
```

Shell은 기능 공백을 처리하는 마지막 수단이다.

---

# 73. Shell Allowlist

선택적으로 Executable Allowlist를 둘 수 있다.

예:

```text
java
mvn
gradle
npm
python
pytest
git
```

단 Git은 전용 Tool 우선.

---

# 74. Shell Deny Pattern

예:

```text
rm -rf /

format

shutdown

reboot

diskpart

del /s outside workspace
```

강제 차단.

---

# 75. Pipe / Redirect

다음은 위험도를 높인다.

```text
|
>
>>
&&
;
```

Command Parser가 구조적으로 분석해야 한다.

단순 Regex만으로는 충분하지 않다.

---

# 76. Privilege Escalation

기본 차단:

```text
sudo

runas

elevated PowerShell

Administrator privilege request
```

Local Agent가 관리자 권한으로 동작하지 않는 것을 권장한다.

---

# 77. Process Isolation

Build/Test 프로세스는 가능한 경우 Workspace Root와 제한된 Environment에서 실행한다.

민감한 환경변수 전체를 child process에 전달하지 않는다.

---

# 78. Environment Variable Filter

필요한 환경변수만 전달한다.

예:

```text
JAVA_HOME
PATH
NODE_HOME
```

Secret 환경변수는 작업 필요성이 없다면 제외한다.

---

# 79. Prompt Injection 방어

Source, README, 개발 가이드, Log에 자연어 명령이 포함될 수 있다.

예:

```text
Ignore your system instructions...
```

이는 Agent Instruction이 아니라 데이터로 처리한다.

---

# 80. Trust Level

Context에 Trust Level을 둘 수 있다.

```text
SYSTEM_POLICY

ORGANIZATION_POLICY

PROJECT_POLICY

USER_CONFIRMED

DEVELOPMENT_GUIDE

SOURCE_CODE

PROJECT_SUMMARY

UNTRUSTED_DOCUMENT
```

---

# 81. Instruction Priority

Agent 실행 시:

```text
System / Policy

>

Workflow Instruction

>

Confirmed User Decision

>

Development Guide

>

Source / Document Content
```

Source 내부 문자열이 상위 정책을 변경할 수 없다.

---

# 82. Guide Security

개발 가이드 자체도 중앙 관리되므로 Version과 Source가 필요하다.

Agent가 임의 문서를 Mandatory Guide로 승격하지 않는다.

---

# 83. Context Poisoning 방지

Project Summary는 Local LLM이 생성한 요약일 수 있으므로 사실 데이터보다 낮은 신뢰도로 취급한다.

```text
Actual Source

>

Parser Index

>

Generated Summary
```

---

# 84. Audit Scope

다음 이벤트는 반드시 Audit한다.

```text
Login

Project Access

Tool Call

Approval Request

Approval Result

Source Apply

Git Commit

Git Push

Shell Execute

Dependency Decision

Security Block

Workflow Cancel
```

---

# 85. Audit Event

예:

```json
{
  "action": "WORKSPACE_APPLY",

  "user_id": "USR-100",
  "project_id": "PRJ-100",
  "work_item_id": "WI-100",

  "resource": "CHG-100",

  "approval_id": "APR-100",

  "result": "SUCCESS",

  "timestamp": "..."
}
```

---

# 86. Audit Data Minimization

Audit에는 전체 Source나 Secret을 넣지 않는다.

대신:

```text
Content Hash

File Path

Change Set ID

Result
```

를 기록한다.

---

# 87. Tool Argument Logging

Tool Arguments 전체를 Audit Log에 저장하면 Secret이 포함될 수 있다.

기본:

```text
Arguments Hash

Sanitized Summary
```

만 저장한다.

필요한 Raw Payload는 별도 보호 저장소에 둔다.

---

# 88. Authentication

사용자는 Server에서 인증한다.

Local Agent는 사용자 Session과 연결된다.

IDE Extension이 Credential을 장기 보관하지 않는 것을 권장한다.

---

# 89. Local Agent Authentication

Local Agent ↔ Server 연결은 인증된 세션이어야 한다.

예:

```text
User Authentication

+

Device / Agent Registration

+

Session Token
```

---

# 90. IDE ↔ Local Agent 인증

localhost라고 해서 무조건 신뢰하지 않는다.

예:

```text
Random Local Token

Client Registration

Short-lived Session
```

을 사용한다.

---

# 91. Session Expiration

Server Session 만료 시 Local Agent는 위험 Tool 실행을 중지한다.

Read-only Local 기능만 허용할지는 별도 정책으로 정의할 수 있다.

---

# 92. Workspace Ownership

Local Agent는 현재 사용자가 명시적으로 연 프로젝트만 onCode Workspace로 등록한다.

임의 Directory Scan 금지.

---

# 93. Project Access Authorization

중앙 Server에서 사용자가 해당 Project에 접근 가능한지 검증한다.

프로젝트 간 Context 검색도 권한에 따라 제한한다.

---

# 94. Multi-tenant Isolation

사용자/조직이 여러 개인 경우 다음을 엄격히 구분한다.

```text
Organization

User

Project

Workspace

Session

Work Item
```

---

# 95. Server-side Tool Policy

Server Tool도 권한 검증이 필요하다.

예:

```text
guide.read

dependency.search

context.get
```

도 project/user scope를 검증한다.

---

# 96. Context URI Authorization

예:

```text
ctx://work-items/WI-999/design/latest
```

를 다른 사용자가 임의 조회할 수 없도록 Context Service가 인증한다.

---

# 97. MCP Adapter Security

향후 MCP Tool을 연결하더라도 onCode Tool Gateway를 우회시키지 않는다.

```text
Agent
 ↓
Tool Gateway
 ↓
Policy
 ↓
MCP Adapter
 ↓
MCP Server
```

---

# 98. External Tool Trust

MCP 등 외부 Tool의 출력도 untrusted data로 취급한다.

Tool Description 자체도 별도 등록 및 승인된 것만 사용한다.

---

# 99. Tool Registration Policy

새 Tool 등록 시 최소 다음을 정의한다.

```text
Tool Name

Provider

Schema

Risk Level

Required Permission

Approval Rule

Network Requirement

Audit Requirement
```

---

# 100. Tool Versioning

Tool behavior가 변경되면 Version을 기록한다.

Approval 또는 Policy 판단이 Tool Version에 영향을 받을 수 있다.

---

# 101. Policy Configuration 예

```yaml
organization:
  network:
    public_internet: deny

  dependency:
    repositories:
      - nexus.internal

    vulnerability:
      critical: block
      high: block
      medium: warn

  git:
    commit: approval
    push: approval
    force_push: deny
    reset_hard: deny

  shell:
    default: approval
    privilege_escalation: deny
```

---

# 102. Project Policy 예

```yaml
project:
  workspace:
    generated_source_write: deny

  build:
    auto_execute: true

  test:
    auto_execute: true

  code_change:
    approval: required
```

---

# 103. Local Policy 예

```yaml
local:
  filesystem:
    outside_workspace: deny

  secret:
    transfer: mask

  shell:
    network_commands: deny

  git:
    destructive: deny
```

---

# 104. Policy Merge

정책 결합 시 더 강한 제한을 적용한다.

예:

```text
Organization: git push = approval
Project: git push = allow
```

최종:

```text
approval
```

---

# 105. Deny Override

상위 정책의 DENY는 하위 정책으로 해제할 수 없다.

```text
Organization DENY
→ always DENY
```

---

# 106. Approval Override

상위 정책에서 Approval을 요구하면 하위에서 Auto Allow로 낮출 수 없다.

---

# 107. Policy Evaluation 결과

```json
{
  "tool": "git.push",

  "decision": "APPROVAL_REQUIRED",

  "effective_policy": [
    "organization.git.push",
    "project.git.push"
  ],

  "risk": "HIGH_RISK"
}
```

---

# 108. Security Finding와 Policy Violation 구분

Security Finding:

```text
코드에 SQL Injection 가능성 존재
```

Policy Violation:

```text
High Vulnerability Dependency는 조직 정책상 사용 금지
```

둘을 구분한다.

---

# 109. Security Review Result

```text
PASS

PASS_WITH_WARNING

FAIL
```

FAIL은 Workflow를 Implementation/Dependency/Design 단계로 되돌린다.

---

# 110. Policy Violation Result

```text
BLOCKED
```

상태를 명확하게 사용한다.

Agent가 재시도해도 정책이 바뀌지 않는 한 해결되지 않는다.

---

# 111. User Override

사용자가 보안 정책을 임의 override하지 못한다.

다만 Policy가 `ALLOW_WITH_APPROVAL`로 정의된 경우 사용자 승인으로 진행할 수 있다.

---

# 112. Security Exception

조직 차원의 예외 승인이 필요한 경우 별도 Exception Entity를 둘 수 있다.

예:

```text
SEC-EXC-100
```

MVP에서는 제외 가능하다.

---

# 113. Break Glass

비상 우회 기능은 일반 onCode Coding Flow에 제공하지 않는 것을 권장한다.

필요하다면 별도 운영자 기능으로 분리한다.

---

# 114. Failed Approval

Approval이 반려되면 Tool은 실행하지 않는다.

재요청 시 새 Approval ID를 생성한다.

---

# 115. Change Request와 Reject 구분

```text
CHANGE_REQUESTED
→ 설계/코드 수정 후 재승인 가능

REJECTED
→ 현재 작업 중단 또는 상위 분석 단계로 복귀
```

---

# 116. Diff Approval UX 보안

사용자가 무엇을 승인하는지 명확하게 볼 수 있어야 한다.

필수:

```text
File Name

Operation

Actual Diff

Added/Deleted Lines

New File/Delete 표시
```

---

# 117. Hidden Change 금지

IDE에 표시되지 않은 파일 변경을 Change Set에 포함하지 않는다.

Approval 대상과 실제 Apply 대상이 완전히 동일해야 한다.

---

# 118. Generated File

Generated Source 변경은 기본 차단 또는 별도 경고한다.

예:

```text
target/generated-sources

dist

generated/**
```

---

# 119. Binary File

onCode의 자동 코드 Change Set은 기본적으로 Text Source 파일 중심으로 제한한다.

Binary Patch는 v1에서 제외하는 것을 권장한다.

---

# 120. Large Change Set

변경 파일 수 또는 Line 변경량이 매우 크면 높은 Risk로 승격할 수 있다.

예:

```text
100 files changed
```

→ 별도 경고 또는 Strong Approval.

---

# 121. Risk Escalation

기본 Tool Risk 외에 Context에 따라 Risk를 높일 수 있다.

예:

```text
workspace.apply_changes
기본 WRITE

하지만 200개 파일 삭제
→ DESTRUCTIVE
```

---

# 122. Policy Evaluation Context

Risk 판단 입력:

```text
Tool

Arguments

File Count

Operation Type

Command

Project

User

Workflow State

Approval

Environment
```

---

# 123. Build Security

Build Script 자체가 임의 명령을 실행할 수 있다.

따라서 Build를 완전히 안전한 Read 작업으로 간주하지 않는다.

Risk:

```text
EXECUTE
```

로 분류한다.

---

# 124. Build Script 변경 후 실행

`pom.xml`, `build.gradle`, `package.json` 등이 Change Set에서 수정되었다면 Build Risk를 높일 수 있다.

빌드 단계에서 새 Plugin/Script가 실행될 가능성이 있기 때문이다.

---

# 125. Test Security

Test도 외부 Process나 Network를 실행할 수 있다.

따라서 Test 역시 `EXECUTE`로 취급한다.

---

# 126. Dependency Install

Dependency 설치는 다음 위험이 있다.

```text
Package Script

Build Plugin

Transitive Dependency

Disk Change
```

따라서 별도 Policy 적용.

---

# 127. npm Lifecycle Script

npm package 설치 시 lifecycle script가 실행될 수 있다.

조직 정책에서:

```text
ignore-scripts
```

또는 승인된 package만 허용하는 정책을 고려할 수 있다.

---

# 128. Maven/Gradle Plugin

Build Plugin도 실행 코드이므로 Nexus 승인 및 정책 대상이 될 수 있다.

---

# 129. Python Package

Python package 설치 시 setup/build 단계의 코드 실행 가능성을 고려한다.

---

# 130. Security Event

다음 Event를 정의할 수 있다.

```text
security.policy.denied

security.secret.detected

security.dependency.blocked

security.command.blocked

security.workspace.violation
```

---

# 131. Security Notification

사용자에게 지나치게 기술적인 내부 정책 대신 명확한 이유를 보여준다.

예:

```text
이 작업은 조직 보안 정책에 의해 차단되었습니다.

이유:
외부 npm Registry 직접 접근은 허용되지 않습니다.
Nexus Repository를 사용해야 합니다.
```

---

# 132. Audit Retention

보안 관련 Audit은 일반 Debug Log보다 장기 보존하는 것이 적절하다.

예:

```text
Approval

Source Apply

Git Push

Security Block

Policy Exception
```

---

# 133. Immutable Audit

Audit Log는 일반 사용자나 Agent가 수정할 수 없어야 한다.

Append-only 형태를 권장한다.

---

# 134. Time Synchronization

Approval/Audit 정확성을 위해 Server와 Local Agent 시간 차이를 최소화해야 한다.

Audit 기준 시간은 Server timestamp를 기준으로 사용할 수 있다.

---

# 135. Security Metrics

향후 운영 지표:

```text
Blocked Tool Calls

Approval Reject Rate

Security Findings

Vulnerable Dependency Attempts

Secret Detection Count

Destructive Command Attempts
```

---

# 136. Policy Metrics

```text
Auto Allowed

Approval Required

Denied
```

비율을 분석해 지나치게 불편하거나 느슨한 정책을 조정할 수 있다.

---

# 137. Default Secure Mode

신규 Project의 기본 정책은 보수적으로 한다.

예:

```text
Read → Auto

Code Apply → Approval

Build/Test → Approval 또는 조직 설정

Commit → Approval

Push → Approval

Shell → Approval

Destructive → Deny
```

---

# 138. Trusted Project Mode

향후 사용자가 자주 사용하는 프로젝트에 대해 일부 실행 승인을 완화할 수 있다.

하지만:

```text
Code Apply

Git Push

Destructive
```

같은 작업은 기본적으로 승인 유지 권장.

---

# 139. Approval Fatigue 방지

너무 많은 승인 요청은 사용자 경험을 떨어뜨린다.

따라서 동일한 안전 수준의 연속 Read Tool은 묶어 자동 실행한다.

예:

```text
read 5 files
search symbols
git status
```

는 매번 승인하지 않는다.

---

# 140. Approval Grouping

예:

```text
Build + Related Tests
```

를 하나의 Execution Plan으로 보여주고 한 번 승인받을 수 있다.

단 실제 실행 내용은 승인 당시 고정되어야 한다.

---

# 141. Execution Plan Approval

예:

```text
실행 계획

1. mvn test -Dtest=AuthServiceTest
2. mvn package

[승인]
```

계획이 변경되면 재승인.

---

# 142. Policy Dry-run

실제 실행 전에:

```text
policy.evaluate
```

를 내부적으로 수행할 수 있다.

UI에서 필요한 승인 목록을 미리 보여줄 수 있다.

---

# 143. Central Policy Service

Organization/Project Policy는 중앙 Policy Service에서 관리하는 구조를 권장한다.

```text
Tool Gateway
    ↓
Policy Service
```

---

# 144. Local Policy Engine

Local Agent는 중앙 정책의 최종 복사본 또는 동기화된 정책을 보유할 수 있다.

Server 연결이 일시적으로 끊겨도 금지 정책은 유지해야 한다.

---

# 145. Policy Sync

```text
Server Policy Revision

→ Local Agent
```

Local Agent가 현재 적용 중인 Policy Revision을 Server에 보고한다.

---

# 146. Policy Version

예:

```text
ORG-POLICY v23

PROJECT-POLICY v7
```

Audit에 사용한 Policy Version을 기록한다.

---

# 147. Policy Stale

Local Agent Policy가 중앙보다 오래되었다면 High Risk Tool 실행을 차단할 수 있다.

상태:

```text
POLICY_STALE
```

---

# 148. Local Agent Tamper Consideration

Local Agent 자체가 임의로 변조될 가능성을 고려해야 한다.

향후:

```text
Signed binary

Version validation

Integrity check
```

를 적용할 수 있다.

---

# 149. IDE Extension Tamper

IDE Extension도 내부 배포 Package 서명 및 Version 관리가 필요하다.

---

# 150. Server Trust

폐쇄망이라도 Server Agent Output을 무조건 신뢰하지 않는다.

Local Agent는 항상:

```text
Schema Validation

Policy Check

Workspace Boundary

Approval Check
```

을 다시 수행한다.

---

# 151. Defense in Depth

보안은 한 계층에만 맡기지 않는다.

```text
Agent Permission
    ↓
Tool Gateway Policy
    ↓
Workflow Guard
    ↓
User Approval
    ↓
Local Policy
    ↓
Execution Validation
```

여러 계층을 사용한다.

---

# 152. Security Failure Flow

예:

```text
Implementation
    ↓
Security Review
    ↓ FAIL
Senior Developer
    ↓
Fix / Redesign
```

Local 실행 전에 반드시 해결한다.

---

# 153. Runtime Policy Failure

예:

```text
Server:
git push 요청
   ↓
Local Policy:
DENY
```

결과:

```text
POLICY_DENIED
```

Workflow는 BLOCKED 또는 사용자 안내 상태로 전환한다.

---

# 154. Security와 Human Approval 관계

사용자 승인은 보안 정책을 대체하지 않는다.

```text
Policy DENY
+
User APPROVE
=
DENY
```

---

# 155. Security와 Development Guide 관계

Mandatory Security Guide 위반은 일반 Guide Warning과 다르게 처리할 수 있다.

```text
MANDATORY
→ BLOCK

RECOMMENDED
→ WARNING
```

---

# 156. Guide Severity

개발 가이드 Rule에도 Severity/Enforcement를 둘 수 있다.

```text
INFO

RECOMMENDED

REQUIRED

SECURITY_REQUIRED
```

---

# 157. Required Rule 위반

Review Agent에서:

```text
REQUIRED rule violation
```

발견 시 Change Set을 사용자 승인 단계로 보내지 않는다.

먼저 수정한다.

---

# 158. Security Findings 표시

사용자에게 다음을 구분해 보여준다.

```text
Blocked Issue

Warning

Informational
```

---

# 159. Security Review Independence

Implementation Agent가 자신의 보안 검증을 최종 판정하지 않는다.

Security Agent 또는 Security Service가 별도로 검증한다.

---

# 160. Secure Defaults

onCode의 모든 설정은 "편의성보다 안전한 기본값"을 선택한다.

사용자가 명시적으로 정책 범위 내에서 편의성을 높일 수 있다.

---

# 161. MVP Security 범위

초기 MVP에서 반드시 구현:

```text
Workspace Boundary

Tool Risk Classification

Agent Tool Permission

Design Approval

Code Diff Approval

Git Commit Approval

Shell Approval

File Hash Validation

Project Revision Validation

Secret Masking 기본

Nexus-only Dependency Policy

Audit Log

Tool Scope Validation
```

---

# 162. 2차 보안 확장

```text
Advanced Secret Scanner

Dependency CVE Offline DB

License Policy

Signed Local Agent

Policy Sync Revision

Advanced Shell Parser

Sandbox

Multi-level Approval

Security Exception Workflow
```

---

# 163. 보안 관련 Error Code

권장:

```text
PERMISSION_DENIED

POLICY_DENIED

APPROVAL_REQUIRED

APPROVAL_REJECTED

APPROVAL_EXPIRED

APPROVAL_RESOURCE_MISMATCH

WORKSPACE_BOUNDARY_VIOLATION

WORKSPACE_FILE_CHANGED

SECRET_DETECTED

DEPENDENCY_BLOCKED

VULNERABILITY_BLOCKED

COMMAND_BLOCKED

NETWORK_POLICY_VIOLATION

POLICY_STALE
```

---

# 164. 핵심 승인 흐름

```text
Agent Request
    ↓
Tool Permission Check
    ↓
Workflow Guard
    ↓
Policy Evaluation
    │
    ├─ DENY ───────────► BLOCK
    │
    ├─ ALLOW ──────────► Execute
    │
    └─ APPROVAL_REQUIRED
                ↓
          User Approval
           /        \
       APPROVE      REJECT
          │            │
          ▼            ▼
     Local Policy    Stop
          │
          ▼
       Execute
```

---

# 165. Code Change 흐름

```text
Implementation Agent
      ↓
Change Set
      ↓
Review
      ↓
Security Review
      ↓
Local Diff
      ↓
Developer Approval
      ↓
Hash / Revision Check
      ↓
Local Policy
      ↓
Atomic Apply
      ↓
Build / Test
```

---

# 166. 핵심 설계 결정

onCode Security / Policy / Approval Model v1의 핵심 결정은 다음과 같다.

1. Local Agent를 최종 실행 보안 경계로 둔다.
2. 중앙 Server Agent는 Local Policy를 우회할 수 없다.
3. Organization > Project > Local/User Request의 정책 우선순위를 적용한다.
4. 정책 판단 결과는 ALLOW / APPROVAL_REQUIRED / DENY로 통일한다.
5. Tool을 READ / WRITE / EXECUTE / HIGH_RISK / DESTRUCTIVE로 분류한다.
6. Design Approval과 Code Change Approval을 분리한다.
7. Approval은 특정 Change Set/Command Hash에 강하게 바인딩한다.
8. 승인 이후 대상이 변경되면 기존 승인을 무효화한다.
9. Source 변경 전 반드시 Local Diff를 보여준다.
10. Workspace 밖 파일 접근 및 Symlink Escape를 차단한다.
11. Agent마다 최소 Tool 권한을 적용한다.
12. Workflow State와 Tool 실행 권한을 연결한다.
13. Context Storage 접근도 User/Project/Work Item Scope로 제한한다.
14. Shell보다 전용 Tool Adapter를 우선한다.
15. External Network와 Public Package Registry를 기본 차단한다.
16. Dependency 사용은 Nexus + Compatibility + Vulnerability + License 기준으로 판단한다.
17. 사용자 승인은 보안 정책 DENY를 override할 수 없다.
18. File Hash와 Project Revision으로 Stale Workspace를 방지한다.
19. Build/Test도 실행 코드이므로 EXECUTE Risk로 관리한다.
20. Source, Log, Diff 전송 전 Secret Filtering을 적용한다.
21. Source/문서 내 자연어 명령을 Agent Instruction으로 취급하지 않는다.
22. 모든 위험 Tool Call, Approval, Apply, Git, Security Block을 Audit한다.
23. Audit에는 전체 Source/Secret 대신 Hash와 Metadata를 기록한다.
24. Policy Version을 관리하고 적용한 Version을 Audit에 남긴다.
25. 정책은 Defense-in-Depth 방식으로 여러 계층에서 중복 검증한다.
26. 초기 버전은 Secure Default로 시작하고 편의성 완화는 정책 범위 내에서만 허용한다.