# SPEC-05 onCode Project Intelligence / PROJECT_INDEX Schema v1

## 1. 목적

본 명세는 onCode Local Agent의 Project Intelligence 기능이 생성하는 프로젝트 인덱스 구조를 정의한다.

Project Intelligence의 목적은 다음과 같다.

1. 중앙 서버가 전체 소스를 받지 않고도 프로젝트 구조를 이해할 수 있도록 한다.
2. 필요한 파일과 Symbol을 빠르게 탐색할 수 있도록 한다.
3. 프로젝트 변경을 revision 단위로 추적한다.
4. 파일·클래스·함수·의존관계·프레임워크 구조를 구조화한다.
5. Server Agent가 `project.search`를 통해 필요한 Source 범위를 결정할 수 있도록 한다.
6. Incremental Indexing과 Delta Sync를 지원한다.
7. 사람과 LLM이 읽기 쉬운 `PROJECT_SUMMARY.md`와 시스템용 `PROJECT_INDEX.json`을 함께 제공한다.

---

# 2. 산출물

Local Agent는 기본적으로 다음 파일을 생성한다.

```text
.codegen/
├─ PROJECT_SUMMARY.md
├─ PROJECT_INDEX.json
└─ state/
   ├─ index-state.json
   └─ cache/
```

역할:

```text
PROJECT_SUMMARY.md
→ Human / LLM friendly overview

PROJECT_INDEX.json
→ Machine-readable canonical index

index-state.json
→ Incremental indexing 내부 상태
```

---

# 3. 핵심 원칙

## 3.1 구조 정보는 Parser 기반

다음 정보는 Parser/AST 기반으로 추출한다.

```text
package
module
class
interface
enum
method
function
field
property
constructor
import
extends
implements
annotation
signature
line range
```

LLM이 추측해서 생성하지 않는다.

---

## 3.2 의미 정보는 Summary Generator 사용

다음 정보는 LLM 또는 별도 요약 모델을 사용할 수 있다.

```text
file summary
class summary
method summary
module summary
relationship description
```

단, Parser 결과를 기반으로 생성해야 한다.

---

## 3.3 Source 전체를 Index에 저장하지 않음

`PROJECT_INDEX.json`에는 Source 전체를 넣지 않는다.

저장 대상:

```text
path
hash
metadata
summary
symbols
relations
dependencies
```

실제 Source는 필요할 때:

```text
workspace.read_file
workspace.read_files
workspace.read_symbol
```

로 요청한다.

---

# 4. PROJECT_INDEX 최상위 구조

```json
{
  "schema": "oncode-project-index/1.0",

  "project": {},
  "workspace": {},
  "revision": {},
  "modules": [],
  "files": [],
  "symbols": [],
  "relations": [],
  "dependencies": [],
  "frameworks": [],
  "statistics": {}
}
```

---

# 5. Schema Version

```json
{
  "schema": "oncode-project-index/1.0"
}
```

Version 정책:

```text
1.x
→ backward compatible

2.x
→ incompatible schema
```

---

# 6. project

프로젝트 자체의 논리적 정보를 나타낸다.

```json
{
  "project": {
    "project_id": "PRJ-100",
    "name": "customer-portal",

    "repository": {
      "type": "GIT",
      "default_branch": "main",
      "remote_fingerprint": "sha256:..."
    },

    "root_modules": [
      "backend",
      "frontend"
    ]
  }
}
```

로컬 절대경로는 중앙으로 전송하지 않는 것을 기본 원칙으로 한다.

---

# 7. workspace

개발자의 실제 Checkout 단위를 나타낸다.

```json
{
  "workspace": {
    "workspace_id": "WS-100",
    "branch": "feature/login-lock",

    "git": {
      "head": "9b2f7...",
      "dirty": true
    },

    "platform": {
      "os": "WINDOWS",
      "case_sensitive": false
    }
  }
}
```

---

# 8. revision

Project Index의 변경 버전을 나타낸다.

```json
{
  "revision": {
    "number": 184,
    "previous": 183,

    "generated_at": "2026-09-09T00:00:00+09:00",

    "type": "INCREMENTAL"
  }
}
```

Type:

```text
FULL
INCREMENTAL
BRANCH_CHANGE
REBUILD
```

---

# 9. Project Revision과 Git Revision 구분

둘은 다른 값이다.

```text
Project Index Revision
→ Local Agent Index 상태

Git HEAD
→ Repository commit 상태
```

예:

```text
index revision = 184
git head = abc123
```

Working Tree 변경 때문에 Git HEAD는 같아도 Index Revision은 증가할 수 있다.

---

# 10. modules

멀티모듈 프로젝트를 지원한다.

```json
{
  "modules": [
    {
      "module_id": "MOD-backend",
      "name": "backend",
      "path": "backend",

      "languages": [
        "java"
      ],

      "frameworks": [
        "spring-boot"
      ],

      "build_tools": [
        "maven"
      ]
    }
  ]
}
```

---

# 11. 멀티 프로젝트 예

```text
customer-portal/
├─ backend/
│  └─ Spring Boot
├─ frontend/
│  └─ React
└─ batch/
   └─ Python
```

Index:

```text
Project
 ├─ MOD-backend
 ├─ MOD-frontend
 └─ MOD-batch
```

---

# 12. files

파일별 메타데이터를 저장한다.

```json
{
  "files": [
    {
      "file_id": "FILE-1001",

      "module_id": "MOD-backend",

      "path": "backend/src/main/java/com/example/auth/AuthService.java",

      "language": "JAVA",

      "file_type": "SOURCE",

      "hash": "sha256:abc123",

      "size": 8432,

      "generated": false,

      "summary": "사용자 인증 및 로그인 상태 처리를 담당한다.",

      "symbol_ids": [
        "SYM-100",
        "SYM-101"
      ]
    }
  ]
}
```

---

# 13. file_type

```text
SOURCE
TEST
CONFIG
BUILD
RESOURCE
SCRIPT
DOCUMENT
GENERATED
OTHER
```

---

# 14. File Classification

예:

```text
*.java
→ SOURCE / TEST

pom.xml
→ BUILD

application.yml
→ CONFIG

package.json
→ BUILD

*.sql
→ RESOURCE 또는 SCRIPT

README.md
→ DOCUMENT
```

---

# 15. 테스트 파일 판별

언어/프레임워크 규칙을 함께 사용한다.

예:

```text
src/test/java/**
*_test.py
test_*.py
*.spec.ts
*.test.ts
```

---

# 16. config metadata

설정 파일은 전체 내용을 요약하지 않고 주요 구조를 추출할 수 있다.

예:

```json
{
  "config": {
    "type": "SPRING_APPLICATION",

    "profiles": [
      "default",
      "dev",
      "prod"
    ]
  }
}
```

Secret 값 자체는 Index에 저장하지 않는다.

---

# 17. symbols

Project Symbol을 독립 배열로 관리하는 것을 권장한다.

```json
{
  "symbols": [
    {
      "symbol_id": "SYM-100",

      "file_id": "FILE-1001",

      "type": "CLASS",

      "name": "AuthService",

      "qualified_name": "com.example.auth.AuthService",

      "visibility": "PUBLIC",

      "line_start": 18,
      "line_end": 142,

      "summary": "사용자 인증 비즈니스 로직을 담당한다."
    }
  ]
}
```

---

# 18. Symbol Type

공통 Type:

```text
MODULE

PACKAGE

CLASS
INTERFACE
ENUM
ANNOTATION

CONSTRUCTOR
METHOD
FUNCTION

FIELD
PROPERTY
CONSTANT

COMPONENT
ROUTE
ENDPOINT
```

언어/Framework별 확장 가능하다.

---

# 19. Method Symbol

```json
{
  "symbol_id": "SYM-101",

  "file_id": "FILE-1001",

  "parent_symbol_id": "SYM-100",

  "type": "METHOD",

  "name": "login",

  "qualified_name": "com.example.auth.AuthService.login",

  "signature": "login(LoginRequest request): LoginResponse",

  "visibility": "PUBLIC",

  "modifiers": [],

  "line_start": 42,
  "line_end": 76,

  "parameters": [
    {
      "name": "request",
      "type": "LoginRequest"
    }
  ],

  "return_type": "LoginResponse",

  "summary": "사용자의 인증 정보를 검증하고 로그인 결과를 반환한다."
}
```

---

# 20. Symbol Annotation

Java/Spring:

```json
{
  "annotations": [
    {
      "name": "Transactional"
    }
  ]
}
```

Controller:

```json
{
  "annotations": [
    {
      "name": "PostMapping",
      "arguments": [
        "/login"
      ]
    }
  ]
}
```

---

# 21. Framework-specific Metadata

공통 Symbol 외에 `framework_metadata`를 둘 수 있다.

예:

```json
{
  "framework_metadata": {
    "spring": {
      "stereotype": "SERVICE",
      "transactional": true
    }
  }
}
```

---

# 22. Spring Controller 예

```json
{
  "type": "METHOD",

  "name": "login",

  "framework_metadata": {
    "spring": {
      "endpoint": {
        "method": "POST",
        "path": "/api/login"
      }
    }
  }
}
```

---

# 23. eGovFramework 확장

eGovFramework 분석 시 별도 metadata를 추가할 수 있다.

```json
{
  "framework_metadata": {
    "egov": {
      "component_type": "SERVICE_IMPL",
      "naming_convention": "COMPLIANT"
    }
  }
}
```

초기에는 일반 Java/Spring 분석을 우선하고 eGov 전용 metadata를 단계적으로 확장한다.

---

# 24. Python Symbol

```json
{
  "type": "FUNCTION",

  "name": "authenticate_user",

  "signature": "authenticate_user(username: str, password: str) -> User",

  "decorators": [],

  "async": false,

  "summary": "사용자 정보를 검증한다."
}
```

---

# 25. React Component

```json
{
  "type": "COMPONENT",

  "name": "LoginForm",

  "framework_metadata": {
    "react": {
      "component_type": "FUNCTION",
      "hooks": [
        "useState",
        "useMutation"
      ]
    }
  }
}
```

---

# 26. Vue Component

```json
{
  "type": "COMPONENT",

  "name": "LoginForm",

  "framework_metadata": {
    "vue": {
      "sfc": true,
      "script_setup": true,

      "props": [
        "redirectUrl"
      ],

      "emits": [
        "success"
      ]
    }
  }
}
```

---

# 27. Node.js Route

```json
{
  "type": "ROUTE",

  "name": "POST /login",

  "framework_metadata": {
    "node": {
      "framework": "express",
      "method": "POST",
      "path": "/login"
    }
  }
}
```

---

# 28. relations

파일 및 Symbol 간 관계를 저장한다.

```json
{
  "relations": [
    {
      "relation_id": "REL-100",

      "type": "CALLS",

      "source": {
        "type": "SYMBOL",
        "id": "SYM-loginController"
      },

      "target": {
        "type": "SYMBOL",
        "id": "SYM-authService-login"
      },

      "confidence": 1.0
    }
  ]
}
```

---

# 29. Relation Type

```text
IMPORTS

USES

CALLS

EXTENDS

IMPLEMENTS

ANNOTATED_BY

READS

WRITES

DEPENDS_ON

ROUTES_TO

INJECTS

REFERENCES

TESTS
```

---

# 30. Relation Confidence

Parser로 확정된 관계:

```text
1.0
```

정적 분석상 추정:

```text
0.5 ~ 0.9
```

LLM 기반 추정 관계는 가능하면 기본 Index에 넣지 않는 것을 권장한다.

---

# 31. Dependency 관계 예

```text
LoginController.login
        ↓ CALLS
AuthService.login
        ↓ CALLS
UserRepository.findByUsername
```

이 관계를 이용해 Server가 구현 영향 범위를 탐색할 수 있다.

---

# 32. imports와 relations

단순 Import 목록은 파일 내부 metadata로도 저장 가능하다.

```json
{
  "imports": [
    "org.springframework.stereotype.Service"
  ]
}
```

프로젝트 내부 Symbol 관계는 `relations`에 별도 관리한다.

---

# 33. dependencies

외부 Package Dependency를 저장한다.

```json
{
  "dependencies": [
    {
      "dependency_id": "DEP-100",

      "module_id": "MOD-backend",

      "ecosystem": "MAVEN",

      "group": "org.springframework.boot",

      "name": "spring-boot-starter-security",

      "version": "3.5.0",

      "scope": "COMPILE",

      "direct": true,

      "source_file": "backend/pom.xml"
    }
  ]
}
```

---

# 34. Ecosystem

```text
MAVEN
GRADLE
NPM
PYPI
```

Gradle Java Dependency도 논리적으로 Maven coordinate로 정규화할 수 있다.

---

# 35. npm Dependency 예

```json
{
  "ecosystem": "NPM",

  "name": "axios",

  "version": "1.8.4",

  "scope": "DEPENDENCY",

  "direct": true
}
```

---

# 36. Python Dependency 예

```json
{
  "ecosystem": "PYPI",

  "name": "fastapi",

  "version_constraint": ">=0.115",

  "source_file": "requirements.txt"
}
```

Lock File이 존재할 경우 resolved version을 별도 저장할 수 있다.

---

# 37. dependency security metadata

Project Index에는 보안 판단 결과를 authoritative하게 저장하지 않는 것을 권장한다.

대신 optional cache 형태:

```json
{
  "security_hint": {
    "last_checked_at": "...",
    "status": "UNKNOWN"
  }
}
```

실제 판단은:

```text
security.check_dependency
```

결과가 기준이다.

---

# 38. frameworks

프로젝트에서 탐지된 Framework 정보를 저장한다.

```json
{
  "frameworks": [
    {
      "framework": "SPRING_BOOT",

      "module_id": "MOD-backend",

      "version": "3.5.0",

      "confidence": 1.0
    }
  ]
}
```

---

# 39. Framework 종류

초기 지원:

```text
SPRING
SPRING_BOOT
EGOVFRAMEWORK

REACT
VUE

NODEJS
EXPRESS

DJANGO
FLASK
FASTAPI
```

Python Web Framework는 향후 확장 가능하다.

---

# 40. Build Tool 정보

```json
{
  "build": {
    "tool": "MAVEN",

    "wrapper": true,

    "files": [
      "pom.xml"
    ]
  }
}
```

멀티모듈이면 Module별로 관리한다.

---

# 41. Runtime Version

탐지 가능한 경우:

```json
{
  "runtime": {
    "java": "17",
    "node": "22",
    "python": "3.12"
  }
}
```

이 값은 Project 설정과 Local Environment를 구분해야 한다.

예:

```text
required_java = 17

installed_java = 21
```

---

# 42. Project Configuration

Project에 선언된 요구 환경:

```json
{
  "requirements": {
    "java": "17",
    "node": ">=20"
  }
}
```

Local Agent Capability는 별도 관리한다.

---

# 43. statistics

Project Overview용 통계이다.

```json
{
  "statistics": {
    "files": 532,
    "source_files": 312,
    "test_files": 84,

    "symbols": {
      "classes": 97,
      "methods": 843,
      "functions": 132
    },

    "languages": {
      "JAVA": 210,
      "TYPESCRIPT": 89,
      "HTML": 13
    }
  }
}
```

---

# 44. PROJECT_SUMMARY.md 구조

권장 Markdown 구조:

```markdown
# Project Summary

## Project
## Technology Stack
## Modules
## Project Tree
## Architecture Overview
## Files
## Main Symbols
## Dependencies
## Framework Information
## Notes
```

---

# 45. Project Tree

전체 Tree가 너무 크면 모든 파일을 넣지 않는다.

권장 방식:

```text
src/
├─ main/
│  ├─ java/
│  │  └─ com/example/
│  │     ├─ controller/
│  │     ├─ service/
│  │     └─ repository/
│  └─ resources/
└─ test/
```

핵심 파일만 확장해서 표시한다.

---

# 46. File Summary Markdown 예

```markdown
## Files

### `src/main/java/com/example/auth/AuthService.java`

사용자 로그인과 인증 상태를 처리하는 Service.

Symbols:

- `AuthService`
- `login(LoginRequest): LoginResponse`
  - 사용자 인증 후 로그인 결과를 반환한다.
- `handleLoginFailure(User)`
  - 로그인 실패 횟수를 증가시킨다.
```

---

# 47. PROJECT_SUMMARY 크기 제한

대형 프로젝트에서 Markdown이 지나치게 커지면 안 된다.

따라서 Level을 지원할 수 있다.

```text
SUMMARY
STANDARD
DETAILED
```

기본은 `STANDARD`.

---

# 48. Summary Level

## SUMMARY

```text
Project
Modules
Technology
Directory Overview
Key Components
```

## STANDARD

```text
+
File Summary
Public/Important Symbols
```

## DETAILED

```text
+
대부분의 Symbol
Dependencies
Relations
```

---

# 49. Important Symbol 판별

기본적으로 다음 Symbol을 우선한다.

```text
Public Class

Controller

Service

Repository

Public API

Route

Component

Business Function

Test
```

다음은 낮은 우선순위:

```text
getter
setter
constructor
simple private utility
```

---

# 50. Summary 생성 최적화

파일이 변경되지 않았으면 Summary를 다시 생성하지 않는다.

키:

```text
file hash
+
summary generator version
```

예:

```text
hash 동일
summary_model_version 동일

→ 기존 summary 사용
```

---

# 51. Summary Provenance

Summary에 내부 metadata를 둘 수 있다.

```json
{
  "summary_metadata": {
    "generator": "LOCAL_LLM",
    "version": "1.2",
    "source_hash": "sha256:abc123"
  }
}
```

Server에 전달할 때는 필요에 따라 제거할 수 있다.

---

# 52. Index State

Local Agent 내부:

```json
{
  "last_revision": 184,

  "files": {
    "AuthService.java": {
      "hash": "sha256:...",
      "indexed_at": "...",
      "parser_version": "java-1.0"
    }
  }
}
```

---

# 53. Full Index 생성

초기 Flow:

```text
Project Open
   ↓
Ignore Rules
   ↓
Workspace Scan
   ↓
Project Detection
   ↓
File Classification
   ↓
Parser
   ↓
Symbol Extraction
   ↓
Relation Analysis
   ↓
Dependency Analysis
   ↓
Summary Generation
   ↓
PROJECT_INDEX
   ↓
PROJECT_SUMMARY
   ↓
Server Sync
```

---

# 54. Incremental Update

```text
File Event
  ↓
Debounce
  ↓
Hash 비교
  ↓
변경 확인
  ↓
File 재분석
  ↓
Old Symbols 제거
  ↓
New Symbols 저장
  ↓
Relations 갱신
  ↓
Revision + 1
```

---

# 55. CREATE Event

새 파일:

```json
{
  "operation": "CREATE",
  "path": "src/main/java/NewService.java"
}
```

새 File/Symbol/Relation을 추가한다.

---

# 56. MODIFY Event

```json
{
  "operation": "MODIFY",
  "path": "AuthService.java",
  "old_hash": "...",
  "new_hash": "..."
}
```

해당 파일 정보만 교체한다.

---

# 57. DELETE Event

```json
{
  "operation": "DELETE",
  "path": "LegacyService.java"
}
```

Server Delta에서도 삭제 사실을 명시해야 한다.

---

# 58. RENAME Event

File System에서 Rename 판단이 어려운 경우:

```text
DELETE + CREATE
```

로 처리해도 된다.

Hash가 동일한 경우 Rename으로 추론 가능하다.

---

# 59. Branch Change

Git Branch 변경 시:

```text
old branch
→ main

new branch
→ feature-x
```

변경 범위가 크면 Full Rebuild를 수행할 수 있다.

---

# 60. Delta Sync

Server에 매번 전체 Index를 보내지 않는다.

```json
{
  "schema": "oncode-project-index-delta/1.0",

  "project_id": "PRJ-100",
  "workspace_id": "WS-100",

  "revision": 185,
  "previous_revision": 184,

  "files": {
    "added": [],
    "modified": [
      "FILE-1001"
    ],
    "deleted": []
  },

  "symbols": {
    "added": [],
    "modified": [
      "SYM-101"
    ],
    "deleted": []
  }
}
```

---

# 61. Delta Payload

단순 ID만 보내지 않고 변경된 Entity 내용을 함께 전달할 수 있다.

```json
{
  "modified_files": [
    {
      "file_id": "FILE-1001",
      "path": "...",
      "hash": "...",
      "summary": "..."
    }
  ]
}
```

---

# 62. Delta Ordering

Server는 반드시 순차 revision만 적용한다.

예:

```text
현재 Server Revision = 184

incoming = 185
→ ACCEPT

incoming = 187
→ REVISION_GAP
```

Gap 발생 시 Full Index 재요청한다.

---

# 63. Revision Gap

Error:

```text
PROJECT_INDEX_REVISION_GAP
```

처리:

```text
Server
  ↓
project.get_index latest
  ↓
Full Sync
```

---

# 64. Index Hash

전체 Index에 optional hash를 둘 수 있다.

```json
{
  "revision_hash": "sha256:..."
}
```

Sync 데이터 손상 검증에 활용한다.

---

# 65. Server-side 저장 구조

Server는 JSON 전체를 하나의 Blob으로만 저장하지 않는다.

다음 구조로 분해한다.

```text
PROJECT

PROJECT_REVISION

PROJECT_FILE

PROJECT_SYMBOL

PROJECT_RELATION
```

`PROJECT_INDEX.json`은 Transport/Snapshot Format이다.

---

# 66. project.search 대상

검색 대상:

```text
file.path
file.summary

symbol.name
symbol.qualified_name
symbol.signature
symbol.summary

framework metadata
```

---

# 67. 검색 Ranking

Vector Search 없이 다음 요소를 조합할 수 있다.

```text
Exact Symbol Match

File Name Match

Keyword Match

Summary Full Text Search

Framework Match

Relation Distance

File Type

Module
```

---

# 68. 검색 Score 예

```text
exact symbol       100

file name           80

qualified name      75

summary keyword     50

dependency relation 30
```

실제 Weight는 운영하면서 조정한다.

---

# 69. PostgreSQL 검색

Server Project Catalog에서는 다음을 활용할 수 있다.

```text
Full Text Search
GIN
pg_trgm
B-Tree
```

예:

```text
"login authentication"
```

→

```text
AuthService.java
LoginController.java
SecurityConfig.java
```

---

# 70. Relation Expansion

첫 검색 결과:

```text
AuthService
```

인 경우 관계를 1~2 depth 확장할 수 있다.

```text
AuthService
 ├─ LoginController
 ├─ UserRepository
 └─ PasswordEncoder
```

이를 이용해 Source 요청 후보를 만든다.

---

# 71. Source Request Strategy

권장 순서:

```text
project.search
   ↓
Top Files
   ↓
project.get_symbols
   ↓
project.get_dependencies
   ↓
workspace.read_files
```

처음부터 Source를 요청하지 않는다.

---

# 72. 대형 파일 처리

파일이 너무 크면:

```text
workspace.read_symbol
```

또는:

```text
workspace.read_range
```

사용을 권장한다.

Index의 `line_start`, `line_end`를 이용한다.

---

# 73. Unsupported Language

지원하지 않는 언어:

```json
{
  "language": "UNKNOWN",
  "parser_status": "UNSUPPORTED"
}
```

File name/path/size/hash 등 기본 metadata는 유지한다.

---

# 74. Parser Error

문법 오류 등으로 AST parsing이 실패할 수 있다.

```json
{
  "parser_status": "FAILED",

  "diagnostics": [
    {
      "line": 18,
      "message": "Unexpected token"
    }
  ]
}
```

Index 전체 실패로 처리하지 않는다.

---

# 75. Partial Index

일부 Parser 실패 시:

```text
index.status = PARTIAL
```

로 관리할 수 있다.

---

# 76. Build Generated Files

다음과 같은 디렉터리는 기본 제외한다.

```text
target/
build/
dist/
coverage/
```

단 프로젝트 특성상 필요한 경우 설정으로 포함할 수 있다.

---

# 77. Dependency Source

Dependency에는 어디에서 선언되었는지를 기록한다.

```json
{
  "source": {
    "file": "pom.xml",
    "line": 84
  }
}
```

가능한 경우 line number를 포함한다.

---

# 78. Dependency Lock 정보

npm:

```text
package.json
+
package-lock.json
```

Python:

```text
requirements.txt
poetry.lock
uv.lock
```

등을 함께 분석한다.

---

# 79. Maven Multi-module

예:

```text
parent pom
├─ core
├─ api
└─ web
```

Module 별 dependency와 build 정보를 분리한다.

---

# 80. Framework Detection 근거

Framework Detection에는 evidence를 저장할 수 있다.

```json
{
  "framework": "SPRING_BOOT",

  "evidence": [
    "dependency:spring-boot-starter-web",
    "annotation:SpringBootApplication"
  ]
}
```

---

# 81. Architecture Hints

Project Intelligence에서 제한적으로 architecture hint를 생성할 수 있다.

예:

```json
{
  "architecture": {
    "patterns": [
      "LAYERED"
    ],

    "layers": [
      "controller",
      "service",
      "repository"
    ]
  }
}
```

Parser/path/package 기반 근거가 충분할 때만 생성한다.

---

# 82. 임의 아키텍처 추정 제한

확신할 수 없는 경우:

```text
UNKNOWN
```

으로 둔다.

LLM이 임의로:

```text
Hexagonal Architecture
DDD
CQRS
```

등을 확정하지 않도록 한다.

---

# 83. Entry Point

프로젝트 주요 Entry Point를 추출할 수 있다.

Java:

```text
@SpringBootApplication
main()
```

Node:

```text
server.ts
app.js
```

Python:

```text
main.py
FastAPI app
```

---

# 84. API Endpoint Index

지원 Framework에서는 endpoint를 별도 정규화할 수 있다.

```json
{
  "endpoints": [
    {
      "method": "POST",
      "path": "/api/login",
      "symbol_id": "SYM-loginController-login"
    }
  ]
}
```

초기 v1에서는 `symbols.framework_metadata`에 포함해도 된다.

---

# 85. Database Access Hint

Repository/DAO 수준에서 Entity/Table 관계를 추출할 수 있다.

예:

```json
{
  "framework_metadata": {
    "persistence": {
      "entity": "User",
      "table": "users"
    }
  }
}
```

JPA Annotation 등 명확한 근거가 있는 경우만 사용한다.

---

# 86. SQL Mapper

eGov/MyBatis 프로젝트에서는:

```text
Mapper Interface
XML Mapper
SQL ID
```

관계를 Index할 수 있다.

예:

```text
UserMapper.findUser
      ↓
UserMapper.xml#findUser
```

이 기능은 eGov/Spring 프로젝트에서 매우 유용하므로 v1 후반 또는 v1.1에서 고려한다.

---

# 87. Test Relation

Test와 대상 Symbol 관계:

```json
{
  "type": "TESTS",

  "source": {
    "id": "SYM-AuthServiceTest-login"
  },

  "target": {
    "id": "SYM-AuthService-login"
  }
}
```

Related Test Selection에 사용한다.

---

# 88. Summary 생성 실패

Summary 생성이 실패해도 Index는 유효해야 한다.

```json
{
  "summary": null,

  "summary_status": "FAILED"
}
```

구조적 metadata를 우선한다.

---

# 89. Summary Model 미사용 모드

폐쇄망 환경이나 자원 부족 시:

```text
summary.mode = NONE
```

또는 heuristic summary를 사용할 수 있다.

```text
Class/Method Name
+
Annotation
+
Docstring/Javadoc
```

기반으로 생성한다.

---

# 90. 기존 주석 활용

우선순위:

```text
Javadoc / Docstring
       ↓
Parser Structure
       ↓
LLM Summary
```

이미 좋은 설명이 있으면 LLM을 다시 호출할 필요가 없다.

---

# 91. Project Index 보안

Index에는 다음을 넣지 않는다.

```text
Password value

API Key

Private Key

Connection Password

Access Token

Secret Environment Value
```

---

# 92. Config Redaction

예:

원본:

```text
password=my-secret-password
```

Index:

```json
{
  "key": "password",
  "value": "[REDACTED]"
}
```

필요하지 않다면 key만 기록하고 value 자체를 저장하지 않는다.

---

# 93. 개인정보 최소화

프로젝트 절대경로:

```text
C:\Users\hong\work\project
```

대신:

```text
workspace relative path
```

만 사용한다.

---

# 94. Ignore Config

`.codegen/config.yaml`:

```yaml
index:
  exclude:
    - "**/generated/**"
    - "**/vendor/**"

  include:
    - "src/**"

  max_file_size: 1048576

  summary:
    level: standard
```

---

# 95. Git Ignore 활용

기본 `.gitignore`도 Index 제외 규칙에 참고할 수 있다.

다만 다음 파일은 Git Ignore 상태여도 Project 분석상 필요한 경우가 있다.

예:

```text
IDE generated config
local build config
```

따라서 `.gitignore == 무조건 제외`는 정책 선택사항으로 둔다.

---

# 96. Default Ignore Priority

```text
Security deny rule

.codegen exclude

Built-in ignore

.gitignore

Include override
```

같은 순서를 명확히 정의해야 한다.

---

# 97. PROJECT_SUMMARY 예시

```markdown
# Project Summary

## Project

- Name: customer-portal
- Branch: feature/login-lock
- Modules: backend, frontend

## Technology Stack

### Backend
- Java 17
- Spring Boot
- Maven

### Frontend
- TypeScript
- React
- npm

## Project Structure

backend/
├─ src/main/java/com/example/
│  ├─ controller/
│  ├─ service/
│  └─ repository/
└─ src/test/

frontend/
└─ src/
   ├─ components/
   └─ services/

## Key Files

### `AuthService.java`

사용자 인증 및 로그인 로직을 담당한다.

- `login(LoginRequest): LoginResponse`
  - 사용자 인증 결과를 반환한다.
- `handleLoginFailure(User)`
  - 로그인 실패 상태를 처리한다.

### `LoginController.java`

로그인 REST API를 제공한다.

- `POST /api/login`

## Dependencies

- spring-boot-starter-web
- spring-boot-starter-security
- react
- axios
```

---

# 98. PROJECT_INDEX 예시 축약본

```json
{
  "schema": "oncode-project-index/1.0",

  "project": {
    "project_id": "PRJ-100",
    "name": "customer-portal"
  },

  "workspace": {
    "workspace_id": "WS-100",
    "branch": "feature/login-lock"
  },

  "revision": {
    "number": 184,
    "previous": 183,
    "type": "INCREMENTAL"
  },

  "modules": [
    {
      "module_id": "MOD-backend",
      "name": "backend",
      "languages": ["JAVA"],
      "frameworks": ["SPRING_BOOT"],
      "build_tools": ["MAVEN"]
    }
  ],

  "files": [
    {
      "file_id": "FILE-100",
      "module_id": "MOD-backend",
      "path": "backend/src/main/java/com/example/AuthService.java",
      "language": "JAVA",
      "hash": "sha256:...",
      "summary": "사용자 인증을 처리한다.",
      "symbol_ids": ["SYM-100", "SYM-101"]
    }
  ],

  "symbols": [
    {
      "symbol_id": "SYM-100",
      "file_id": "FILE-100",
      "type": "CLASS",
      "name": "AuthService"
    },
    {
      "symbol_id": "SYM-101",
      "file_id": "FILE-100",
      "parent_symbol_id": "SYM-100",
      "type": "METHOD",
      "name": "login",
      "signature": "login(LoginRequest): LoginResponse",
      "line_start": 40,
      "line_end": 72,
      "summary": "로그인 인증을 수행한다."
    }
  ],

  "relations": [],

  "dependencies": []
}
```

---

# 99. Server Sync Flow

```text
Local Project Intelligence
        ↓
PROJECT_INDEX revision 184
        ↓
Project Sync
        ↓
Server Project Context Service
        ↓
PROJECT_REVISION
PROJECT_FILE
PROJECT_SYMBOL
PROJECT_RELATION
```

---

# 100. 최초 Sync

```text
Server Revision 없음
       ↓
Full Index Upload
       ↓
revision 1
```

---

# 101. 이후 Sync

```text
revision 184
       ↓
file changed
       ↓
revision 185
       ↓
Delta Upload
```

---

# 102. Server 요청 기반 재동기화

Server가 inconsistency를 발견하면:

```text
project.index.resync_required
```

이벤트 또는 Tool 요청을 보낼 수 있다.

Local Agent:

```text
Full Snapshot
```

으로 재동기화한다.

---

# 103. File Hash 알고리즘

기본:

```text
SHA-256
```

파일 내용 기준으로 계산한다.

Line ending normalize 여부는 일관되게 정의해야 한다.

권장:

```text
raw file bytes 기준
```

이다.

---

# 104. Stable Symbol ID

Symbol ID를 매번 랜덤 생성하면 Delta 추적이 어려워진다.

가능하면 deterministic key를 생성한다.

예:

```text
hash(
 project relative path
 + symbol type
 + qualified name
 + normalized signature
)
```

---

# 105. Symbol 이동 문제

메서드가 파일 내에서 단순 이동하면 line number가 바뀌더라도 Symbol ID는 유지될 수 있다.

따라서 line number는 Symbol Identity에 포함하지 않는다.

---

# 106. Signature 변경

```text
login(String)
```

→

```text
login(LoginRequest)
```

처럼 signature가 변경되면 새 Symbol로 판단할 수 있다.

이 정책은 언어별로 설정 가능하다.

---

# 107. File ID

File ID도 상대경로 기반 deterministic ID를 고려한다.

Rename 시에는 새 ID가 생성될 수 있다.

별도로 Rename Relation을 기록할 수도 있다.

---

# 108. Index Consistency

다음 조건을 검증한다.

```text
모든 symbol.file_id 존재

모든 relation source 존재

모든 relation target 존재

module_id 유효

revision 순서 정상
```

---

# 109. Validation Tool

Local 내부에서:

```text
project.validate_index
```

와 같은 내부 기능을 두는 것을 권장한다.

Server Tool로 공개할 필요는 없다.

---

# 110. Schema Validation

JSON Schema를 별도 정의하는 것이 좋다.

예:

```text
schemas/
├─ project-index-v1.schema.json
└─ project-index-delta-v1.schema.json
```

Local Agent와 Server 모두 동일 Schema로 검증한다.

---

# 111. Index Size 관리

대형 프로젝트에서는 Index 자체도 커질 수 있다.

따라서 Server Tool 응답으로 전체 Index를 LLM에 반환하지 않는다.

```text
project.search
project.get_symbols
project.get_dependencies
```

를 통해 필요한 데이터만 반환한다.

---

# 112. Full Index 접근 권한

`project.get_index`는 시스템 동기화나 운영 목적으로 사용하고, 일반 Agent가 자주 호출하지 않도록 한다.

---

# 113. Search Result 형태

```json
{
  "matches": [
    {
      "type": "FILE",

      "file_id": "FILE-100",

      "path": "src/main/java/AuthService.java",

      "score": 98,

      "summary": "사용자 인증 처리",

      "matched_symbols": [
        {
          "symbol_id": "SYM-101",
          "name": "login",
          "signature": "login(LoginRequest): LoginResponse"
        }
      ]
    }
  ]
}
```

---

# 114. Search Explain

Senior Developer가 왜 파일을 요청하는지 판단할 수 있도록 검색 근거를 반환한다.

```json
{
  "reason": [
    "method name matched: login",
    "summary matched: authentication",
    "called by LoginController"
  ]
}
```

이 기능은 매우 유용하다.

---

# 115. Search Filter

지원 권장 필터:

```text
module

language

file_type

symbol_type

framework

path

dependency

test_only

generated
```

---

# 116. Search Relation Depth

선택적으로:

```json
{
  "relation_depth": 1
}
```

을 지원한다.

너무 깊은 Graph 확장은 검색 Noise를 증가시키므로 기본 0 또는 1을 권장한다.

---

# 117. Project Intelligence의 책임 범위

Project Intelligence가 담당한다.

```text
구조 분석
Symbol Index
Dependency Metadata
Framework Detection
Summary
Relation
Revision
```

담당하지 않는다.

```text
기능 구현 설계
코드 변경 결정
취약점 최종 판단
개발 가이드 판단
```

이 영역은 Server Agent의 책임이다.

---

# 118. MVP 범위

초기 MVP에는 다음을 포함한다.

```text
Project Detection

File Scan

File Hash

Java / Python / JS / TS Parser

Class / Method / Function Symbol

Import

Basic Dependency

File Summary

Method Summary

PROJECT_SUMMARY.md

PROJECT_INDEX.json

Incremental File Update

Revision

Delta Sync
```

---

# 119. MVP 후순위

```text
Full Call Graph

Data Flow Analysis

Deep Reference Resolution

MyBatis XML Mapping

JPA Entity Graph

Advanced API Graph

React State Graph

Cross-language Call Graph

Architecture Pattern Detection
```

---

# 120. 핵심 설계 결정

onCode Project Intelligence / PROJECT_INDEX v1의 핵심 결정은 다음과 같다.

1. `PROJECT_SUMMARY.md`와 `PROJECT_INDEX.json`을 분리한다.
2. JSON Index를 시스템의 공식 프로젝트 Map으로 사용한다.
3. Source 전체는 Index에 저장하지 않는다.
4. 구조 정보는 Parser 기반으로 추출한다.
5. LLM은 의미 요약에 제한적으로 사용한다.
6. Project와 Local Workspace를 구분한다.
7. Git Revision과 Project Index Revision을 구분한다.
8. Multi-module 구조를 지원한다.
9. File, Symbol, Relation, Dependency를 독립 Entity로 관리한다.
10. File Hash와 deterministic Symbol ID를 사용한다.
11. Incremental Indexing을 기본으로 한다.
12. Delta Sync에 revision sequence를 적용한다.
13. Revision Gap 발생 시 Full Resync한다.
14. Server는 Index 전체 대신 `project.search` 중심으로 사용한다.
15. Vector DB 없이 RDBMS Full Text Search와 Symbol/Relation 검색을 사용한다.
16. Config Secret 값은 Index에 저장하지 않는다.
17. Unsupported Language 또는 Parser Failure가 전체 Index 실패로 이어지지 않게 한다.
18. Summary 실패보다 구조 Index 성공을 우선한다.
19. 대형 프로젝트에서는 PROJECT_SUMMARY 크기를 제한한다.
20. 초기 구현은 정밀 Call Graph보다 파일·Symbol 탐색 정확도를 우선한다.