-- 워크플로 SSOT 최소 테이블. 스키마 oncode는 Flyway default-schema가 만든다.

CREATE TABLE work_item (
  work_item_id     text        PRIMARY KEY,
  title            text,
  original_request text        NOT NULL,
  work_item_type   text        NOT NULL,
  status           text        NOT NULL,
  created_at       timestamptz NOT NULL,
  updated_at       timestamptz NOT NULL
);

COMMENT ON TABLE work_item IS '개발자 요청 단위. 워크플로보다 수명이 길다.';
COMMENT ON COLUMN work_item.work_item_id IS '워크 아이템 식별자 (WI-...).';
COMMENT ON COLUMN work_item.title IS '요청 요약. 없으면 null.';
COMMENT ON COLUMN work_item.original_request IS '사용자가 보낸 원문 요청.';
COMMENT ON COLUMN work_item.work_item_type IS '요청 유형. 1일차는 IMPLEMENTATION.';
COMMENT ON COLUMN work_item.status IS 'OPEN / COMPLETED / CANCELLED.';
COMMENT ON COLUMN work_item.created_at IS '생성 시각 (UTC).';
COMMENT ON COLUMN work_item.updated_at IS '마지막 변경 시각 (UTC).';

CREATE TABLE workflow (
  workflow_id     text        PRIMARY KEY,
  work_item_id    text        NOT NULL,
  workflow_type   text        NOT NULL,
  current_state   text        NOT NULL,
  previous_state  text,
  resume_state    text,
  revision        integer     NOT NULL,
  status          text        NOT NULL,
  retry_count     integer     NOT NULL,
  lease_owner     text,
  lease_until     timestamptz,
  created_at      timestamptz NOT NULL,
  updated_at      timestamptz NOT NULL,
  CONSTRAINT fk_workflow_work_item_id
    FOREIGN KEY (work_item_id) REFERENCES work_item (work_item_id),
  CONSTRAINT uq_workflow_work_item_id UNIQUE (work_item_id)
);

COMMENT ON TABLE workflow IS 'Work Item의 공식 실행 상태. 프로세스 메모리가 아니라 이 행이 SSOT다.';
COMMENT ON COLUMN workflow.workflow_id IS '워크플로 식별자 (WF-...).';
COMMENT ON COLUMN workflow.work_item_id IS '부모 워크 아이템.';
COMMENT ON COLUMN workflow.workflow_type IS '워크플로 유형. 1일차는 IMPLEMENTATION.';
COMMENT ON COLUMN workflow.current_state IS '현재 공식 상태. LLM 출력이 이 값을 바꾸지 않는다.';
COMMENT ON COLUMN workflow.previous_state IS '직전 공식 상태.';
COMMENT ON COLUMN workflow.resume_state IS '재개 시 참고 상태. 없으면 current_state를 쓴다.';
COMMENT ON COLUMN workflow.revision IS '낙관적 잠금 버전. 전이마다 1 증가.';
COMMENT ON COLUMN workflow.status IS 'RUNNING / WAITING / COMPLETED / FAILED.';
COMMENT ON COLUMN workflow.retry_count IS '재시도 횟수. 0부터 증가.';
COMMENT ON COLUMN workflow.lease_owner IS '실행 리스를 가진 서버 식별자. WAITING_*에서는 null.';
COMMENT ON COLUMN workflow.lease_until IS '리스 만료 시각 (UTC). 만료되면 다른 노드가 회수한다.';
COMMENT ON COLUMN workflow.created_at IS '생성 시각 (UTC).';
COMMENT ON COLUMN workflow.updated_at IS '마지막 변경 시각 (UTC).';

CREATE INDEX idx_workflow_current_state ON workflow (current_state);
CREATE INDEX idx_workflow_lease_until ON workflow (lease_until);

CREATE TABLE workflow_transition (
  transition_id  text        PRIMARY KEY,
  workflow_id    text        NOT NULL,
  from_state     text,
  to_state       text        NOT NULL,
  trigger_type   text        NOT NULL,
  trigger_id     text,
  reason         text,
  created_at     timestamptz NOT NULL,
  CONSTRAINT fk_workflow_transition_workflow_id
    FOREIGN KEY (workflow_id) REFERENCES workflow (workflow_id)
);

COMMENT ON TABLE workflow_transition IS '공식 상태 전이 이력. 한 행이 한 번의 허용된 전이다.';
COMMENT ON COLUMN workflow_transition.transition_id IS '전이 식별자 (TR-...). SPEC-03 transition_id.';
COMMENT ON COLUMN workflow_transition.workflow_id IS '전이가 속한 워크플로.';
COMMENT ON COLUMN workflow_transition.from_state IS '출발 상태. 최초 RECEIVED 진입은 null.';
COMMENT ON COLUMN workflow_transition.to_state IS '도착 상태.';
COMMENT ON COLUMN workflow_transition.trigger_type IS '전이 원인. 1일차는 ENGINE 또는 SYSTEM.';
COMMENT ON COLUMN workflow_transition.trigger_id IS '원인 상관 식별자. 없으면 null.';
COMMENT ON COLUMN workflow_transition.reason IS '전이를 설명하는 짧은 이유.';
COMMENT ON COLUMN workflow_transition.created_at IS '전이 시각 (UTC).';

CREATE INDEX idx_workflow_transition_workflow_id ON workflow_transition (workflow_id);

CREATE TABLE context_entry (
  context_ref   text        PRIMARY KEY,
  work_item_id  text        NOT NULL,
  ref_type      text        NOT NULL,
  payload       jsonb       NOT NULL,
  created_at    timestamptz NOT NULL,
  updated_at    timestamptz NOT NULL,
  CONSTRAINT fk_context_entry_work_item_id
    FOREIGN KEY (work_item_id) REFERENCES work_item (work_item_id)
);

COMMENT ON TABLE context_entry IS 'Context Storage 최소 행. 에이전트는 SQL이 아니라 context_ref로만 조회한다.';
COMMENT ON COLUMN context_entry.context_ref IS '컨텍스트 참조 URI (ctx://...).';
COMMENT ON COLUMN context_entry.work_item_id IS '이 컨텍스트가 속한 워크 아이템.';
COMMENT ON COLUMN context_entry.ref_type IS '컨텍스트 종류. 예: DESIGN, REQUIREMENT.';
COMMENT ON COLUMN context_entry.payload IS '구조화된 JSON. hidden reasoning을 넣지 않는다.';
COMMENT ON COLUMN context_entry.created_at IS '최초 저장 시각 (UTC).';
COMMENT ON COLUMN context_entry.updated_at IS '마지막 put 시각 (UTC).';

CREATE INDEX idx_context_entry_work_item_id ON context_entry (work_item_id);
