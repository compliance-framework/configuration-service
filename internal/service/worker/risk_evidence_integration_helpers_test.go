//go:build integration

package worker

import (
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// createStreamEvidence creates an evidence of the given stream with the given state and end.
func createStreamEvidence(t *testing.T, db *gorm.DB, stream uuid.UUID, state string, end time.Time) *relational.Evidence {
	t.Helper()
	id := uuid.New()
	evidence := &relational.Evidence{
		UUIDModel: relational.UUIDModel{ID: &id},
		UUID:      stream,
		Title:     "Test Evidence",
		Start:     end.Add(-time.Hour),
		End:       end,
		Status:    datatypes.NewJSONType(oscalTypes_1_1_3.ObjectiveStatus{State: state}),
	}
	require.NoError(t, db.Create(evidence).Error)
	return evidence
}
