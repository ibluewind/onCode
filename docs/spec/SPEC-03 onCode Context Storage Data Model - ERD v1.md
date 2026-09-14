# SPEC-03 onCode Context Storage Data Model / ERD v1

## 1. 목적

본 명세는 onCode의 중앙 서버에서 사용하는 Context Storage의 논리 데이터 모델을 정의한다.

Context Storage는 단순한 대화 이력 저장소가 아니라 다음 정보를 관리하는 **Workflow 중심 상태 저장소**이다.

- 사용자 요청
- 프로젝트 정보
- 프로젝트 인덱스
- 요구사항
- 설계
- 사용자 결정
- 개발 가이드 참조
- Dependency 결정
- 코드 변경안
- Review 결과
- Security 결과
- Build/Test 결과
- Tool 호출 이력
- Workflow 상태
- Audit Event

핵심 목표는 다음과 같다.

1. Agent 간 직접 Context 전달 제거
2. Agent 간 결합도 최소화
3. Workflow 상태 복구 가능
4. 모든 결정과 변경 이력 추적 가능
5. Context Versioning 지원
6. Project Context와 Work Item Context 분리
7. 대용량 데이터를 별도 Artifact로 분리
8. 최소 필요 Context만 조회 가능하도록 구조화

---

# 2. Context Storage 기본 원칙

## 2.1 Single Source of Truth

Agent가 자체적으로 중요한 상태를 보유하지 않는다.

공식 상태는 Context Storage에 저장된 데이터만을 기준으로 한다.

```text
Agent Memory
   X

Context Storage
   O
```

예:

```text
Design Agent가 생성한 설계
        ↓
Context Storage
        ↓
Review Agent가 조회
```

---

## 2.2 Agent 간 직접 Context 전달 금지

다음 구조는 사용하지 않는다.

```text
Requirement Agent
      ↓ 전체 결과
Design Agent
      ↓ 전체 결과
Implementation Agent
```

대신:

```text
Requirement Agent
       ↓
Context Storage
       ↑
Design Agent
       ↓
Context Storage
       ↑
Implementation Agent
```

을 사용한다.

---

## 2.3 Immutable Versioning

중요 Context는 overwrite하지 않는다.

예:

```text
Design v1
Design v2
Design v3
```

최신 버전 참조:

```text
ctx://work-items/WI-100/design/latest
```

과거 버전:

```text
ctx://work-items/WI-100/design/1
ctx://work-items/WI-100/design/2
```

---

# 3. Context 영역 구분

Context Storage는 크게 네 영역으로 나눈다.

```text
1. Identity / Session Context

2. Project Context

3. Work Item / Workflow Context

4. Execution / Audit Context
```

---

# 4. 상위 Entity 관계

```text
USER
  │
  └── SESSION
         │
         └── PROJECT
                │
                ├── PROJECT_REVISION
                │      │
                │      ├── PROJECT_FILE
                │      └── PROJECT_SYMBOL
                │
                └── WORK_ITEM
                       │
                       └── WORKFLOW
                              │
                              ├── REQUIREMENT
                              ├── USER_DECISION
                              ├── GUIDE_REFERENCE
                              ├── DESIGN
                              ├── DEPENDENCY_DECISION
                              ├── CHANGE_SET
                              ├── CODE_REVIEW
                              ├── SECURITY_FINDING
                              ├── BUILD_RESULT
                              ├── TEST_RESULT
                              ├── TOOL_CALL
                              └── WORKFLOW_EVENT
```

---

# 5. USER

사용자를 나타낸다.

주요 필드:

```text
user_id
login_id
display_name
status
created_at
updated_at
```

예:

```json
{
  "user_id": "USR-100",
  "login_id": "honggildong",
  "display_name": "홍길동",
  "status": "ACTIVE"
}
```

민감정보는 Context Storage의 일반 Entity에 저장하지 않는 것을 권장한다.

인증정보는 별도 Identity Store에서 관리한다.

---

# 6. SESSION

사용자의 onCode 접속 세션을 나타낸다.

```text
session_id
user_id
client_type
ide_type
local_agent_id
started_at
last_active_at
status
```

IDE 예:

```text
ECLIPSE
VSCODE
INTELLIJ
```

---

# 7. PROJECT

지속적인 프로젝트 단위이다.

`WORK_ITEM`보다 수명이 길다.

필드:

```text
project_id
name
workspace_fingerprint
repository_type
repository_url_or_id
default_branch
status
created_at
updated_at
```

예:

```json
{
  "project_id": "PRJ-100",
  "name": "customer-portal",
  "repository_type": "GIT",
  "default_branch": "main"
}
```

---

# 8. PROJECT_PROFILE

프로젝트 기술 스택 정보를 저장한다.

```text
project_id
profile_version
languages
frameworks
build_tools
package_managers
runtime_versions
root_modules
generated_at
```

예:

```json
{
  "languages": [
    "java",
    "typescript"
  ],
  "frameworks": [
    "spring-boot",
    "react"
  ],
  "build_tools": [
    "maven",
    "npm"
  ],
  "runtime_versions": {
    "java": "17",
    "node": "22"
  }
}
```

---

# 9. PROJECT_REVISION

Local Project Intelligence에서 관리하는 프로젝트 인덱스의 revision이다.

필드:

```text
project_revision_id
project_id
revision
base_revision
source
generated_at
sync_status
```

예:

```text
revision = 183
```

Server가 설계를 시작할 때 사용한 revision을 Work Item에 기록한다.

---

# 10. PROJECT_FILE

Project Index의 파일 정보를 저장한다.

필드:

```text
project_file_id
project_id
project_revision
path
language
file_type
package_name
module_name
hash
summary
size
last_modified_at
is_deleted
```

예:

```json
{
  "path": "src/main/java/com/example/AuthService.java",
  "language": "java",
  "hash": "sha256:...",
  "summary": "사용자 인증과 로그인 처리를 담당한다."
}
```

---

# 11. PROJECT_SYMBOL

파일 내부의 구조화된 Symbol을 저장한다.

지원 type:

```text
CLASS
INTERFACE
ENUM
METHOD
FUNCTION
FIELD
PROPERTY
CONSTRUCTOR
MODULE
```

필드:

```text
symbol_id
project_file_id
symbol_type
name
signature
visibility
line_start
line_end
summary
parent_symbol_id
```

예:

```json
{
  "symbol_type": "METHOD",
  "name": "login",
  "signature": "login(LoginRequest request): LoginResponse",
  "line_start": 42,
  "line_end": 76,
  "summary": "사용자 인증을 처리한다."
}
```

---

# 12. PROJECT_RELATION

Symbol 및 파일 간 관계를 저장한다.

지원 relation:

```text
IMPORTS
CALLS
USES
EXTENDS
IMPLEMENTS
READS
WRITES
DEPENDS_ON
REFERENCED_BY
```

필드:

```text
relation_id
project_id
source_type
source_id
target_type
target_id
relation_type
confidence
project_revision
```

이를 이용해:

```text
LoginController
      ↓ calls
AuthService
      ↓ uses
UserRepository
```

구조를 표현할 수 있다.

---

# 13. WORK_ITEM

사용자의 개별 요청 단위이다.

예:

```text
"로그인 실패 5회 시 계정을 잠그도록 수정해줘."
```

필드:

```text
work_item_id
project_id
session_id
user_id
title
original_request
normalized_request
intent
work_item_type
priority
status
base_project_revision
created_at
updated_at
completed_at
```

---

# 14. WORK_ITEM TYPE

```text
GENERAL_QA
ANALYSIS
CODE_REVIEW
IMPLEMENTATION
MODIFICATION
FIX
TEST
BUILD
GIT
SECURITY_REVIEW
DEPENDENCY_REVIEW
```

---

# 15. WORKFLOW

Work Item의 실행 상태를 관리한다.

필드:

```text
workflow_id
work_item_id
workflow_type
current_state
previous_state
resume_state
revision
status
retry_count
created_at
updated_at
completed_at
```

예:

```json
{
  "workflow_id": "WF-100",
  "workflow_type": "IMPLEMENTATION",
  "current_state": "IMPLEMENTING",
  "revision": 17
}
```

---

# 16. WORKFLOW_TRANSITION

상태 변경 이력을 별도 저장하는 것을 권장한다.

필드:

```text
transition_id
workflow_id
from_state
to_state
trigger_type
trigger_id
reason_code
reason
created_at
```

예:

```text
DESIGNING
→ WAITING_DESIGN_APPROVAL
```

---

# 17. REQUIREMENT

Requirement Agent가 구조화한 요구사항을 저장한다.

필드:

```text
requirement_id
work_item_id
version
status
goal
functional_requirements
non_functional_requirements
constraints
unknowns
assumptions
created_by
created_at
```

---

# 18. Requirement 예

```json
{
  "version": 2,
  "goal": "로그인 실패 5회 이상 시 계정 잠금",
  "functional_requirements": [
    "로그인 실패 횟수를 기록한다.",
    "5회 이상 실패하면 계정을 잠근다.",
    "로그인 성공 시 실패 횟수를 초기화한다."
  ],
  "constraints": [
    "기존 Repository 구조 유지"
  ],
  "unknowns": []
}
```

---

# 19. USER_DECISION

Senior Developer가 질문하고 사용자가 결정한 내용을 저장한다.

필드:

```text
decision_id
work_item_id
question_id
subject
question
decision_type
options
selected_value
reason
status
created_at
```

예:

```json
{
  "decision_id": "DEC-100",
  "subject": "account-lock-policy",
  "selected_value": "EXISTING_POLICY",
  "status": "CONFIRMED"
}
```

이 데이터는 이후 Agent가 반드시 준수해야 할 Constraint이다.

---

# 20. APPROVAL

Human-in-the-Loop 승인 상태를 저장한다.

필드:

```text
approval_id
work_item_id
workflow_id
approval_type
resource_type
resource_id
status
requested_at
responded_at
approved_by
decision
reason
expires_at
```

Approval Type:

```text
DESIGN
CODE_CHANGE
COMMAND
DEPENDENCY_INSTALL
GIT_COMMIT
GIT_PUSH
DESTRUCTIVE_OPERATION
```

---

# 21. GUIDE_REFERENCE

해당 Work Item에서 사용한 개발 가이드 정보를 저장한다.

필드:

```text
guide_reference_id
work_item_id
guide_id
guide_version
section_id
category
priority
purpose
retrieved_at
```

Purpose 예:

```text
DESIGN
IMPLEMENTATION
REVIEW
SECURITY
```

---

# 22. DESIGN

설계 결과를 version 단위로 저장한다.

필드:

```text
design_id
work_item_id
version
status
goal
scope
affected_files
components
data_flow
logic
exception_strategy
dependency_strategy
security_considerations
test_strategy
guide_references
base_project_revision
created_by
created_at
```

---

# 23. Design Status

```text
DRAFT
WAITING_APPROVAL
APPROVED
CHANGE_REQUESTED
REJECTED
SUPERSEDED
```

---

# 24. DEPENDENCY_DECISION

라이브러리 선정 결과를 저장한다.

필드:

```text
dependency_decision_id
work_item_id
ecosystem
package_name
current_version
selected_version
source
decision
reason
security_status
license_status
nexus_repository
created_at
```

Decision:

```text
USE_EXISTING
ADD
UPGRADE
DOWNGRADE
REPLACE
AVOID
CUSTOM_IMPLEMENTATION
```

---

# 25. Dependency Decision 예

```json
{
  "ecosystem": "MAVEN",
  "package_name": "org.example:example-lib",
  "selected_version": "2.1.4",
  "decision": "ADD",
  "security_status": "PASS",
  "license_status": "PASS"
}
```

---

# 26. CHANGE_SET

Implementation Agent가 생성한 코드 변경 단위이다.

파일 변경은 반드시 Change Set 단위로 관리한다.

필드:

```text
change_set_id
work_item_id
design_id
version
status
base_project_revision
created_by
created_at
approved_at
applied_at
```

Status:

```text
PROPOSED
REVIEW_FAILED
SECURITY_FAILED
WAITING_APPROVAL
APPROVED
REJECTED
STALE
APPLIED
FAILED
```

---

# 27. FILE_CHANGE

Change Set에 속하는 개별 파일 변경이다.

필드:

```text
file_change_id
change_set_id
path
operation
target_path
base_hash
result_hash
content_ref
diff_ref
status
```

Operation:

```text
CREATE
MODIFY
DELETE
RENAME
```

대용량 content는 DB에 직접 저장하지 않고 Artifact Store를 사용할 수 있다.

---

# 28. Artifact Reference

예:

```text
artifact://changes/CHG-100/AuthService.java
artifact://diffs/CHG-100/AuthService.diff
```

Context DB는 reference만 저장한다.

---

# 29. CODE_REVIEW

Review Agent의 검토 결과를 저장한다.

필드:

```text
code_review_id
work_item_id
change_set_id
reviewer
status
summary
created_at
```

Status:

```text
PASS
FAIL
WARNING
```

---

# 30. REVIEW_FINDING

Review 결과를 세부 항목으로 저장한다.

필드:

```text
finding_id
code_review_id
severity
category
file_path
symbol
line
rule_id
guide_reference_id
message
suggestion
status
```

Severity:

```text
INFO
LOW
MEDIUM
HIGH
CRITICAL
```

---

# 31. SECURITY_REVIEW

코드/Dependency 보안 검토의 상위 Entity이다.

필드:

```text
security_review_id
work_item_id
change_set_id
status
created_at
```

---

# 32. SECURITY_FINDING

필드:

```text
security_finding_id
security_review_id
finding_type
severity
file_path
line
package_name
package_version
vulnerability_id
rule_id
message
remediation
status
```

Finding Type:

```text
VULNERABILITY
SECRET
INJECTION
AUTHENTICATION
AUTHORIZATION
CRYPTOGRAPHY
PATH_TRAVERSAL
COMMAND_EXECUTION
DESERIALIZATION
LICENSE
```

---

# 33. BUILD_RESULT

Build Tool 실행 결과를 저장한다.

필드:

```text
build_result_id
work_item_id
change_set_id
execution_id
build_tool
status
exit_code
duration_ms
error_count
warning_count
log_ref
created_at
```

---

# 34. BUILD_ERROR

구조화된 Build 오류이다.

필드:

```text
build_error_id
build_result_id
error_type
file_path
line
column
message
category
```

Category:

```text
CODE_ERROR
DEPENDENCY_ERROR
CONFIGURATION_ERROR
ENVIRONMENT_ERROR
INFRASTRUCTURE_ERROR
```

---

# 35. TEST_RESULT

필드:

```text
test_result_id
work_item_id
change_set_id
execution_id
scope
status
total
passed
failed
skipped
duration_ms
log_ref
created_at
```

---

# 36. TEST_FAILURE

필드:

```text
test_failure_id
test_result_id
test_name
failure_type
file_path
line
message
stack_ref
```

Failure Type:

```text
IMPLEMENTATION_ERROR
TEST_ERROR
EXISTING_FAILURE
ENVIRONMENT_ERROR
FLAKY_TEST
```

---

# 37. TOOL_CALL

모든 Tool 호출을 구조적으로 기록한다.

필드:

```text
tool_call_id
message_id
session_id
project_id
workflow_id
work_item_id
task_id
actor_type
actor_id
tool_name
tool_version
risk_level
request_hash
status
approval_id
started_at
completed_at
duration_ms
execution_id
error_code
```

---

# 38. TOOL_CALL_PAYLOAD

Arguments 전체를 반드시 TOOL_CALL 테이블에 넣을 필요는 없다.

별도로 저장하거나 Artifact Store를 사용할 수 있다.

```text
tool_call_payload_id
tool_call_id
request_ref
response_ref
```

특히 소스코드 전체가 Audit DB에 중복 저장되지 않도록 한다.

---

# 39. WORKFLOW_EVENT

진행 상황 및 내부 이벤트를 저장한다.

필드:

```text
event_id
workflow_id
work_item_id
event_type
stage
severity
message
data_ref
created_at
```

예:

```text
workflow.started
workflow.state.changed
project.index.updated
approval.requested
tool.completed
```

---

# 40. AUDIT_LOG

보안 감사 및 운영 감사를 위한 별도 Log를 두는 것이 좋다.

필드:

```text
audit_id
user_id
session_id
project_id
work_item_id
workflow_id
actor_type
actor_id
action
resource_type
resource_id
result
client_ip_or_node_id
created_at
```

Audit Log에는 Secret 또는 전체 Source를 기록하지 않는다.

---

# 41. TASK

Workflow 내부에서 Agent/Tool 작업을 세분화할 필요가 있을 경우 사용한다.

필드:

```text
task_id
workflow_id
task_type
assigned_actor
status
attempt
started_at
completed_at
parent_task_id
```

Task 예:

```text
REQUIREMENT_ANALYSIS
PROJECT_SEARCH
GUIDE_SEARCH
DESIGN
IMPLEMENT
REVIEW
BUILD
TEST
```

---

# 42. AGENT_EXECUTION

LLM Agent 실행 자체를 별도 추적할 수도 있다.

필드:

```text
agent_execution_id
task_id
agent_type
model_id
prompt_template_version
input_context_refs
output_context_ref
status
started_at
completed_at
duration_ms
token_usage
```

폐쇄망 운영에서도 품질 및 성능 분석에 유용하다.

---

# 43. Context Reference Scheme

Agent가 DB ID 구조를 직접 알 필요가 없도록 Context URI를 사용한다.

예:

```text
ctx://projects/PRJ-100/profile

ctx://projects/PRJ-100/revisions/latest

ctx://work-items/WI-100/requirement/latest

ctx://work-items/WI-100/design/latest

ctx://work-items/WI-100/decisions

ctx://work-items/WI-100/guides

ctx://work-items/WI-100/dependencies

ctx://work-items/WI-100/change-sets/latest

ctx://work-items/WI-100/reviews/latest

ctx://work-items/WI-100/build/latest

ctx://work-items/WI-100/test/latest
```

---

# 44. context.get

Agent는 Context Storage를 SQL로 직접 조회하지 않는다.

예:

```json
{
  "tool": "context.get",
  "arguments": {
    "ref": "ctx://work-items/WI-100/design/latest"
  }
}
```

---

# 45. context.put

예:

```json
{
  "tool": "context.put",
  "arguments": {
    "ref_type": "DESIGN",
    "work_item_id": "WI-100",
    "data": {}
  }
}
```

Context Service가 version과 저장 방식을 결정한다.

---

# 46. context.query

여러 정보를 조합할 필요가 있는 경우 사용할 수 있다.

예:

```json
{
  "tool": "context.query",
  "arguments": {
    "work_item_id": "WI-100",
    "types": [
      "REQUIREMENT",
      "USER_DECISION",
      "DESIGN",
      "GUIDE_REFERENCE"
    ]
  }
}
```

---

# 47. Agent별 Context 접근 범위

## Requirement Agent

Read:

```text
Original Request
Project Profile
User Decisions
```

Write:

```text
REQUIREMENT
```

---

## Guide Agent

Read:

```text
REQUIREMENT
Project Profile
```

Write:

```text
GUIDE_REFERENCE
```

---

## Design Agent

Read:

```text
REQUIREMENT
USER_DECISION
PROJECT_FILE
PROJECT_SYMBOL
GUIDE_REFERENCE
```

Write:

```text
DESIGN
```

---

## Dependency Agent

Read:

```text
DESIGN
Project Profile
Current Dependencies
```

Write:

```text
DEPENDENCY_DECISION
```

---

## Implementation Agent

Read:

```text
REQUIREMENT
DESIGN
USER_DECISION
GUIDE_REFERENCE
DEPENDENCY_DECISION
Requested Source Files
```

Write:

```text
CHANGE_SET
FILE_CHANGE
```

---

## Review Agent

Read:

```text
REQUIREMENT
DESIGN
GUIDE_REFERENCE
CHANGE_SET
Relevant Source
```

Write:

```text
CODE_REVIEW
REVIEW_FINDING
```

---

## Security Agent

Read:

```text
CHANGE_SET
DEPENDENCY_DECISION
Relevant Source
```

Write:

```text
SECURITY_REVIEW
SECURITY_FINDING
```

---

## Senior Developer Agent

Read 권한은 비교적 넓게 허용한다.

주로 조회:

```text
REQUIREMENT
USER_DECISION
DESIGN
DEPENDENCY_DECISION
REVIEW
SECURITY_FINDING
BUILD_RESULT
TEST_RESULT
```

직접 대량 Source 저장은 하지 않는다.

---

# 48. Context Snapshot

LLM 호출 시 Context Storage의 데이터를 그대로 전부 Prompt에 넣지 않는다.

Context Builder가 필요한 정보만 조합한다.

```text
Context Storage
       ↓
Context Resolver
       ↓
Context Builder
       ↓
LLM
```

예:

Implementation Agent 입력:

```text
1. Requirement Summary
2. Approved User Decisions
3. Approved Design
4. Relevant Guide Sections
5. Selected Dependency
6. Requested Source Files
```

---

# 49. Context Budget

각 Agent별 Context Budget을 정의할 수 있다.

예:

```text
Requirement Agent
→ 프로젝트 Source 최소

Design Agent
→ 관련 파일 + 가이드 중심

Implementation Agent
→ 직접 수정/참조 파일 중심

Review Agent
→ Change Set + 관련 가이드
```

이 방식으로 Context Rot 및 불필요한 Token 사용을 줄인다.

---

# 50. Project Context와 Work Item Context 분리

매우 중요한 원칙이다.

## Project Context

지속적 정보:

```text
Project Profile
File Index
Symbols
Dependencies
Architecture Summary
```

## Work Item Context

일회성 작업 정보:

```text
Request
Requirement
Decision
Design
Change Set
Review
Build
Test
```

---

# 51. 사용자 채팅 이력

전체 채팅 이력 자체를 모든 Agent가 항상 읽게 하지 않는다.

대신:

```text
Conversation
      ↓
Relevant Message Extraction
      ↓
Requirement / Decision
```

형태로 구조화한다.

원본 대화는 별도 Conversation Store에 저장할 수 있다.

---

# 52. CONVERSATION_MESSAGE

필요한 경우:

```text
message_id
session_id
work_item_id
role
content_ref
created_at
```

다만 Agent 간 상태 공유에는 직접 사용하지 않는 것을 원칙으로 한다.

---

# 53. 대용량 데이터 처리

다음 정보는 RDBMS에 그대로 넣지 않는 것이 좋다.

```text
전체 Source File
전체 Build Log
전체 Test Log
대형 Diff
Stack Trace
첨부 파일
```

별도 Artifact Storage에 저장한다.

---

# 54. Artifact Store

폐쇄망 환경에서는 다음과 같은 구현이 가능하다.

```text
Shared File Storage

Object Storage compatible system

NFS

NAS

DB Large Object
```

Context Storage에는 Reference만 저장한다.

---

# 55. Artifact Entity

```text
artifact_id
artifact_type
storage_uri
content_hash
size
created_by
created_at
expires_at
```

Artifact Type:

```text
SOURCE
DIFF
BUILD_LOG
TEST_LOG
STACK_TRACE
AGENT_OUTPUT
PROJECT_INDEX
```

---

# 56. Artifact Immutable 정책

Build Log, Test Log, Change Set 등의 Artifact는 수정하지 않는 것을 권장한다.

새 실행은 새 Artifact를 만든다.

```text
BUILD-1
BUILD-2
BUILD-3
```

---

# 57. Entity Version 관리

Version이 필요한 Entity:

```text
PROJECT_PROFILE
REQUIREMENT
DESIGN
CHANGE_SET
PROJECT_REVISION
```

Version을 직접 숫자로 관리한다.

예:

```text
DESIGN
v1
v2
v3
```

---

# 58. Superseded 관계

새 버전이 생성되면 이전 버전은 삭제하지 않는다.

```text
DESIGN v1
status = SUPERSEDED

DESIGN v2
status = APPROVED
```

---

# 59. Optimistic Locking

중앙 서버의 Workflow와 Context 변경에는 optimistic lock을 사용하는 것을 권장한다.

예:

```text
workflow_revision = 17
```

업데이트:

```sql
UPDATE workflow
SET current_state = ...
    revision = 18
WHERE workflow_id = ?
AND revision = 17
```

0 row이면 concurrent modification으로 판단한다.

---

# 60. Project Revision과 Workflow Revision 구분

둘은 서로 다른 개념이다.

```text
Project Revision
→ Local Workspace 변화

Workflow Revision
→ 중앙 Workflow 상태 변화
```

예:

```text
Project Revision = 183
Workflow Revision = 17
```

---

# 61. Source Staleness 검증

Design에는:

```text
base_project_revision
```

Change Set에는:

```text
base_project_revision
base_file_hash
```

를 기록한다.

적용 전:

```text
현재 Project Revision
현재 File Hash
```

를 검증한다.

---

# 62. Data Retention

폐쇄망 및 감사 환경을 고려하여 Entity별 보존 정책을 분리한다.

장기 보존 후보:

```text
WORK_ITEM
DESIGN
USER_DECISION
APPROVAL
CHANGE_SET
AUDIT_LOG
SECURITY_FINDING
```

단기 또는 정책 기반:

```text
전체 Source Artifact
Build Log
Test Log
Agent Input Context Snapshot
```

---

# 63. Secret / Sensitive Data

Context Storage에는 다음을 평문으로 저장하지 않는다.

```text
Password
Access Token
Private Key
Credential
Secret
```

Local Agent의 Secret Scanner와 연계하여 Masking 처리한다.

예:

```text
Authorization: Bearer ****
```

---

# 64. Audit와 Context 분리

Context Storage와 Audit Store는 논리적으로 분리한다.

```text
Context Storage
→ 현재/과거 작업 상태

Audit Store
→ 누가 무엇을 했는가
```

한 DB에 구현하더라도 Schema를 분리하는 것이 좋다.

---

# 65. Project Intelligence Sync 구조

```text
Local Agent
   │
   │ PROJECT_INDEX revision 183
   ▼
Project Context Service
   │
   ├─ PROJECT_REVISION
   ├─ PROJECT_FILE
   ├─ PROJECT_SYMBOL
   └─ PROJECT_RELATION
```

파일 전체 Source는 Project Catalog에 저장하지 않는다.

---

# 66. Incremental Index Update

Local Agent:

```json
{
  "revision": 184,
  "previous_revision": 183,
  "changed": [
    "AuthService.java"
  ],
  "deleted": []
}
```

서버에서는 해당 파일과 Symbol만 갱신한다.

---

# 67. PROJECT_SUMMARY.md와 Context Storage

`PROJECT_SUMMARY.md`는 프로젝트 전체를 사람이 읽을 수 있는 형태로 제공한다.

하지만 중앙 Project Context의 공식 검색 데이터는 다음 구조화 데이터가 기준이다.

```text
PROJECT_PROFILE
PROJECT_FILE
PROJECT_SYMBOL
PROJECT_RELATION
```

즉:

```text
PROJECT_SUMMARY.md
→ Human / LLM Overview

PROJECT_INDEX.json
→ Sync Format

Project Context DB
→ Server-side Operational Index
```

으로 구분한다.

---

# 68. 검색 구조

Server Agent가:

```text
"로그인 처리 파일이 어디지?"
```

라고 판단하면 직접 DB query를 작성하지 않는다.

Tool:

```text
project.search
```

를 사용한다.

Project Context Service가:

```text
File summary
Symbol
Relation
Keyword
Category
```

기반으로 검색한다.

---

# 69. 개발 가이드 저장소와 Context Storage 분리

Guide Repository는 Context Storage의 일부가 아니다.

```text
Guide Repository
      ↓
guide.search
      ↓
GUIDE_REFERENCE
      ↓
Context Storage
```

Context Storage에는 실제 사용한 Guide Reference만 기록한다.

---

# 70. Nexus와 Context Storage 분리

Nexus 역시 독립 시스템이다.

```text
Nexus
   ↓
dependency.search
   ↓
Dependency Agent
   ↓
DEPENDENCY_DECISION
```

Context Storage에는 결정 결과만 저장한다.

---

# 71. Vulnerability DB와 Context Storage 분리

```text
Offline Vulnerability DB
          ↓
security.check_dependency
          ↓
SECURITY_FINDING
```

---

# 72. ERD 개념도

```text
USER
 │
 └── SESSION
       │
       └── PROJECT
             │
             ├── PROJECT_PROFILE
             │
             ├── PROJECT_REVISION
             │      │
             │      ├── PROJECT_FILE
             │      │      │
             │      │      └── PROJECT_SYMBOL
             │      │
             │      └── PROJECT_RELATION
             │
             └── WORK_ITEM
                    │
                    ├── WORKFLOW
                    │      │
                    │      ├── WORKFLOW_TRANSITION
                    │      ├── TASK
                    │      ├── TOOL_CALL
                    │      └── WORKFLOW_EVENT
                    │
                    ├── REQUIREMENT
                    │
                    ├── USER_DECISION
                    │
                    ├── APPROVAL
                    │
                    ├── GUIDE_REFERENCE
                    │
                    ├── DESIGN
                    │
                    ├── DEPENDENCY_DECISION
                    │
                    ├── CHANGE_SET
                    │      │
                    │      └── FILE_CHANGE
                    │
                    ├── CODE_REVIEW
                    │      │
                    │      └── REVIEW_FINDING
                    │
                    ├── SECURITY_REVIEW
                    │      │
                    │      └── SECURITY_FINDING
                    │
                    ├── BUILD_RESULT
                    │      │
                    │      └── BUILD_ERROR
                    │
                    └── TEST_RESULT
                           │
                           └── TEST_FAILURE
```

---

# 73. 물리 DB 권장 구분

초기에는 PostgreSQL 하나로 구현해도 충분하다.

Schema를 다음 정도로 구분하는 것을 권장한다.

```text
identity.*

project.*

workflow.*

context.*

execution.*

audit.*
```

예:

```text
project.project_file

workflow.work_item

context.design

execution.build_result

audit.audit_log
```

---

# 74. 검색용 RDBMS 활용

Project Catalog와 Guide Repository 모두 Vector DB 없이 구성한다면 PostgreSQL 기준으로 다음 기능을 활용할 수 있다.

```text
B-Tree Index

GIN Index

Full Text Search

JSONB

Trigram Search
```

Project Summary 검색에서도:

```text
file.summary
symbol.name
symbol.signature
symbol.summary
```

를 검색 대상에 포함한다.

---

# 75. JSONB 사용 범위

구조가 자주 변경되는 일부 데이터는 JSONB가 적합하다.

예:

```text
Requirement constraints

Design components

Tool arguments

User question options

Project runtime_versions
```

하지만 핵심 Relation까지 모두 JSONB로 넣지는 않는다.

---

# 76. 정규화 권장 대상

반드시 별도 테이블을 권장하는 데이터:

```text
PROJECT_FILE

PROJECT_SYMBOL

WORK_ITEM

WORKFLOW

USER_DECISION

APPROVAL

CHANGE_SET

BUILD_RESULT

TEST_RESULT

TOOL_CALL
```

검색, 참조, 감사가 많기 때문이다.

---

# 77. Context Service

Agent가 DB에 직접 접근하지 않도록 Server에 별도 Context Service를 둔다.

```text
Agent
  ↓
Context Tool
  ↓
Context Service
  ↓
Context DB
```

Context Service 역할:

```text
Reference Resolution

Version 관리

Authorization

Context Filtering

Data Masking

Artifact Resolution

Audit Logging
```

---

# 78. Project Context Service

Project Index는 일반 Context와 성격이 다르므로 별도 Service로 둘 수 있다.

```text
Project Intelligence
        ↓
Project Context Service
        ↓
Project Index DB
```

Tool:

```text
project.get_profile
project.search
project.get_symbols
project.get_dependencies
```

---

# 79. Context Resolver

`ctx://` URI를 실제 Entity로 변환한다.

예:

```text
ctx://work-items/WI-100/design/latest
```

↓

```text
DESIGN
work_item_id = WI-100
ORDER BY version DESC
LIMIT 1
```

Agent는 DB 구조를 몰라도 된다.

---

# 80. Agent Context Builder

LLM 호출 직전에 필요한 Context를 구성한다.

예:

```text
Implementation Agent

Requirement
    +
Approved Design
    +
User Decisions
    +
Relevant Guides
    +
Relevant Source
    +
Dependency Decisions
```

만 구성한다.

전체 Work Item 기록을 전부 전달하지 않는다.

---

# 81. Context Provenance

Agent가 생성한 결과에는 가능하면 입력 근거를 기록한다.

예:

```json
{
  "created_from": [
    "ctx://work-items/WI-100/requirement/2",
    "ctx://work-items/WI-100/design/3",
    "guide://GUIDE-87/account-lock",
    "source://AuthService.java@sha256:abc123"
  ]
}
```

이를 통해 결과의 근거 추적이 가능하다.

---

# 82. 설계 변경 추적

예:

```text
Design v1
    ↓
Developer Request Change
    ↓
Decision DEC-101
    ↓
Requirement v2
    ↓
Design v2
```

각 Entity 간 관계를 저장하면 반려 이유와 변경 이력을 추적할 수 있다.

---

# 83. Change Set Provenance

Change Set 역시 다음을 참조한다.

```text
Requirement Version

Design Version

Guide References

Dependency Decisions

Source File Hashes
```

이 정보는 향후 다음 질문에 답할 수 있게 한다.

```text
"이 코드가 왜 이렇게 수정되었는가?"
```

---

# 84. Context Lifecycle

대표 Lifecycle:

```text
User Request
    ↓
WORK_ITEM
    ↓
REQUIREMENT
    ↓
USER_DECISION
    ↓
GUIDE_REFERENCE
    ↓
DESIGN
    ↓
DEPENDENCY_DECISION
    ↓
CHANGE_SET
    ↓
CODE_REVIEW
    ↓
SECURITY_REVIEW
    ↓
APPROVAL
    ↓
BUILD_RESULT
    ↓
TEST_RESULT
    ↓
COMPLETED
```

---

# 85. 삭제 정책

Work Item 완료 후 즉시 Context를 삭제하지 않는다.

기본 상태:

```text
ACTIVE
COMPLETED
ARCHIVED
PURGED
```

보존 기간은 운영 정책으로 관리한다.

---

# 86. Context Storage에서 제외할 정보

다음 정보는 공식 Context로 사용하지 않는 것을 권장한다.

```text
Agent private reasoning

Chain-of-thought

미검증 임시 추론

모델 내부 scratchpad
```

저장 대상은 구조화된 결과와 근거만이어야 한다.

---

# 87. MVP Entity

초기 구현에서는 다음 Entity만으로 시작할 수 있다.

```text
USER
SESSION

PROJECT
PROJECT_PROFILE
PROJECT_REVISION
PROJECT_FILE
PROJECT_SYMBOL

WORK_ITEM
WORKFLOW
WORKFLOW_TRANSITION

REQUIREMENT
USER_DECISION
APPROVAL
GUIDE_REFERENCE
DESIGN

CHANGE_SET
FILE_CHANGE

CODE_REVIEW

BUILD_RESULT
BUILD_ERROR

TEST_RESULT
TEST_FAILURE

TOOL_CALL
WORKFLOW_EVENT
```

---

# 88. 2차 확장 Entity

```text
PROJECT_RELATION

DEPENDENCY_DECISION

SECURITY_REVIEW
SECURITY_FINDING

TASK
AGENT_EXECUTION

ARTIFACT

CONVERSATION_MESSAGE
```

---

# 89. 핵심 설계 결정

onCode Context Storage v1의 핵심 원칙은 다음과 같다.

1. Context Storage를 Agent 간 Single Source of Truth로 사용한다.
2. Agent 간 전체 Context 직접 전달을 금지한다.
3. Project Context와 Work Item Context를 분리한다.
4. 주요 결과는 Immutable Versioning을 적용한다.
5. Agent는 DB에 직접 접근하지 않고 Context Tool을 사용한다.
6. Project Source 전체는 중앙 DB에 상시 저장하지 않는다.
7. Project Index를 별도 구조화 데이터로 관리한다.
8. File Hash와 Project Revision으로 Source 일관성을 검증한다.
9. 대용량 Source/Log/Diff는 Artifact Store로 분리한다.
10. 사용자 Decision과 Approval을 공식 Context Entity로 관리한다.
11. 모든 Change Set은 Requirement와 Design의 Provenance를 보유한다.
12. Build/Test 결과는 구조화해서 저장한다.
13. Tool Call과 Workflow Transition을 추적한다.
14. Audit Store와 Context Store를 논리적으로 분리한다.
15. Agent Context Builder가 필요한 정보만 LLM에 제공한다.
16. 개발 가이드, Nexus, Vulnerability DB는 독립 시스템으로 유지한다.
17. Context에는 검증된 결과만 저장하고 private reasoning은 저장하지 않는다.

---

# 90. 다음 명세

다음 단계는 다음 두 가지 중 **SPEC-04 Local Agent Architecture**를 먼저 작성하는 것이 적절하다.

```text
SPEC-04
Local Agent Architecture
        ↓
SPEC-05
Project Intelligence / PROJECT_INDEX Schema
```

Local Agent 내부 구조를 먼저 정하면 Project Intelligence, Workspace Manager, Build/Test/Git Adapter, Policy Engine, IDE Bridge의 경계를 먼저 확정할 수 있고, 이후 PROJECT_INDEX 스키마를 실제 Parser 및 Indexer 구조에 맞춰 더 정확하게 설계할 수 있다.