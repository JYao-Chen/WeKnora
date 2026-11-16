package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

func TestWikiIngestTriggerDeadlinesStayAligned(t *testing.T) {
	cases := map[string]func(*wikiIngestService){
		"initial": func(s *wikiIngestService) {
			require.NoError(t, enqueueWikiIngestTrigger(context.Background(), s.task, 7, "kb-1"))
		},
		"followup": func(s *wikiIngestService) {
			require.True(t, s.scheduleFollowUp(context.Background(), WikiIngestPayload{KnowledgeBaseID: "kb-1"}, wikiFollowUpDelay, 1, 1, false))
		},
		"capped": func(s *wikiIngestService) {
			s.scheduleCappedRetry(context.Background(), WikiIngestPayload{KnowledgeBaseID: "kb-1"})
		},
		"stale claims": func(s *wikiIngestService) {
			require.True(t, s.scheduleStaleClaimRecheck(context.Background(), WikiIngestPayload{KnowledgeBaseID: "kb-1"}))
		},
		"reconcile": func(s *wikiIngestService) {
			require.True(t, s.scheduleFollowUpRecheck(context.Background(), WikiIngestPayload{KnowledgeBaseID: "kb-1"}))
		},
	}
	for name, enqueue := range cases {
		t.Run(name, func(t *testing.T) {
			svc, inspector := newQueuedFollowUpTestService(t)
			enqueue(svc)
			tasks, err := inspector.ListScheduledTasks(types.QueueWiki, asynq.PageSize(100))
			require.NoError(t, err)
			require.Len(t, tasks, 1)
			require.Equal(t, types.WikiIngestTaskTimeout, tasks[0].Timeout)
		})
	}
	require.Greater(t, wikiClaimStaleAfter, types.WikiIngestTaskTimeout, "active document claims must outlive the task deadline")
	require.Greater(t, wikiDeletedTTL, types.WikiIngestTaskTimeout, "deletion tombstones must cover all live tasks")
	require.Greater(t, wikiIdentityClaimTTL, types.WikiIngestTaskTimeout, "identity reservations must cover all live tasks")
}
