# SPEC-13 onCode Audit / Logging / Observability Architecture v1

## 1. 목적

본 명세는 onCode의 감사(Audit), 애플리케이션 로그(Logging), 메트릭(Metrics), 트레이싱(Tracing), 운영 모니터링(Observability) 구조를 정의한다.

onCode는 중앙 서버, 다중 Agent, Local Agent, IDE Extension, Guide Repository, Nexus, Security Service, Build/Test Tool 등 여러 컴포넌트가 협력하는 구조이므로 단순 애플리케이션 로그만으로는 장애 원인과 사용자 행위를 추적하기 어렵다.

본 시스템의 목표는 다음과 같다.

1. 사용자의 요청부터 최종 코드 변경까지 전체 흐름을 추적한다.
2. Workflow / Agent / Tool / Approval 실행 관계를 상호 연결한다.
3. 보안 및 승인 관련 이벤트를 장기 보존 가능한 Audit로 관리한다.
4. 운영 로그와 감사 로그를 분리한다.
5. Source, Secret, Credential의 불필요한 로그 저장을 방지한다.
6. Agent 품질, 모델 성능, Tool 실패율을 정량화한다.
7. Build/Test 실패 원인과 재시도 과정을 추적한다.
8. Local Agent와 중앙 서버 간 연결 상태를 모니터링한다.
9. 장애 발생 시 Work Item 단위로 전체 실행 타임라인을 복원한다.
10. 폐쇄망 환경에서도 외부 SaaS 없이 내부 관측 체계를 구축한다.

---

# 2. 핵심 원칙

onCode의 관측 데이터는 다음 세 종류로 구분한다.

```text
Audit
→ 누가 무엇을 했는가

Log
→ 시스템 내부에서 무슨 일이 발생했는가

Metric / Trace
→ 시스템이 얼마나 잘 동작하고 있는가
```

세 데이터를 하나의 테이블이나 파일에 모두 저장하지 않는다.

---

# 3. 전체 구조

```text
┌──────────────────────── onCode Components ───────────────────────┐
│                                                                 │
│ API / Session                                                   │
│ Workflow Orchestrator                                           │
│ Agent Runtime                                                   │
│ Tool Gateway                                                    │
│ Context Service                                                 │
│ Guide Retrieval                                                 │
│ Dependency Service                                              │
│ Security Service                                                │
│ Local Agent                                                     │
│ IDE Extension                                                   │
│                                                                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
                    Telemetry SDK / Adapter
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
            Audit           Logs          Metrics/Trace
              │              │              │
              ▼              ▼              ▼
        Audit Store      Log Store      Observability Store
              │              │              │
              └──────────────┼──────────────┘
                             ▼
                    Operations Console
```

---

# 4. Audit와 Log 분리

Audit Log는 법적/보안/운영 추적을 위한 기록이다.

예:

```text
사용자 로그인

Design 승인

Code Change 승인

파일 변경 적용

Git Commit

Git Push

Security Policy Block

Project Access Denied
```

Application Log는 개발/운영 장애 분석을 위한 기록이다.

예:

```text
HTTP request failed

LLM timeout

Database connection reset

Parser failed

Nexus timeout
```

---

# 5. Audit Store 성격

Audit Store는 가능한 한:

```text
Append-only

Immutable

Long Retention

Restricted Write Access
```

특성을 갖도록 한다.

일반 Agent나 사용자에게 수정 권한을 제공하지 않는다.

---

# 6. Logging Store 성격

Application Log는:

```text
Short / Medium Retention

Searchable

Level-based

High Volume
```

운영 데이터를 저장한다.

Audit보다 보존 기간이 짧을 수 있다.

---

# 7. Metrics Store

Metric은 집계 가능한 숫자 데이터다.

예:

```text
workflow_duration_seconds

agent_execution_duration

tool_failure_count

build_success_rate

active_local_agents
```

---

# 8. Distributed Trace

onCode에서는 사용자 한 요청이 여러 컴포넌트를 통과한다.

예:

```text
IDE
 ↓
Local Agent
 ↓
Server API
 ↓
Workflow
 ↓
Design Agent
 ↓
Guide Search
 ↓
Implementation
 ↓
Local Tool
```

따라서 Trace ID를 공통으로 전달한다.

---

# 9. Correlation ID 계층

다음 ID를 공통 상관관계 식별자로 사용한다.

```text
trace_id

session_id

project_id

workspace_id

work_item_id

workflow_id

task_id

agent_execution_id

tool_call_id

execution_id

approval_id

change_set_id
```

---

# 10. trace_id

사용자의 단일 요청 흐름을 추적하는 최상위 기술 Trace ID다.

예:

```text
TRACE-01J...
```

한 Work Item에서 여러 사용자 후속 요청이 있다면 여러 Trace가 존재할 수 있다.

---

# 11. Work Item과 Trace 구분

```text
Work Item
→ 논리적 개발 작업

Trace
→ 특정 실행 흐름
```

예:

```text
WI-100
로그인 잠금 기능

Trace 1
→ 최초 설계

Trace 2
→ 사용자 설계 변경 요청

Trace 3
→ Build 실패 수정
```

---

# 12. Workflow Timeline

Work Item 상세 화면에서 다음을 복원할 수 있어야 한다.

```text
10:01 Request received
10:01 Requirement analysis
10:02 Guide search
10:02 Source read
10:03 Design generated
10:04 Design approved
10:05 Implementation generated
10:06 Review passed
10:06 Security passed
10:07 Code approved
10:07 Files applied
10:08 Build failed
10:09 Fix generated
10:10 Build succeeded
10:11 Tests passed
10:11 Completed
```

---

# 13. 공통 Event Envelope

모든 주요 Telemetry Event는 공통 구조를 사용한다.

예:

```json
{
  "event_id": "EVT-100",
  "event_type": "workflow.state.changed",
  "timestamp": "2026-09-09T10:10:00+09:00",

  "trace_id": "TRACE-100",
  "session_id": "SES-100",
  "project_id": "PRJ-100",
  "workspace_id": "WS-100",
  "work_item_id": "WI-100",
  "workflow_id": "WF-100",

  "actor": {
    "type": "AGENT",
    "id": "senior-developer-agent"
  },

  "data": {}
}
```

---

# 14. Event Category

권장 Category:

```text
AUTH

SESSION

PROJECT

WORKFLOW

AGENT

TOOL

APPROVAL

WORKSPACE

BUILD

TEST

GIT

GUIDE

DEPENDENCY

SECURITY

POLICY

SYSTEM
```

---

# 15. Event Severity

```text
DEBUG

INFO

WARN

ERROR

CRITICAL
```

Audit Event는 Severity와 별개로 기록 여부를 결정한다.

---

# 16. Audit Event

Audit는 최소 다음 필드를 가진다.

```text
audit_id

timestamp

user_id

session_id

project_id

workspace_id

work_item_id

workflow_id

actor_type

actor_id

action

resource_type

resource_id

result

reason_code

approval_id

policy_version

client_node
```

---

# 17. Audit Action 예

```text
AUTH_LOGIN

AUTH_LOGOUT

SESSION_REVOKE

PROJECT_ACCESS

WORKFLOW_CREATE

WORKFLOW_CANCEL

DESIGN_APPROVE

DESIGN_REJECT

CODE_CHANGE_APPROVE

CODE_CHANGE_REJECT

WORKSPACE_APPLY

GIT_COMMIT

GIT_PUSH

COMMAND_EXECUTE

DEPENDENCY_ADD

DEPENDENCY_BLOCK

SECURITY_BLOCK

GUIDE_PUBLISH
```

---

# 18. Audit Result

```text
SUCCESS

FAILED

DENIED

REJECTED

CANCELLED
```

---

# 19. Audit Reason Code

예:

```text
POLICY_DENIED

APPROVAL_REJECTED

ACCESS_DENIED

WORKSPACE_STALE

SECURITY_VIOLATION
```

자유 텍스트보다 Code를 우선한다.

---

# 20. Audit 최소화

Audit에는 다음을 기본적으로 저장하지 않는다.

```text
전체 Source

전체 Diff

전체 Prompt

Password

Token

Private Key

Full Build Log

Full Test Log
```

대신:

```text
hash

resource ID

file path

summary

artifact reference
```

를 기록한다.

---

# 21. Change Audit

Source 변경 Audit 예:

```json
{
  "action": "WORKSPACE_APPLY",
  "change_set_id": "CHG-100",
  "approval_id": "APR-200",
  "files": [
    {
      "path": "src/main/java/AuthService.java",
      "operation": "MODIFY",
      "before_hash": "sha256:...",
      "after_hash": "sha256:..."
    }
  ],
  "result": "SUCCESS"
}
```

---

# 22. Git Audit

Git Commit:

```text
repository

branch

commit_id

commit_message_hash

user

approval
```

를 기록한다.

Source 내용 전체는 기록하지 않는다.

---

# 23. Git Push Audit

```text
remote

branch

commit range

force 여부

approval_id
```

를 기록한다.

---

# 24. Approval Audit

Approval에는 다음을 기록한다.

```text
approval_id

approval_type

request_user

approved_by

resource_id

resource_hash

requested_at

responded_at

decision

reason
```

---

# 25. Security Audit

다음은 반드시 Audit한다.

```text
Policy Denied

Secret Detected

Vulnerable Dependency Blocked

External Repository Access Attempt

Workspace Boundary Violation

Unauthorized Tool Request
```

---

# 26. Authentication Audit

```text
Login Success

Login Failure

Logout

Session Expired

Session Revoked

Local Agent Registration

Local Agent Revocation
```

---

# 27. Admin Audit

Admin Console에서:

```text
Guide Publish

Policy Change

Role Change

Project Permission Change

Security Exception

Release Rollback
```

등을 기록한다.

---

# 28. Application Logging

공통 로그 포맷은 Structured JSON을 권장한다.

예:

```json
{
  "timestamp": "...",
  "level": "ERROR",
  "service": "agent-runtime",
  "logger": "ModelClient",
  "message": "Model request timed out",
  "trace_id": "TRACE-100",
  "work_item_id": "WI-100",
  "error_code": "MODEL_TIMEOUT"
}
```

---

# 29. Structured Log

단순 문자열:

```text
Build failed!!!
```

보다:

```json
{
  "event": "build.failed",
  "build_tool": "MAVEN",
  "exit_code": 1,
  "error_count": 3
}
```

형태를 우선한다.

---

# 30. Log Level 기준

## DEBUG

개발/상세 진단.

운영 기본 비활성 또는 제한.

## INFO

정상적인 주요 상태 변화.

## WARN

비정상 가능성이 있으나 처리 가능.

## ERROR

요청/작업 실패.

## CRITICAL

서비스 전체 또는 보안에 큰 영향.

---

# 31. DEBUG Log 주의

DEBUG에서도 Source 전체, Credential, Prompt 전체를 출력하지 않는다.

---

# 32. Prompt Logging

LLM Prompt는 민감한 Source와 사용자 데이터를 포함할 수 있다.

따라서 기본:

```text
prompt_id

prompt_template_version

input_context_refs

input_token_count
```

만 기록한다.

---

# 33. Agent Output Logging

Agent의 최종 구조화 Output 전체를 Log에 중복 저장하지 않는다.

Context Storage의 Output Reference를 기록한다.

예:

```text
output_ref =
ctx://work-items/WI-100/design/2
```

---

# 34. Chain-of-thought 저장 금지

다음은 Observability 시스템에서도 저장하지 않는다.

```text
private reasoning

hidden scratchpad

chain-of-thought
```

저장 대상:

```text
decision

result

finding

reason code

evidence reference
```

이다.

---

# 35. Agent Execution Metric

Agent별:

```text
agent_execution_count

agent_success_count

agent_failure_count

agent_retry_count

agent_duration_seconds

agent_input_tokens

agent_output_tokens
```

---

# 36. Agent Quality Metrics

예:

```text
Requirement revision count

Design rejection rate

Implementation review fail rate

Security review fail rate

Build fix loop count

Test fix loop count
```

---

# 37. Model Metrics

Inference Gateway:

```text
model_request_count

model_latency

time_to_first_token

input_tokens

output_tokens

timeout_count

error_count

queue_wait_time
```

---

# 38. Model별 분리

Label:

```text
model_id

model_profile

agent_type
```

을 사용한다.

---

# 39. Tool Metrics

Tool별:

```text
tool_call_count

tool_success_rate

tool_failure_rate

tool_duration

tool_timeout_count

tool_retry_count
```

---

# 40. Local Tool Metrics

특히:

```text
workspace.read

workspace.apply

build.run

test.run

git.*
```

를 별도로 모니터링한다.

---

# 41. Workflow Metrics

```text
workflow_created

workflow_completed

workflow_failed

workflow_cancelled

workflow_duration

workflow_wait_user_duration

workflow_wait_approval_duration
```

---

# 42. Workflow Stage Metrics

각 단계:

```text
DISCOVERING_CONTEXT

DESIGNING

IMPLEMENTING

REVIEWING

BUILDING

TESTING
```

의 평균 소요시간을 측정한다.

병목 분석에 유용하다.

---

# 43. Human Wait Metric

AI 처리 시간이 아니라 사용자 대기시간도 분리한다.

```text
approval_wait_time

user_question_wait_time
```

전체 Workflow Latency 분석 시 중요하다.

---

# 44. Build Metrics

```text
build_count

build_success

build_failure

build_duration

build_retry

build_failure_by_category
```

Category:

```text
CODE_ERROR

DEPENDENCY_ERROR

ENVIRONMENT_ERROR

CONFIGURATION_ERROR
```

---

# 45. Test Metrics

```text
test_run_count

test_pass_rate

test_failure_count

flaky_test_count

existing_failure_count

test_duration
```

---

# 46. Dependency Metrics

```text
dependency_search_count

dependency_added

dependency_reused

dependency_upgrade

dependency_blocked

vulnerability_found

license_blocked
```

---

# 47. Guide Metrics

```text
guide_search_count

guide_no_result_rate

guide_search_latency

guide_selected_count

mandatory_rule_hit

guide_conflict_count
```

---

# 48. Guide Adoption Rate

```text
selected guide sections
/
search results returned
```

등으로 계산할 수 있다.

검색 품질 개선에 사용한다.

---

# 49. Security Metrics

```text
policy_denied_count

secret_detected_count

workspace_violation_count

dependency_blocked_count

destructive_command_attempt_count

unauthorized_access_count
```

---

# 50. Session Metrics

```text
active_sessions

active_local_agents

offline_local_agents

reconnect_count

session_expiration_count

agent_version_distribution
```

---

# 51. Project Metrics

프로젝트별:

```text
active_work_items

workflow_success_rate

average_build_duration

test_success_rate

guide_usage

security_findings
```

---

# 52. IDE Metrics

개인 사용자 행동 Telemetry는 최소화한다.

운영에 필요한 범위:

```text
extension_version

connection_error

protocol_error

IDE type
```

정도만 수집한다.

키 입력이나 편집 행동 같은 불필요한 Telemetry는 수집하지 않는다.

---

# 53. Trace Span

대표 Trace:

```text
workflow
 ├─ requirement.agent
 ├─ project.search
 ├─ workspace.read_files
 ├─ guide.search
 ├─ design.agent
 ├─ approval.design
 ├─ implementation.agent
 ├─ review.agent
 ├─ workspace.apply
 ├─ build.run
 └─ test.run
```

---

# 54. Span 공통 필드

```text
trace_id

span_id

parent_span_id

operation

start_time

end_time

status

service

attributes
```

---

# 55. Trace Sampling

모든 Trace를 영구 저장할 필요는 없다.

권장:

```text
ERROR Trace
→ 100%

Security-related Trace
→ 100%

Normal Success Trace
→ sampling
```

Audit는 Sampling 대상이 아니다.

---

# 56. Error Trace

실패한 Workflow는 전체 Trace를 일정 기간 보관하는 것을 권장한다.

---

# 57. Artifact Reference

대용량 로그는 Artifact Store로 분리한다.

예:

```text
artifact://build/EXEC-100/full.log

artifact://test/EXEC-200/full.log
```

Observability에서는 Reference만 저장한다.

---

# 58. Build Log

Server로 전송:

```text
structured summary

important excerpts

full_log_ref
```

형태.

---

# 59. Test Log

동일:

```text
result summary

failure information

stack_ref

full_log_ref
```

---

# 60. Log Sanitization

모든 Log Pipeline 앞에서 Sanitizer를 적용한다.

```text
Application
  ↓
Sanitizer
  ↓
Log Sink
```

---

# 61. Sanitization 대상

```text
Password

Access Token

Authorization Header

Private Key

API Key

Nexus Credential

Database Credential

Sensitive Environment Variable
```

---

# 62. Source Sanitization

소스 파일 전체를 일반 로그에 출력하지 않는다.

필요하면:

```text
path

line

symbol

hash
```

만 남긴다.

---

# 63. Stack Trace

Stack Trace는 장애 분석에 필요하지만 내부 경로가 포함될 수 있다.

Local 절대경로는 가능하면 Workspace-relative path로 정규화한다.

---

# 64. Log Retention

예시:

```text
DEBUG
7 days

INFO
30 days

ERROR
90 days

Audit
1~5 years 또는 조직 정책
```

실제 기간은 보안/감사 정책으로 정한다.

---

# 65. Artifact Retention

Build/Test Full Log는 Audit보다 짧게 보관 가능.

예:

```text
Successful Build Log
30 days

Failed Build Log
90 days
```

정책화한다.

---

# 66. Audit Retention

Approval/Git/Security Audit은 장기 보존을 권장한다.

---

# 67. Audit Integrity

Audit Record에 Hash Chain을 적용할 수 있다.

예:

```text
record_hash =
hash(
 previous_record_hash
 + current_record
)
```

변조 탐지에 활용 가능.

---

# 68. Audit Signing

고보안 환경에서는 Audit Batch에 전자서명을 적용할 수 있다.

MVP 필수는 아니다.

---

# 69. Audit Store 권한

```text
Application
→ INSERT

Audit Viewer
→ SELECT

일반 Admin
→ UPDATE/DELETE 불가
```

구조 권장.

---

# 70. Log Store 권한

운영자에게 조회 권한을 제공하되 사용자 Source나 Secret이 없도록 Sanitization이 선행되어야 한다.

---

# 71. Audit Query

주요 조회 방식:

```text
User

Project

Work Item

Date

Action

Approval ID

Change Set

Git Commit

Security Event
```

---

# 72. Work Item Audit View

예:

```text
WI-100

User: hong
Project: customer-portal

09:10 Design approved
09:12 Code approved
09:13 AuthService.java modified
09:14 Build succeeded
09:15 Tests passed
```

---

# 73. User Audit View

특정 사용자가:

```text
어떤 프로젝트에서

어떤 작업을

어떤 시각에

승인했는지
```

조회 가능.

권한 있는 관리자에게만 제공한다.

---

# 74. Security Incident View

예:

```text
Policy Denied
External npm registry access

User:
USR-100

Project:
PRJ-300

Tool:
shell.execute

Command Hash:
...

Time:
...
```

---

# 75. Operations Console

운영 Console은 다음 화면을 제공할 수 있다.

```text
System Overview

Active Workflows

Failed Workflows

Agent Runtime

Model Metrics

Local Agent Status

Build/Test Metrics

Guide Metrics

Dependency/Security

Audit Search

Trace Search
```

---

# 76. System Overview

예:

```text
Active Users       34

Local Agents       28 / 30 online

Running Workflows  17

Failed Workflows   2

Model Queue        4

Build Queue        3
```

---

# 77. Workflow Dashboard

```text
Workflow

User

Project

State

Duration

Agent

Last Event
```

표시.

---

# 78. Workflow Drill-down

클릭:

```text
Timeline

Agent Calls

Tool Calls

Approvals

Build/Test

Errors

Context References
```

확인 가능.

---

# 79. Agent Dashboard

```text
Agent Type

Executions

Success %

Average Time

Retry %

Invalid Output %

Token Usage
```

---

# 80. Model Dashboard

```text
Model

Active Requests

Queue

Latency P50/P95/P99

Tokens/sec

Error %

GPU Utilization
```

Inference Engine metrics와 연계할 수 있다.

---

# 81. Local Agent Dashboard

```text
User

Device

Version

Status

Last Heartbeat

Active Workspace

Capabilities
```

---

# 82. Version Distribution

```text
Local Agent 1.4.0  80%

1.3.0              15%

1.2.0               5%
```

구버전 관리에 유용하다.

---

# 83. Build/Test Dashboard

프로젝트별:

```text
Build success rate

Average duration

Most common errors

Test pass rate

Flaky tests
```

---

# 84. Guide Dashboard

```text
Search Count

No Result

Most Used Guides

Mandatory Guide Miss

Conflict

Average Search Latency
```

---

# 85. Security Dashboard

```text
Policy Denials

Blocked Dependencies

Secret Detection

Workspace Violations

High/Critical Findings
```

---

# 86. Alerts

Alert 조건 예:

```text
Model Error Rate > threshold

Nexus Unavailable

Vulnerability DB Stale

Context DB Down

Audit Write Failure

Local Agent Offline Spike

Build Failure Spike

Guide Search No-result Spike
```

---

# 87. Audit Write Failure

Audit 기록 실패는 일반 Log 실패보다 더 심각하게 취급한다.

High Risk Tool의 경우 정책에 따라:

```text
Audit unavailable
→ execution deny
```

를 고려할 수 있다.

---

# 88. Fail-closed Audit

특히 다음 작업:

```text
Git Push

Destructive Command

Security Exception
```

은 Audit 저장이 불가능하면 실행하지 않는 정책을 사용할 수 있다.

---

# 89. Metric Label Cardinality

Metrics에 다음과 같은 고유 ID를 Label로 직접 넣지 않는 것을 권장한다.

```text
work_item_id

user_id

file_path
```

Cardinality가 너무 높아진다.

이 정보는 Trace/Log에서 조회한다.

---

# 90. 권장 Metric Label

낮은 cardinality:

```text
agent_type

tool_name

status

error_category

model_profile

IDE type

project_group
```

---

# 91. Performance Percentile

평균값뿐 아니라:

```text
P50

P90

P95

P99
```

를 측정한다.

특히 LLM/Build/Search Latency에 중요.

---

# 92. SLI

주요 Service Level Indicator:

```text
Request success rate

Workflow success rate

Guide search latency

Tool availability

LLM latency

Local Agent connectivity
```

---

# 93. SLO 예

초기 예시:

```text
Server API availability
99.5%

Guide Search P95
< 1 sec

Tool Gateway P95
< 500 ms

Local Agent reconnect success
> 99%
```

실제 환경에서 조정한다.

---

# 94. Workflow E2E Metric

사용자가 요청한 시점부터 시스템 처리가 끝나는 시점까지:

```text
workflow_total_duration
```

측정.

다만 사용자 승인 대기시간을 분리한다.

---

# 95. Active Processing Time

```text
total_duration
-
human_wait
```

를 계산하여 실제 시스템 처리시간을 분석한다.

---

# 96. Development Productivity Metrics

조직에서 필요할 수 있으나 주의해서 사용한다.

예:

```text
Work Items Completed

Build Fix Loops

Review Failure Rate
```

개인 개발자 평가 지표로 직접 사용하는 것은 별도 조직 정책이 필요하다.

---

# 97. Privacy 원칙

Observability는 운영 목적에 필요한 최소 정보만 수집한다.

다음은 기본 수집하지 않는다.

```text
Keystroke

Clipboard

전체 Editor 내용

개인 폴더 탐색

불필요한 IDE 행동
```

---

# 98. User Message Logging

원본 사용자 요청을 Application Log에 중복 저장하지 않는다.

Context Storage의 Message Reference를 사용한다.

Audit에는 요청 원문 대신 Request ID 또는 Summary 정도만 기록한다.

---

# 99. Search Query Logging

Guide Search나 Project Search Query는 운영 품질 분석에 유용하지만 민감한 정보가 포함될 수 있다.

정책에 따라:

```text
query_hash

normalized_terms

category
```

만 저장하거나 Sanitization 후 저장한다.

---

# 100. Database Slow Query

RDBMS 운영을 위해:

```text
slow query

connection pool

lock wait
```

Metrics/Logs를 수집한다.

SQL parameter 전체는 민감정보에 유의한다.

---

# 101. Context Storage Metric

```text
context_get_latency

context_put_latency

context_version_conflict

context_not_found
```

---

# 102. Project Search Metric

```text
project_search_latency

project_search_no_result

workspace_read_count

source_bytes_transferred
```

---

# 103. Source Transfer Metric

Local → Server 전송량을 측정한다.

```text
source_transfer_bytes
```

Context 최소화 효과를 확인할 수 있다.

---

# 104. Index Metrics

Local Project Intelligence:

```text
index_file_count

index_duration

incremental_update_duration

parser_failure_count

index_revision_gap

full_resync_count
```

---

# 105. Guide Ingestion Metrics

SPEC-09:

```text
document_upload_count

conversion_failure

docling_duration

section_count

manual_review_count

publish_count
```

---

# 106. Dependency Security Metrics

```text
nexus_search_latency

nexus_error

vulnerability_check_latency

security_db_age

dependency_resolution_failure
```

---

# 107. Health Check

각 서비스는:

```text
/health

/readiness

/liveness
```

개념을 제공하는 것이 좋다.

---

# 108. Readiness와 Liveness 구분

```text
Liveness
→ 프로세스가 살아 있는가

Readiness
→ 실제 요청을 처리할 수 있는가
```

예:

Context DB가 Down이면 Server 프로세스는 살아 있어도 Ready가 아니다.

---

# 109. Dependency Health

Readiness 검사에:

```text
Context DB

Workflow DB

Inference Gateway

Tool Gateway
```

등 핵심 의존성을 포함한다.

Nexus 장애는 Dependency 기능만 DEGRADED로 만들 수도 있다.

---

# 110. Degraded Mode

서비스 일부 장애 시 전체 onCode를 Down시키지 않을 수 있다.

예:

```text
Guide Search Down
→ Implementation policy에 따라 block/warn

Nexus Down
→ 신규 Dependency 작업만 block

Build unavailable
→ Local validation 단계 대기
```

---

# 111. Component Status

```text
HEALTHY

DEGRADED

UNAVAILABLE
```

공통 상태 사용.

---

# 112. Operational Event

예:

```text
system.component.degraded

system.component.recovered

system.capacity.warning
```

---

# 113. Capacity Metrics

```text
workflow_queue_depth

agent_queue_depth

model_queue_depth

build_queue_depth

active_process_count
```

---

# 114. Resource Metrics

Server:

```text
CPU

Memory

Disk

Network

DB Pool
```

Local Agent:

```text
CPU

Memory

running build/test count
```

민감하거나 불필요한 상세 시스템 정보는 중앙으로 과도하게 보내지 않는다.

---

# 115. GPU Metrics

Inference Server:

```text
GPU utilization

VRAM usage

KV cache

request concurrency

queue depth
```

추론 병목 분석에 사용.

---

# 116. Log Collection 구조

폐쇄망에서 다음 중 선택 가능.

```text
File + Central Collector

Fluent Bit

Vector

Logstash 계열

OpenTelemetry Collector
```

제품 종속성은 최소화한다.

---

# 117. Observability 표준

가능하면 OpenTelemetry 개념을 채택하는 것을 권장한다.

```text
Trace

Metric

Log correlation
```

을 표준화할 수 있다.

---

# 118. Backend 독립성

```text
Telemetry SDK
    ↓
Collector
    ↓
Backend
```

구조를 사용하면 특정 제품에 종속되지 않는다.

---

# 119. 폐쇄망 Backend 후보

조직 환경에 따라:

```text
Prometheus

Grafana

Loki

Tempo

OpenSearch

Elasticsearch compatible stack
```

등을 사용할 수 있다.

본 명세는 특정 제품을 강제하지 않는다.

---

# 120. RDBMS Audit

Audit Store는 PostgreSQL 같은 RDBMS로 시작해도 충분하다.

고성능 검색이 필요해지면 별도 Index/Search 시스템을 추가한다.

---

# 121. Audit Partition

대량 Audit를 위해 날짜 기준 Partition을 권장할 수 있다.

예:

```text
audit_event_2026_09
```

---

# 122. Log Rotation

Local Agent 로그는 Disk를 무한 사용하지 않도록:

```text
size based rotation

retention

compression
```

적용.

---

# 123. Local Agent Offline Buffer

Server와 연결이 끊겨도 Local Agent 운영 로그를 임시 Buffer할 수 있다.

재연결 후 필요 범위만 전송한다.

---

# 124. Audit Offline 처리

High Risk 실행은 Server 연결이 필요하므로 정상적으로는 중앙 Audit가 가능하다.

Read-only Local Event 정도만 Offline Buffer 대상이다.

---

# 125. Event Delivery

중요 Audit Event는 Best-effort가 아니라 durable하게 저장해야 한다.

가능하면 DB Transaction 또는 Durable Queue 사용.

---

# 126. Transaction Boundary

예:

```text
workspace.apply 성공
+
Audit 기록
```

을 최대한 일관되게 처리한다.

분산 시스템이라 완전한 단일 Transaction이 어려우면 Outbox Pattern을 고려한다.

---

# 127. Transactional Outbox

예:

```text
Business Transaction
   ↓
Result Table
+
Outbox Event
   ↓
Event Publisher
```

이벤트 유실을 줄인다.

---

# 128. Workflow Event Store

Workflow Transition은 운영 로그가 아니라 공식 상태 이력으로 저장한다.

```text
WORKFLOW_TRANSITION
```

과 Observability Event를 구분한다.

---

# 129. Official State vs Telemetry

```text
Context/Workflow DB
→ 공식 상태

Audit
→ 공식 행위 기록

Telemetry
→ 운영 관측
```

Telemetry 손실이 Workflow 상태 손실로 이어지면 안 된다.

---

# 130. Event Deduplication

Network Retry로 동일 Event가 두 번 전송될 수 있다.

```text
event_id
```

기반 deduplication을 지원한다.

---

# 131. Event Ordering

같은 Workflow에서:

```text
sequence
```

값을 둘 수 있다.

예:

```text
101
102
103
```

Timeline 정렬에 유용하다.

---

# 132. Clock Skew

Local Agent와 Server 시간 차이가 있을 수 있다.

각 Event에:

```text
event_time

received_time
```

을 둘 수 있다.

Audit 기준은 Server received time을 우선할 수 있다.

---

# 133. Error Classification

모든 오류는 공통 Category를 사용한다.

```text
AUTH

PERMISSION

PROJECT

WORKSPACE

AGENT

MODEL

TOOL

BUILD

TEST

GIT

GUIDE

DEPENDENCY

SECURITY

NETWORK

DATABASE

SYSTEM
```

---

# 134. Root Cause와 Symptom 분리

예:

```text
Build Failed
```

은 Symptom.

실제 Root Cause:

```text
DEPENDENCY_ERROR
```

일 수 있다.

Senior Developer 분석 결과를 별도 field로 기록한다.

---

# 135. Incident Correlation

같은 Nexus 장애로 100개의 Work Item이 실패해도 100개의 독립 문제로 보지 않도록 Component Error와 Workflow Error를 연결한다.

---

# 136. Incident ID

운영 장애 발생 시:

```text
INC-20260909-001
```

같은 Incident ID로 연관 Event를 묶을 수 있다.

---

# 137. Alert Deduplication

동일 Root Cause Alert를 반복 전송하지 않는다.

---

# 138. Recovery Event

장애 복구 시:

```text
system.component.recovered
```

Event를 생성한다.

---

# 139. SLA/SLO Report

주간/월간:

```text
Availability

P95 Latency

Failure Rate

Top Error

Security Incident

Capacity Trend
```

Report 생성 가능.

---

# 140. Agent Evaluation 데이터

운영 Metrics와 별도로 품질 평가용 Dataset을 만들 수 있다.

예:

```text
request

design version

implementation

review findings

build result

test result
```

단 Source와 사용자 데이터의 보존 정책을 별도로 고려한다.

---

# 141. Prompt/Model A-B 비교

폐쇄망 환경에서도 Agent Prompt나 Model 버전 비교를 할 수 있다.

Agent Execution에:

```text
agent_version

prompt_version

model_version
```

을 기록한다.

---

# 142. 품질 회귀 분석

예:

```text
implementation-agent v1.3

Review fail:
10%

v1.4

Review fail:
18%
```

처럼 회귀를 탐지할 수 있다.

---

# 143. Workflow Loop Metrics

```text
design_revision_count

implementation_retry_count

build_fix_count

test_fix_count
```

품질 및 비용 지표로 중요하다.

---

# 144. Token Cost 대신 Resource Cost

폐쇄망 자체 모델 환경에서는 외부 API 비용보다:

```text
GPU time

token throughput

queue time

VRAM utilization
```

이 운영 비용 지표가 된다.

---

# 145. Search Quality Metrics

Project Search:

```text
search → source request → actual modified file
```

관계를 분석하면 Project Intelligence 검색 품질을 평가할 수 있다.

---

# 146. Source Retrieval Efficiency

예:

```text
requested source files = 8

actually modified/referenced = 3
```

과도한 Context 요청을 개선할 수 있다.

---

# 147. Observability Access Control

모든 개발자가 전체 운영 로그를 볼 수 있게 하지 않는다.

Role 예:

```text
SYSTEM_ADMIN

OPERATIONS

SECURITY_ADMIN

PROJECT_LEAD
```

별 조회 범위를 적용한다.

---

# 148. Project-level Log Access

Project Lead는 자기 프로젝트 관련 정보만 조회.

System Admin은 시스템 전체 조회.

---

# 149. User Data Access

개별 사용자 작업 이력 조회는 감사 권한이 있는 사용자에게만 허용한다.

---

# 150. Security Audit Access

보안 Event는 일반 운영 로그보다 더 제한된 권한을 적용할 수 있다.

---

# 151. Export

감사 요청 시:

```text
CSV

JSON

PDF report
```

등으로 내보낼 수 있다.

Export 행위 자체도 Audit한다.

---

# 152. PII / Sensitive Field Mask

UI에서도:

```text
token

credential

secret

private path
```

등은 표시하지 않는다.

---

# 153. Data Classification

Observability 데이터도 분류한다.

예:

```text
PUBLIC_INTERNAL

INTERNAL

SENSITIVE

SECURITY
```

---

# 154. Retention Policy by Classification

```text
INTERNAL
30 days

SENSITIVE
90 days

SECURITY AUDIT
long term
```

같이 설정 가능.

---

# 155. Deletion / Purge

보존 기간이 지나면 자동 Purge한다.

Audit는 조직 정책에 따라 예외.

---

# 156. Backup

Audit Store와 핵심 Workflow DB는 백업 대상이다.

Application Debug Log는 반드시 백업할 필요는 없다.

---

# 157. Disaster Recovery

복구 우선순위:

```text
Workflow State

Context

Audit

Guide Release

Policy

Observability Metrics
```

Metric 데이터는 일부 유실 허용 가능.

---

# 158. 운영 설정

예:

```yaml
observability:

  logging:
    level: INFO
    retention_days: 30

  tracing:
    enabled: true
    success_sample_rate: 0.1
    error_sample_rate: 1.0

  audit:
    enabled: true
    fail_closed_high_risk: true

  metrics:
    enabled: true
```

---

# 159. Component별 Logger

```text
api

workflow

agent-runtime

tool-gateway

context

project-search

guide

dependency

security

local-agent
```

Logger namespace를 분리한다.

---

# 160. Error Code 표준화

SPEC-01의 Error Model을 모든 Log/Metric/Trace에서 재사용한다.

이를 통해:

```text
DEPENDENCY_NOT_FOUND

BUILD_FAILED

POLICY_DENIED
```

같은 오류를 시스템 전체에서 동일하게 집계한다.

---

# 161. Work Item 완료 Summary

Workflow 완료 시 운영용 Summary Event 생성:

```json
{
  "event": "workflow.completed",
  "work_item_id": "WI-100",
  "duration_ms": 48200,
  "agent_executions": 6,
  "tool_calls": 14,
  "build_attempts": 2,
  "test_attempts": 1,
  "design_revisions": 1
}
```

---

# 162. 실패 Summary

```json
{
  "event": "workflow.failed",
  "failure_stage": "BUILDING",
  "error_category": "ENVIRONMENT_ERROR",
  "retry_count": 3
}
```

---

# 163. User-facing Trace 제한

일반 사용자에게 내부 Trace 전체를 노출하지 않는다.

IDE에는:

```text
현재 단계

간단한 이유

실행 결과
```

만 보여준다.

---

# 164. 운영자 Detail

운영 Console에서만:

```text
Trace

Span

Tool

Agent

Internal Error
```

를 자세히 조회한다.

---

# 165. Local Agent 로그

Local Log 경로 예:

```text
~/.oncode/logs/
```

또는 OS 표준 Application Data 경로를 사용한다.

Project Workspace 내부 `.codegen`에 시스템 로그를 넣지 않는 것을 권장한다.

---

# 166. Project Artifact와 System Log 분리

```text
.codegen/
→ 프로젝트 Index/Change Artifact

Agent Application Data
→ Local Agent 운영 로그
```

분리한다.

---

# 167. Security of Log Files

Local 로그 파일 접근 권한은 현재 사용자로 제한한다.

---

# 168. Crash Dump

Crash Dump에 Source/Token이 포함될 수 있으므로 자동 중앙 전송하지 않는다.

관리자 선택 또는 Sanitization 후 수집한다.

---

# 169. Protocol Observability

Server ↔ Local Agent:

```text
connection count

message count

round trip latency

protocol error

reconnect
```

를 측정한다.

---

# 170. Tool Call Latency Breakdown

예:

```text
queue_wait

network

local_execution

response_parse
```

로 나누면 병목 분석이 쉽다.

---

# 171. Agent Latency Breakdown

```text
context_build

model_queue

model_generation

tool_wait

output_validation
```

분리 권장.

---

# 172. Context Builder Metrics

```text
context_token_count

source_file_count

guide_section_count

context_trim_count
```

Context Rot/과다 Context 분석에 유용하다.

---

# 173. Context Budget Warning

Agent Context가 설정한 Budget의 일정 비율을 초과하면 Warning Metric 발생.

---

# 174. Token Truncation

Context가 잘린 경우:

```text
context.truncated
```

Event 기록.

품질 저하 원인이 될 수 있다.

---

# 175. Retry Observability

모든 Retry에는:

```text
attempt

reason

backoff

final_result
```

를 기록한다.

---

# 176. Retry 폭주 탐지

특정 Tool/Agent가 반복 Retry되면 Alert.

---

# 177. Circuit Breaker

Nexus, LLM 등 외부 내부서비스가 지속 실패하면 Circuit Breaker를 사용할 수 있다.

상태:

```text
CLOSED

OPEN

HALF_OPEN
```

Metric으로 노출.

---

# 178. Queue Monitoring

Agent/Tool Job Queue에는:

```text
depth

oldest_wait

processing

failed
```

Metric 필요.

---

# 179. Backpressure

Queue가 너무 커지면 신규 Workflow 제한 또는 사용자에게 Busy 상태 안내.

---

# 180. Capacity Alert

예:

```text
model_queue_depth > 100

build_queue_depth > 30
```

Alert.

---

# 181. 운영 이벤트와 Workflow Event 연결

Workflow failure가 시스템 장애 때문이면:

```text
workflow_event.incident_id
```

로 운영 Incident와 연결한다.

---

# 182. Deployment Version

모든 Log/Trace에는 가능하면:

```text
service_version

build_version
```

을 포함한다.

배포 후 오류 증가 분석에 중요하다.

---

# 183. Config Version

중요 설정도 Version 관리:

```text
policy_version

ranking_version

prompt_version
```

가능한 범위에서 Trace에 기록한다.

---

# 184. Audit Schema 권장

```text
audit.audit_event

audit.security_event

audit.admin_event
```

로 세분화하거나 단일 Event Table + category로 구성 가능.

초기에는 단일 테이블이 단순하다.

---

# 185. Telemetry Schema 권장

```text
observability.event

observability.incident
```

등 최소 Entity만 RDBMS에 두고 대용량 Log/Metric은 전용 backend로 분리한다.

---

# 186. MVP 구현

초기에는 다음을 반드시 구현한다.

```text
Structured Application Log

Trace ID / Correlation ID

Workflow Timeline

Tool Call Logging

Agent Execution Logging

Approval Audit

Workspace Apply Audit

Git Audit

Security Audit

Build/Test Structured Result

Local Agent Heartbeat

Basic Metrics

Operations Dashboard

Log Sanitization
```

---

# 187. MVP Backend 단순화

초기에는 다음 정도로 시작 가능하다.

```text
PostgreSQL
→ Audit / Workflow Event

Application JSON Log
→ File + Central Collector

Prometheus-compatible Metrics

Grafana-compatible Dashboard
```

Trace Backend는 2차 도입 가능.

---

# 188. 2차 확장

```text
Distributed Tracing

OpenTelemetry Collector

Dedicated Log Search

Audit Hash Chain

Incident Management

Advanced Alerting

Agent Quality Analytics

Prompt/Model Comparison Dashboard
```

---

# 189. 권장 운영 화면 MVP

```text
System Status

Active Workflows

Failed Workflows

Local Agent Status

Model Status

Build/Test Status

Security Alerts

Audit Search
```

이 정도로 시작한다.

---

# 190. 핵심 설계 결정

onCode Audit / Logging / Observability Architecture v1의 핵심 결정은 다음과 같다.

1. Audit, Application Log, Metric, Trace를 서로 다른 목적의 데이터로 분리한다.
2. Work Item / Workflow / Tool / Agent / Approval ID를 공통 Correlation 체계로 사용한다.
3. 사용자 요청부터 최종 Build/Test까지 Workflow Timeline을 복원할 수 있어야 한다.
4. Audit Store는 Append-only 및 장기 보존을 기본으로 한다.
5. 일반 Log는 Structured JSON 형태를 권장한다.
6. Source 전체, Prompt 전체, Token, Credential은 일반 Log에 기록하지 않는다.
7. Agent private reasoning과 chain-of-thought은 어떠한 관측 저장소에도 기록하지 않는다.
8. 대용량 Build/Test Log는 Artifact Store에 두고 Reference만 저장한다.
9. 모든 Log Pipeline에 Secret Sanitization을 적용한다.
10. Design Approval, Code Approval, Source Apply, Git, Security Block을 반드시 Audit한다.
11. Agent/Tool/Workflow/Build/Test/Guide/Dependency/Security Metrics를 별도로 수집한다.
12. 사용자 승인 대기시간과 실제 시스템 처리시간을 분리해서 측정한다.
13. Local Agent 상태와 재접속을 운영 Metrics로 관리한다.
14. Agent Prompt/Model/Version별 품질과 실패율을 분석할 수 있어야 한다.
15. Error Code는 SPEC-01 공통 Error Model을 재사용한다.
16. Distributed Trace에서는 trace_id/span_id를 사용한다.
17. 정상 성공 Trace는 Sampling할 수 있지만 Error/Security Trace는 높은 보존율을 사용한다.
18. Audit는 Sampling하지 않는다.
19. Metric Label의 고유값 Cardinality를 제한한다.
20. Workflow 공식 상태는 Workflow DB에 저장하며 Telemetry를 공식 상태로 사용하지 않는다.
21. 중요 Event 유실 방지를 위해 Transactional Outbox를 고려한다.
22. Audit 기록 실패 시 High Risk Tool을 차단하는 Fail-closed 정책을 지원한다.
23. 폐쇄망 내부에서 OpenTelemetry 개념을 사용할 수 있도록 Backend 독립적으로 설계한다.
24. Project/사용자 정보 조회는 Role 기반으로 제한한다.
25. IDE 행동 Telemetry는 최소화하고 불필요한 사용자 행동 추적을 하지 않는다.
26. 장애는 Workflow 실패와 Component Incident를 연결해서 분석할 수 있어야 한다.
27. 서비스별 Version, Policy Version, Prompt Version 등을 가능한 범위에서 Trace에 남긴다.
28. MVP에서는 Audit + Structured Log + Basic Metrics + Workflow Dashboard를 우선 구현한다.