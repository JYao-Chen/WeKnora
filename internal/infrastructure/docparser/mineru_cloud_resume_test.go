package docparser

import (
	"context"
	"fmt"
	"github.com/Tencent/WeKnora/internal/infrastructure/checkpoint"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCloudPartResumeDoesNotUpload(t *testing.T) {
	t.Setenv("WEKNORA_PROCESSING_CHECKPOINT_DIR", t.TempDir())
	req := &types.ReadRequest{FileName: "part.pdf", FileContent: []byte("input")}
	if e := checkpoint.Save("mineru-results-v090", ReadCheckpointKey(req), &types.ReadResult{MarkdownContent: "completed", SourceBlocks: []types.SourceBlock{{Start: 0, End: 9}}}); e != nil {
		t.Fatal(e)
	}
	c := &MinerUCloudReader{}
	got, e := c.readCloudPart(context.Background(), req)
	if e != nil || got.MarkdownContent != "completed" || len(got.SourceBlocks) != 1 {
		t.Fatalf("resume %v %v", got, e)
	}
}
func TestCloudPartResumesExistingJob(t *testing.T) {
	t.Setenv("WEKNORA_PROCESSING_CHECKPOINT_DIR", t.TempDir())
	utils.SetSSRFWhitelistFromRaw("127.0.0.1")
	t.Cleanup(utils.ResetSSRFWhitelistForTest)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"code":0,"data":{"extract_result":[{"state":"done","markdown":"resumed"}]}}`)
	}))
	defer srv.Close()
	req := &types.ReadRequest{FileName: "part.pdf", FileContent: []byte("input")}
	checkpoint.Save("mineru-jobs", ReadCheckpointKey(req), cloudJobCheckpoint{BatchID: "existing"})
	c := &MinerUCloudReader{baseURL: srv.URL, timeout: time.Minute, pollInterval: time.Millisecond}
	got, e := c.readCloudPart(context.Background(), req)
	if e != nil || got.MarkdownContent != "resumed" || calls != 1 {
		t.Fatalf("resume %v %v calls=%d", got, e, calls)
	}
}
