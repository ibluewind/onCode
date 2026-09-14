# SPEC-04 onCode Local Agent Architecture v1

## 1. 목적

본 명세는 onCode Local Agent의 내부 아키텍처를 정의한다.

Local Agent는 개발자 PC에서 동작하며 다음 역할을 담당한다.

- IDE와 중앙 서버 간 중계
- 프로젝트 구조 분석
- 파일 및 Symbol 인덱싱
- 필요한 소스 제공
- Diff 생성
- 승인된 변경 적용
- 빌드
- 테스트
- Git 작업
- Shell 실행
- 로컬 정책 검증
- 실행 결과 구조화
- 프로젝트 변경 감지

Local Agent는 코딩 의사결정의 주체가 아니다.

핵심 원칙은 다음과 같다.

> 중앙 서버는 판단하고, Local Agent는 검증된 요청을 안전하게 실행한다.

---

# 2. 전체 구조

```text
┌──────────────────────────── IDE ────────────────────────────┐
│                                                           │
│ Eclipse / VS Code / IntelliJ                              │
│                                                           │
│ Chat / Diff / Approval / Progress / Selection Context     │
└────────────────────────────┬───────────────────────────────┘
                             │
                             ▼
┌────────────────────── Local Agent ─────────────────────────┐
│                                                           │
│                    IDE Bridge                             │
│                       │                                   │
│                       ▼                                   │
│                Local Agent Core                           │
│                       │                                   │
│      ┌────────────────┼─────────────────────┐             │
│      │                │                     │             │
│      ▼                ▼                     ▼             │
│ Project           Workspace              Execution        │
│ Intelligence      Manager                Runtime          │
│      │                │                     │             │
│      ├─ Scanner       ├─ Read              ├─ Build       │
│      ├─ Parser        ├─ Search            ├─ Test        │
│      ├─ Symbols       ├─ Diff              ├─ Git         │
│      ├─ Relations     ├─ Patch             └─ Shell       │
│      └─ Index         └─ File Write                       │
│                                                           │
│                Policy / Approval Engine                   │
│                           │                               │
│                           ▼                               │
│                   Server Connector                        │
│                                                           │
└────────────────────────────┬───────────────────────────────┘
                             │
                             ▼
                       onCode Server
```

---

# 3. 주요 컴포넌트

Local Agent는 다음 7개 주요 컴포넌트로 구성한다.

```text
1. IDE Bridge
2. Server Connector
3. Local Agent Core
4. Project Intelligence
5. Workspace Manager
6. Execution Runtime
7. Policy & Approval Engine
```

---

# 4. IDE Bridge

IDE Extension과 Local Agent 간 인터페이스를 담당한다.

IDE별 구현 차이를 Local Agent Core에서 분리한다.

지원 IDE:

```text
Eclipse
VS Code
IntelliJ IDEA
```

IDE Bridge 주요 역할:

```text
Chat Message 전달

Current File 전달

Selection Context 전달

Cursor Position 전달

Open Files 정보 전달

Diff UI 요청

Approval UI 요청

Question UI 요청

Progress 표시

Notification 전달
```

---

# 5. IDE Bridge 설계 원칙

IDE별 Plugin은 최대한 Thin Client로 유지한다.

```text
IDE Plugin
    ↓
UI / Context 수집
    ↓
Local Agent
```

다음 기능은 IDE Plugin에서 직접 처리하지 않는다.

```text
Maven 실행
Gradle 실행
npm 실행
Git 명령
파일 분석
AST 분석
Project Index 생성
Security Policy 판단
```

이 기능들은 Local Agent에 집중한다.

---

# 6. IDE ↔ Local Agent 연결

Local IPC를 기본으로 한다.

후보:

```text
Local HTTP

WebSocket

Named Pipe

Unix Domain Socket

gRPC Local
```

권장 구조:

```text
VS Code Extension
      │
      ├──┐
IntelliJ Plugin
      │  │
      ├──┼── Local WebSocket / gRPC
Eclipse Plugin
      │  │
      └──┘
           ↓
       Local Agent
```

Localhost 외부 접근은 기본 차단한다.

---

# 7. Server Connector

중앙 onCode Server와 통신하는 컴포넌트이다.

역할:

```text
Authentication

Session 연결

Tool Request 수신

Tool Response 전송

Event 전송

Capability Negotiation

Heartbeat

Reconnect

Request Correlation

Streaming
```

---

# 8. 연결 방식

서버가 Local Agent에 Tool을 호출해야 하므로 지속 연결을 권장한다.

폐쇄망 환경에서는 다음 구조가 적합하다.

```text
Local Agent
    │
    │ outbound persistent connection
    ▼
onCode Server
```

즉 Server가 개발자 PC에 inbound 접속하지 않는다.

이 방식은 방화벽 및 보안 정책상 유리하다.

---

# 9. 연결 수립

Local Agent 실행:

```text
Local Agent Start
       ↓
Server Authentication
       ↓
Session Registration
       ↓
system.get_capabilities
       ↓
Project Registration
       ↓
Persistent Connection
```

---

# 10. Heartbeat

Local Agent 상태를 Server가 확인할 수 있어야 한다.

예:

```json
{
  "event": "agent.heartbeat",
  "data": {
    "agent_id": "LOCAL-100",
    "status": "AVAILABLE",
    "active_project": "PRJ-100"
  }
}
```

권장 interval:

```text
15 ~ 30 seconds
```

환경에 따라 설정 가능하도록 한다.

---

# 11. Reconnect

네트워크 연결이 끊어진 경우:

```text
CONNECTED
   ↓
DISCONNECTED
   ↓
RECONNECTING
   ↓
CONNECTED
```

재연결 후 반드시 다음을 확인한다.

```text
Session 상태

Pending Tool Call

Pending Approval

Project Revision

Capability 변경 여부
```

---

# 12. Local Agent Core

Local Agent 내부 요청을 Routing하는 핵심 컴포넌트이다.

역할:

```text
Tool Dispatch

Request Validation

Policy 적용

Component Routing

Execution Context 관리

Error Mapping

Audit Event 생성
```

구조:

```text
Tool Request
    ↓
Local Agent Core
    ↓
Tool Registry
    ↓
Policy Engine
    ↓
Handler
```

---

# 13. Tool Registry

Local Agent가 제공하는 Tool 목록을 관리한다.

예:

```text
project.*
workspace.*
build.*
test.*
git.*
shell.*
system.*
```

Tool마다 다음 Metadata를 관리한다.

```json
{
  "name": "workspace.apply_changes",
  "version": "1.0",
  "risk": "WRITE",
  "requires_approval": true,
  "idempotent": false
}
```

---

# 14. Project Intelligence

프로젝트를 서버가 효율적으로 탐색할 수 있도록 구조화하는 핵심 기능이다.

하위 구성:

```text
Project Detector
Workspace Scanner
Language Detector
Parser Adapter
Symbol Analyzer
Dependency Analyzer
Relation Analyzer
Summary Generator
Incremental Indexer
Project Index Store
Project Index Sync
```

---

# 15. Project Detector

프로젝트 최초 분석 시 주요 기술 환경을 탐지한다.

대상 파일:

```text
pom.xml
build.gradle
settings.gradle

package.json

requirements.txt
pyproject.toml
setup.py

vue.config.*
vite.config.*
tsconfig.json

application.yml
application.properties
```

결과:

```json
{
  "languages": ["java", "typescript"],
  "frameworks": ["spring-boot", "react"],
  "build_tools": ["maven", "npm"],
  "package_managers": ["maven", "npm"]
}
```

---

# 16. 지원 기술

초기 요구사항 기준:

```text
Java
Python
JavaScript
TypeScript

Spring Framework
Spring Boot

eGovFramework

React
Vue.js

Node.js
```

---

# 17. Parser Adapter

언어별 Parser를 추상화한다.

```text
ParserAdapter
   ├─ JavaParserAdapter
   ├─ PythonParserAdapter
   ├─ JavaScriptParserAdapter
   ├─ TypeScriptParserAdapter
   └─ VueParserAdapter
```

공통 인터페이스 예:

```text
supports(file)

parse(file)

extractSymbols()

extractImports()

extractRelations()

extractDiagnostics()
```

---

# 18. 구조 정보는 Parser 우선

다음 정보는 LLM이 아닌 Parser를 통해 수집한다.

```text
Class
Interface
Method
Function
Field
Import
Package
Extends
Implements
Signature
Line Number
```

원칙:

```text
Parser
→ 사실 정보

LLM
→ 의미 요약
```

---

# 19. Summary Generator

LLM 또는 내부 요약 모델을 이용하여 다음을 생성할 수 있다.

```text
파일 역할 요약
클래스 역할 요약
메서드 역할 요약
```

예:

```text
AuthService.java
→ 사용자 인증 및 로그인 상태를 관리한다.

login(...)
→ 사용자 인증 정보를 확인하고 로그인 결과를 반환한다.
```

---

# 20. Summary 생성 정책

모든 함수를 무조건 LLM으로 요약할 필요는 없다.

우선순위:

```text
Public API
Service Method
Controller Method
Repository Method
중요 Business Logic
```

단순 getter/setter 등은 제외 가능하다.

---

# 21. Project Index Store

Local Agent 내부에 다음 데이터를 저장한다.

```text
.codegen/
├─ PROJECT_SUMMARY.md
├─ PROJECT_INDEX.json
└─ state/
```

`.codegen`은 프로젝트 메타데이터 디렉터리로 사용한다.

---

# 22. PROJECT_SUMMARY.md

사람과 LLM이 읽기 위한 프로젝트 개요이다.

포함:

```text
Project Profile

Directory Tree

File Summary

Class/Function Summary

주요 Dependency

주요 Module 관계
```

---

# 23. PROJECT_INDEX.json

Machine-readable 구조화 Index이다.

서버와 동기화할 공식 데이터로 사용한다.

주요 항목:

```text
Project Metadata

File

Symbol

Relation

Dependency

Hash

Revision
```

세부 Schema는 SPEC-05에서 정의한다.

---

# 24. Incremental Indexer

전체 Index를 매번 재생성하지 않는다.

File Watcher를 이용한다.

```text
File Changed
     ↓
Hash Check
     ↓
해당 File 재분석
     ↓
Symbol 변경
     ↓
Relation 변경
     ↓
Summary 필요 시 재생성
     ↓
Index Revision 증가
```

---

# 25. Ignore 정책

다음 디렉터리는 기본적으로 Indexing 대상에서 제외한다.

```text
.git

node_modules

target

build

dist

out

.venv

venv

__pycache__

.idea

.vscode
```

프로젝트 설정으로 추가 제외 가능하다.

---

# 26. Binary File 제외

다음은 Source Index에서 제외한다.

```text
jar
class
exe
dll
so
png
jpg
pdf
zip
```

단, 필요 시 metadata만 저장할 수 있다.

---

# 27. Generated Source

자동 생성 파일은 별도 표시한다.

예:

```json
{
  "generated": true
}
```

기본적으로 Implementation Agent가 Generated Source를 직접 수정하지 않도록 정책화한다.

---

# 28. Workspace Manager

파일 및 Workspace 작업을 담당한다.

주요 기능:

```text
List
Read
Search
Diff
Patch
Create
Modify
Delete
Rename
Hash
Revision Check
```

---

# 29. Workspace Boundary

Workspace Root를 명확히 제한한다.

```text
workspace_root
    ↓
허용
```

다음은 차단한다.

```text
../

../../

absolute path outside workspace
```

---

# 30. Symbol-aware Read

단순 파일 전체 Read 외에도 Symbol 또는 Line Range 기반 읽기를 향후 지원할 수 있다.

예:

```text
workspace.read_symbol
workspace.read_range
```

이 기능은 대형 파일의 Context 절감을 위해 유용하다.

---

# 31. Workspace Search

검색 방식:

```text
Literal
Regex
File Name
Symbol
```

향후 구조 검색:

```text
AST Query
Reference Search
```

까지 확장할 수 있다.

---

# 32. Change Proposal

Server가 보낸 코드를 즉시 적용하지 않는다.

```text
Server
   ↓
workspace.propose_changes
   ↓
Local Agent
   ↓
현재 Source와 비교
   ↓
Diff 생성
   ↓
Developer 확인
```

---

# 33. Diff Generator

Diff는 Local Agent가 생성한다.

이유:

Server가 분석 이후 파일이 변경되었을 수 있기 때문이다.

입력:

```text
Current Workspace File

+

Proposed File Content
```

결과:

```text
Unified Diff
Structured Diff
```

IDE Extension은 이를 시각화한다.

---

# 34. Diff Metadata

각 Diff에는 다음 정보를 포함한다.

```text
file
operation
base_hash
current_hash
line_changes
change_set_id
```

---

# 35. Approval과 Apply 분리

반드시 다음 두 단계를 분리한다.

```text
PROPOSE
    ↓
APPROVE
    ↓
APPLY
```

Approval 없이 APPLY할 수 없다.

---

# 36. Apply 직전 검증

`workspace.apply_changes` 실행 전 확인:

```text
Approval valid

Approval 대상 Change Set 일치

base_hash == current_hash

Workspace Root 정상

파일 Lock 문제 없음
```

불일치:

```text
WORKSPACE_FILE_CHANGED
```

---

# 37. Atomic Change Set

기본 정책으로 Change Set 단위 Atomic Apply를 권장한다.

```text
5개 파일 변경
     ↓
모두 적용 가능 확인
     ↓
전체 적용
```

중간 오류가 나면 가능한 경우 rollback한다.

---

# 38. Local Backup

Apply 직전 임시 Snapshot 또는 Backup을 생성하는 것을 권장한다.

예:

```text
.codegen/state/changes/CHG-100/
```

문제 발생 시 Local Agent가 적용 전 상태로 되돌릴 수 있다.

---

# 39. Execution Runtime

로컬 명령 실행 계층이다.

하위 Adapter:

```text
Build Adapter
Test Adapter
Git Adapter
Shell Adapter
```

---

# 40. Execution Sandbox

가능한 경우 실행 범위를 Workspace Root로 제한한다.

기본 working directory:

```text
project root
```

Tool별로 명시적으로 변경할 수 있으나 Workspace 범위 검증을 거친다.

---

# 41. Build Adapter

공통 인터페이스:

```text
BuildAdapter

detect()

supports()

build()

parseResult()
```

구현:

```text
MavenBuildAdapter
GradleBuildAdapter
NpmBuildAdapter
PythonBuildAdapter
```

---

# 42. Maven Adapter

Project Profile 또는 pom.xml 기준으로 실행한다.

가능한 명령:

```text
mvn compile

mvn package

mvn test
```

실제 명령 선택은 `build.run` 옵션에 따라 결정한다.

---

# 43. Gradle Adapter

Wrapper가 존재하면 우선 사용한다.

```text
./gradlew
gradlew.bat
```

시스템 Gradle보다 프로젝트 Wrapper를 우선한다.

---

# 44. Node Adapter

Package Manager 탐지:

```text
package-lock.json → npm

yarn.lock → yarn

pnpm-lock.yaml → pnpm
```

폐쇄망 정책상 허용된 Package Manager만 실행한다.

---

# 45. Python Adapter

탐지:

```text
pytest
unittest
pyproject.toml
requirements.txt
```

가상환경 정보도 Project Profile에 저장할 수 있다.

---

# 46. Build Output Parser

LLM에게 전체 로그를 바로 보내지 않는다.

Adapter가 다음을 추출한다.

```text
exit_code

error_type

file

line

column

message

warning

relevant_log
```

---

# 47. Test Adapter

공통 인터페이스:

```text
TestAdapter

discover()

run()

parseResult()
```

지원 후보:

```text
JUnit
Maven Surefire
Gradle Test

pytest
unittest

npm test
Jest
Vitest
```

---

# 48. Related Test Selection

변경 파일과 관련 있는 테스트를 우선 실행할 수 있다.

```text
AuthService.java
      ↓
AuthServiceTest.java
```

Project Relation Index를 활용할 수 있다.

---

# 49. Test Scope

```text
RELATED
FILE
MODULE
PROJECT
CUSTOM
```

---

# 50. Git Adapter

Git 실행은 Shell 직접 실행보다 전용 Adapter를 우선한다.

Tool:

```text
git.status
git.diff
git.log
git.branch
git.checkout
git.add
git.commit
git.merge
git.rebase
git.push
git.reset
git.clean
```

---

# 51. Git Status Check

변경 작업 전 최소한 다음을 확인한다.

```text
Repository 여부

Current Branch

Working Tree Status

Conflict 존재 여부
```

---

# 52. Git Dangerous Operations

다음 작업은 높은 Risk로 분류한다.

```text
git reset --hard
git clean -fd
git push --force
git rebase
```

기본적으로 자동 실행을 금지한다.

---

# 53. Shell Adapter

Shell은 가장 위험한 범용 Tool이다.

기본 원칙:

```text
전용 Tool로 처리 가능한 작업은 Shell 금지
```

예:

```text
git status
```

는:

```text
shell.execute("git status")
```

보다:

```text
git.status
```

를 사용한다.

---

# 54. Shell Policy

Shell 명령은 Parsing 및 Policy 검증을 거친다.

검토 대상:

```text
Executable

Arguments

Working Directory

Pipe

Redirect

Delete

Network Access

Privilege Escalation
```

---

# 55. Network Access

폐쇄망 Local Agent에서도 임의 외부 Network 호출을 허용하지 않는다.

예:

```text
curl
wget
pip direct internet
npm public registry
maven central
```

기본 차단한다.

Package 다운로드는 Nexus를 이용해야 한다.

---

# 56. Policy & Approval Engine

Local Agent 보안의 핵심 계층이다.

Server가 요청했다고 해서 무조건 실행하지 않는다.

```text
Server Tool Request
       ↓
Local Policy Engine
       ↓
ALLOW / APPROVAL / DENY
```

---

# 57. Risk Level

기본 등급:

```text
READ

WRITE

EXECUTE

HIGH_RISK

DESTRUCTIVE
```

---

# 58. 정책 예

```text
workspace.read_file
→ READ
→ AUTO

workspace.apply_changes
→ WRITE
→ CODE APPROVAL 필요

build.run
→ EXECUTE

test.run
→ EXECUTE

git.commit
→ HIGH_RISK
→ Approval

git.push
→ HIGH_RISK
→ Approval

git.reset --hard
→ DESTRUCTIVE
→ Strong Approval 또는 DENY
```

---

# 59. Policy Decision

```json
{
  "decision": "APPROVAL_REQUIRED",
  "risk": "HIGH_RISK",
  "reason": "git.commit modifies repository history."
}
```

가능한 결과:

```text
ALLOW
APPROVAL_REQUIRED
DENY
```

---

# 60. Approval Validation

Local Agent는 Server가 전달한 `approval_id`만 신뢰하지 않는다.

다음 정보를 함께 검증해야 한다.

```text
approval_id

project_id

work_item_id

resource_id

change_set_id 또는 command hash

approved action

expiration
```

---

# 61. Command Hash

Shell/Git 명령 승인은 실제 승인한 명령과 실행 명령이 동일한지 확인해야 한다.

예:

```text
SHA256(
 executable
 + args
 + working_directory
)
```

승인 이후 Command가 변경되면 승인 무효 처리한다.

---

# 62. Secret Scanner

서버로 Source를 전송하기 전 Secret Pattern을 검사할 수 있다.

검사 예:

```text
Private Key
Password
API Key
Access Token
Credential
```

---

# 63. Secret Policy

발견 시 프로젝트 정책에 따라:

```text
MASK

BLOCK

ASK_USER
```

를 적용한다.

---

# 64. Source Transfer Filter

Server가 Source를 요청하면 Local Agent는 전송 전 다음을 검증한다.

```text
Workspace 내부 파일인가

Binary 아닌가

Ignore 대상 아닌가

Secret 포함 여부

File Size Limit

Approval 필요 여부
```

---

# 65. File Size Limit

매우 큰 Source를 통째로 전송하지 않도록 제한한다.

예:

```text
max_source_file_size = configurable
```

초과하면:

```text
workspace.read_range

workspace.read_symbol
```

방식으로 나눠 제공한다.

---

# 66. Local Artifact Store

Local Agent는 실행 결과를 임시 저장할 수 있다.

```text
.codegen/state/
├─ logs/
├─ diffs/
├─ changes/
└─ executions/
```

예:

```text
Build Full Log

Test Full Log

Diff

Backup
```

---

# 67. Artifact Lifecycle

Artifact는 무기한 저장하지 않는다.

예:

```text
active workflow
→ 유지

workflow completed
→ retention policy 적용

expired
→ 삭제
```

---

# 68. Execution Manager

동시 실행 Tool을 관리한다.

필드:

```text
execution_id

tool

process_id

status

started_at

timeout

cancel_token
```

---

# 69. Process Cancellation

Build/Test/Shell은 취소 가능해야 한다.

```text
CANCEL
   ↓
Execution Manager
   ↓
Process terminate
   ↓
tool.cancelled
```

가능하면 child process까지 정리한다.

---

# 70. Timeout Handling

각 Tool별 기본 timeout을 가진다.

예:

```text
workspace.read_file
→ 짧음

build.run
→ 김

test.run
→ 김
```

프로젝트 정책에서 override 가능하다.

---

# 71. Capability Manager

Local Agent 실행 환경 정보를 관리한다.

예:

```json
{
  "os": "WINDOWS",
  "arch": "X86_64",
  "java": ["17", "21"],
  "node": ["22"],
  "python": ["3.12"],
  "git": true,
  "maven": true,
  "gradle": true
}
```

Server가 지원 불가능한 Tool을 요청하지 않도록 한다.

---

# 72. Capability 변경

JDK 설치 등 환경이 바뀔 수 있다.

변경 감지 시:

```text
system.capabilities.changed
```

Event를 전송한다.

---

# 73. Local Agent 상태

```text
STARTING

CONNECTING

READY

BUSY

DISCONNECTED

DEGRADED

STOPPING
```

---

# 74. Project 상태

Local Agent는 프로젝트별 상태를 관리한다.

```text
UNREGISTERED

SCANNING

INDEXING

READY

STALE

ERROR
```

---

# 75. IDE Context

개발자가 요청할 때 IDE Extension은 다음을 함께 전달할 수 있다.

```text
Current File

Selection

Cursor Position

Open Files

Current Module

Current Error Marker
```

---

# 76. IDE Context 최소화

모든 열린 파일 내용을 서버로 보내지 않는다.

예:

```json
{
  "current_file": "AuthService.java",
  "selection": {
    "start_line": 34,
    "end_line": 58
  }
}
```

실제 소스는 Server가 필요할 때 Tool로 요청한다.

---

# 77. Project Registration

Local Agent에서 프로젝트를 처음 열면:

```text
Project Detection
       ↓
Project Fingerprint
       ↓
Server Register
       ↓
Project Index 생성
       ↓
Index Sync
```

---

# 78. Project Fingerprint

같은 Repository/Workspace를 식별하기 위해 Fingerprint를 사용한다.

후보 입력:

```text
Git Remote ID

Repository Root

Initial Commit

Project Metadata
```

민감한 로컬 절대경로 자체를 중앙 식별자로 사용하는 것은 피한다.

---

# 79. Local Workspace ID

동일 Project를 여러 개발자가 각자 checkout할 수 있다.

따라서:

```text
PROJECT
```

와:

```text
LOCAL_WORKSPACE
```

를 구분하는 것을 권장한다.

예:

```text
PRJ-100
  ├─ WS-Alice
  ├─ WS-Bob
  └─ WS-Charlie
```

---

# 80. Workspace Revision

Local Workspace마다 별도 revision을 가진다.

```text
workspace_revision = 183
```

Server가 저장한 Project Catalog는 이 revision과 연결한다.

---

# 81. File Watcher

감시 대상:

```text
CREATE
MODIFY
DELETE
RENAME
```

File Event를 debounce해서 처리한다.

IDE 저장 시 연속 이벤트가 여러 번 발생할 수 있기 때문이다.

---

# 82. Index Update Batch

여러 파일이 짧은 시간 내 변경되면 하나의 Revision으로 처리할 수 있다.

```text
10 files changed
     ↓
debounce 500ms
     ↓
revision 184
```

값은 설정 가능하다.

---

# 83. Git Branch 변경 감지

Branch가 변경되면 프로젝트 구조가 크게 달라질 수 있다.

```text
git checkout feature-x
```

감지 후:

```text
Project Revision 증가

변경 파일 재분석

필요하면 Full Rescan
```

을 수행한다.

---

# 84. Dependency File 변경

다음 파일 변경은 중요한 이벤트로 본다.

```text
pom.xml
build.gradle
package.json
requirements.txt
pyproject.toml
```

Project Profile과 Dependency Index를 재생성한다.

---

# 85. Configurable Project Policy

프로젝트 루트에 설정 파일을 둘 수 있다.

예:

```text
.codegen/config.yaml
```

예:

```yaml
index:
  exclude:
    - generated/**
    - vendor/**

policy:
  shell: restricted
  git_push: approval
  destructive_git: deny

build:
  preferred: maven

security:
  secret_transfer: mask
```

---

# 86. Server Policy와 Local Policy

정책 우선순위:

```text
Local Security Policy

    >=

Server Policy
```

즉 Server가 `ALLOW`해도 Local Agent가 `DENY`할 수 있다.

반대는 허용하지 않는다.

---

# 87. 정책 우선순위

예:

```text
Organization Policy
    >
Project Policy
    >
User Preference
    >
Server Request
```

로컬 보안 정책은 중앙 요청보다 강해야 한다.

---

# 88. Audit

Local Agent도 최소한의 로컬 Audit를 남길 수 있다.

예:

```text
Tool Call
Approval
File Apply
Git Command
Shell Command
Build
Test
```

중앙 Audit Store와 동기화할 수 있다.

---

# 89. Privacy

중앙 Server에는 필요한 Source만 제공한다.

기본 원칙:

```text
Project Index
→ 먼저 전달

Source
→ 필요 시 요청

Full Workspace
→ 전달 금지
```

---

# 90. Local Agent Tool Flow

대표적인 파일 조회:

```text
Server
  ↓ workspace.read_files
Server Connector
  ↓
Local Agent Core
  ↓
Policy Engine
  ↓
Workspace Manager
  ↓
Secret Filter
  ↓
Response
```

---

# 91. Code Apply Flow

```text
Server
  ↓ workspace.propose_changes
Local Agent
  ↓
Diff Generator
  ↓
IDE
  ↓
Developer Approval
  ↓
Server Workflow
  ↓ workspace.apply_changes
Local Policy Engine
  ↓
Hash Check
  ↓
Backup
  ↓
Apply
  ↓
Project Index Update
  ↓
Result
```

---

# 92. Build Flow

```text
Server
  ↓ build.run
Local Agent
  ↓
Policy Check
  ↓
Build Adapter
  ↓
Process Runner
  ↓
Log Parser
  ↓
Structured Build Result
  ↓
Server
```

---

# 93. Git Flow

```text
Developer
   ↓
"commit 해줘"
   ↓
Server
   ↓ git.status
Local
   ↓
Server analyzes
   ↓ user.request_approval
Developer
   ↓ approve
Server
   ↓ git.commit + approval_id
Local Policy
   ↓
Git Adapter
```

---

# 94. 오류 구조화

Local Agent 내부 예외를 그대로 Server에 노출하지 않는다.

공통 Error Model로 변환한다.

예:

```json
{
  "code": "BUILD_TOOL_NOT_FOUND",
  "category": "BUILD",
  "retryable": false,
  "details": {
    "build_tool": "maven"
  }
}
```

---

# 95. Log 관리

Server로 전체 log를 무조건 보내지 않는다.

Local:

```text
full.log
```

Server:

```text
structured result
+
필요 시 log excerpt
```

원칙을 사용한다.

---

# 96. Resource Limits

Local Agent가 개발자 PC를 과도하게 사용하는 것을 막기 위해 제한을 둘 수 있다.

```text
max_processes

max_parallel_build

max_parallel_test

cpu_limit

memory_limit
```

가능한 범위에서 적용한다.

---

# 97. Task Queue

동시에 여러 Tool Request가 들어올 수 있다.

```text
READ Tool
→ 병렬 가능

WRITE Tool
→ 파일 충돌 확인

BUILD
→ 프로젝트별 1개 권장

GIT WRITE
→ Repository별 직렬화
```

---

# 98. File Lock

강제 OS Lock보다는 논리적 Soft Lock을 권장한다.

개발자가 IDE로 수정하는 것을 막지 않는다.

```text
Server change preparation
     ↓
base_hash
     ↓
Developer edits file
     ↓
Apply 시 hash mismatch
```

방식으로 안전하게 실패시킨다.

---

# 99. Local Agent 업데이트

폐쇄망 환경을 고려하면 자동 인터넷 업데이트를 전제하지 않는다.

업데이트 방식은 별도 내부 배포 체계를 사용한다.

Local Agent는 Version/Protocol 정보를 Server에 제공한다.

---

# 100. Protocol Compatibility

연결 시:

```text
Server Protocol
Local Protocol
```

을 비교한다.

호환되지 않을 경우:

```text
PROTOCOL_VERSION_UNSUPPORTED
```

으로 연결을 제한한다.

---

# 101. Plugin Architecture

향후 기술 스택 추가를 위해 Adapter 구조를 권장한다.

```text
Language Parser Plugin

Build Plugin

Test Plugin

SCM Plugin
```

예:

```text
Java
Python
Go
C#
```

확장이 가능해야 한다.

---

# 102. Java 지원

초기 Java 분석은 다음 정보까지 제공하는 것을 권장한다.

```text
Package

Imports

Class

Interface

Annotation

Method

Field

Extends

Implements

Method Calls
```

Spring 추가 분석:

```text
@Controller
@RestController
@Service
@Repository
@Component

@RequestMapping
@GetMapping
@PostMapping

@Autowired

@Transactional
```

---

# 103. eGovFramework 지원

eGovFramework는 Java Parser에 별도 Framework Analyzer를 추가하는 구조가 적절하다.

```text
Java Parser
    +
Spring Analyzer
    +
eGovFramework Analyzer
```

특정 eGov Naming/Structure를 인식하여 Project Summary에 반영한다.

---

# 104. React/Vue 지원

JavaScript/TypeScript Symbol 외에 다음을 분석한다.

```text
Component

Props

Hooks

State

Routes

API Calls

Imports
```

Vue:

```text
SFC
script
template
style
defineProps
defineEmits
```

---

# 105. Node.js 지원

추가 분석:

```text
Express Route

Middleware

Module Export

Package Dependency
```

---

# 106. Python 지원

추출:

```text
Module

Class

Function

Decorator

Import

Function Calls
```

Framework 확장:

```text
FastAPI
Flask
Django
```

등을 향후 Adapter로 지원 가능하다.

---

# 107. 추천 구현 모듈

실제 코드 프로젝트 구조 예:

```text
local-agent/

├─ core/
│  ├─ tool-registry
│  ├─ dispatcher
│  └─ execution-context
│
├─ connector/
│  ├─ server
│  └─ protocol
│
├─ ide/
│  └─ bridge
│
├─ project/
│  ├─ scanner
│  ├─ parser
│  ├─ analyzer
│  ├─ indexer
│  └─ sync
│
├─ workspace/
│  ├─ reader
│  ├─ search
│  ├─ diff
│  └─ patch
│
├─ execution/
│  ├─ build
│  ├─ test
│  ├─ git
│  └─ shell
│
├─ policy/
│  ├─ risk
│  ├─ approval
│  └─ secret
│
└─ storage/
   ├─ project-index
   └─ artifacts
```

---

# 108. Local Agent MVP

초기 MVP에서는 다음 기능을 먼저 구현한다.

## Core

```text
Server Connection

Tool Registry

Tool Dispatch

Policy Check
```

## Project Intelligence

```text
Workspace Scan

Project Detection

File List

Java/Python/JS/TS Symbol Extraction

PROJECT_INDEX 생성

Incremental File Update
```

## Workspace

```text
read_file

read_files

search

propose_changes

apply_changes

diff
```

## Execution

```text
build.run

test.run

git.status

git.diff

git.commit
```

## Human

```text
IDE Diff UI 연동

Approval 연동

Question UI 연동
```

---

# 109. MVP에서 후순위로 둘 기능

```text
Call Graph 정밀 분석

Symbol Reference 전체 추적

고급 Semantic Summary

부분 Change Set 승인

Git Rebase

Git Push

Destructive Shell

Full Sandbox

다중 Workspace 병렬 제어
```

---

# 110. Local Agent 핵심 보안 원칙

1. Server가 Local File System에 직접 접근하지 않는다.
2. 모든 Local 작업은 Tool Registry를 통한다.
3. Workspace Root 밖 파일 접근을 차단한다.
4. Source 변경은 승인된 Change Set만 적용한다.
5. File Hash로 stale change를 차단한다.
6. Shell보다 전용 Adapter를 우선한다.
7. 위험 Git 명령은 명시적 승인 없이는 실행하지 않는다.
8. 외부 인터넷 Package Repository 접근을 금지한다.
9. Secret 포함 Source의 중앙 전송을 정책으로 제어한다.
10. Local Policy는 Server 요청보다 우선한다.
11. 실행 결과는 구조화하여 Server로 반환한다.
12. 전체 로그는 필요 시에만 전달한다.
13. 모든 위험 작업은 Audit 가능해야 한다.

---

# 111. 핵심 설계 결정

onCode Local Agent Architecture v1의 핵심 결정은 다음과 같다.

1. IDE Plugin과 Local Agent Runtime을 분리한다.
2. IDE별 차이는 IDE Bridge에 한정한다.
3. Local Agent가 프로젝트 분석과 Project Index 생성을 담당한다.
4. 구조 정보는 Parser 기반으로 추출한다.
5. LLM은 의미 요약에 제한적으로 사용한다.
6. PROJECT_SUMMARY.md와 PROJECT_INDEX.json을 모두 생성한다.
7. File Watcher 기반 Incremental Indexing을 사용한다.
8. 서버는 Project Index를 통해 필요한 Source만 요청한다.
9. Source Proposal과 실제 Apply를 분리한다.
10. Diff는 현재 Workspace 기준으로 Local Agent가 생성한다.
11. Change Set Apply 직전 File Hash를 재검증한다.
12. Build/Test/Git은 전용 Adapter로 추상화한다.
13. Shell은 마지막 수단으로 사용한다.
14. Local Policy Engine은 Server 요청보다 우선한다.
15. Server 연결은 Local Agent가 시작하는 Persistent Outbound Connection을 기본으로 한다.
16. Project Revision과 File Hash로 Workspace 일관성을 관리한다.
17. Full Log와 대용량 Artifact는 Local Artifact Store에 두고 구조화 결과만 전달한다.
18. Local Agent의 기능은 Adapter/Plugin 구조로 확장 가능하게 한다.