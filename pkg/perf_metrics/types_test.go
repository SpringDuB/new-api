package perfmetrics

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestModelSummaryIncludesObservedRequestCount(t *testing.T) {
	encoded, err := common.Marshal(SummaryAllResult{Models: []ModelSummary{{
		ModelName: "seedance-test", RequestCount: 1, SuccessRate: 100,
	}}})
	require.NoError(t, err)
	var payload struct {
		Models []map[string]any `json:"models"`
	}
	require.NoError(t, common.Unmarshal(encoded, &payload))
	require.Equal(t, float64(1), payload.Models[0]["request_count"])
}
