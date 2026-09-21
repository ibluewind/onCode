# onCode local infra

로컬 PostgreSQL만 둔다. CodeGEN `codegen-postgres`(127.0.0.1:5434) 및 HRE Postgres(5433/5435/5436)와 **볼륨·포트·계정·DB를 공유하지 않는다**. Nexus는 포함하지 않는다.

운영/HA 배포는 이후 `deployment/` (SPEC-14). 이 디렉터리는 개발자 PC용이다.

## Postgres

| 항목 | 값 |
|------|-----|
| 이미지 | `postgres:16-alpine` |
| 호스트 | `127.0.0.1:5437` |
| DB | `oncode` |
| 스키마 | `oncode` (`search_path` 기본) |
| 사용자 / 비밀번호 | `oncode` / `oncode` (로컬 전용) |
| 볼륨 | Docker volume `oncode-pg-data` |

JDBC:

```text
jdbc:postgresql://127.0.0.1:5437/oncode?currentSchema=oncode
```

## 기동

저장소 루트에서:

```text
docker compose -f infra/docker-compose.yml up -d
docker compose -f infra/docker-compose.yml ps
```

중지:

```text
docker compose -f infra/docker-compose.yml down
```

데이터를 지우려면 `down -v` (볼륨 `oncode-pg-data` 삭제).

`postgres/init/` 스크립트는 **빈 볼륨 최초 기동**에만 적용된다. 스키마를 바꾼 뒤에는 볼륨을 지우거나 마이그레이션으로 맞춘다.

## 테스트

서버 통합 테스트는 이 compose에 묶지 않고, 이후 Testcontainers로 격리하는 것을 권장한다. compose는 `spring-boot:run` / 수동 확인용이다.
