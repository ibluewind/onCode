package contract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"oncode/protocol/contract"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/ids"
	"oncode/protocol/messages"
	"oncode/protocol/versioning"
)

func TestIDKindsAndUUIDv7(t *testing.T) {
	required := []ids.Kind{
		ids.Message, ids.Call, ids.Session, ids.Project, ids.Workspace,
		ids.WorkItem, ids.Workflow, ids.Approval, ids.ChangeSet,
	}
	for _, k := range required {
		id, err := ids.New(k)
		if err != nil {
			t.Fatalf("New(%s): %v", k, err)
		}
		if err := ids.Validate(k, id); err != nil {
			t.Fatalf("Validate(%s, %s): %v", k, id, err)
		}
	}
}

func TestErrorCatalogStable(t *testing.T) {
	must := []protoerr.Code{
		protoerr.ProtocolVersionUnsupported,
		protoerr.InvalidRequest,
		protoerr.WorkspaceFileNotFound,
		protoerr.WorkspaceFileChanged,
		protoerr.ApprovalRequired,
		protoerr.InternalError,
	}
	for _, c := range must {
		if !protoerr.Known(c) {
			t.Fatalf("missing code %s", c)
		}
		e, err := protoerr.New(c, "test")
		if err != nil {
			t.Fatal(err)
		}
		if e.Code != c || e.Message == "" {
			t.Fatalf("structured error incomplete: %+v", e)
		}
		if e.Category == "" {
			t.Fatalf("category missing for %s", c)
		}
	}
	codes := protoerr.Codes()
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	if len(codes) < 20 {
		t.Fatalf("catalog too small: %d", len(codes))
	}
}

func TestUnknownProtocolVersionRejected(t *testing.T) {
	if _, err := versioning.ParseCompatible("oncode-tool/2.0"); err == nil {
		t.Fatal("expected 2.0 rejected")
	}
	if _, err := versioning.ParseCompatible("other/1.0"); err == nil {
		t.Fatal("expected unknown name rejected")
	}
	if _, err := versioning.ParseCompatible(versioning.Current); err != nil {
		t.Fatal(err)
	}
	if _, err := versioning.ParseCompatible("oncode-tool/1.1"); err != nil {
		t.Fatalf("1.1 should be compatible: %v", err)
	}
}

func TestToolRequestResponseJSONRoundTrip(t *testing.T) {
	reqPath := filepath.Join(contract.FixtureDir(), "tool-request.success.json")
	raw, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatal(err)
	}
	var env messages.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if err := env.Validate(); err != nil {
		t.Fatal(err)
	}
	var payload messages.ToolRequestPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if err := payload.Validate(); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var env2 messages.Envelope
	if err := json.Unmarshal(out, &env2); err != nil {
		t.Fatal(err)
	}
	if env2.MessageID != env.MessageID || env2.Protocol != env.Protocol {
		t.Fatalf("round-trip mismatch")
	}
	if err := contract.ValidateRequired(filepath.Join(contract.SchemaDir(), "envelope.schema.json"), mustMap(t, raw)); err != nil {
		t.Fatal(err)
	}
	if err := contract.ValidateRequired(filepath.Join(contract.SchemaDir(), "tool-request.schema.json"), mustMap(t, env.Payload)); err != nil {
		t.Fatal(err)
	}

	respRaw, err := os.ReadFile(filepath.Join(contract.FixtureDir(), "tool-response.success.json"))
	if err != nil {
		t.Fatal(err)
	}
	var resp messages.Envelope
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		t.Fatal(err)
	}
	if err := resp.Validate(); err != nil {
		t.Fatal(err)
	}
	var rp messages.ToolResponsePayload
	if err := json.Unmarshal(resp.Payload, &rp); err != nil {
		t.Fatal(err)
	}
	if err := rp.Validate(); err != nil {
		t.Fatal(err)
	}
	back, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var resp2 messages.Envelope
	if err := json.Unmarshal(back, &resp2); err != nil {
		t.Fatal(err)
	}
	if resp2.MessageID != resp.MessageID {
		t.Fatal("response round-trip mismatch")
	}
}

func TestFailedResponseRequiresStructuredError(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(contract.FixtureDir(), "tool-response.validation-failure.json"))
	if err != nil {
		t.Fatal(err)
	}
	var env messages.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if err := env.Validate(); err != nil {
		t.Fatal(err)
	}
	var p messages.ToolResponsePayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if p.Error == nil || p.Error.Code != protoerr.WorkspaceFileNotFound {
		t.Fatalf("want WORKSPACE_FILE_NOT_FOUND, got %+v", p.Error)
	}
	if err := contract.ValidateRequired(filepath.Join(contract.SchemaDir(), "tool-error.schema.json"), mustMap(t, mustJSON(t, p.Error))); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedVersionFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(contract.FixtureDir(), "tool-request.unsupported-version.json"))
	if err != nil {
		t.Fatal(err)
	}
	var env messages.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if err := env.Validate(); err == nil {
		t.Fatal("expected unsupported version to fail")
	}
}

func TestCorePayloadFixtures(t *testing.T) {
	cases := []struct {
		file   string
		schema string
		check  func(t *testing.T, raw []byte)
	}{
		{"heartbeat.json", "heartbeat.schema.json", func(t *testing.T, raw []byte) {
			var h messages.HeartbeatPayload
			if err := json.Unmarshal(raw, &h); err != nil {
				t.Fatal(err)
			}
			if err := h.Validate(); err != nil {
				t.Fatal(err)
			}
		}},
		{"approval.json", "approval.schema.json", func(t *testing.T, raw []byte) {
			var a messages.Approval
			if err := json.Unmarshal(raw, &a); err != nil {
				t.Fatal(err)
			}
			if err := a.Validate(); err != nil {
				t.Fatal(err)
			}
		}},
		{"proposed-change-set.json", "proposed-change-set.schema.json", func(t *testing.T, raw []byte) {
			var p messages.ProposedChangeSet
			if err := json.Unmarshal(raw, &p); err != nil {
				t.Fatal(err)
			}
			if err := p.Validate(); err != nil {
				t.Fatal(err)
			}
		}},
		{"actual-diff.json", "actual-diff.schema.json", func(t *testing.T, raw []byte) {
			var d messages.ActualDiff
			if err := json.Unmarshal(raw, &d); err != nil {
				t.Fatal(err)
			}
			if err := d.Validate(); err != nil {
				t.Fatal(err)
			}
		}},
		{"capabilities.json", "capabilities.schema.json", func(t *testing.T, raw []byte) {
			var c messages.CapabilitiesResult
			if err := json.Unmarshal(raw, &c); err != nil {
				t.Fatal(err)
			}
			if c.ProtocolVersion != versioning.Current {
				t.Fatalf("protocol_version=%q", c.ProtocolVersion)
			}
			if len(c.SupportedTools) == 0 {
				t.Fatal("supported_tools required")
			}
			if c.AgentVersion == "" || c.Platform.OS == "" {
				t.Fatalf("incomplete capabilities: %+v", c)
			}
		}},
		{"workflow-event.json", "envelope.schema.json", func(t *testing.T, raw []byte) {
			var env messages.Envelope
			if err := json.Unmarshal(raw, &env); err != nil {
				t.Fatal(err)
			}
			if err := env.Validate(); err != nil {
				t.Fatal(err)
			}
			if err := contract.ValidateRequired(filepath.Join(contract.SchemaDir(), "workflow-event.schema.json"), mustMap(t, env.Payload)); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(contract.FixtureDir(), tc.file))
			if err != nil {
				t.Fatal(err)
			}
			if err := contract.ValidateRequired(filepath.Join(contract.SchemaDir(), tc.schema), mustMap(t, raw)); err != nil {
				t.Fatal(err)
			}
			tc.check(t, raw)
		})
	}
}

func TestProtobufJSONLogicalFieldMapping(t *testing.T) {
	names, err := contract.LoadProtoJSONNames()
	if err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["Envelope"],
		"protocol", "message_type", "message_id", "timestamp",
		"session_id", "project_id", "workspace_id", "workflow_id",
		"work_item_id",
	); err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["ToolRequestPayload"], "call_id", "tool"); err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["ToolResponsePayload"], "call_id", "status", "error"); err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["ToolError"], "code", "message", "retryable"); err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["Heartbeat"], "agent_id", "protocol_version", "timestamp"); err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["CapabilitiesResult"],
		"agent_version", "platform", "capabilities", "protocol_version", "supported_tools",
	); err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["Approval"], "approval_id", "decision"); err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["ProposedChangeSet"], "change_set_id", "changes"); err != nil {
		t.Fatal(err)
	}
	if err := contract.ContainsAll(names["ActualDiff"], "change_set_id", "diff", "diff_hash"); err != nil {
		t.Fatal(err)
	}
}

func mustMap(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
