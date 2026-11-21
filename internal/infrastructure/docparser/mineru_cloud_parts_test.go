package docparser

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Tencent/WeKnora/internal/infrastructure/checkpoint"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func testPDF(pages int) []byte {
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offset := []int{0}
	obj := func(n int, s string) {
		offset = append(offset, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", n, s)
	}
	obj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	var kids strings.Builder
	for i := 0; i < pages; i++ {
		fmt.Fprintf(&kids, "%d 0 R ", i+3)
	}
	obj(2, fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", pages, kids.String()))
	for i := 0; i < pages; i++ {
		obj(i+3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 400] /Resources << >> >>")
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offset))
	for _, o := range offset[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offset), xref)
	return []byte(b.String())
}
func TestSplitCloudPDFPageCoverage(t *testing.T) {
	if _, e := exec.LookPath("qpdf"); e != nil {
		t.Skip("qpdf required")
	}
	parts, e := splitCloudPDF(context.Background(), testPDF(205))
	if e != nil {
		t.Fatal(e)
	}
	again, err := splitCloudPDF(context.Background(), testPDF(205))
	if err != nil {
		t.Fatal(err)
	}
	for i := range parts {
		if !bytes.Equal(parts[i].content, again[i].content) {
			t.Fatal("splitting changed PDF bytes and would invalidate resume checkpoints")
		}
	}
	if len(parts) != 3 || parts[0].first != 1 || parts[0].last != 100 || parts[1].first != 101 || parts[1].last != 200 || parts[2].first != 201 || parts[2].last != 205 {
		t.Fatalf("incorrect page coverage: %v", parts)
	}
}
func TestCloudPartResumeDoesNotUpload(t *testing.T) {
	t.Setenv("WEKNORA_PROCESSING_CHECKPOINT_DIR", t.TempDir())
	req := &types.ReadRequest{FileName: "part.pdf", FileContent: []byte("input")}
	if e := checkpoint.Save("mineru-results", ReadCheckpointKey(req), &types.ReadResult{MarkdownContent: "completed"}); e != nil {
		t.Fatal(e)
	}
	c := &MinerUCloudReader{}
	got, e := c.readCloudPart(context.Background(), req)
	if e != nil || got.MarkdownContent != "completed" {
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
	c := &MinerUCloudReader{baseURL: srv.URL}
	got, e := c.readCloudPart(context.Background(), req)
	if e != nil || got.MarkdownContent != "resumed" || calls != 1 {
		t.Fatalf("resume %v %v calls=%d", got, e, calls)
	}
}
