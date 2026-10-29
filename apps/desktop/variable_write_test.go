package main

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

type preparedWriteClient struct {
	read          func(int64) (string, error)
	write         func(string, int64) (string, error)
	reads, writes int
}

func (c *preparedWriteClient) SnapshotJSON(_ int64, _ string, timeout int64) (string, error) {
	c.reads++
	return c.read(timeout)
}
func (c *preparedWriteClient) OperateJSON(_ int64, _ string, capability, schema, request string, timeout int64) (string, error) {
	c.writes++
	if capability != "write" || schema != protocol.SchemaVariableWriteV3 {
		panic("unexpected operation")
	}
	return c.write(request, timeout)
}

const preparedWriteBase = `{"version":1,"instance_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","revision":"10","definition":{"type":"boolean"},"value":false,"status":"ready","observed_at_unix_ms":100}`
const preparedWriteRequest = `{"version":3,"expected_instance_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","expected_revision":"10","value":true}`

func TestPreparedWriteValueEqualityPreservesJSONStructureAndSignedZero(t *testing.T) {
	if sameJSONVariableValue(float64(0), math.Copysign(0, -1)) {
		t.Fatal("signed-zero change was ignored")
	}
	if sameJSONVariableValue(map[string]any{"x": nil}, map[string]any{"y": nil}) {
		t.Fatal("changed object key was ignored")
	}
	if !sameJSONVariableValue(map[string]any{"a": []any{float64(1), false}, "b": nil}, map[string]any{"b": nil, "a": []any{float64(1), false}}) {
		t.Fatal("equal JSON values rejected")
	}
}

func TestPreparedVariableWriteRefreshesMetadataOnceWithinOneBudget(t *testing.T) {
	now := time.Now()
	client := &preparedWriteClient{
		read: func(timeout int64) (string, error) {
			if timeout != 5000 {
				t.Fatalf("read budget %d", timeout)
			}
			now = now.Add(2 * time.Second)
			return strings.Replace(preparedWriteBase, `"revision":"10"`, `"revision":"11"`, 1), nil
		},
		write: func(raw string, timeout int64) (string, error) {
			var request protocol.VariableWriteV3
			if err := json.Unmarshal([]byte(raw), &request); err != nil {
				t.Fatal(err)
			}
			if request.ExpectedRevision != "11" || string(request.Value) != "true" || timeout != 3000 {
				t.Fatalf("write changed intent/budget: %s, %d", raw, timeout)
			}
			return `{"schema":"mfh.variable-value.v1","payload":{}}`, nil
		},
	}
	result := writeVariableFromSnapshot(client, 3, "example/value", preparedWriteBase, preparedWriteRequest, func() time.Time { return now })
	if !result.Sent || result.Error != "" || result.ExpectedRevision != "11" || client.reads != 1 || client.writes != 1 {
		t.Fatalf("unexpected attempt: %+v", result)
	}
}

func TestPreparedVariableWriteRejectsChangedBaseAndInvalidInputsBeforeDispatch(t *testing.T) {
	for _, tc := range []struct{ name, base, request, current string }{
		{"value", preparedWriteBase, preparedWriteRequest, strings.Replace(preparedWriteBase, `"value":false`, `"value":true`, 1)},
		{"instance", preparedWriteBase, preparedWriteRequest, strings.ReplaceAll(preparedWriteBase, strings.Repeat("a", 32), strings.Repeat("b", 32))},
		{"revision", preparedWriteBase, preparedWriteRequest, strings.Replace(preparedWriteBase, `"revision":"10"`, `"revision":"9"`, 1)},
		{"malformed-base", `{}`, preparedWriteRequest, preparedWriteBase},
		{"mismatched-intent", preparedWriteBase, strings.Replace(preparedWriteRequest, `"expected_revision":"10"`, `"expected_revision":"9"`, 1), preparedWriteBase},
		{"invalid-draft", preparedWriteBase, strings.Replace(preparedWriteRequest, `"value":true`, `"value":1`, 1), preparedWriteBase},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &preparedWriteClient{read: func(int64) (string, error) { return tc.current, nil }, write: func(string, int64) (string, error) { t.Fatal("unexpected side effect"); return "", nil }}
			result := writeVariableFromSnapshot(client, 3, "example/value", tc.base, tc.request, time.Now)
			if result.Sent || result.Error == "" || client.writes != 0 {
				t.Fatalf("invalid attempt accepted: %+v", result)
			}
		})
	}
}

func TestPreparedVariableWriteReportsDispatchBoundaryWithoutRetry(t *testing.T) {
	for _, sent := range []bool{false, true} {
		t.Run(map[bool]string{false: "read-failure", true: "write-failure"}[sent], func(t *testing.T) {
			client := &preparedWriteClient{read: func(int64) (string, error) {
				if !sent {
					return "", errors.New("disconnected")
				}
				return preparedWriteBase, nil
			}, write: func(string, int64) (string, error) { return "", errors.New("conflict: concurrent sampling") }}
			result := writeVariableFromSnapshot(client, 3, "example/value", preparedWriteBase, preparedWriteRequest, time.Now)
			if result.Sent != sent || result.Error == "" || client.writes > 1 || (sent && client.writes != 1) {
				t.Fatalf("wrong boundary or retry: %+v writes=%d", result, client.writes)
			}
		})
	}
	now := time.Now()
	client := &preparedWriteClient{read: func(int64) (string, error) { now = now.Add(5 * time.Second); return preparedWriteBase, nil }, write: func(string, int64) (string, error) { t.Fatal("write after deadline"); return "", nil }}
	result := writeVariableFromSnapshot(client, 3, "example/value", preparedWriteBase, preparedWriteRequest, func() time.Time { return now })
	if result.Sent || result.Error == "" || client.writes != 0 {
		t.Fatalf("late dispatch: %+v", result)
	}
}
