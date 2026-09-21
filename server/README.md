# onCode Server

Java + Spring Boot modular monolith (ADR-001, PHASE_02).

```text
server/
└─ com.oncode.server
   ├─ api
   ├─ workflow
   ├─ context
   ├─ toolgateway
   ├─ agentruntime
   ├─ inferencegateway
   ├─ approval
   └─ persistence
```

Day-1: PostgreSQL SSOT, workflow transitions through WAITING_DESIGN_APPROVAL, context get/put. Agent bodies and gRPC are still stubs.

## Local PostgreSQL

CodeGEN(5434)과 분리된 compose: `infra/docker-compose.yml` → `127.0.0.1:5437`, DB/스키마/사용자 `oncode`.

```text
docker compose -f infra/docker-compose.yml up -d
```

상세는 `infra/README.md`.

## Requirements

- JDK 25 LTS (Eclipse Temurin 25.0.4.1+)
- Maven 3.6.3+

If `java -version` still shows 17, this session or user `JAVA_HOME` is pointing at Temurin 17. Use:

```text
C:\Program Files\Eclipse Adoptium\jdk-25.0.4.101-hotspot
```

## Build / test

onCode Maven 설정(`.mvn/settings.xml`)이 사용자 `~/.m2/settings.xml`의 로컬 Nexus 미러를 덮어쓴다. `server/` 또는 저장소 루트에서 실행한다.

```text
cd server
mvn test
```

저장소 루트에서:

```text
mvn -s .mvn/settings.xml -f server/pom.xml test
```

## Run

Postgres가 떠 있어야 한다 (`infra/docker-compose.yml`, `127.0.0.1:5437`). 저장소 루트에서:

```text
mvn -s .mvn/settings.xml -f server/pom.xml spring-boot:run
```

- HTTP: `http://localhost:18080/actuator/health` (8080은 이 PC의 HRE Keycloak)
- gRPC (Local Agent만): `127.0.0.1:9443`

로컬에서 에이전트·IDE까지 붙이는 순서는 [`docs/LOCAL_RUN.md`](../docs/LOCAL_RUN.md).

## Out of this slice

- Tool Gateway gRPC to Local Agent
- Senior / Implementation / Review agent bodies
- Inference Gateway real model calls
- Approval adapter completion
- JPA / Flyway / PostgreSQL
