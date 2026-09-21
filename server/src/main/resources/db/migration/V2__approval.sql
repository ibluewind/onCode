-- 설계/코드 승인. 워크플로 상태와 별도 행이며 재사용하지 않는다.

CREATE TABLE approval (
  approval_id    text        PRIMARY KEY,
  work_item_id   text        NOT NULL,
  workflow_id    text        NOT NULL,
  approval_type  text        NOT NULL,
  resource_type  text        NOT NULL,
  resource_id    text        NOT NULL,
  status         text        NOT NULL,
  requested_at   timestamptz NOT NULL,
  responded_at   timestamptz,
  approved_by    text,
  decision       text,
  reason         text,
  expires_at     timestamptz,
  is_consumed    boolean     NOT NULL,
  consumed_at    timestamptz,
  CONSTRAINT fk_approval_work_item_id
    FOREIGN KEY (work_item_id) REFERENCES work_item (work_item_id),
  CONSTRAINT fk_approval_workflow_id
    FOREIGN KEY (workflow_id) REFERENCES workflow (workflow_id)
);

COMMENT ON TABLE approval IS 'Human-in-the-loop 승인. 설계와 코드 변경은 서로 다른 행이다.';
COMMENT ON COLUMN approval.approval_id IS '승인 식별자 (APR-...).';
COMMENT ON COLUMN approval.work_item_id IS '승인이 속한 워크 아이템.';
COMMENT ON COLUMN approval.workflow_id IS '승인이 속한 워크플로.';
COMMENT ON COLUMN approval.approval_type IS 'DESIGN 또는 CODE_CHANGE. 서로 대체할 수 없다.';
COMMENT ON COLUMN approval.resource_type IS '바인딩 대상 종류. DESIGN_REF 또는 CHANGE_SET.';
COMMENT ON COLUMN approval.resource_id IS '승인한 리소스 ID. 다른 리소스에 재사용 금지.';
COMMENT ON COLUMN approval.status IS 'REQUESTED / APPROVED / REJECTED / EXPIRED / CONSUMED.';
COMMENT ON COLUMN approval.requested_at IS '승인 요청 시각 (UTC).';
COMMENT ON COLUMN approval.responded_at IS '승인 또는 거절 시각 (UTC). 대기 중이면 null.';
COMMENT ON COLUMN approval.approved_by IS '결정한 주체. 대기 중이면 null.';
COMMENT ON COLUMN approval.decision IS 'APPROVE 또는 REJECT. 대기 중이면 null.';
COMMENT ON COLUMN approval.reason IS '결정 사유. 없으면 null.';
COMMENT ON COLUMN approval.expires_at IS '만료 시각 (UTC). 지나면 fail-closed.';
COMMENT ON COLUMN approval.is_consumed IS '한 번 쓰였으면 true. 재사용 금지.';
COMMENT ON COLUMN approval.consumed_at IS 'consumed 시각 (UTC). 아니면 null.';

CREATE INDEX idx_approval_workflow_id ON approval (workflow_id);
CREATE INDEX idx_approval_work_item_id ON approval (work_item_id);

CREATE UNIQUE INDEX uq_approval_workflow_id_approval_type_requested
  ON approval (workflow_id, approval_type)
  WHERE status = 'REQUESTED';
