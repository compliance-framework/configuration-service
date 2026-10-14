//go:build integration

package oscal

import (
	"fmt"
	"os"
	"time"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// complianceStreamVersion is one evidence record of a stream; versions are stored
// oldest first, so the last one is the stream's latest evidence.
type complianceStreamVersion struct {
	state  string
	labels map[string]string
}

type evidenceLabelRow struct {
	EvidenceID  uuid.UUID
	LabelsName  string
	LabelsValue string
}

// seedComplianceProfile creates a catalog with the given control ids and a profile
// carrying all of them, and returns the profile and catalog ids.
func (suite *ProfileIntegrationSuite) seedComplianceProfile(controlIDs []string) (uuid.UUID, uuid.UUID) {
	catID := uuid.New()
	controls := make([]relational.Control, 0, len(controlIDs))
	for _, id := range controlIDs {
		controls = append(controls, relational.Control{CatalogID: catID, ID: id, Title: "Control " + id})
	}
	suite.Require().NoError(suite.DB.CreateInBatches(&controls, 500).Error)

	profileID := uuid.New()
	profile := relational.Profile{
		UUIDModel: relational.UUIDModel{ID: &profileID},
		Metadata:  relational.Metadata{Title: "Compliance profile", Version: "1.0.0", OscalVersion: "1.1.3"},
	}
	suite.Require().NoError(suite.DB.Create(&profile).Error)
	suite.Require().NoError(suite.DB.Model(&profile).Association("Controls").Append(&controls))
	return profileID, catID
}

// attachComplianceFilter creates a filter and links it to the given controls.
func (suite *ProfileIntegrationSuite) attachComplianceFilter(catID uuid.UUID, filter labelfilter.Filter, controlIDs ...string) {
	f := relational.Filter{Name: "f-" + uuid.NewString(), Filter: datatypes.NewJSONType(filter)}
	suite.Require().NoError(suite.DB.Create(&f).Error)
	for _, controlID := range controlIDs {
		suite.Require().NoError(suite.DB.Exec(
			"INSERT INTO filter_controls (filter_id, control_catalog_id, control_id) VALUES (?, ?, ?)",
			f.ID, catID, controlID,
		).Error)
	}
}

// seedComplianceStreams stores every stream's versions, one minute apart, oldest first.
func (suite *ProfileIntegrationSuite) seedComplianceStreams(streams [][]complianceStreamVersion) {
	base := time.Now().UTC().Add(-time.Hour)
	evidences := []relational.Evidence{}
	labelRows := []evidenceLabelRow{}
	labels := map[relational.Labels]struct{}{}
	for _, versions := range streams {
		streamUUID := uuid.New()
		for v, version := range versions {
			id := uuid.New()
			at := base.Add(time.Duration(v) * time.Minute)
			evidences = append(evidences, relational.Evidence{
				UUIDModel: relational.UUIDModel{ID: &id},
				UUID:      streamUUID,
				Title:     "e",
				Start:     at,
				End:       at,
				Status:    datatypes.NewJSONType(oscalTypes_1_1_3.ObjectiveStatus{State: version.state}),
			})
			for name, value := range version.labels {
				labelRows = append(labelRows, evidenceLabelRow{EvidenceID: id, LabelsName: name, LabelsValue: value})
				labels[relational.Labels{Name: name, Value: value}] = struct{}{}
			}
		}
	}
	suite.Require().NoError(suite.DB.CreateInBatches(&evidences, 1000).Error)
	for label := range labels {
		suite.Require().NoError(suite.DB.Exec(
			"INSERT INTO labels (name, value) VALUES (?, ?) ON CONFLICT DO NOTHING", label.Name, label.Value).Error)
	}
	if len(labelRows) > 0 {
		suite.Require().NoError(suite.DB.Table("evidence_labels").CreateInBatches(&labelRows, 1000).Error)
	}
}

// filtersForControl reads a control's filters the way ComplianceProgress does.
func (suite *ProfileIntegrationSuite) filtersForControl(catID uuid.UUID, controlID string) []labelfilter.Filter {
	var control relational.Control
	suite.Require().NoError(suite.DB.Preload("Filters").
		First(&control, "catalog_id = ? AND id = ?", catID, controlID).Error)
	filters := make([]labelfilter.Filter, 0, len(control.Filters))
	for _, f := range control.Filters {
		filters = append(filters, f.Filter.Data())
	}
	return filters
}

func labelScope(label, operator, value string) *labelfilter.Scope {
	return &labelfilter.Scope{Condition: &labelfilter.Condition{Label: label, Operator: operator, Value: value}}
}

// TestComplianceProgressMatchesPerControlCounts pins the batched evaluation to the
// per-control SQL aggregate it replaced (relational.EvidenceStatusCountsForFilters):
// every control's status counts and computed status must be identical.
func (suite *ProfileIntegrationSuite) TestComplianceProgressMatchesPerControlCounts() {
	suite.Require().NoError(suite.Migrator.Refresh())

	controlIDs := []string{"c-0", "c-1", "c-2", "c-3", "c-4", "c-5", "c-6", "c-7", "c-8"}
	profileID, catID := suite.seedComplianceProfile(controlIDs)

	ec2 := labelfilter.Filter{Scope: labelScope("svc", "=", "ec2")}
	// c-0 has no filter.
	suite.attachComplianceFilter(catID, ec2, "c-1", "c-6") // one filter shared by two controls
	suite.attachComplianceFilter(catID, labelfilter.Filter{Scope: labelScope("svc", "!=", "ec2")}, "c-2")
	suite.attachComplianceFilter(catID, ec2, "c-3") // c-3: two filters, OR-ed
	suite.attachComplianceFilter(catID, labelfilter.Filter{Scope: labelScope("env", "=", "prod")}, "c-3")
	suite.attachComplianceFilter(catID, labelfilter.Filter{Scope: &labelfilter.Scope{Query: &labelfilter.Query{
		Operator: "and",
		Scopes:   []labelfilter.Scope{*labelScope("svc", "=", "ec2"), *labelScope("env", "=", "prod")},
	}}}, "c-4")
	suite.attachComplianceFilter(catID, labelfilter.Filter{Scope: &labelfilter.Scope{Query: &labelfilter.Query{
		Operator: "or",
		Scopes: []labelfilter.Scope{*labelScope("svc", "=", "s3"), {Query: &labelfilter.Query{
			Operator: "and",
			Scopes:   []labelfilter.Scope{*labelScope("team", "=", "a"), *labelScope("env", "!=", "dev")},
		}}},
	}}}, "c-5")
	suite.attachComplianceFilter(catID, labelfilter.Filter{Scope: labelScope("svc", "=", "rds")}, "c-7")
	suite.attachComplianceFilter(catID, labelfilter.Filter{}, "c-8") // unscoped: matches every stream

	suite.seedComplianceStreams([][]complianceStreamVersion{
		{{"not-satisfied", map[string]string{"svc": "ec2"}}, {"satisfied", map[string]string{"svc": "ec2", "env": "prod"}}},
		{{"satisfied", map[string]string{"svc": "ec2"}}, {"not-satisfied", map[string]string{"svc": "ec2", "env": "dev"}}},
		// Only the latest version's labels count: this stream left ec2 for s3.
		{{"not-satisfied", map[string]string{"svc": "ec2"}}, {"satisfied", map[string]string{"svc": "s3", "env": "prod"}}},
		{{"satisfied", map[string]string{"Svc": "EC2", "team": "a"}}},
		{{"satisfied", map[string]string{"team": "a", "env": "dev"}}},
		{{"not-satisfied", nil}},
	})

	data := suite.complianceProgress(profileID.String(), "")
	suite.Require().Len(data.Controls, len(controlIDs))

	byID := map[string]ProfileComplianceControl{}
	for _, c := range data.Controls {
		byID[c.ControlID] = c
	}
	for _, controlID := range controlIDs {
		control, ok := byID[controlID]
		suite.Require().True(ok, controlID)

		expected, err := relational.EvidenceStatusCountsForFilters(suite.DB, suite.filtersForControl(catID, controlID))
		suite.Require().NoError(err)
		expectedRows := make([]ProfileComplianceStatusCount, len(expected))
		for i, r := range expected {
			expectedRows[i] = ProfileComplianceStatusCount{Count: r.Count, Status: r.Status}
		}
		suite.Equal(expectedRows, control.StatusCounts, controlID)
		suite.Equal(relational.CollapseEvidenceStatus(expected), control.ComputedStatus, controlID)
	}

	// Spot checks, so the comparison above can't pass by both sides being empty.
	suite.Equal("unknown", byID["c-0"].ComputedStatus)
	suite.Equal([]ProfileComplianceStatusCount{{Count: 1, Status: "not-satisfied"}, {Count: 2, Status: "satisfied"}}, byID["c-1"].StatusCounts)
	suite.Equal(byID["c-1"].StatusCounts, byID["c-6"].StatusCounts)
	suite.Equal([]ProfileComplianceStatusCount{{Count: 1, Status: "satisfied"}}, byID["c-4"].StatusCounts)
	suite.Equal([]ProfileComplianceStatusCount{{Count: 2, Status: "satisfied"}}, byID["c-5"].StatusCounts)
	suite.Equal("unknown", byID["c-7"].ComputedStatus)
	suite.Equal([]ProfileComplianceStatusCount{{Count: 2, Status: "not-satisfied"}, {Count: 4, Status: "satisfied"}}, byID["c-8"].StatusCounts)
	// c-1, c-3 and c-6 see the failing ec2/dev stream; c-2 and c-8 the unlabelled one.
	suite.Equal(5, data.Summary.NotSatisfied)
}

// TestComplianceProgressScale guards against the per-control N+1 returning: 150
// filtered controls over 5000 streams with three versions each must stay well under
// the old cost (one full evidence aggregate per control). Set
// CCF_PROFILE_COMPLIANCE_COMPARE=1 to also time the per-control path and log both.
func (suite *ProfileIntegrationSuite) TestComplianceProgressScale() {
	suite.Require().NoError(suite.Migrator.Refresh())

	const (
		nControls = 150
		nStreams  = 5000
		nVersions = 3
	)
	controlIDs := make([]string, 0, nControls)
	for i := 0; i < nControls; i++ {
		controlIDs = append(controlIDs, fmt.Sprintf("ac-%d", i))
	}
	profileID, catID := suite.seedComplianceProfile(controlIDs)

	services := []string{"ec2", "s3", "rds", "eks", "iam"}
	for i, controlID := range controlIDs {
		svc := services[i%len(services)]
		env := []string{"prod", "dev"}[i%2]
		suite.attachComplianceFilter(catID, labelfilter.Filter{Scope: &labelfilter.Scope{Query: &labelfilter.Query{
			Operator: "and",
			Scopes:   []labelfilter.Scope{*labelScope("svc", "=", svc), *labelScope("env", "=", env)},
		}}}, controlID)
	}

	streams := make([][]complianceStreamVersion, 0, nStreams)
	for i := 0; i < nStreams; i++ {
		versions := make([]complianceStreamVersion, 0, nVersions)
		for v := 0; v < nVersions; v++ {
			state := "satisfied"
			if (i+v)%7 == 0 {
				state = "not-satisfied"
			}
			versions = append(versions, complianceStreamVersion{state, map[string]string{
				"svc":  services[i%len(services)],
				"env":  []string{"prod", "dev"}[i%2],
				"team": fmt.Sprintf("t-%d", i%10),
			}})
		}
		streams = append(streams, versions)
	}
	suite.seedComplianceStreams(streams)

	start := time.Now()
	data := suite.complianceProgress(profileID.String(), "")
	batched := time.Since(start)
	suite.Equal(nControls, data.Summary.TotalControls)
	suite.Equal(0, data.Summary.Unknown, "every control has matching evidence")
	suite.T().Logf("compliance-progress, batched (%d controls, %d streams x %d versions): %s", nControls, nStreams, nVersions, batched)

	if os.Getenv("CCF_PROFILE_COMPLIANCE_COMPARE") != "" {
		start = time.Now()
		for _, controlID := range controlIDs {
			_, err := relational.EvidenceStatusCountsForFilters(suite.DB, suite.filtersForControl(catID, controlID))
			suite.Require().NoError(err)
		}
		perControl := time.Since(start)
		suite.T().Logf("compliance-progress, per-control queries: %s (%.1fx the batched time)", perControl, float64(perControl)/float64(batched))
	}

	suite.Less(batched, 5*time.Second, "compliance-progress must not regress into a per-control N+1")
}
