# SPEC-10 onCode Runtime Guide Repository / Retrieval Architecture v1

## 1. 목적

본 명세는 onCode 런타임에서 사용하는 개발 가이드 저장소와 검색 아키텍처를 정의한다.

SPEC-09에서 전처리 및 검수 완료된 Guide Section은 Publish 과정을 거쳐 Runtime Guide Repository에 반영된다.

Runtime Guide Repository의 주요 목적은 다음과 같다.

1. Agent가 개발 요청에 적합한 Guide를 빠르게 조회할 수 있도록 한다.
2. Vector DB 없이 RDBMS 기반 검색을 제공한다.
3. 기술, 카테고리, 버전, 프로젝트, 조직 정책을 기준으로 검색 범위를 제한한다.
4. Mandatory / Security Guide의 우선순위를 강제한다.
5. 검색 결과의 근거와 출처를 명확히 제공한다.
6. 동일한 Guide Release 기준으로 검색 결과를 재현할 수 있도록 한다.
7. Agent가 과도한 Guide Context를 받지 않도록 Retrieval 결과를 제한한다.
8. Guide 검색 결과를 Review 및 Security 단계에서 동일하게 재사용할 수 있도록 한다.

---

# 2. 핵심 원칙

Runtime Guide Retrieval은 다음 원칙을 따른다.

```text
Metadata Filtering
        ↓
Keyword / Full Text Search
        ↓
Exact Match / Symbol Match
        ↓
Priority / Rule Type Ranking
        ↓
Version Compatibility
        ↓
Result Diversification
        ↓
Top-K Selection
```

Vector Embedding Search는 v1에서 사용하지 않는다.

---

# 3. 전체 아키텍처

```text
┌──────────────────────── onCode Agent ────────────────────────┐
│                                                             │
│ Requirement / Design / Review / Security Agent              │
│                          │                                  │
│                          ▼                                  │
│                     guide.search                            │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌──────────────────── Guide Retrieval Service ─────────────────┐
│                                                             │
│ Query Analyzer                                              │
│      │                                                      │
│      ├─ Technology Resolver                                 │
│      ├─ Category Resolver                                   │
│      ├─ Version Resolver                                    │
│      ├─ Keyword Expander                                    │
│      └─ Scope Resolver                                      │
│                                                             │
│ Retrieval Engine                                            │
│      │                                                      │
│      ├─ Metadata Filter                                     │
│      ├─ Exact Match                                         │
│      ├─ Keyword Search                                      │
│      ├─ PostgreSQL FTS                                      │
│      ├─ Trigram Search                                      │
│      ├─ Alias Search                                        │
│      └─ Rule Priority Ranking                               │
│                                                             │
│ Result Ranker                                               │
│                                                             │
│ Context Packager                                            │
│                                                             │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌──────────────── Runtime Guide Repository ────────────────────┐
│                                                             │
│ GUIDE_RELEASE                                               │
│ GUIDE_DOCUMENT                                              │
│ GUIDE_SECTION                                               │
│ CATEGORY                                                    │
│ TECHNOLOGY                                                  │
│ KEYWORD                                                     │
│ ALIAS                                                       │
│ RULE                                                        │
│ VERSION_SCOPE                                               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

# 4. Runtime Repository와 Curation Repository 분리

SPEC-09의 Curation 영역과 Runtime Repository는 논리적으로 분리한다.

```text
Curation Repository
    ↓ Publish
Runtime Repository
```

Curation Repository에는 다음 상태가 존재할 수 있다.

```text
DRAFT
REVIEW_REQUIRED
APPROVED
PUBLISHED
ARCHIVED
```

Runtime Repository는 기본적으로:

```text
PUBLISHED
```

상태의 데이터만 가진다.

---

# 5. Guide Release

Runtime 검색은 특정 Guide Release를 기준으로 수행하는 것을 권장한다.

예:

```text
GUIDE-REL-20260909-01
```

Guide Release에는 다음 정보가 포함된다.

```text
release_id

version

created_at

created_by

status

description
```

---

# 6. Release 목적

Guide Release를 사용하면 다음이 가능하다.

```text
검색 결과 재현

Rollback

Agent 실행 당시 Guide Set 추적

변경 전후 비교

운영 안정성
```

---

# 7. Runtime Guide Entity

Runtime 핵심 Entity는 다음과 같다.

```text
GUIDE_RELEASE

GUIDE_DOCUMENT

GUIDE_DOCUMENT_VERSION

GUIDE_SECTION

GUIDE_SECTION_VERSION

CATEGORY

SECTION_CATEGORY

TECHNOLOGY

SECTION_TECHNOLOGY

KEYWORD

SECTION_KEYWORD

ALIAS

GUIDE_RULE

GUIDE_PRIORITY

VERSION_SCOPE
```

---

# 8. GUIDE_SECTION_VERSION

Runtime 검색의 핵심 단위이다.

주요 필드:

```text
section_version_id

section_id

document_version_id

release_id

title

description

summary

content

heading_path

priority

rule_type

status

language

page_start

page_end

content_hash
```

---

# 9. Runtime Search Unit

검색 단위는 문서 전체가 아니라 Section이다.

잘못된 방식:

```text
Spring Development Guide.pdf
```

단위 검색.

권장:

```text
Spring Transaction Rollback Policy

Spring Controller Exception Handling

eGov Service Layer Coding Rule
```

같은 Section 단위 검색.

---

# 10. Query 입력

`guide.search`는 다음 정보를 입력으로 받을 수 있다.

```json
{
  "query": "로그인 실패 시 계정 잠금 구현",

  "technologies": [
    "JAVA",
    "SPRING_BOOT"
  ],

  "categories": [
    "SECURITY",
    "AUTHENTICATION"
  ],

  "project_id": "PRJ-100",

  "runtime": {
    "java": "17",
    "spring_boot": "3.3"
  },

  "purpose": "DESIGN",

  "limit": 10
}
```

---

# 11. purpose

Guide가 어느 단계에서 필요한지를 명시한다.

```text
REQUIREMENT

DESIGN

IMPLEMENTATION

REVIEW

SECURITY

TEST

GENERAL_QA
```

Ranking에 영향을 줄 수 있다.

---

# 12. Query Analyzer

Query Analyzer는 자연어 요청을 검색 조건으로 구조화한다.

입력:

```text
"Spring 로그인 실패 5회 시 계정 잠금 구현"
```

출력 예:

```json
{
  "terms": [
    "login failure",
    "account lock",
    "authentication"
  ],

  "technologies": [
    "SPRING",
    "JAVA"
  ],

  "categories": [
    "SECURITY",
    "AUTHENTICATION"
  ]
}
```

---

# 13. Query Analyzer 구현

LLM을 사용할 수 있지만 결과는 구조화한다.

LLM이 직접 SQL을 작성하도록 하지 않는다.

```text
Natural Language
    ↓
Query Analysis
    ↓
Structured Search Request
    ↓
Retrieval Engine
```

---

# 14. Project Profile 활용

Agent가 Technology를 명시하지 않아도 Project Context를 활용할 수 있다.

예:

```text
Project:
Java 17
Spring Boot 3.3
```

사용자:

```text
"트랜잭션 처리 규칙 찾아줘."
```

자동 Filter:

```text
Technology = JAVA, SPRING_BOOT
```

---

# 15. Scope Resolver

Guide Scope를 다음 순서로 적용한다.

```text
PROJECT

ORGANIZATION

FRAMEWORK

LANGUAGE

GENERAL
```

---

# 16. Guide Priority

SPEC-08과 동일한 우선순위를 적용한다.

```text
PROJECT_RULE

>

ORGANIZATION_RULE

>

FRAMEWORK_RULE

>

LANGUAGE_RULE

>

GENERAL_RULE
```

---

# 17. Priority Score

예:

```text
PROJECT_RULE       1000

ORGANIZATION_RULE   800

FRAMEWORK_RULE      600

LANGUAGE_RULE       400

GENERAL_RULE        200
```

검색 Score와 별도로 강하게 반영한다.

---

# 18. Rule Type

Guide Section의 Rule Type:

```text
REFERENCE

RECOMMENDED

REQUIRED

SECURITY_REQUIRED

PROHIBITED
```

---

# 19. Rule Score

예:

```text
SECURITY_REQUIRED  1000

PROHIBITED          900

REQUIRED            700

RECOMMENDED         400

REFERENCE           100
```

검색 relevance가 조금 낮더라도 필수 규칙은 상위에 노출할 수 있다.

---

# 20. Mandatory Guide Injection

특정 기술/카테고리에 Mandatory Rule이 존재하면 일반 검색 결과와 별도로 반드시 포함한다.

예:

```text
검색:
"파일 업로드 구현"
```

결과:

```text
Relevant Guides

+

Mandatory Security Rules
```

---

# 21. Mandatory Rule 조건

예:

```text
technology = SPRING
category = FILE_UPLOAD
rule_type = SECURITY_REQUIRED
```

검색어와 정확히 일치하지 않더라도 적용 가능성이 있으면 후보에 포함한다.

---

# 22. Search Pipeline

권장 처리 순서:

```text
1. Release 선택
2. Project/Organization Scope 적용
3. Technology Filter
4. Version Filter
5. Category Filter
6. Mandatory Rule 조회
7. Exact Keyword Match
8. Symbol/API Match
9. Full Text Search
10. Alias Search
11. Trigram Search
12. Ranking
13. Deduplication
14. Top-K
15. Context Packaging
```

---

# 23. Metadata Filter

먼저 가능한 검색 범위를 줄인다.

예:

```sql
technology IN ('SPRING', 'JAVA')
AND status = 'PUBLISHED'
AND release_id = :release
```

이후 Text Search를 수행한다.

---

# 24. Version Filtering

Guide가 적용되는 Version 범위를 고려한다.

예:

```text
Spring Boot >= 3.0
Spring Boot < 4.0
```

현재 Project:

```text
Spring Boot 3.3
```

이면 검색 가능.

---

# 25. Version 불일치

다음 Guide는 기본 검색 결과에서 제외한다.

```text
Spring Boot 2.x 전용
```

현재 프로젝트가 3.x인 경우.

단 검색 결과가 부족하면:

```text
version_mismatch = true
```

표시와 함께 낮은 Ranking으로 반환할 수 있다.

---

# 26. Exact Keyword Search

가장 높은 검색 신뢰도를 가진다.

예:

```text
@Transactional
```

Keyword Table Exact Match.

---

# 27. Symbol / API Search

다음 항목은 일반 자연어보다 높은 Weight를 준다.

```text
Class

Annotation

Method

API

Configuration Key
```

예:

```text
@RestControllerAdvice
```

---

# 28. Full Text Search

PostgreSQL FTS를 사용한다.

검색 대상:

```text
title

description

summary

content
```

필드별 Weight를 다르게 설정한다.

---

# 29. FTS Weight

예:

```text
title        A

keywords     A

description  B

summary      B

content      C
```

---

# 30. Korean / English 검색

개발 문서는 한글과 영문 기술 용어가 혼재한다.

예:

```text
트랜잭션 rollback

인증 authentication

의존성 dependency
```

Alias/Synonym Table을 적극 활용한다.

---

# 31. Alias Table

예:

```text
전자정부프레임워크
→ egovframework
→ egovframe
→ egov

트랜잭션
→ transaction

롤백
→ rollback
```

---

# 32. Keyword Expansion

Query:

```text
"계정 잠금"
```

확장:

```text
account lock
lockout
authentication failure
login failure
```

LLM 또는 사전 정의 Alias를 사용할 수 있다.

---

# 33. Expansion 우선순위

권장:

```text
관리자 등록 Alias

>

Technology Dictionary

>

LLM Generated Expansion
```

LLM 확장은 보조적으로 사용한다.

---

# 34. Trigram Search

오타 및 유사 문자열 검색에 사용한다.

예:

```text
transaciton
```

→

```text
transaction
```

PostgreSQL `pg_trgm` 활용 가능.

---

# 35. Search Index

권장 Index:

```text
B-Tree

GIN Full Text Search

GIN/GIN Trigram

Category FK Index

Technology FK Index

Priority Index

Rule Type Index

Release Index
```

---

# 36. Search Document Column

검색 성능을 위해 Section별 결합 Search Text를 둘 수 있다.

예:

```text
title
+
description
+
summary
+
keywords
+
aliases
+
content
```

에서 `tsvector` 생성.

---

# 37. Search Score

최종 Score 예:

```text
score =

exact_match_score

+ title_score

+ keyword_score

+ fts_score

+ trigram_score

+ technology_score

+ category_score

+ priority_score

+ rule_type_score

+ version_score
```

---

# 38. Priority가 Relevance를 완전히 덮으면 안 됨

PROJECT_RULE이라고 해서 관련 없는 Guide가 항상 상위에 오르면 안 된다.

따라서 최소 relevance threshold를 둔다.

예:

```text
relevance < threshold
→ 제외
```

Mandatory Rule은 예외.

---

# 39. Ranking 예

검색:

```text
"Spring 로그인 실패 처리"
```

후보:

```text
A. 프로젝트 로그인 실패 처리 규칙
B. Spring Security 인증 정책
C. Java Exception 일반 규칙
```

예상 Ranking:

```text
A
B
C
```

---

# 40. Search Result

```json
{
  "results": [
    {
      "section_id": "SEC-100",
      "section_version_id": "SECV-202",

      "title": "로그인 실패 및 계정 잠금 처리 규칙",

      "description": "로그인 실패 횟수와 계정 잠금 처리 기준",

      "summary": "...",

      "priority": "PROJECT_RULE",

      "rule_type": "REQUIRED",

      "technology": [
        "SPRING_BOOT"
      ],

      "categories": [
        "SECURITY",
        "AUTHENTICATION"
      ],

      "score": 0.96,

      "match_reason": [
        "keyword: login failure",
        "category: authentication",
        "project rule"
      ],

      "source": {
        "document": "프로젝트 보안 개발 가이드",
        "page_start": 42,
        "page_end": 44
      }
    }
  ]
}
```

---

# 41. Match Reason

검색 결과에는 왜 검색되었는지 포함하는 것을 권장한다.

예:

```text
Exact keyword match

Title match

Technology match

Mandatory rule

Project-specific rule
```

Senior Developer의 판단 근거가 된다.

---

# 42. guide.read

검색 후 실제 Section 내용을 조회한다.

예:

```json
{
  "tool": "guide.read",

  "arguments": {
    "section_version_id": "SECV-202"
  }
}
```

---

# 43. guide.search와 guide.read 분리

`guide.search`에서는 전체 Content를 반환하지 않는다.

검색 결과:

```text
title
summary
metadata
score
```

중심.

필요한 Section만:

```text
guide.read
```

로 가져온다.

Context 사용량을 줄인다.

---

# 44. Batch Read

여러 Section이 필요한 경우:

```text
guide.read_many
```

를 지원할 수 있다.

예:

```json
{
  "section_version_ids": [
    "SECV-101",
    "SECV-102",
    "SECV-110"
  ]
}
```

---

# 45. Context Packaging

Agent에 Guide 원문 전체를 그대로 전달하지 않는다.

필요 Section만 묶는다.

예:

```text
Guide Context

[GUIDE-87]
Title
Rule Type
Summary
Relevant Content
Source
```

---

# 46. Guide Context Format

예:

```text
Guide ID: GUIDE-87
Section: account-lock
Priority: PROJECT_RULE
Rule Type: REQUIRED

Rule:
로그인 실패 횟수가 지정 기준을 초과하면 계정을 잠금 처리한다.

Source:
프로젝트 보안 개발 가이드 / p.42
```

---

# 47. Mandatory / Reference 구분

Agent Context에서도 분리한다.

```text
MANDATORY RULES

RECOMMENDED GUIDES

REFERENCE MATERIAL
```

Agent가 강제 규칙과 참고사항을 혼동하지 않도록 한다.

---

# 48. Result Limit

Top-K를 무제한 반환하지 않는다.

권장 기본:

```text
Mandatory Rules: max 10

Primary Results: max 10

Reference Results: max 5
```

실제 값은 운영 데이터로 조정한다.

---

# 49. Result Diversification

같은 문서의 비슷한 Section이 검색 결과를 모두 차지하지 않도록 한다.

예:

```text
max_sections_per_document = 3
```

단 Mandatory Rule은 예외.

---

# 50. Parent Context Inclusion

검색된 Section이 지나치게 짧은 경우 Parent Heading 또는 Summary를 함께 제공한다.

예:

```text
Transaction
 └─ Rollback Rules
```

검색 결과:

```text
Parent: Transaction Management
Section: Rollback Rules
```

---

# 51. Child Section Expansion

검색된 Section이 상위 개념일 경우 필요한 Child Section을 추가 후보로 넣을 수 있다.

예:

```text
Authentication
 ├─ Login Failure
 ├─ Account Lock
 └─ Password Policy
```

---

# 52. Search Intent

Query가 어떤 목적의 Guide를 찾는지 구분한다.

예:

```text
HOW_TO

RULE

SECURITY_RULE

API_USAGE

ERROR_HANDLING

TEST_STANDARD

ARCHITECTURE
```

---

# 53. Intent별 Ranking

예:

```text
"반드시 지켜야 할 보안 규칙"
```

이면:

```text
SECURITY_REQUIRED
REQUIRED
```

가 높은 Weight.

---

# 54. General Q&A 검색

일반 질문에도 Guide를 사용할 수 있다.

예:

```text
"우리 회사 Spring Controller 예외처리 방식이 뭐야?"
```

Guide Retrieval:

```text
Project / Organization Guide
```

를 먼저 검색한다.

---

# 55. Runtime Retrieval Cache

검색 성능 개선을 위해 Cache를 사용할 수 있다.

Cache Key 예:

```text
release_id
+
normalized query
+
technology
+
category
+
version
+
project_id
```

---

# 56. Cache Invalidation

새 Guide Release가 Publish되면 기존 Release Cache는 그대로 둘 수 있다.

새 Release는 새로운 Cache Namespace를 사용한다.

이것이 Release 기반 구조의 장점이다.

---

# 57. Search Audit

Agent가 어떤 Guide를 조회했는지 기록한다.

예:

```text
work_item_id

agent

query

guide_release

selected_sections

timestamp
```

---

# 58. Guide Reference 저장

실제 사용된 Guide는 SPEC-03의 `GUIDE_REFERENCE`에 기록한다.

```text
GUIDE_REFERENCE

work_item_id
section_version_id
release_id
purpose
```

---

# 59. 검색된 Guide와 실제 사용 Guide 구분

모든 검색 결과를 Guide Reference로 저장하지 않는다.

```text
Search Candidate
≠
Used Guide
```

Agent가 실제 설계/검토 근거로 채택한 Section만 기록한다.

---

# 60. Design Provenance

Design에는 사용한 Guide Reference를 연결한다.

예:

```text
DESIGN v3

Guide:
GUIDE-87
GUIDE-91
```

---

# 61. Review 재사용

Implementation 후 Review Agent는 Design 시 사용한 Guide를 기본 입력으로 받는다.

추가 검색도 가능하다.

```text
Design Guide Set

+

Review-specific Search
```

---

# 62. Security 재사용

Security Agent는:

```text
Design Security Rules

+

Security-specific Mandatory Guide
```

를 사용한다.

---

# 63. Guide Conflict

두 Guide가 서로 충돌할 수 있다.

예:

```text
Organization Guide:
Service에서 transaction 처리

Framework Guide:
Controller에서도 가능
```

Priority Rule을 적용한다.

```text
ORGANIZATION_RULE
>
FRAMEWORK_RULE
```

---

# 64. 동일 Priority 충돌

동일 Priority Guide가 충돌하면 다음 기준을 적용한다.

```text
Higher Rule Type

More Specific Scope

More Specific Technology

Newer Effective Version

Manual Conflict Resolution
```

---

# 65. Conflict Detection

검색 결과를 Context Packager가 검토한다.

예:

```text
Rule A:
MUST use X

Rule B:
MUST NOT use X
```

충돌이 의심되면:

```text
GUIDE_CONFLICT
```

상태 또는 Warning을 생성할 수 있다.

---

# 66. Guide Conflict 처리

Critical Conflict:

```text
Senior Developer
→ User/Admin clarification
```

또는 사전 큐레이션 단계에서 해결하도록 한다.

Runtime에서 자동 임의 선택은 피한다.

---

# 67. Effective Date

현재 유효하지 않은 Guide는 기본 제외한다.

조건:

```text
effective_from <= now

AND

effective_to IS NULL OR effective_to >= now
```

---

# 68. Deprecated Guide

상태:

```text
DEPRECATED
```

는 검색 대상에서 기본 제외한다.

사용자가:

```text
"기존 방식이 어떻게 되어 있었어?"
```

같은 역사 검색을 요청한 경우에만 포함할 수 있다.

---

# 69. Archived Guide

`ARCHIVED` Guide는 Runtime Search에서 제외한다.

---

# 70. Project-specific Guide

특정 프로젝트에만 적용되는 Guide:

```text
project_scope = PRJ-100
```

다른 Project에서는 검색되지 않는다.

---

# 71. Organization Guide

조직 전체:

```text
organization_scope = ORG-100
```

---

# 72. Global Framework Guide

Framework 공통 Guide는 Project Scope가 없다.

예:

```text
Spring Framework Official Guide
```

---

# 73. Scope Hierarchy

검색 시:

```text
Current Project
      ↓
Organization
      ↓
Global Framework
      ↓
Language
      ↓
General
```

순으로 확장한다.

---

# 74. Scope Search Strategy

처음부터 모든 Guide를 동일 Weight로 검색하지 않는다.

예:

```text
Phase 1:
Project + Organization

결과 부족 시

Phase 2:
Framework + Language

결과 부족 시

Phase 3:
General
```

방식도 가능하다.

---

# 75. 검색 Recall과 Precision

개발 가이드는 일반 웹 검색과 다르게 Precision이 중요하다.

잘못된 Guide를 제공하면 코드 구현 방향이 틀어질 수 있기 때문이다.

따라서 v1에서는:

```text
Precision > Recall
```

을 기본 목표로 한다.

---

# 76. Low Confidence Result

검색 Score가 낮으면 Agent에 자동 제공하지 않는다.

예:

```text
score < 0.35
→ 제외
```

값은 운영 데이터로 조정한다.

---

# 77. No Result

검색 결과가 없으면:

```text
GUIDE_NOT_FOUND
```

를 명시한다.

LLM이 Guide가 있는 것처럼 추측하지 않도록 한다.

---

# 78. No Result 정책

SPEC-02 정책과 연결한다.

```text
ALLOW_WITH_WARNING

ASK_USER

BLOCK_IMPLEMENTATION
```

Project Policy에 따라 결정한다.

---

# 79. Search Fallback

검색 결과가 없을 때 다음 순서로 Query를 완화할 수 있다.

```text
Exact Technology + Category

↓

Technology only

↓

Alias Expansion

↓

Framework/Language broad search
```

---

# 80. 무제한 Query Expansion 금지

LLM이 임의의 대량 Keyword를 만들어 검색 범위를 과도하게 넓히지 않도록 제한한다.

예:

```text
max_expanded_terms = 10
```

---

# 81. Query Normalization

예:

```text
@Transactional 처리 방법
```

정규화:

```text
transactional
transaction
rollback
```

---

# 82. 한국어 형태 처리

필요하면 한국어 형태소 분석기를 사용할 수 있지만 필수는 아니다.

초기에는:

```text
Trigram
+
Keyword
+
Alias
+
Title/Summary FTS
```

조합으로도 충분히 시작 가능하다.

---

# 83. 영어 기술 용어 보존

다음은 과도하게 한국어 형태소 처리하지 않는다.

```text
Spring Security

@Transactional

JWT

useEffect

ResponseEntity
```

Technical Token으로 그대로 유지한다.

---

# 84. Keyword Dictionary

별도 Technical Dictionary를 권장한다.

예:

```text
SPRING_SECURITY

TRANSACTION

JPA

MYBATIS

EGOV_SERVICE

REACT_HOOK
```

Alias와 검색 확장에 사용한다.

---

# 85. Framework Version Resolver

Project Profile에서 정확한 버전을 가져온다.

예:

```text
Spring Boot 3.3.2
```

Guide는:

```text
>=3.0 <4.0
```

범위와 비교한다.

---

# 86. Version Unknown

Project Version을 알 수 없는 경우:

```text
version_status = UNKNOWN
```

Version 제한 Guide를 낮은 Confidence로 반환하거나 Warning을 붙인다.

---

# 87. Category Filter

카테고리가 여러 개일 수 있다.

예:

```text
SECURITY
+
AUTHENTICATION
```

기본은 OR보다는 일부 AND Weight를 줄 수 있다.

---

# 88. Multi-category Ranking

Query가:

```text
보안 로그인 처리
```

이면:

```text
SECURITY + AUTHENTICATION
```

둘 다 일치하는 Section이 한쪽만 일치하는 Section보다 상위에 온다.

---

# 89. Title / Description 생성 품질 영향

SPEC-09에서 생성한 Title/Description이 Runtime 검색 품질에 직접 영향을 미친다.

따라서 Runtime Search 개선을 위해 Curation Console에서 검색용 Metadata 품질을 관리해야 한다.

---

# 90. Search Simulation

Admin Console에 다음 기능을 추가하는 것을 권장한다.

```text
Search Preview
```

예:

```text
Query:
"트랜잭션 rollback"

결과:
1. ...
2. ...
3. ...
```

Publish 전에 Retrieval 품질을 확인할 수 있다.

---

# 91. Golden Query

중요 Guide에는 대표 검색어를 등록할 수 있다.

예:

```text
GUIDE-87

Golden Queries:
- 로그인 실패
- 계정 잠금
- authentication lockout
```

운영 테스트에 활용한다.

---

# 92. Retrieval Evaluation

Guide Repository에 대한 평가 데이터셋을 만든다.

예:

```text
Query
Expected Section
Expected Mandatory Rule
```

---

# 93. 품질 지표

예:

```text
Precision@5

Recall@10

MRR

Mandatory Rule Recall

No-result Rate
```

Vector DB가 없어도 검색 품질을 정량적으로 개선할 수 있다.

---

# 94. Mandatory Rule Recall

특히 중요하다.

보안/필수 Rule이 적용 대상인데 검색되지 않는 문제를 별도 지표로 관리한다.

목표:

```text
Mandatory Rule Recall ≈ 100%
```

에 가깝게 잡는다.

---

# 95. Search Explain API

관리자용으로:

```text
guide.search.explain
```

을 제공할 수 있다.

예:

```text
Base FTS Score: 0.61
Keyword Match: +0.20
Project Priority: +0.15
Rule Type: +0.10
Version Match: +0.05
```

운영 튜닝에 유용하다.

---

# 96. Runtime DB 구성

초기에는 PostgreSQL 하나로 충분하다.

Schema 예:

```text
guide_runtime.*
```

핵심 Table:

```text
guide_release

document

section

section_version

category

section_category

technology

section_technology

keyword

section_keyword

alias

version_scope

guide_rule
```

---

# 97. Materialized Search View

검색 성능을 위해 Materialized View 또는 Search Table을 둘 수 있다.

예:

```text
guide_search_index
```

필드:

```text
section_version_id

release_id

search_vector

normalized_title

keyword_text

alias_text

priority

rule_type

technology_codes

category_codes
```

---

# 98. Publish 시 Search Index 생성

SPEC-09 Publish 과정:

```text
Curated Guide
    ↓
Release 생성
    ↓
Runtime Table 반영
    ↓
Search Index Build
    ↓
Validation
    ↓
Release Activate
```

---

# 99. Release Activation

새 Release를 바로 Current로 바꾸지 않고 검증 후 활성화한다.

상태:

```text
BUILDING

VALIDATING

ACTIVE

INACTIVE

FAILED
```

---

# 100. Current Release

Runtime Service는 기본적으로:

```text
CURRENT_ACTIVE_RELEASE
```

를 사용한다.

특정 Work Item이 시작되면 Release ID를 고정할 수 있다.

---

# 101. Work Item Guide Release Pinning

Work Item 시작:

```text
guide_release_id = REL-100
```

작업 도중 새 Release가 나와도 기존 Work Item은 REL-100을 유지할 수 있다.

장점:

```text
작업 중 규칙 변화 방지

재현 가능성

Review 일관성
```

---

# 102. 새로운 Work Item

새로운 Work Item은 Current Active Release를 사용한다.

---

# 103. 긴 작업 중 Security Hotfix

예외적으로 Critical Security Guide가 Publish된 경우 기존 Work Item에 강제 적용할 필요가 있을 수 있다.

이 경우:

```text
SECURITY_FORCE_UPDATE
```

같은 별도 정책을 정의할 수 있다.

MVP에서는 제외 가능하다.

---

# 104. Retrieval Service API

주요 Tool/API:

```text
guide.search

guide.read

guide.read_many

guide.get_mandatory_rules

guide.get_release

guide.search_explain
```

---

# 105. guide.get_mandatory_rules

예:

```json
{
  "technologies": [
    "SPRING_BOOT"
  ],

  "categories": [
    "SECURITY"
  ],

  "project_id": "PRJ-100"
}
```

필수 Rule을 별도 조회한다.

---

# 106. Agent별 Retrieval 정책

## Requirement Agent

주로:

```text
Project Rule

Business Rule

Architecture Rule
```

---

## Design Agent

```text
Architecture

Framework

Security

Error Handling

Dependency
```

---

## Implementation Agent

```text
Coding Standard

Framework API Usage

Naming

Error Handling
```

---

## Review Agent

```text
Approved Design Guides

Coding Standard

Mandatory Rules
```

---

## Security Agent

```text
SECURITY_REQUIRED

PROHIBITED

Dependency Security
```

---

# 107. Retrieval Policy Profile

Agent별 기본 Search Filter/Profile을 정의할 수 있다.

예:

```json
{
  "profile": "SECURITY_REVIEW",

  "rule_types": [
    "SECURITY_REQUIRED",
    "PROHIBITED",
    "REQUIRED"
  ]
}
```

---

# 108. Agent 임의 Priority 변경 금지

Agent가:

```text
"이 규칙은 덜 중요해 보여."
```

라고 판단하여 Repository Priority를 변경할 수 없다.

Priority와 Rule Type은 Repository Metadata가 기준이다.

---

# 109. Context Token Budget

Guide Context도 Agent별 Budget을 둔다.

예:

```text
Design Agent:
8,000 tokens

Implementation:
6,000 tokens

Review:
5,000 tokens
```

실제 모델에 따라 조정한다.

---

# 110. Guide Context Compression

Section이 크면:

```text
Mandatory Rule
→ 원문 유지

Recommended Guide
→ Relevant Paragraph + Summary

Reference
→ Summary only
```

방식으로 Context를 줄일 수 있다.

---

# 111. Mandatory Rule 원문 우선

필수 정책은 Summary만 전달하지 않는다.

원문 또는 검수된 Canonical Content를 전달한다.

Summary 왜곡 위험을 줄인다.

---

# 112. Relevant Passage

긴 Section 내에서 검색 Query와 관련된 Paragraph를 선택할 수 있다.

단 원문 Section 구조를 잃지 않도록 Source Reference를 유지한다.

---

# 113. Search Result Provenance

Agent가 받은 Guide에는 반드시 다음을 포함한다.

```text
Guide ID

Section ID

Version

Release

Source Document

Page/Heading Path

Priority

Rule Type
```

---

# 114. Runtime Guide URI

Context Reference 예:

```text
guide://REL-100/sections/SEC-200/version/3
```

---

# 115. Context Storage 연결

Agent가 Guide를 실제 채택하면:

```text
context.put(GUIDE_REFERENCE)
```

를 수행한다.

---

# 116. Guide 결과 Cache와 Context 분리

검색 Cache는 운영 최적화용이다.

`GUIDE_REFERENCE`는 Work Item 공식 상태다.

둘을 혼동하지 않는다.

---

# 117. Search Security

Guide Search도 사용자 Project Scope를 검증한다.

다른 프로젝트 전용 Guide가 노출되지 않도록 한다.

---

# 118. Confidential Guide

일부 Guide는 특정 Team/Project만 접근 가능할 수 있다.

필드:

```text
visibility_scope

organization_id

project_id

role_requirement
```

---

# 119. Role-based Guide Access

예:

```text
GENERAL_DEVELOPER

LEAD

SECURITY

ADMIN
```

필요한 경우 RBAC를 적용한다.

---

# 120. Runtime Guide Admin 변경 금지

Runtime Repository는 Admin Console에서 직접 수정하지 않는다.

수정은:

```text
Curation
→ Publish
```

과정으로만 반영한다.

---

# 121. Rollback

새 Release 검색 품질에 문제가 있으면:

```text
REL-102 ACTIVE
     ↓
Rollback
     ↓
REL-101 ACTIVE
```

가능해야 한다.

---

# 122. Release 삭제

이미 Work Item이 참조한 Release는 물리 삭제하지 않는 것을 권장한다.

`INACTIVE` 상태로 보존한다.

---

# 123. Performance 목표

초기 목표 예:

```text
Metadata Filter + Search
< 500 ms

Guide Read
< 100 ms

Top-10 Search
< 1 sec
```

실제 데이터량에 따라 조정한다.

---

# 124. Scale 예상

Guide Section 수가:

```text
10,000
100,000
1,000,000
```

수준으로 증가하더라도 PostgreSQL FTS + Metadata Index 구조로 일정 규모까지 대응 가능하도록 설계한다.

대규모화 시 Search Engine 도입은 별도 검토한다.

---

# 125. Vector DB 제외 이유

v1에서는 다음 이유로 Vector DB를 사용하지 않는다.

```text
Guide Domain이 제한적

Category가 명확함

Technology Metadata가 존재함

Keyword/Title/Summary를 전처리 단계에서 생성함

Mandatory Rule은 의미 유사도보다 정책 적용이 중요함

검색 근거 설명이 쉬움
```

---

# 126. 향후 확장

검색 품질이 부족해지면 다음을 단계적으로 검토할 수 있다.

```text
BM25 Search Engine

Hybrid Search

Reranker

Embedding Search
```

하지만 Runtime API는 동일하게 유지하여 Backend를 교체 가능하게 한다.

---

# 127. Retrieval Provider 추상화

```text
GuideRetrievalProvider

search()
read()
mandatoryRules()
```

구현:

```text
PostgresGuideRetrievalProvider
```

향후:

```text
OpenSearchGuideRetrievalProvider
HybridGuideRetrievalProvider
```

확장 가능.

---

# 128. Search Backend와 Agent 분리

Agent는 검색 Backend가 PostgreSQL인지 알 필요가 없다.

```text
Agent
 ↓
guide.search
 ↓
Guide Retrieval Service
 ↓
Search Provider
```

---

# 129. Retrieval 실패

오류 예:

```text
GUIDE_RELEASE_NOT_FOUND

GUIDE_SECTION_NOT_FOUND

GUIDE_SEARCH_FAILED

GUIDE_ACCESS_DENIED

GUIDE_VERSION_MISMATCH
```

---

# 130. Search Degraded Mode

Full Text Index에 문제가 생긴 경우:

```text
Exact Keyword
+
Category
+
Title Search
```

정도로 제한된 Degraded Search를 제공할 수 있다.

---

# 131. Runtime Monitoring

운영 지표:

```text
Search Count

Average Latency

No Result Rate

Top-K Click/Usage

Guide Adoption Rate

Mandatory Rule Hit Rate

Guide Conflict Count

Search Error Rate
```

---

# 132. Guide Adoption Rate

검색된 Guide 중 실제 Agent가 Guide Reference로 채택한 비율.

검색 품질 튜닝에 유용하다.

---

# 133. Search Feedback

Review 단계에서:

```text
Guide not relevant
```

같은 내부 Feedback을 기록할 수 있다.

Admin Search Tuning에 활용한다.

Agent가 직접 Guide를 삭제하거나 Ranking Weight를 수정할 수는 없다.

---

# 134. Golden Query Regression

새 Guide Release Publish 전 Golden Query 테스트를 수행한다.

예:

```text
100개 Query
```

각 Query에 대해:

```text
Expected Guide

Forbidden Guide

Mandatory Rule
```

를 검증한다.

---

# 135. Release Validation Gate

Publish Pipeline:

```text
Build Runtime Index
     ↓
Golden Query Test
     ↓
Mandatory Rule Test
     ↓
Search Performance Test
     ↓
PASS
     ↓
Activate Release
```

---

# 136. Mandatory Rule Test

예:

```text
Query:
"파일 업로드 구현"

Expected Mandatory:
SEC-FILE-001
```

반드시 검색 결과에 포함되는지 확인한다.

---

# 137. Runtime Search Console

관리자에게 검색 테스트 Console을 제공하는 것이 좋다.

입력:

```text
Project

Technology

Version

Query
```

결과:

```text
Rank

Score

Match Reason

Rule Type

Priority
```

---

# 138. Search Explain UI

각 결과 클릭:

```text
Title match +0.25

Keyword +0.20

Project priority +0.15

Required rule +0.10
```

등을 확인할 수 있다.

---

# 139. 검색 Weight Config

Weight를 DB 또는 Config로 관리한다.

예:

```yaml
ranking:
  exact_keyword: 1.0
  title: 0.8
  summary: 0.5
  content: 0.2
  project_priority: 0.8
  required_rule: 0.7
```

---

# 140. Weight Versioning

Ranking 정책도 Version 관리한다.

예:

```text
retrieval-ranking/1.0
```

Agent Execution에 사용한 Ranking Version을 기록할 수 있다.

---

# 141. Runtime Retrieval 핵심 Flow

```text
Senior Developer
      ↓
guide.search
      ↓
Query Analyzer
      ↓
Scope / Technology / Version Filter
      ↓
Mandatory Rule Search
      ↓
Keyword / FTS / Alias Search
      ↓
Ranking
      ↓
Top-K
      ↓
Guide Metadata
      ↓
guide.read
      ↓
Relevant Guide Content
      ↓
Context Builder
      ↓
Agent
```

---

# 142. Design 단계 Flow

```text
Requirement
   ↓
Project Profile
   ↓
Guide Search
   ↓
Mandatory Rules
   +
Relevant Guides
   ↓
Design Agent
   ↓
Design
   ↓
GUIDE_REFERENCE 저장
```

---

# 143. Review 단계 Flow

```text
Approved Design
   +
Design Guide References
   +
Review Guide Search
        ↓
Review Agent
        ↓
Guide Compliance Findings
```

---

# 144. Security 단계 Flow

```text
Change Set
   ↓
Technology / Risk Detection
   ↓
Security Mandatory Rules
   ↓
Security Agent
   ↓
Security Finding
```

---

# 145. 핵심 데이터 관계

```text
GUIDE_RELEASE
     │
     └── GUIDE_DOCUMENT_VERSION
             │
             └── GUIDE_SECTION_VERSION
                    │
                    ├── CATEGORY
                    ├── TECHNOLOGY
                    ├── KEYWORD
                    ├── VERSION_SCOPE
                    ├── RULE_TYPE
                    └── PRIORITY
```

---

# 146. 핵심 설계 결정

onCode Runtime Guide Repository / Retrieval Architecture v1의 핵심 결정은 다음과 같다.

1. Runtime Guide Repository와 Curation Repository를 분리한다.
2. Published Guide만 Runtime 검색에 사용한다.
3. 검색 단위는 Document가 아니라 Guide Section이다.
4. Vector DB를 v1에서 사용하지 않는다.
5. PostgreSQL Metadata Filter + FTS + Keyword + Trigram + Alias 검색을 사용한다.
6. Project, Organization, Framework, Language Scope를 계층적으로 적용한다.
7. PROJECT_RULE > ORGANIZATION_RULE > FRAMEWORK_RULE > LANGUAGE_RULE > GENERAL_RULE 우선순위를 적용한다.
8. Rule Type을 REFERENCE / RECOMMENDED / REQUIRED / SECURITY_REQUIRED / PROHIBITED로 관리한다.
9. Mandatory Rule은 일반 relevance 검색과 별도로 강제 조회한다.
10. Technology와 Version Compatibility를 Retrieval Ranking에 반영한다.
11. `guide.search`와 `guide.read`를 분리한다.
12. 검색 결과에는 전체 Content 대신 Metadata/Summary를 우선 반환한다.
13. 실제 필요한 Section만 `guide.read`로 가져온다.
14. Guide Context를 Mandatory / Recommended / Reference로 구분한다.
15. Mandatory Rule은 Summary보다 원문 Canonical Content를 우선 제공한다.
16. Agent가 실제 사용한 Guide만 `GUIDE_REFERENCE`로 저장한다.
17. Design 시 사용한 Guide를 Review/Security 단계에서 재사용한다.
18. Guide Release를 사용해 검색 결과를 재현 가능하게 한다.
19. Work Item 시작 시 Guide Release를 Pinning하는 것을 기본으로 한다.
20. 새 Release는 Build → Validation → Activate 과정을 거친다.
21. Golden Query와 Mandatory Rule Test로 검색 품질을 회귀 검증한다.
22. 검색 결과에 Match Reason과 Provenance를 제공한다.
23. 검색 품질은 Precision을 Recall보다 우선한다.
24. No Result를 명확하게 반환하고 Agent가 Guide를 추측하지 못하게 한다.
25. Guide Conflict를 자동으로 임의 해결하지 않는다.
26. Alias/Technical Dictionary를 이용해 한글/영문 기술 용어를 연결한다.
27. Ranking Weight도 Versioning할 수 있도록 한다.
28. Retrieval Backend는 추상화하여 향후 OpenSearch/Hybrid Search로 교체 가능하게 한다.
29. Runtime Guide Repository는 직접 수정하지 않고 Curation → Publish로만 갱신한다.
30. 검색 사용 이력과 실제 Guide 채택 이력을 운영 지표로 활용한다.