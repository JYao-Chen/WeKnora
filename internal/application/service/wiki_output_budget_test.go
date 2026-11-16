package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type wikiOutputBudgetModels struct {
	interfaces.ModelService
	budget int
	err    error
}

func (m wikiOutputBudgetModels) GetModelByID(context.Context, string) (*types.Model, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &types.Model{Parameters: types.ModelParameters{MaxOutputTokens: m.budget}}, nil
}

func TestWikiTemplateUsesConfiguredOutputBudget(t *testing.T) {
	for _, budget := range []int{0, 8192, 900000} {
		model := &templateCaptureChatModel{response: `{"entities":[],"concepts":[]}`}
		svc := &wikiIngestService{modelService: wikiOutputBudgetModels{budget: budget}}
		_, err := svc.generateWithTemplate(context.Background(), model, `{{.Content}}`, map[string]string{"Content": "hello"})
		require.NoError(t, err)
		want := budget
		if want == 0 {
			want = wikiLLMMaxTokens
		}
		require.Equal(t, want, model.options.MaxTokens)
	}
}
func TestWikiTemplateReturnsOutputBudgetLookupError(t *testing.T) {
	lookupErr := errors.New("model lookup failed")
	svc := &wikiIngestService{modelService: wikiOutputBudgetModels{err: lookupErr}}
	model := &templateCaptureChatModel{}
	_, err := svc.generateWithTemplate(context.Background(), model, `{{.Content}}`, map[string]string{"Content": "hello"})
	require.ErrorIs(t, err, lookupErr)
	require.Zero(t, model.options.MaxTokens)
}
