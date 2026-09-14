# SPEC-09 onCode Guide Ingestion / Preprocessing / Curation Architecture v1

## 1. 목적

본 명세는 onCode에서 사용하는 개발 가이드 문서를 수집하고, 변환하고, 구조화하고, 분류하고, 요약하고, 검수하여 Guide Repository에 적재하기 위한 전처리 및 큐레이션 아키텍처를 정의한다.

본 시스템은 단순한 문서 업로드 기능이 아니다.

다음 전체 Lifecycle을 관리한다.

```text
원본 문서
   ↓
Format Detection
   ↓
Document Conversion
   ↓
Markdown / Structured Document 생성
   ↓
Document Normalization
   ↓
Section 분리
   ↓
Category 분류
   ↓
Summary 생성
   ↓
Keyword 생성
   ↓
Metadata 생성
   ↓
관리자 검수 / 수정
   ↓
Publish
   ↓
Guide Repository
```

---

# 2. 핵심 목표

Guide Ingestion 시스템의 목표는 다음과 같다.

1. 다양한 전자문서 형식을 폐쇄망에서 처리한다.
2. 문서의 원본 구조를 최대한 유지한다.
3. Markdown을 공통 편집/확인 포맷으로 사용한다.
4. 문서를 의미 있는 Guide Section 단위로 분리한다.
5. LLM을 이용하여 제목, 설명, 요약, 키워드 후보를 생성한다.
6. 자동 생성 결과를 관리자가 검수할 수 있도록 한다.
7. Vector DB 없이 RDBMS 검색에 최적화된 Metadata를 생성한다.
8. 원본 문서와 변환 문서의 추적성을 유지한다.
9. 재처리/Re-ingestion을 지원한다.
10. 게시된 Guide와 작업 중인 Guide를 분리한다.

---

# 3. 전체 아키텍처

```text
┌──────────────────── Guide Admin Console ───────────────────┐
│                                                           │
│ Upload                                                    │
│ Conversion Status                                         │
│ Document Preview                                          │
│ Category Editor                                           │
│ Section Editor                                            │
│ Summary Editor                                            │
│ Keyword Editor                                            │
│ Validation                                                │
│ Publish / Reprocess                                       │
│                                                           │
└───────────────────────────┬───────────────────────────────┘
                            │
                            ▼
┌──────────────── Guide Ingestion Service ──────────────────┐
│                                                           │
│ Ingestion Orchestrator                                    │
│                                                           │
│ ├─ File Type Detector                                     │
│ ├─ Conversion Router                                      │
│ │    ├─ Docling Direct Converter                          │
│ │    ├─ LibreOffice Converter                             │
│ │    └─ PDF Pipeline                                      │
│ │                                                         │
│ ├─ Document Normalizer                                    │
│ ├─ Structure Analyzer                                     │
│ ├─ Section Splitter                                       │
│ ├─ Category Classifier                                    │
│ ├─ Metadata Generator                                     │
│ │    ├─ Title Generator                                   │
│ │    ├─ Description Generator                             │
│ │    ├─ Summary Generator                                 │
│ │    └─ Keyword Extractor                                 │
│ │                                                         │
│ ├─ Validation Engine                                      │
│ └─ Publisher                                              │
│                                                           │
└──────────────┬──────────────────┬─────────────────────────┘
               │                  │
               ▼                  ▼
        Artifact Storage       RDBMS
        ├─ Original             ├─ Documents
        ├─ PDF                  ├─ Sections
        ├─ Markdown             ├─ Categories
        └─ Docling JSON         ├─ Keywords
                               └─ Versions
```

---

# 4. 지원 입력 문서

초기 지원 대상:

```text
PDF

DOC
DOCX

PPT
PPTX

XLS
XLSX

ODT
ODS
ODP

HWP
HWPX

Markdown

HTML

TXT

Image-based scanned documents
```

향후 필요에 따라:

```text
CSV
EPUB
Email
XML
```

등을 확장할 수 있다.

---

# 5. 변환 전략

모든 문서를 동일한 경로로 처리하지 않는다.

Format에 따라 Conversion Route를 결정한다.

기본 원칙:

```text
Docling이 안정적으로 직접 처리 가능한 문서
→ Direct Docling Conversion

Docling 직접 처리 불가 또는 품질이 낮은 문서
→ LibreOffice / 별도 Converter
→ PDF
→ Docling

이미 PDF인 문서
→ Docling PDF Pipeline
```

---

# 6. 권장 Conversion Routing

## 6.1 PDF

```text
PDF
 ↓
Docling
 ↓
Markdown + Docling JSON
```

---

## 6.2 DOC / DOCX

기본:

```text
DOC/DOCX
   ↓
Docling Direct
   ↓
Markdown + Structured JSON
```

Fallback:

```text
DOC/DOCX
   ↓
LibreOffice
   ↓
PDF
   ↓
Docling
```

---

## 6.3 PPT / PPTX

기본:

```text
PPT/PPTX
   ↓
Docling Direct
   ↓
Markdown + Structured JSON
```

레이아웃 보존 또는 변환 품질 문제가 있는 경우:

```text
PPT/PPTX
   ↓
LibreOffice PDF
   ↓
Docling
```

---

## 6.4 HWP / HWPX

현재 onCode 전처리 시스템에서는 다음 경로를 기본으로 한다.

```text
HWP / HWPX
      ↓
LibreOffice Conversion Container
      ↓
PDF
      ↓
Docling
      ↓
Markdown + Structured JSON
```

HWP/HWPX 전용 Parser가 향후 도입되더라도 이 경로는 fallback으로 유지할 수 있다.

---

# 7. HWP/HWPX Conversion Service

HWP 계열 문서는 별도 LibreOffice Container를 사용한다.

```text
Guide Ingestion Service
       ↓
Conversion Request
       ↓
LibreOffice Container
       ↓
PDF
```

Container 내부에서는 headless conversion을 수행한다.

개념:

```text
input.hwp
   ↓
LibreOffice Headless
   ↓
output.pdf
```

---

# 8. Conversion Service 분리

LibreOffice는 Main Ingestion Application 내부 라이브러리로 포함하지 않고 별도 Container/Worker로 분리한다.

장점:

```text
프로세스 격리

LibreOffice 장애 격리

파일 형식별 의존성 분리

Worker 수평 확장 가능

Timeout / Kill 처리 용이
```

---

# 9. Conversion Router

Conversion Router가 파일 형식 및 정책에 따라 Converter를 결정한다.

예:

```json
{
  "input_format": "HWPX",
  "route": [
    "LIBREOFFICE_PDF",
    "DOCLING_PDF"
  ]
}
```

DOCX:

```json
{
  "input_format": "DOCX",
  "route": [
    "DOCLING_DIRECT"
  ],
  "fallback": [
    "LIBREOFFICE_PDF",
    "DOCLING_PDF"
  ]
}
```

---

# 10. Direct Conversion 우선 이유

DOCX/PPTX 등을 무조건 PDF로 변환하면 원문에 존재하는 구조 정보가 손실될 수 있다.

예:

```text
Heading

Table Structure

Paragraph Hierarchy

List

Slide Object
```

따라서 직접 구조를 읽을 수 있는 포맷은 Direct Parse를 우선한다.

---

# 11. PDF 경유가 유리한 경우

다음 상황에서는 PDF intermediate를 사용할 수 있다.

```text
지원하지 않는 Format

문서 Renderer 결과가 중요한 경우

복잡한 HWP Layout

Direct Parsing Failure

Direct Parsing Quality가 낮은 경우
```

---

# 12. Docling Output

Docling 처리 결과는 최소 두 종류를 보관하는 것을 권장한다.

```text
Markdown

Docling Structured JSON
```

Markdown:

```text
Human Review
LLM Input
Section Editing
```

에 사용한다.

Structured JSON:

```text
Heading Hierarchy
Table
Page
Block
Document Structure
```

를 보존하기 위한 Canonical Conversion Artifact로 활용한다.

Docling은 현재 다양한 문서를 통합 표현으로 변환하고 Markdown 및 JSON 출력 등을 지원한다.

---

# 13. PDF Artifact

PDF를 중간 변환한 경우에도 삭제하지 않고 Artifact로 유지한다.

```text
Original HWP

Converted PDF

Docling JSON

Markdown
```

을 연결한다.

문제 발생 시 변환 과정을 추적할 수 있다.

---

# 14. Artifact 구조

예:

```text
guide-artifacts/

DOC-100/
├─ original/
│  └─ development-guide.hwpx
│
├─ converted/
│  └─ development-guide.pdf
│
├─ docling/
│  ├─ document.json
│  └─ document.md
│
└─ processing/
   └─ metadata.json
```

---

# 15. Document Entity

```text
document_id

original_filename

original_format

mime_type

original_hash

size

source

status

created_at

created_by
```

---

# 16. Document Processing Status

```text
UPLOADED

QUEUED

CONVERTING

CONVERTED

NORMALIZING

SPLITTING

ENRICHING

WAITING_REVIEW

PUBLISHED

FAILED

ARCHIVED
```

---

# 17. Pipeline State

보다 세부적으로 처리 Job을 분리한다.

```text
FORMAT_DETECTION

DOCUMENT_CONVERSION

MARKDOWN_GENERATION

NORMALIZATION

STRUCTURE_ANALYSIS

SECTION_SPLIT

CATEGORY_CLASSIFICATION

SUMMARY_GENERATION

KEYWORD_EXTRACTION

VALIDATION

REVIEW

PUBLISH
```

---

# 18. Ingestion Job

모든 처리 단계를 하나의 Job으로 관리한다.

```text
ingestion_job_id

document_id

pipeline_version

current_stage

status

retry_count

started_at

completed_at

error_code
```

---

# 19. Pipeline Version

전처리 알고리즘은 지속적으로 변경될 수 있으므로 Version을 저장한다.

예:

```text
guide-ingestion/1.0
```

또한 다음도 기록한다.

```text
docling_version

summary_model

prompt_version

classifier_version
```

---

# 20. Markdown Normalization

Docling 생성 Markdown을 그대로 최종 Guide로 사용하지 않는다.

Normalization 단계에서 다음을 처리한다.

```text
Heading Level 정리

불필요 Page Header 제거

Footer 제거

Page Number 제거

중복 문장 제거

공백 정리

List 정규화

Table 정규화

깨진 Line Break 정리
```

---

# 21. 원문 손실 최소화

Normalization은 의미 있는 내용을 삭제하는 작업이 아니어야 한다.

원본 Markdown도 반드시 보존한다.

```text
RAW_MARKDOWN

NORMALIZED_MARKDOWN
```

으로 분리한다.

---

# 22. Document Structure Analyzer

문서의 논리 구조를 분석한다.

예:

```text
Document
 ├─ Chapter
 │   ├─ Section
 │   │   ├─ Paragraph
 │   │   ├─ List
 │   │   └─ Table
```

Heading hierarchy를 최대한 우선한다.

---

# 23. Section Splitter

Guide 검색 단위가 될 Section을 생성한다.

단순 고정 token chunking을 기본 전략으로 사용하지 않는다.

우선순위:

```text
Heading

Subheading

Semantic Boundary

Paragraph Group

Table Boundary
```

---

# 24. Section Split 기본 원칙

하나의 Section은 가능한 한 하나의 독립된 개발 지침 또는 개념을 포함한다.

좋은 예:

```text
Spring Transaction 처리 기준
```

나쁜 예:

```text
문서 4~6페이지
```

---

# 25. Section 크기

크기 제한은 필요하지만 의미 구조보다 우선하지 않는다.

예:

```text
target size:
500 ~ 2,000 tokens
```

정도에서 운영 데이터에 따라 조정할 수 있다.

너무 큰 Section은 하위 Heading 기준으로 다시 분할한다.

---

# 26. Parent / Child Section

계층 구조를 유지한다.

```text
Spring Guide
   ↓
Transaction
   ↓
Rollback Policy
```

DB:

```text
section_id

parent_section_id

depth
```

---

# 27. Section Metadata

각 Section에는 다음 정보를 저장한다.

```text
section_id

document_id

parent_section_id

sequence

heading_path

source_page

content

content_hash

token_count

status
```

---

# 28. Heading Path

예:

```text
Spring Framework
 >
Transaction Management
 >
Rollback Policy
```

검색 결과의 Context 설명에 유용하다.

---

# 29. Page Provenance

PDF 기반 문서에서는 원문 페이지 정보를 가능한 한 유지한다.

예:

```text
page_start = 42
page_end = 44
```

향후 사용자가:

```text
"이 가이드 원문 어디야?"
```

라고 물었을 때 근거를 제공할 수 있다.

---

# 30. Category Model

문서를 카테고리별로 구조화한다.

초기 Category 예:

```text
LANGUAGE

FRAMEWORK

SECURITY

ARCHITECTURE

CODING_STANDARD

DATABASE

TEST

BUILD

DEPLOYMENT

UI

API

ERROR_HANDLING
```

---

# 31. Technology Category

별도 Technology Dimension도 두는 것이 좋다.

예:

```text
JAVA

PYTHON

JAVASCRIPT

TYPESCRIPT

SPRING

SPRING_BOOT

EGOVFRAMEWORK

REACT

VUE

NODEJS
```

즉:

```text
Category
≠
Technology
```

로 분리한다.

---

# 32. Hierarchical Category

eGovFramework의 llms.txt와 유사한 계층형 구조를 지원한다.

예:

```text
EGOVFRAMEWORK
 ├─ RUNTIME
 │   ├─ PRESENTATION
 │   ├─ BUSINESS
 │   └─ PERSISTENCE
 │
 └─ COMMON_COMPONENT
```

---

# 33. Category Assignment

Section마다 하나 이상의 Category를 지정할 수 있다.

예:

```text
Technology:
SPRING

Category:
SECURITY
AUTHENTICATION
```

---

# 34. Category 분류 방식

자동 분류:

```text
Heading
+
Content
+
Document Metadata
+
LLM
```

를 사용한다.

자동 결과는 관리자가 수정할 수 있다.

---

# 35. Category Confidence

예:

```json
{
  "category": "SECURITY",
  "confidence": 0.94
}
```

낮은 Confidence는 Review 대상으로 표시한다.

---

# 36. Metadata Enrichment

Section 분리 후 LLM을 이용하여 Metadata를 생성한다.

생성 대상:

```text
Display Title

Short Description

Summary

Keywords

Technology

Category Suggestions

Applicable Version

Rule Type
```

---

# 37. Title

원문 Heading이 충분히 명확하면 그대로 사용한다.

그렇지 않을 경우 LLM이 사용자 검색에 적합한 제목을 제안한다.

예:

원문:

```text
4.2.1 처리 방법
```

보정:

```text
Spring Transaction 예외 발생 시 Rollback 처리
```

---

# 38. Description

검색 결과에서 사용하기 위한 짧은 설명이다.

예:

```text
Spring Transaction에서 Checked Exception과 Runtime Exception에 따른 Rollback 처리 기준을 설명한다.
```

---

# 39. Summary

Section 전체를 압축한 검색/LLM용 요약이다.

원칙:

```text
원문에 없는 규칙을 추가하지 않는다.

MUST / SHOULD 의미를 변경하지 않는다.

수치/버전 정보를 임의 변경하지 않는다.

금지 조건을 완화하지 않는다.
```

---

# 40. Search Summary와 Display Summary

필요하다면 두 종류로 분리할 수 있다.

```text
display_summary
→ 관리자/사용자 표시

search_summary
→ 검색 recall 개선
```

MVP에서는 하나의 Summary로 시작해도 된다.

---

# 41. Keyword Extraction

LLM이 Section에서 검색 Keyword 후보를 추출한다.

예:

```text
@Transactional

rollback

RuntimeException

checked exception

transaction management
```

---

# 42. Keyword Type

Keyword를 단순 문자열 배열보다 유형화하면 검색 품질이 좋아진다.

```text
TECHNOLOGY

FRAMEWORK

CLASS

ANNOTATION

API

CONCEPT

ERROR

SECURITY

GENERAL
```

---

# 43. Keyword 예

```json
[
  {
    "value": "@Transactional",
    "type": "ANNOTATION",
    "weight": 1.0
  },
  {
    "value": "rollback",
    "type": "CONCEPT",
    "weight": 0.9
  }
]
```

---

# 44. Synonym / Alias

검색 개선을 위해 Alias도 관리할 수 있다.

예:

```text
전자정부프레임워크
eGovFrame
eGovFramework
egov
```

또는:

```text
트랜잭션
transaction
@Transactional
```

---

# 45. Keyword 자동 생성과 관리자 편집

LLM 출력:

```text
Suggested Keywords
```

관리자가:

```text
추가
삭제
Weight 변경
```

가능해야 한다.

---

# 46. Rule Type

Guide Section이 규칙 성격을 갖는 경우 별도 Metadata를 둔다.

```text
REFERENCE

RECOMMENDED

REQUIRED

SECURITY_REQUIRED

PROHIBITED
```

---

# 47. Enforcement

예:

```text
REQUIRED
→ Code Review에서 강제 검증

SECURITY_REQUIRED
→ Security Review에서 강제 검증

REFERENCE
→ 참고
```

SPEC-08 Policy Model과 연계한다.

---

# 48. Version Metadata

가이드가 특정 Framework 버전에 적용될 수 있다.

예:

```text
Spring Boot 3.x

Java 17+

eGovFramework 4.3
```

Section에:

```text
applies_to_version

min_version

max_version
```

등을 둘 수 있다.

---

# 49. Source Metadata

각 Guide는 원본 출처를 기록한다.

```text
Organization

Project

Framework Official

External Standard
```

---

# 50. Guide Priority

SPEC-08에서 정의한 우선순위와 연계한다.

```text
PROJECT_RULE

ORGANIZATION_RULE

FRAMEWORK_RULE

LANGUAGE_RULE

GENERAL_RULE
```

Document 또는 Section 수준으로 지정할 수 있다.

---

# 51. Admin Console

별도의 Web Console을 구현한다.

주요 화면:

```text
Document Upload

Ingestion Jobs

Conversion Monitor

Document Preview

Section Editor

Category Editor

Keyword Editor

Summary Editor

Validation

Publishing

Version History
```

---

# 52. Document Upload 화면

지원 기능:

```text
Drag & Drop

Multiple Upload

Document Type 확인

Source 설정

Guide Priority 설정

Technology 지정

Processing 시작
```

---

# 53. Processing Monitor

예:

```text
development-guide.hwpx

✓ Upload
✓ HWPX → PDF
✓ PDF → Markdown
✓ Structure Analysis
● Section Splitting
○ Metadata Generation
○ Review
```

---

# 54. Failed Job

실패 시:

```text
Stage

Error

Retry

Download Artifact

View Log
```

를 제공한다.

---

# 55. Document Preview

가능하면 세 화면을 제공한다.

```text
Original/PDF Preview

Normalized Markdown

Structured Sections
```

관리자가 변환 품질을 비교할 수 있어야 한다.

---

# 56. Section Editor

왼쪽:

```text
Document Tree
```

중앙:

```text
Section Content
```

오른쪽:

```text
Metadata
```

구조를 권장한다.

---

# 57. Section Merge

자동 분할이 지나치게 세분화된 경우 관리자가:

```text
Merge with Previous

Merge with Next
```

할 수 있어야 한다.

---

# 58. Section Split

반대로 하나의 Section이 너무 큰 경우:

```text
Split Here
```

기능을 지원한다.

---

# 59. Content Edit

관리자가 Markdown Content 자체를 수정할 수 있는지 정책 결정이 필요하다.

권장:

```text
Normalized content 수정 가능
+
Original artifact 유지
```

수정 이력은 Versioning한다.

---

# 60. Metadata Editor

편집 항목:

```text
Title

Description

Summary

Technology

Category

Keywords

Priority

Rule Type

Applicable Versions
```

---

# 61. AI Regenerate

관리자가 다음 항목만 선택해서 재생성할 수 있다.

```text
Regenerate Title

Regenerate Summary

Regenerate Keywords

Reclassify
```

전체 Pipeline을 다시 돌릴 필요가 없다.

---

# 62. Validation

Publish 전 자동 검증한다.

예:

```text
제목 없음

Content 없음

Category 없음

Summary 없음

Duplicate Content

Too Large Section

Invalid Parent

Broken Markdown

Secret 포함
```

---

# 63. Duplicate Detection

같은 문서가 반복 업로드될 수 있다.

우선:

```text
Original File Hash
```

로 Exact Duplicate를 검출한다.

---

# 64. Near Duplicate

같은 내용의 다른 파일도 존재할 수 있다.

Vector DB 없이:

```text
Normalized Content Hash

SimHash

MinHash

Trigram similarity
```

등을 검토할 수 있다.

MVP에서는 Exact Hash + Title/Content similarity 정도로 시작한다.

---

# 65. Secret Scanner

가이드 문서에도 실제 Password/Key가 포함될 수 있다.

Publish 전 검사한다.

```text
API Key

Password

Private Key

Credential
```

탐지 시 Review 경고 또는 Block.

---

# 66. Publish Model

처리 완료와 검색 가능 상태를 분리한다.

```text
DRAFT

REVIEW_REQUIRED

APPROVED

PUBLISHED

ARCHIVED
```

Guide Search는 기본적으로 `PUBLISHED`만 조회한다.

---

# 67. Publish Approval

가이드 중요도에 따라 게시 승인 절차를 둘 수 있다.

초기:

```text
Guide Administrator
→ Publish
```

향후:

```text
Editor
→ Reviewer
→ Publish
```

확장 가능하다.

---

# 68. Versioning

동일 문서가 업데이트되면 이전 버전을 덮어쓰지 않는다.

```text
Guide Document v1
Guide Document v2
```

Section도 Version 관계를 유지한다.

---

# 69. Effective Date

정책성 가이드에는:

```text
effective_from

effective_to
```

를 둘 수 있다.

Search는 현재 유효한 Guide를 우선한다.

---

# 70. Superseded

새 가이드가 기존 가이드를 대체하는 경우:

```text
SUPERSEDES
```

관계를 저장한다.

예:

```text
GUIDE-200 v2
→ supersedes
GUIDE-200 v1
```

---

# 71. Re-ingestion

전처리 알고리즘이 개선되면 기존 원본에서 다시 처리할 수 있어야 한다.

```text
Original Artifact
      ↓
Re-ingestion
      ↓
New Processing Version
```

---

# 72. Re-ingestion과 Content Edit

관리자가 직접 편집한 내용이 있다면 자동 재처리로 덮어쓰면 안 된다.

다음 상태를 구분한다.

```text
AUTO_GENERATED

MANUALLY_EDITED
```

---

# 73. Re-ingestion Diff

기존 Published Guide와 새 처리 결과를 비교한다.

```text
Old Section

New Section

Metadata Difference
```

관리자가 변경 내용을 확인 후 Publish한다.

---

# 74. Conversion Quality Score

문서 변환 품질을 정량화할 수 있다.

예:

```text
Parser Success

Heading Count

Broken Character Ratio

Empty Page Ratio

OCR Confidence

Table Parse Failure
```

---

# 75. Low Quality Processing

품질 기준 이하:

```text
NEEDS_MANUAL_REVIEW
```

상태로 보낸다.

자동 Publish하지 않는다.

---

# 76. OCR

PDF가 이미지 기반인 경우 Docling OCR 기능을 사용할 수 있다.

Docling은 PDF/이미지 처리에서 OCR 옵션을 지원한다.

OCR 사용 여부는 문서 특성을 보고 결정한다.

---

# 77. OCR 전략

```text
Text PDF
→ OCR OFF 또는 기본

Scanned PDF
→ OCR ON

Mixed PDF
→ 필요 Page OCR
```

---

# 78. Table 처리

개발 가이드에는 표가 중요할 수 있다.

예:

```text
지원 버전 표

설정값 표

규칙 비교표
```

Markdown 변환 시 Table 구조를 최대한 유지한다.

---

# 79. Code Block 처리

코드 예제가 깨지지 않도록 보호한다.

```text
Java

XML

YAML

JSON

Shell

SQL
```

Code Fence를 유지한다.

---

# 80. 코드와 설명 분리

Section Metadata에서 Code Block을 별도로 추출할 수도 있다.

예:

```text
has_code = true

languages = JAVA, XML
```

검색 시 유용하다.

---

# 81. Image 처리

이미지 자체를 Guide 검색에 직접 활용하지 않는다면:

```text
Caption

Alt Text

Nearby Description
```

정도만 Markdown에 유지할 수 있다.

중요 Architecture Diagram은 향후 별도 분석 기능으로 확장할 수 있다.

---

# 82. Broken Conversion

한글 깨짐, 표 붕괴, 이미지 누락 등이 발견되면 관리자가 다른 Conversion Route를 선택할 수 있어야 한다.

예:

```text
Retry with:
○ Docling Direct
○ LibreOffice → PDF → Docling
○ OCR
```

---

# 83. Manual Route Override

관리자가 특정 문서의 변환 Route를 강제로 지정할 수 있다.

```text
AUTO

DOCLING_DIRECT

LIBREOFFICE_PDF

PDF_OCR
```

---

# 84. Conversion Route 기록

Artifact에 반드시 기록한다.

예:

```json
{
  "route": [
    {
      "step": "LIBREOFFICE",
      "from": "HWPX",
      "to": "PDF"
    },
    {
      "step": "DOCLING",
      "from": "PDF",
      "to": "MARKDOWN"
    }
  ]
}
```

---

# 85. Worker Architecture

변환/LLM 작업은 Web Application Process에서 직접 하지 않는다.

권장:

```text
Admin API
   ↓
Job Queue
   ↓
Worker
```

Worker 종류:

```text
Conversion Worker

Docling Worker

LLM Enrichment Worker

Validation Worker
```

---

# 86. Job Queue

폐쇄망 내부 Message Broker 또는 DB Queue를 사용할 수 있다.

중요한 것은 처리 상태를 persistent하게 유지하는 것이다.

---

# 87. Worker Retry

재시도 가능한 오류:

```text
Temporary Worker Failure

Conversion Timeout

LLM Timeout
```

재시도 불가:

```text
Unsupported Format

Corrupted File

Password Protected Document
```

---

# 88. Timeout

LibreOffice나 Docling 변환은 문서에 따라 Hang 가능성이 있으므로 반드시 timeout과 process kill을 지원한다.

---

# 89. Resource Isolation

LibreOffice Worker와 Docling Worker는 Container 기반 분리를 권장한다.

특히 PDF 처리와 OCR은 CPU/GPU/Memory 사용량이 클 수 있으므로 동시 실행 수를 제한한다.

---

# 90. Antivirus / File Validation

폐쇄망이라도 업로드 문서는 untrusted input으로 취급한다.

가능하면:

```text
File Signature Check

MIME Verification

Size Limit

Archive Bomb Check

Malware Scan
```

등을 적용한다.

---

# 91. Password-protected Document

암호화 문서는 자동 처리하지 않는다.

상태:

```text
PASSWORD_REQUIRED
```

관리자에게 안내한다.

암호 자체를 장기 저장하지 않는다.

---

# 92. DB 저장 모델

핵심 Entity:

```text
GUIDE_DOCUMENT

GUIDE_DOCUMENT_VERSION

INGESTION_JOB

DOCUMENT_ARTIFACT

GUIDE_SECTION

GUIDE_SECTION_VERSION

CATEGORY

SECTION_CATEGORY

KEYWORD

SECTION_KEYWORD

TECHNOLOGY

SECTION_TECHNOLOGY

GUIDE_RELATION

PROCESSING_LOG
```

---

# 93. GUIDE_DOCUMENT

```text
document_id

name

source_type

priority

status

owner

created_at
```

---

# 94. GUIDE_DOCUMENT_VERSION

```text
document_version_id

document_id

version

original_artifact_id

pipeline_version

effective_from

effective_to

status
```

---

# 95. GUIDE_SECTION

논리적 Guide Section 식별자.

```text
section_id

document_id

parent_section_id

canonical_key
```

---

# 96. GUIDE_SECTION_VERSION

```text
section_version_id

section_id

document_version_id

version

title

description

summary

content

heading_path

sequence

page_start

page_end

rule_type

status

content_hash
```

---

# 97. KEYWORD

```text
keyword_id

normalized_value

display_value

keyword_type
```

---

# 98. SECTION_KEYWORD

```text
section_version_id

keyword_id

weight

source
```

Source:

```text
LLM

ADMIN

AUTO_EXTRACT
```

---

# 99. CATEGORY

```text
category_id

parent_category_id

code

name

description

sequence

active
```

계층 구조 지원.

---

# 100. Technology

```text
technology_id

code

name

aliases

version_scheme
```

---

# 101. Processing Metadata

LLM 생성 결과의 출처를 기록한다.

예:

```text
generated_by_model

prompt_version

generated_at

confidence
```

관리자 수정 후에는:

```text
edited_by

edited_at
```

을 기록한다.

---

# 102. Provenance

Guide Section의 최종 내용이 어디에서 왔는지 추적 가능해야 한다.

```text
Original File
   ↓
Converted PDF
   ↓
Docling Markdown
   ↓
Normalized Markdown
   ↓
Section
   ↓
Admin Edited Section
```

---

# 103. Admin Audit

다음 활동을 기록한다.

```text
Document Upload

Conversion Retry

Section Edit

Category Change

Keyword Change

Summary Regenerate

Publish

Archive
```

---

# 104. Guide Search와 Ingestion 분리

중요한 원칙:

```text
Ingestion DB/Workflow
≠
Runtime Guide Search
```

Ingestion이 처리 중이어도 Published Guide 검색에는 영향을 주지 않는다.

---

# 105. Publishing

Publisher가 검수된 Guide를 Runtime Guide Repository로 반영한다.

```text
Curation DB
     ↓
Publisher
     ↓
Runtime Guide Repository
```

동일 DB를 사용하더라도 Schema/Status를 논리적으로 분리하는 것이 좋다.

---

# 106. Publish Snapshot

Publish 시점에 Immutable Snapshot을 만드는 것을 권장한다.

예:

```text
GUIDE_RELEASE-20260909-01
```

Runtime 검색은 특정 Release를 기준으로 동작할 수 있다.

---

# 107. Release 장점

```text
검색 결과 재현 가능

Rollback 가능

Agent 실행 시 사용한 Guide Set 추적 가능

일괄 배포 가능
```

---

# 108. Runtime Guide Reference

Agent가 Guide를 사용하면 다음을 기록할 수 있다.

```text
guide_release_id

section_version_id
```

따라서 이후 동일 근거를 재현할 수 있다.

---

# 109. 폐쇄망 고려

전 처리 Worker에서 사용하는 다음 자원은 사전 반입해야 한다.

```text
Docling Model

OCR Model

LLM Model

Tokenizer

Python Package

LibreOffice Image
```

Runtime Download를 전제로 하지 않는다.

---

# 110. Model Registry

전처리에 사용하는 내부 모델은 Version을 명시적으로 관리한다.

예:

```text
summary-model/1

keyword-model/2

classification-model/1
```

---

# 111. LLM Prompt Version

다음 Prompt를 별도 관리한다.

```text
TITLE_GENERATION

SUMMARY_GENERATION

KEYWORD_EXTRACTION

CATEGORY_CLASSIFICATION
```

모두 Versioning한다.

---

# 112. LLM Structured Output

Metadata Generator는 JSON Schema 기반 결과를 반환한다.

예:

```json
{
  "title": "...",
  "description": "...",
  "summary": "...",

  "keywords": [
    {
      "value": "@Transactional",
      "type": "ANNOTATION",
      "weight": 1.0
    }
  ],

  "categories": [
    {
      "code": "TRANSACTION",
      "confidence": 0.97
    }
  ]
}
```

---

# 113. Hallucination Validation

LLM이 생성한 Metadata를 원문과 검증한다.

특히:

```text
Version

API Name

Class Name

Annotation

Numeric Value
```

등이 원문에 없는 경우 경고할 수 있다.

---

# 114. Keyword Validation

가능하면 다음 Keyword는 원문/구조 데이터와 일치 여부를 확인한다.

```text
Class Name

Method

Annotation

Package

API

Framework Version
```

---

# 115. Summary Quality Check

Summary가 원문 대비 지나치게 길거나 짧은 경우 재생성 대상이 될 수 있다.

또한 Mandatory/Prohibited 의미가 반전되지 않았는지 검증한다.

---

# 116. Processing Configuration

예:

```yaml
ingestion:

  conversion:
    direct_docling:
      enabled: true

    libreoffice_fallback:
      enabled: true

  section:
    target_tokens: 1200
    max_tokens: 2400

  enrichment:
    title: true
    summary: true
    keywords: true
    classification: true

  review:
    auto_publish: false
```

---

# 117. Format Route Config

```yaml
formats:

  pdf:
    route:
      - docling

  docx:
    route:
      - docling
    fallback:
      - libreoffice_pdf
      - docling

  pptx:
    route:
      - docling
    fallback:
      - libreoffice_pdf
      - docling

  hwp:
    route:
      - libreoffice_pdf
      - docling

  hwpx:
    route:
      - libreoffice_pdf
      - docling
```

---

# 118. Admin Console MVP

초기 Console에서는 다음 기능을 먼저 구현한다.

```text
Upload

Document List

Processing Status

Retry

Markdown Preview

Section Tree

Section Split/Merge

Title Edit

Description Edit

Summary Edit

Category Assignment

Keyword Edit

Publish
```

---

# 119. MVP Conversion

초기 지원 우선순위:

```text
PDF

DOCX

PPTX

HWP/HWPX

Markdown
```

필요에 따라 XLSX, HTML 등을 후속 지원한다.

---

# 120. MVP Metadata

MVP에서는 Section마다 최소 다음을 생성한다.

```text
Title

Description

Summary

Technology

Category

Keywords

Priority

Source Reference
```

---

# 121. MVP에서 제외 가능

```text
Near Duplicate 고급 탐지

Architecture Diagram 분석

Image Semantic Extraction

자동 Multi-level Approval

고급 Rule Extraction

Semantic Relation Graph

Fully Automatic Publish
```

---

# 122. 권장 구현 모듈

```text
guide-ingestion/

├─ api/
├─ jobs/
├─ conversion/
│  ├─ router
│  ├─ docling
│  ├─ libreoffice
│  └─ pdf
│
├─ normalization/
├─ structure/
├─ splitter/
├─ classification/
├─ enrichment/
│  ├─ title
│  ├─ summary
│  └─ keyword
│
├─ validation/
├─ publishing/
├─ artifact/
└─ persistence/
```

Admin:

```text
guide-admin-web/

├─ documents
├─ jobs
├─ preview
├─ sections
├─ metadata
├─ validation
└─ publishing
```

---

# 123. 대표 HWPX Flow

```text
Administrator
      ↓
Upload development-guide.hwpx
      ↓
File Validation
      ↓
HWPX Route Detection
      ↓
LibreOffice Worker
      ↓
PDF
      ↓
Docling Worker
      ↓
Markdown + JSON
      ↓
Normalization
      ↓
Section Split
      ↓
Category Classification
      ↓
Summary / Keyword Generation
      ↓
WAITING_REVIEW
      ↓
Admin Console
      ↓
Edit / Approve
      ↓
Publish
      ↓
Runtime Guide Repository
```

---

# 124. 대표 DOCX Flow

```text
Upload guide.docx
     ↓
Docling Direct
     ↓
Markdown + JSON
     ↓
Quality Check
     │
     ├─ PASS
     │     ↓
     │  Normalize
     │
     └─ FAIL
           ↓
       LibreOffice
           ↓
          PDF
           ↓
        Docling
```

---

# 125. Runtime과 전처리 서버 분리

가능하다면 다음을 논리적으로 분리한다.

```text
onCode Runtime Server
→ 개발자 요청 처리

Guide Ingestion Server
→ 문서 변환 및 큐레이션
```

문서 변환은 자원 사용량과 장애 특성이 Runtime Agent 처리와 다르기 때문이다.

---

# 126. Resource Isolation

특히 Docling/OCR/LibreOffice 처리 Worker가 Runtime LLM Resource와 경쟁하지 않도록 Resource Limit을 두는 것을 권장한다.

---

# 127. 운영 모니터링

지표:

```text
Uploaded Documents

Conversion Success Rate

Average Processing Time

Failed Jobs

Manual Review Count

Published Sections

Average Sections per Document

LLM Metadata Failure Rate
```

---

# 128. 핵심 설계 결정

onCode Guide Ingestion / Preprocessing / Curation Architecture v1의 핵심 결정은 다음과 같다.

1. Guide Repository에 앞서 독립적인 Ingestion/Curation Pipeline을 둔다.
2. Markdown을 공통 편집 포맷으로 사용한다.
3. Docling Structured JSON도 함께 보관하여 구조 정보를 잃지 않는다.
4. DOCX/PPTX 등 Docling이 직접 처리 가능한 문서는 Direct Conversion을 우선한다.
5. HWP/HWPX는 LibreOffice → PDF → Docling 경로를 기본으로 한다.
6. Direct Conversion 실패 시 PDF 기반 fallback을 지원한다.
7. Original, PDF, Markdown, Structured JSON을 Artifact로 추적한다.
8. 변환 Worker는 Main Web Process와 분리한다.
9. 문서는 Heading/Semantic Structure 기반 Section으로 분리한다.
10. Fixed Token Chunking은 보조 수단으로만 사용한다.
11. Category와 Technology를 별도 Dimension으로 관리한다.
12. LLM을 Title/Description/Summary/Keyword/Category 후보 생성에 사용한다.
13. LLM 결과는 자동 확정하지 않고 Admin Console에서 검수 가능해야 한다.
14. Section Split/Merge 및 Metadata 수정을 지원한다.
15. 원본과 수정된 Guide Content 모두 Versioning한다.
16. Re-ingestion 시 관리자 수동 수정 내용을 보호한다.
17. RDBMS 검색에 필요한 Keyword, Summary, Category, Alias를 명시적으로 생성한다.
18. Guide Section에 Priority와 Rule Type을 부여한다.
19. Mandatory/Security Guide는 SPEC-08 Policy Model과 연계한다.
20. Published 상태의 Guide만 Runtime 검색 대상이 된다.
21. Publish 시 Guide Release Snapshot을 생성할 수 있도록 한다.
22. Runtime Agent가 사용한 Guide Release와 Section Version을 추적한다.
23. 폐쇄망에서 Docling/OCR/LLM Model의 Runtime 다운로드를 금지하고 사전 반입한다.
24. 전처리 Pipeline/Model/Prompt/Converter Version을 모두 기록한다.
25. 변환 품질이 낮은 문서는 자동 Publish하지 않는다.
26. 전처리와 Runtime Guide Search를 서로 독립적으로 운영한다.