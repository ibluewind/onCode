# SPEC-12 onCode Authentication / Session / Authorization Architecture v1

## 1. 목적

본 명세는 onCode의 사용자 인증(Authentication), 세션 관리(Session), 권한 제어(Authorization), Local Agent 등록 및 재접속 구조를 정의한다.

onCode는 중앙 서버와 Local Agent가 협업하며, 여러 사용자·프로젝트·워크스페이스·Work Item이 동시에 존재할 수 있으므로 모든 요청에 명확한 Identity와 Scope가 필요하다.

핵심 목표는 다음과 같다.

1. 사용자를 중앙 서버에서 안전하게 인증한다.
2. IDE Extension이 Credential을 직접 장기 보관하지 않도록 한다.
3. Local Agent와 사용자 Session을 안전하게 연결한다.
4. Project / Workspace / Work Item 단위 접근 권한을 분리한다.
5. Agent와 Tool이 항상 명시적 사용자 Scope 아래 동작하도록 한다.
6. Session 재접속과 Workflow Resume를 지원한다.
7. 다른 사용자·다른 프로젝트의 Context가 섞이지 않도록 한다.
8. Local Agent와 Server 사이 상호 신뢰를 검증한다.
9. Role과 Project 권한에 따라 Guide/Tool/Approval 접근을 제한한다.
10. 인증·세션·권한 변경을 Audit 가능하게 한다.

---

# 2. 전체 구조

```text
Developer
   │
   ▼
IDE Extension
   │
   ▼
Local Agent
   │
   │ Authenticated Persistent Connection
   ▼
onCode Server
   │
   ├─ Authentication Service
   ├─ Session Service
   ├─ Authorization Service
   ├─ Project Access Service
   ├─ Agent Registration Service
   └─ Audit Service
```

---

# 3. 기본 원칙

## 3.1 인증은 중앙 서버에서 수행

사용자 Identity의 Source of Truth는 중앙 서버다.

```text
IDE
 ↓
Local Agent
 ↓
Server Authentication
```

IDE Extension 자체가 인증 체계를 별도로 갖지 않는다.

---

## 3.2 Local Agent는 사용자 Session과 연결

Local Agent는 단순한 익명 실행 데몬이 아니다.

다음과 연결되어야 한다.

```text
User
Session
Device/Agent
Workspace
Project
```

---

## 3.3 모든 Tool Call은 Scope를 가져야 함

최소 Scope:

```text
user_id

session_id

project_id

workspace_id

work_item_id
```

Tool 성격에 따라 일부는 optional일 수 있으나 Project 작업에서는 명시적으로 유지한다.

---

# 4. Identity Model

핵심 Entity:

```text
USER

ROLE

USER_ROLE

ORGANIZATION

PROJECT

PROJECT_MEMBER

LOCAL_AGENT

DEVICE

SESSION

WORKSPACE

AUTH_TOKEN

PERMISSION
```

---

# 5. USER

필드:

```text
user_id

login_id

display_name

organization_id

status

created_at

updated_at
```

Status:

```text
ACTIVE
LOCKED
DISABLED
EXPIRED
```

---

# 6. 인증 방식

폐쇄망 환경을 고려하여 다음 방식을 지원할 수 있다.

```text
LDAP

Active Directory

OIDC-compatible Internal IdP

SAML-based SSO

Local Account
```

초기 구현은 기존 조직 인증 체계를 재사용하는 것을 권장한다.

---

# 7. 인증 우선순위

권장:

```text
기존 사내 IdP / AD / LDAP
        ↓
onCode Adapter
        ↓
User Session
```

onCode 자체에서 별도 Password Store를 새로 운영하는 것은 가능하면 피한다.

---

# 8. Authentication Adapter

```text
Authentication Service
      ↓
Authentication Provider
```

구현 예:

```text
LdapAuthProvider

OidcAuthProvider

LocalAuthProvider
```

---

# 9. 로그인 Flow

```text
IDE
 ↓
Local Agent
 ↓
Login Request
 ↓
Server Authentication Service
 ↓
Identity Provider
 ↓
Success
 ↓
Server Session 생성
 ↓
Local Agent Session Bind
```

---

# 10. IDE와 Credential

IDE Extension은 가능하면 다음만 유지한다.

```text
session_id

local_agent_connection_info
```

장기 Access Token이나 Password를 직접 보관하지 않는다.

---

# 11. Local Secure Store

Credential이 필요한 경우 Local Agent의 Secure Store에 저장한다.

예:

```text
Windows Credential Manager

macOS Keychain

Linux Secret Service
```

OS별 Adapter를 둔다.

---

# 12. Session Token

인증 성공 후 서버는 Session Token을 발급한다.

필드 예:

```text
session_token

user_id

session_id

issued_at

expires_at

scope
```

---

# 13. Token 형식

폐쇄망이라고 해서 특정 Token 형식을 강제할 필요는 없다.

가능:

```text
Opaque Session Token

Signed JWT
```

권장:

내부 시스템이고 즉시 Revocation이 중요하다면 Opaque Token + Server-side Session Store도 적합하다.

---

# 14. Access Token과 Refresh

장시간 IDE Session을 고려하면:

```text
Access Token
+
Refresh Token
```

구조를 사용할 수 있다.

단 Refresh Token은 IDE가 아닌 Local Agent Secure Store에 둔다.

---

# 15. Session Entity

```text
session_id

user_id

local_agent_id

status

created_at

last_active_at

expires_at

terminated_at

termination_reason
```

---

# 16. Session Status

```text
ACTIVE

IDLE

DISCONNECTED

EXPIRED

REVOKED

TERMINATED
```

---

# 17. Session과 Connection 분리

중요한 원칙:

```text
Session
≠
Network Connection
```

WebSocket이 끊겨도 Session 자체는 일정 시간 유지할 수 있다.

---

# 18. Persistent Connection

Local Agent가 Server에 outbound persistent connection을 생성한다.

```text
Local Agent
    ↓ outbound
Server
```

Server가 개발자 PC에 inbound 접속하지 않는다.

---

# 19. Connection Binding

Persistent Connection은 특정:

```text
session_id

local_agent_id

user_id
```

와 연결한다.

---

# 20. Local Agent Registration

Local Agent 최초 실행 시 Server에 등록한다.

예:

```json
{
  "agent_version": "1.0.0",
  "protocol": "oncode-tool/1.0",
  "platform": {
    "os": "WINDOWS",
    "arch": "X86_64"
  }
}
```

Server가:

```text
local_agent_id
```

를 발급한다.

---

# 21. Local Agent Identity

필드:

```text
local_agent_id

device_id

user_id

agent_version

protocol_version

platform

registered_at

last_seen_at

status
```

---

# 22. Agent Registration 방식

권장:

```text
사용자 로그인
   ↓
Server에서 Registration Challenge
   ↓
Local Agent 등록
```

무인 자동 등록보다 사용자 인증 이후 등록을 권장한다.

---

# 23. Device Identity

동일 사용자가 여러 PC에서 사용할 수 있다.

예:

```text
USR-100
 ├─ DEV-001 회사 데스크톱
 └─ DEV-002 노트북
```

Device와 Local Agent를 분리하면 재설치/업데이트 처리에 유리하다.

---

# 24. Device Fingerprint

과도한 개인정보 수집 없이 다음 정도를 조합할 수 있다.

```text
Generated Device ID

OS

Agent install ID
```

MAC Address나 Hardware Serial을 필수로 사용할 필요는 없다.

---

# 25. Agent Re-registration

Agent 재설치나 초기화 시 새 local_agent_id를 발급할 수 있다.

과거 Audit 기록은 기존 ID를 유지한다.

---

# 26. Capability Binding

Local Agent Session에는 Capability 정보가 연결된다.

예:

```text
Java 17
Maven
Git
Node 22
Python 3.12
```

Server는 Session별 Capability를 기준으로 Tool 사용 가능 여부를 판단한다.

---

# 27. Project Authorization

사용자가 Project를 사용하려면 명시적 권한이 있어야 한다.

Entity:

```text
PROJECT_MEMBER
```

필드:

```text
project_id

user_id

role

status
```

---

# 28. Project Role

초기 Role 예:

```text
DEVELOPER

LEAD

REVIEWER

SECURITY_REVIEWER

PROJECT_ADMIN
```

---

# 29. Organization Role

조직 전역 Role:

```text
USER

GUIDE_ADMIN

SECURITY_ADMIN

SYSTEM_ADMIN
```

Project Role과 분리한다.

---

# 30. RBAC

권한 모델은 기본적으로 RBAC를 사용한다.

```text
Role
 ↓
Permission
```

예:

```text
DEVELOPER
→ project.read
→ code.request
→ build.request
→ test.request
```

---

# 31. Permission 예

```text
project.read

project.modify

workflow.create

workflow.cancel

guide.read

guide.publish

dependency.request

security.review

git.commit

git.push

approval.respond

project.admin
```

---

# 32. Tool 권한과 RBAC

Tool Gateway는 Agent Permission뿐 아니라 사용자 권한도 확인한다.

예:

```text
Agent permission = allow

User permission = deny

→ DENY
```

---

# 33. Authorization Guard

Tool 실행 전:

```text
Authentication valid

Session active

User active

Project permission

Workspace ownership

Workflow scope

Agent tool permission

Policy

Approval
```

을 순차 검증한다.

---

# 34. Project Scope

모든 Project Context 접근은:

```text
user can access project
```

조건을 만족해야 한다.

---

# 35. Workspace Model

Project와 Local Workspace는 구분한다.

```text
PROJECT
  ├─ WORKSPACE-1
  ├─ WORKSPACE-2
  └─ WORKSPACE-3
```

---

# 36. WORKSPACE

필드:

```text
workspace_id

project_id

local_agent_id

user_id

branch

fingerprint

status

last_revision
```

---

# 37. Workspace Ownership

기본적으로 Workspace는:

```text
user_id
+
local_agent_id
```

에 바인딩한다.

다른 사용자가 같은 workspace_id를 사용할 수 없다.

---

# 38. Workspace Registration

IDE에서 프로젝트를 열면:

```text
IDE
 ↓
Local Agent
 ↓
Project Detection
 ↓
Project Fingerprint
 ↓
Server Project Lookup
 ↓
Workspace Register
```

---

# 39. Project Fingerprint

논리 Project 식별:

```text
Git Remote Fingerprint

Repository Identity

Optional project metadata
```

로 구성한다.

로컬 절대경로는 Project ID 기준으로 사용하지 않는다.

---

# 40. Workspace Fingerprint

동일 Project의 로컬 Checkout 식별:

```text
project_id

local_agent_id

generated workspace UUID
```

정도로 충분하다.

---

# 41. Project 미등록

새 프로젝트라면:

```text
PROJECT_NOT_REGISTERED
```

상태.

권한에 따라:

```text
Register Project

Request Access

Deny
```

중 처리한다.

---

# 42. Project 자동 생성

개발자가 아무 프로젝트나 열었다고 중앙 Project를 자동 생성하는 것은 권장하지 않는다.

조직 정책에 따라 승인된 프로젝트만 등록한다.

---

# 43. Work Item Authorization

Work Item은 생성한 사용자와 Project Scope에 연결된다.

```text
work_item_id

user_id

project_id

workspace_id
```

---

# 44. Work Item Ownership

기본적으로 Work Item은 생성 사용자만 수정/승인 가능하다.

필요하면 Lead/Reviewer가 조회 또는 승인 가능하도록 확장한다.

---

# 45. Shared Work Item

향후 협업 기능에서는:

```text
owner_user_id

participants

reviewers
```

를 둘 수 있다.

MVP에서는 단일 사용자 중심으로 시작할 수 있다.

---

# 46. Approval Authorization

Approval Response는 아무 사용자가 할 수 없다.

Approval에는:

```text
requested_to_user

required_role
```

중 하나 또는 둘 다 명시한다.

---

# 47. Design Approval

기본:

```text
request owner
```

가 승인한다.

---

# 48. Security Approval

향후 조직 정책상:

```text
SECURITY_REVIEWER
```

Role만 승인 가능하도록 설정할 수 있다.

---

# 49. Approval Scope Validation

다음이 일치해야 한다.

```text
approval.user

approval.project

approval.work_item

approval.resource

session user
```

---

# 50. Impersonation 금지

Agent가 사용자 대신 Approval을 생성하거나 승인 결과를 위조할 수 없다.

Approval 응답은 IDE/Authorized User Interaction에서만 생성된다.

---

# 51. Session Context

Workflow 생성 시:

```text
session_id
```

를 기록하지만 Work Item 자체는 Session보다 오래 살 수 있다.

---

# 52. Session 종료 후 Work Item

IDE 종료:

```text
Session disconnected
```

되어도 Work Item은:

```text
PAUSED
WAITING_RECONNECT
```

상태로 유지 가능하다.

---

# 53. Resume Token

재접속 시 기존 Work Item을 이어가기 위해 별도 Resume Token보다 Server-side Session/Work Item ID를 사용하는 것이 단순하다.

사용자가 재인증 후 권한이 동일하면 Resume 가능.

---

# 54. Session Resume Flow

```text
IDE Restart
   ↓
Local Agent Start
   ↓
User Re-authentication or Refresh
   ↓
Server Session Re-established
   ↓
Workspace Register
   ↓
Pending Work Items 조회
   ↓
Resume
```

---

# 55. Resume 전 검증

반드시 확인:

```text
User 동일

Project 권한 유효

Workspace 동일

Project Revision

Pending Approval

Policy Version

Local Agent Capability
```

---

# 56. Workspace Changed During Disconnect

재접속 시:

```text
stored revision != current revision
```

이면:

```text
STALE_WORKSPACE
```

처리한다.

---

# 57. Session Expired During Approval

Approval 대기 중 Session이 만료되면:

```text
Approval remains pending
```

일 수 있으나 사용자 재인증 후에만 응답 가능하다.

---

# 58. Approval Expiration

Session 복구와 별개로 Approval 자체 만료시간을 적용한다.

Expired Approval은 재요청한다.

---

# 59. Local Agent Disconnect

Server:

```text
LOCAL_AGENT_UNAVAILABLE
```

상태로 표시.

Local Tool을 필요로 하는 Workflow는 대기한다.

---

# 60. Read-only Server Task

Local Agent가 없어도 가능한 작업:

```text
General QA

Guide Search

일부 Context 조회
```

는 계속 처리 가능할 수 있다.

Project Source 접근이 필요한 작업은 중단한다.

---

# 61. Heartbeat

Local Agent는 주기적으로:

```text
agent heartbeat
```

전송.

Server는:

```text
AVAILABLE
DEGRADED
OFFLINE
```

상태를 관리한다.

---

# 62. Session Idle Timeout

IDE가 장시간 비활성일 경우:

```text
IDLE
```

상태로 전환할 수 있다.

High Risk Tool 실행 시 재인증을 요구하는 정책도 가능하다.

---

# 63. Re-authentication

다음 작업에 선택적으로 재인증을 요구할 수 있다.

```text
Git Push

Destructive Operation

Security Exception

Project Admin
```

---

# 64. Step-up Authentication

평상시 Session은 유지하되 위험 작업에서 추가 인증을 요구하는 구조다.

MVP에서는 생략 가능하지만 Architecture는 고려한다.

---

# 65. Session Revocation

관리자가 Session을 강제 종료할 수 있다.

```text
REVOKED
```

Local Agent는 이후 Tool 실행을 중단한다.

---

# 66. User Disabled

사용자 Account가 Disabled되면 모든 Active Session을 즉시 Revocation한다.

---

# 67. Role 변경

사용자의 Role/Project Permission이 변경되면 다음 Tool Call부터 새 권한을 적용한다.

오래된 권한을 Session에 영구 Cache하지 않는다.

---

# 68. Authorization Cache

성능상 Cache 가능하지만 짧은 TTL을 사용한다.

중요 권한 변경 이벤트에서 Cache invalidate.

---

# 69. Policy와 Authorization 구분

```text
Authorization
→ 누가 할 수 있는가

Policy
→ 어떤 조건에서 허용되는가
```

예:

```text
사용자에게 git.push 권한 있음
+
Project Policy가 approval 요구

→ APPROVAL_REQUIRED
```

---

# 70. Authentication과 Agent Permission 구분

```text
User Authorization

Agent Permission

Tool Policy

Approval
```

모두 통과해야 한다.

---

# 71. Scope Propagation

Orchestrator가 Agent Task를 생성할 때 Scope를 함께 전달한다.

예:

```json
{
  "user_id": "USR-100",
  "project_id": "PRJ-100",
  "workspace_id": "WS-100",
  "work_item_id": "WI-100"
}
```

---

# 72. Agent가 Scope 변경 금지

Agent가 Tool 호출에서 임의로:

```text
project_id
```

를 다른 값으로 바꿀 수 없다.

Agent Runtime이 Task Scope를 강제한다.

---

# 73. Tool Gateway Scope Injection

더 안전한 방법:

Agent가 Scope를 직접 생성하지 않고 Agent Runtime/Tool Gateway가 자동 주입한다.

```text
Agent output
   ↓
Tool Gateway
   ↓
Task Scope injected
```

---

# 74. Context URI Access

예:

```text
ctx://work-items/WI-200/design/latest
```

요청 시 Context Service가 현재 Scope와 WI-200의 Project/User 관계를 검증한다.

---

# 75. Guide Access Scope

Project-specific Guide:

```text
PRJ-100 only
```

는 해당 Project 권한이 있는 사용자에게만 노출한다.

---

# 76. Confidential Guide

특정 Role만 접근 가능한 Guide도 지원 가능.

예:

```text
SECURITY_REVIEWER
```

전용 보안 가이드.

---

# 77. Dependency Access

Nexus Package 검색 자체는 일반 Developer에 허용할 수 있으나, 특정 Restricted Package는 Role/Policy로 제한할 수 있다.

---

# 78. Admin Console Authentication

Guide Admin, Policy Admin, System Admin Console도 동일 Authentication Service를 사용한다.

---

# 79. Admin Role

예:

```text
GUIDE_ADMIN

POLICY_ADMIN

SECURITY_ADMIN

SYSTEM_ADMIN
```

일반 Developer Role과 분리한다.

---

# 80. Least Privilege

관리자도 필요한 권한만 가진다.

예:

```text
GUIDE_ADMIN
→ Guide Publish 가능
→ System Policy 변경 불가
```

---

# 81. API Authorization

IDE/Local Agent뿐 아니라 모든 Server API에서 Authorization Filter를 공통 적용한다.

---

# 82. Service-to-Service 인증

Server 내부 서비스를 Microservice로 분리할 경우:

```text
Service Identity
+
mTLS
```

또는 내부 Token을 사용할 수 있다.

초기 Monolith 구조에서는 필수 아님.

---

# 83. Local Agent ↔ Server TLS

폐쇄망이라도 TLS를 적용하는 것을 권장한다.

내부 CA 기반 인증서 사용 가능.

---

# 84. Mutual TLS

보안 요구가 높은 환경에서는 Local Agent와 Server 간 mTLS를 사용할 수 있다.

구조:

```text
Server Certificate

+

Client Agent Certificate
```

---

# 85. Agent Certificate 발급

초기 Registration 과정에서 내부 PKI와 연계할 수 있다.

MVP에서는 Token 기반 인증 후 향후 mTLS로 확장 가능.

---

# 86. Certificate Rotation

mTLS 사용 시 만료/교체 정책을 별도 관리한다.

---

# 87. IDE ↔ Local Agent

Local IPC는:

```text
127.0.0.1 only
+
local session token
```

을 사용한다.

---

# 88. Local IPC Token

Local Agent 시작 시 랜덤 토큰 생성.

IDE Extension이 안전한 방식으로 전달받아 연결한다.

---

# 89. IPC Token Rotation

Local Agent 재시작 시 새 Token 생성 권장.

---

# 90. CSRF / Localhost 공격

Localhost HTTP/WebSocket이라고 해서 신뢰하지 않는다.

필수:

```text
Origin Check

Auth Token

Random Port 또는 Controlled Port

Loopback Binding
```

---

# 91. Multiple IDE Clients

Local Agent는 여러 IDE Client를 등록할 수 있다.

필드:

```text
ide_client_id

ide_type

workspace_id

connected_at
```

---

# 92. Active Client

질문/Approval은 해당 Work Item을 생성한 IDE Client로 우선 전달한다.

Client가 없으면 다른 동일 사용자 Client 또는 Pending 상태 유지.

---

# 93. Approval Delivery

Approval Event에는:

```text
target_user_id
target_ide_client_id
```

를 선택적으로 포함한다.

---

# 94. Notification 중복 방지

동일 사용자가 VS Code와 IntelliJ를 동시에 사용 중일 때 동일 Approval이 양쪽에서 중복 처리되지 않도록 Approval 상태는 중앙 기준으로 atomic 처리한다.

---

# 95. Concurrent Approval Response

두 Client가 동시에 응답하면 첫 성공 응답만 적용.

이후 응답:

```text
APPROVAL_ALREADY_RESOLVED
```

---

# 96. Optimistic Lock

Session, Approval, Workflow 상태 변경에는 revision 기반 optimistic locking 권장.

---

# 97. Authorization Decision

공통 결과:

```text
ALLOW

DENY

REAUTH_REQUIRED
```

Policy Layer는 이후:

```text
ALLOW

APPROVAL_REQUIRED

DENY
```

를 판단한다.

---

# 98. Error Code

권장:

```text
AUTHENTICATION_REQUIRED

AUTHENTICATION_FAILED

SESSION_EXPIRED

SESSION_REVOKED

USER_DISABLED

ACCESS_DENIED

PROJECT_ACCESS_DENIED

WORKSPACE_ACCESS_DENIED

WORK_ITEM_ACCESS_DENIED

ROLE_REQUIRED

LOCAL_AGENT_NOT_REGISTERED

LOCAL_AGENT_OFFLINE

DEVICE_NOT_TRUSTED

REAUTHENTICATION_REQUIRED

APPROVAL_NOT_AUTHORIZED
```

---

# 99. Audit Scope

다음은 반드시 Audit한다.

```text
Login Success

Login Failure

Logout

Session Creation

Session Revocation

Agent Registration

Workspace Registration

Project Access Denied

Role Change

Approval Authorization Failure

Admin Action
```

---

# 100. Login Failure Log

비밀번호 자체는 절대 기록하지 않는다.

기록:

```text
login_id

time

source device

failure code
```

---

# 101. Audit Correlation

Session ID와 User ID를 Tool/Audit Log 전체에서 사용한다.

```text
USR-100

SES-200

WI-300

CALL-400
```

연결 추적 가능.

---

# 102. Session Storage

초기에는 RDBMS 또는 Redis + DB 조합 사용 가능.

권장:

```text
Active Session
→ Redis or fast store

Audit / History
→ RDBMS
```

폐쇄망 환경에서 운영 단순성을 위해 RDBMS만으로 시작해도 된다.

---

# 103. Session Token Revocation

Opaque Token 방식은 Server-side Session 상태 확인으로 즉시 Revocation이 쉽다.

JWT 사용 시 denylist나 짧은 expiry가 필요하다.

---

# 104. Time Synchronization

Token Expiry와 Audit를 위해 Server/Local Agent 시간 동기화가 중요하다.

Server time을 기준으로 판단한다.

---

# 105. Project Permission 캐시

Local Agent에 프로젝트 권한을 장기 저장하지 않는다.

권한 판단은 Server가 수행한다.

---

# 106. Offline Authorization

Server 연결이 끊긴 상태에서 Local Agent가 High Risk Tool을 독자적으로 허용하지 않는다.

기본:

```text
Server disconnected
→ risky operation denied
```

---

# 107. Local-only Read

정책에 따라:

```text
workspace.read

git.status
```

같은 Local-only 기능은 Server 없이 허용할 수 있지만 MVP에서는 단순화를 위해 연결 필수로 둘 수 있다.

---

# 108. Session Migration

사용자가 PC A에서 시작한 Work Item을 PC B에서 이어갈 경우 Workspace가 다르므로 자동 Resume하지 않는다.

```text
Same Work Item
+
Different Workspace
```

는 Source Revision 검증 후 새 Workspace Context를 만들어야 한다.

---

# 109. Workspace Transfer

향후:

```text
Resume on another workspace
```

기능을 제공할 수 있으나 v1에서는 별도 Workflow로 처리한다.

---

# 110. Branch 변경

동일 Workspace에서 branch 변경 시:

```text
workspace revision changed
```

로 판단.

Pending Change Set/Approval이 있으면 stale 처리한다.

---

# 111. Session과 Branch

Session 자체는 Branch에 종속되지 않는다.

Workspace State가 Branch를 관리한다.

---

# 112. Project Archive

Project가 ARCHIVED 상태가 되면 신규 Work Item 생성 금지.

기존 Work Item은 정책에 따라 종료 또는 Read-only.

---

# 113. Project Lock

보안 사고 등으로 Project 접근을 일시 차단할 수 있다.

```text
PROJECT_LOCKED
```

Local Tool execution도 중지.

---

# 114. Maintenance Mode

System 전체 Maintenance:

```text
new workflow denied

existing risky execution paused
```

정책 가능.

---

# 115. User Logout

Logout 시:

```text
Session terminate

Local Agent token invalidate

Pending High Risk approval invalidation
```

을 권장한다.

---

# 116. IDE Close와 Logout 구분

IDE를 닫았다고 사용자 Logout으로 처리하지 않는다.

Local Agent Session은 계속 유지될 수 있다.

---

# 117. Local Agent Shutdown

Local Agent 종료 시 Server에 graceful disconnect Event 전송.

미전송 종료는 Heartbeat timeout으로 감지.

---

# 118. Local Agent Status

```text
REGISTERED

ONLINE

OFFLINE

REVOKED

UPGRADE_REQUIRED

BLOCKED
```

---

# 119. Version Compatibility

Server가 최소 Local Agent Version을 강제할 수 있다.

예:

```text
min_agent_version = 1.4.0
```

낮은 버전:

```text
UPGRADE_REQUIRED
```

---

# 120. Protocol Compatibility

`oncode-tool/1.x` 호환 여부 확인.

호환 불가:

```text
PROTOCOL_VERSION_UNSUPPORTED
```

---

# 121. Extension Compatibility

Local Agent가 IDE Extension Version도 확인할 수 있다.

보안 취약 버전은 차단 가능.

---

# 122. Session Security Context

Session에 다음 snapshot을 보관할 수 있다.

```text
roles

organization

auth_method

device trust

created_at
```

다만 권한 판단은 최신 Role 정보를 우선한다.

---

# 123. Device Trust

향후 Device 상태:

```text
TRUSTED

UNTRUSTED

REVOKED
```

를 관리할 수 있다.

MVP에서는 사용자 로그인 + Local Agent 등록만으로 시작 가능.

---

# 124. Concurrent Sessions

사용자별 여러 Session 허용 가능.

예:

```text
Desktop
Laptop
```

조직 정책에 따라 최대 수 제한 가능.

---

# 125. Session Takeover 방지

동일 Token을 다른 Device에서 사용하지 못하도록:

```text
session token
+
local_agent_id
```

에 바인딩할 수 있다.

---

# 126. Token Binding

Session Token이 특정 local_agent_id에만 유효하도록 한다.

복사된 Token 재사용 위험 감소.

---

# 127. Workspace Scope Binding

High Risk Approval은 특정:

```text
workspace_id
```

에도 바인딩한다.

다른 Checkout에서 재사용 금지.

---

# 128. Context Leakage 방지

Agent Runtime이 Prompt 구성 시 반드시:

```text
current work_item
+
current project
```

만 조회하도록 Context Builder가 Scope를 강제한다.

---

# 129. Search Scope

`project.search`도 현재 Project Catalog만 검색한다.

Cross-project Search는 별도 권한/Tool로 분리.

---

# 130. Guide Search Scope

Runtime Guide Retrieval은:

```text
Project Scope

Organization Scope

Global Scope
```

를 Authorization 기반으로 적용한다.

---

# 131. Admin Search

Guide Admin이 다른 프로젝트 Guide를 조회하는 것은 별도 Role 권한으로 처리한다.

---

# 132. Authentication Service 책임

담당:

```text
Login

Identity Provider 연계

Token 발급

Token 검증

Logout

Revocation
```

---

# 133. Session Service 책임

```text
Session Lifecycle

Heartbeat

Connection Binding

Reconnect

Idle/Expiry

Agent Session Mapping
```

---

# 134. Authorization Service 책임

```text
Role

Permission

Project Membership

Resource Scope

Authorization Decision
```

---

# 135. Project Access Service 책임

```text
Project Registration

Project Member

Workspace Registration

Project Status

Access Validation
```

---

# 136. Agent Registration Service 책임

```text
Local Agent 등록

Version 관리

Device Mapping

Status

Capability Binding
```

---

# 137. 권장 Server 구조

```text
auth/
├─ authentication
├─ session
├─ authorization
├─ project-access
├─ agent-registration
└─ audit
```

---

# 138. Authorization API 예

내부:

```text
authorize(
 user,
 permission,
 resource,
 context
)
```

결과:

```json
{
  "decision": "ALLOW",
  "reason": "PROJECT_DEVELOPER"
}
```

---

# 139. Tool Gateway Integration

```text
Tool Request
    ↓
Session Validation
    ↓
Authorization
    ↓
Agent Permission
    ↓
Policy
    ↓
Approval
    ↓
Execute
```

---

# 140. Context Service Integration

```text
context.get
    ↓
Session
    ↓
Project/Work Item Authorization
    ↓
Resolve
```

---

# 141. Workflow Creation

신규 Work Item:

```text
Authenticated user

+

Project access

+

Registered workspace
```

필수.

---

# 142. GENERAL_QA

Project가 필요 없는 일반 질문은:

```text
user + session
```

만으로 생성 가능하다.

---

# 143. Project-aware QA

프로젝트 설명/검색 요청은:

```text
project_id
workspace_id
```

필수.

---

# 144. Git Permission

예:

```text
DEVELOPER
→ git.commit

LEAD
→ git.push

```

처럼 조직 정책에 따라 세분화 가능.

---

# 145. Security Review Permission

일반 Developer가 Security Finding을 볼 수는 있지만 최종 Security Approval은:

```text
SECURITY_REVIEWER
```

Role로 제한할 수 있다.

---

# 146. Guide Publish Permission

```text
GUIDE_ADMIN
```

만 Publish 가능.

---

# 147. Policy Change Permission

```text
POLICY_ADMIN
```

전용.

---

# 148. Separation of Duties

보안 요구가 높은 환경에서는 다음 역할을 분리할 수 있다.

```text
Developer

Guide Admin

Security Reviewer

Policy Admin

System Admin
```

한 사용자가 모든 권한을 가지지 않도록 할 수 있다.

---

# 149. Emergency Access

Break-glass 계정은 일반 Developer Session과 분리한다.

사용 시 강한 Audit를 적용.

MVP에서는 제외 가능.

---

# 150. Authentication Failure 정책

반복 실패 시 IdP 정책을 따른다.

onCode 자체에서 별도 Password Lockout 정책을 중복 구현할 필요는 없다.

---

# 151. Account Deprovisioning

IdP에서 사용자 삭제/비활성화 시 onCode Session도 종료되어야 한다.

주기 Sync 또는 Login/Authorization 시 확인.

---

# 152. Role Sync

Role을 외부 Directory에서 가져올 수도 있고 onCode 내부 Project Membership으로 관리할 수도 있다.

권장:

```text
Organization Role
→ IdP/Directory

Project Role
→ onCode
```

---

# 153. Data Retention

Session History와 Login Audit은 조직 정책에 따라 장기 보존.

Token 자체는 만료 후 삭제 가능.

---

# 154. Sensitive Data

저장 금지 또는 보호 대상:

```text
Password

Raw Refresh Token

Private Key

Credential
```

Refresh Token은 암호화/OS Secure Store.

---

# 155. Database Encryption

민감 Token을 DB에 저장해야 한다면 application-level encryption 또는 DB encryption을 적용.

---

# 156. Log Masking

다음 값은 Log에서 Masking:

```text
Authorization header

Session Token

Refresh Token

Client Certificate Secret
```

---

# 157. Security Event

권장 Event:

```text
auth.login.success

auth.login.failed

auth.session.revoked

auth.access.denied

agent.registered

agent.revoked

workspace.registered

project.access.denied
```

---

# 158. IDE 사용자 표시

권한 오류는 내부 Error Code를 그대로 노출하지 않는다.

예:

```text
PROJECT_ACCESS_DENIED
```

→

```text
이 프로젝트에 대한 onCode 사용 권한이 없습니다.
```

---

# 159. Session UX

IDE Status:

```text
onCode: Signed in

onCode: Session expired

onCode: Re-authentication required

onCode: Local Agent disconnected
```

---

# 160. MVP 범위

초기 구현에서 필수:

```text
Central Login

Session Token

Local Agent Registration

Persistent Session Binding

Project Membership

Workspace Registration

Basic RBAC

Tool Authorization

Approval Authorization

Reconnect

Session Expiration

Audit

IDE ↔ Local Agent Local Token
```

---

# 161. 2차 확장

```text
mTLS

Device Trust

Step-up Authentication

Multi-level Approval

Cross-device Work Item Resume

Advanced Role Hierarchy

Break-glass

Certificate Rotation

Fine-grained ABAC
```

---

# 162. RBAC 이후 ABAC 확장

향후 조건 기반 권한이 필요하면:

```text
Role
+
Project
+
Environment
+
Risk
+
Time
```

등을 포함한 ABAC를 일부 도입할 수 있다.

MVP는 RBAC + Policy 구조가 적절하다.

---

# 163. 대표 로그인 흐름

```text
Developer
   ↓
IDE Extension
   ↓
Local Agent
   ↓
Authentication Request
   ↓
onCode Server
   ↓
Internal IdP
   ↓
Authentication Success
   ↓
Session Created
   ↓
Local Agent Bound
   ↓
IDE Connected
```

---

# 164. 대표 Project 작업 흐름

```text
User Request
   ↓
Session Validation
   ↓
Project Access
   ↓
Workspace Validation
   ↓
Work Item Create
   ↓
Workflow
```

---

# 165. Tool 실행 흐름

```text
Agent Tool Call
   ↓
Task Scope
   ↓
Session Validation
   ↓
User Authorization
   ↓
Agent Permission
   ↓
Project/Workspace Scope
   ↓
Policy
   ↓
Approval
   ↓
Execution
```

---

# 166. Reconnect 흐름

```text
Network Disconnect
   ↓
Session = DISCONNECTED
   ↓
Workflow Pause if local tool required
   ↓
Local Agent Reconnect
   ↓
Token Validation
   ↓
Agent Binding
   ↓
Workspace Revision Check
   ↓
Pending Approval Check
   ↓
Resume
```

---

# 167. Session Revocation 흐름

```text
Admin/User Logout
   ↓
Session REVOKED
   ↓
Persistent Connection close
   ↓
Pending Local Tool cancel
   ↓
High-risk Approval invalidated
```

---

# 168. 핵심 설계 결정

onCode Authentication / Session / Authorization Architecture v1의 핵심 결정은 다음과 같다.

1. 사용자 인증은 중앙 서버에서 수행한다.
2. 기존 사내 IdP/AD/LDAP/SSO를 우선 재사용한다.
3. IDE Extension은 Credential을 장기 저장하지 않는다.
4. Token과 Server 연결 관리는 Local Agent에 집중한다.
5. Session과 Network Connection을 분리한다.
6. Local Agent는 사용자 Session 및 Device와 명시적으로 바인딩한다.
7. Local Agent는 Server에 outbound persistent connection을 생성한다.
8. Project와 Local Workspace를 서로 다른 Entity로 관리한다.
9. Workspace는 User + Local Agent + Project Scope에 바인딩한다.
10. 모든 Project Tool Call에 User/Session/Project/Workspace/Work Item Scope를 적용한다.
11. Project Membership과 Role을 기반으로 기본 RBAC를 적용한다.
12. Organization Role과 Project Role을 분리한다.
13. Agent Permission과 User Authorization을 별도로 검증한다.
14. Tool Gateway에서 Session → Authorization → Agent Permission → Policy → Approval 순으로 검증한다.
15. Agent가 Tool Scope를 임의 변경하지 못하게 Agent Runtime이 Scope를 강제한다.
16. Context 및 Guide Retrieval에도 동일한 Authorization Scope를 적용한다.
17. Approval은 요청 대상 사용자/Role과 Resource Scope를 검증한다.
18. 동일 Approval의 중복 응답은 원자적으로 방지한다.
19. Session 종료와 IDE 종료를 구분한다.
20. Work Item은 Session보다 오래 유지될 수 있다.
21. 재접속 시 User/Project/Workspace/Revision/Policy/Capability를 다시 검증한다.
22. 다른 Workspace에서의 자동 Resume는 허용하지 않는다.
23. Server 연결이 없는 상태에서 High Risk Tool의 독립 실행을 금지한다.
24. localhost IDE IPC에도 인증 Token과 Origin 검증을 적용한다.
25. Token/Refresh Token/Credential을 Audit/Log에 남기지 않는다.
26. 폐쇄망에서도 TLS 사용을 기본 권장한다.
27. mTLS와 Device Trust는 후속 보안 확장으로 고려한다.
28. 관리 Role을 Developer Role과 분리한다.
29. 모든 Login/Session/Agent Registration/Access Denied를 Audit한다.
30. MVP에서는 RBAC + Policy 구조로 시작하고 필요 시 ABAC로 확장한다.