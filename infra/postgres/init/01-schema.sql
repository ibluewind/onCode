-- onCode 공식 상태 스키마. public에 업무 테이블을 두지 않는다.
-- 이 스크립트는 볼륨이 비어 있을 때 한 번만 실행된다.

CREATE SCHEMA IF NOT EXISTS oncode;

COMMENT ON SCHEMA oncode IS 'onCode 워크플로·컨텍스트 공식 상태. 사용자 워크스페이스 파일이 아니다.';

ALTER ROLE oncode SET search_path TO oncode, public;

GRANT USAGE, CREATE ON SCHEMA oncode TO oncode;
