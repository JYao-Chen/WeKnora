package docparser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/infrastructure/checkpoint"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

type pdfPart struct {
	content     []byte
	first, last int
}

// splitCloudPDF bounds both page count and upload size, without rasterizing PDFs.
func splitCloudPDF(ctx context.Context, content []byte) ([]pdfPart, error) {
	dir, err := os.MkdirTemp("", "mineru-pdf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "input.pdf")
	if err = os.WriteFile(input, content, 0600); err != nil {
		return nil, err
	}
	out, err := exec.CommandContext(ctx, "qpdf", "--show-npages", input).Output()
	if err != nil {
		if ex, ok := err.(*exec.ExitError); !ok || ex.ExitCode() != 3 {
			return nil, fmt.Errorf("PDF page inspection: %w", err)
		}
	}
	pages, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || pages < 1 {
		return nil, fmt.Errorf("invalid PDF page count %q", out)
	}
	if pages <= 100 && len(content) <= 10*1024*1024 {
		return []pdfPart{{content, 1, pages}}, nil
	}
	var parts []pdfPart
	var extract func(int, int) error
	extract = func(first, last int) error {
		path := filepath.Join(dir, fmt.Sprintf("%d-%d.pdf", first, last))
		output, e := exec.CommandContext(ctx, "qpdf", "--deterministic-id", input, "--pages", ".", fmt.Sprintf("%d-%d", first, last), "--", path).CombinedOutput()
		// qpdf exit 3 means a repaired PDF was written with warnings.
		if e != nil {
			if ex, ok := e.(*exec.ExitError); !ok || ex.ExitCode() != 3 {
				return fmt.Errorf("split PDF pages %d-%d: %s: %w", first, last, output, e)
			}
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		_ = os.Remove(path)
		if len(b) > 10*1024*1024 && first < last {
			mid := (first + last) / 2
			if e = extract(first, mid); e != nil {
				return e
			}
			return extract(mid+1, last)
		}
		parts = append(parts, pdfPart{b, first, last})
		return nil
	}
	for first := 1; first <= pages; first += 100 {
		last := first + 99
		if last > pages {
			last = pages
		}
		if err = extract(first, last); err != nil {
			return nil, err
		}
	}
	return parts, nil
}

type cloudJobCheckpoint struct{ BatchID string }

func (c *MinerUCloudReader) readCloudPart(ctx context.Context, req *types.ReadRequest) (*types.ReadResult, error) {
	key := ReadCheckpointKey(req)
	var saved types.ReadResult
	if checkpoint.Load("mineru-results", key, &saved) {
		logger.Infof(ctx, "[MinerUCloud] Resuming completed part %s", req.FileName)
		return &saved, nil
	}
	var job cloudJobCheckpoint
	checkpoint.Load("mineru-jobs", key, &job)
	if job.BatchID == "" {
		batch, url, err := c.applyUploadURLs(ctx, req.FileName, filepath.Ext(req.FileName))
		if err != nil {
			return nil, err
		}
		if err = c.uploadFile(ctx, url, req.FileContent); err != nil {
			return nil, err
		}
		job.BatchID = batch
		if err = checkpoint.Save("mineru-jobs", key, job); err != nil {
			return nil, fmt.Errorf("save OCR job checkpoint: %w", err)
		}
	}
	md, refs, err := c.pollBatchResult(ctx, job.BatchID)
	if err != nil {
		// A cloud timeout can resume polling. A terminal failed job must be resubmitted.
		if strings.Contains(err.Error(), "task failed:") {
			if p := checkpoint.Path("mineru-jobs", key); p != "" {
				_ = os.Remove(p)
			}
		}
		return nil, err
	}
	result := &types.ReadResult{MarkdownContent: md, ImageRefs: refs}
	if err = checkpoint.Save("mineru-results", key, result); err != nil {
		return nil, fmt.Errorf("save OCR result checkpoint: %w", err)
	}
	return result, nil
}

func (c *MinerUCloudReader) readSplitPDF(ctx context.Context, req *types.ReadRequest) (*types.ReadResult, error) {
	parts, err := splitCloudPDF(ctx, req.FileContent)
	if err != nil {
		return nil, err
	}
	logger.Infof(ctx, "[MinerUCloud] PDF split into %d parts: %s", len(parts), req.FileName)
	merged := &types.ReadResult{Metadata: map[string]string{"pages": strconv.Itoa(parts[len(parts)-1].last), "ocr_parts": strconv.Itoa(len(parts))}}
	var markdown strings.Builder
	for i, part := range parts {
		sub := *req
		sub.FileContent = part.content
		sub.FileName = fmt.Sprintf("%s-pages-%d-%d.pdf", strings.TrimSuffix(req.FileName, filepath.Ext(req.FileName)), part.first, part.last)
		result, e := c.readCloudPart(ctx, &sub)
		if e != nil {
			return nil, fmt.Errorf("OCR part %d/%d (pages %d-%d): %w", i+1, len(parts), part.first, part.last, e)
		}
		md := result.MarkdownContent
		// MinerU parts can reuse image filenames. Namespace references before merging.
		for _, ref := range result.ImageRefs {
			old := ref.OriginalRef
			ref.Filename = fmt.Sprintf("part-%d-%s", i+1, ref.Filename)
			ref.OriginalRef = fmt.Sprintf("part-%d/%s", i+1, old)
			md = strings.ReplaceAll(md, "]("+old+")", "]("+ref.OriginalRef+")")
			merged.ImageRefs = append(merged.ImageRefs, ref)
		}
		fmt.Fprintf(&markdown, "\n\n<!-- Original PDF pages %d-%d -->\n\n%s", part.first, part.last, md)
		logger.Infof(ctx, "[MinerUCloud] Completed part %d/%d pages %d-%d", i+1, len(parts), part.first, part.last)
	}
	merged.MarkdownContent = strings.TrimSpace(markdown.String())
	return merged, nil
}
