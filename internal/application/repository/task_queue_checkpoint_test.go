package repository

import (
	"context"
	"encoding/json"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTaskPendingPayloadCheckpointCASAndTenantIsolation(t *testing.T) {
	db := setupTaskQueueTestDB(t)
	repo := NewTaskPendingOpsRepository(db)
	store := repo.(interfaces.TaskPendingOpsPayloadCheckpointer)
	row := &types.TaskPendingOp{TenantID: 1, TaskType: "wiki:ingest", Scope: "knowledge_base", ScopeID: "kb", Op: "ingest", DedupKey: "doc", Payload: json.RawMessage(`{"knowledge_id":"doc"}`), FailCount: 4}
	require.NoError(t, repo.Enqueue(context.Background(), row))
	before := append(json.RawMessage(nil), row.Payload...)
	after := json.RawMessage(`{"knowledge_id":"doc","extraction_checkpoint":{"segments":{"first":{}}}}`)
	updated, err := store.CompareAndSwapPayload(context.Background(), 2, row.ID, before, after)
	require.NoError(t, err)
	require.False(t, updated, "another tenant cannot update progress")
	updated, err = store.CompareAndSwapPayload(context.Background(), 1, row.ID, before, after)
	require.NoError(t, err)
	require.True(t, updated)
	var saved types.TaskPendingOp
	require.NoError(t, db.First(&saved, row.ID).Error)
	require.JSONEq(t, string(after), string(saved.Payload))
	require.Zero(t, saved.FailCount)
	updated, err = store.CompareAndSwapPayload(context.Background(), 1, row.ID, before, json.RawMessage(`{"stale":true}`))
	require.NoError(t, err)
	require.False(t, updated, "a stale worker cannot overwrite a newer completion")
	updated, err = store.CompareAndSwapPayload(context.Background(), 1, row.ID, after, json.RawMessage(`{`))
	require.Error(t, err)
	require.False(t, updated)
	require.NoError(t, repo.DeleteByIDs(context.Background(), []int64{row.ID}))
	updated, err = store.CompareAndSwapPayload(context.Background(), 1, row.ID, after, before)
	require.NoError(t, err)
	require.False(t, updated, "a cancelled/deleted task cannot be recreated")
}
