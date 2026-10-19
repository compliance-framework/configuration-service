//go:build integration

package sspscope_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	"github.com/compliance-framework/api/internal/tests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ScopeServiceIntegrationSuite struct {
	tests.IntegrationTestSuite
	svc   *sspscope.Service
	sspID uuid.UUID
}

func TestScopeServiceIntegration(t *testing.T) {
	suite.Run(t, new(ScopeServiceIntegrationSuite))
}

func (s *ScopeServiceIntegrationSuite) SetupTest() {
	s.Require().NoError(s.Migrator.Refresh())
	s.svc = sspscope.NewService(s.DB)
	s.sspID = s.createSSP()
}

func (s *ScopeServiceIntegrationSuite) createSSP() uuid.UUID {
	id := uuid.New()
	s.Require().NoError(s.DB.Create(&relational.SystemSecurityPlan{UUIDModel: relational.UUIDModel{ID: &id}}).Error)
	return id
}

func labels(pairs ...string) []relational.ScopeLabel {
	out := make([]relational.ScopeLabel, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, relational.ScopeLabel{Key: pairs[i], Value: pairs[i+1]})
	}
	return out
}

func (s *ScopeServiceIntegrationSuite) TestGetEmptyScope() {
	scope, err := s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.False(scope.AutoFilters)
	s.Empty(scope.Entries)
	s.Empty(scope.Exceptions)

	_, ok, err := s.svc.Filter(s.sspID)
	s.Require().NoError(err)
	s.False(ok, "an empty scope is not 'everything'")
}

func (s *ScopeServiceIntegrationSuite) TestUnknownSSP() {
	missing := uuid.New()
	_, err := s.svc.Get(missing)
	s.ErrorIs(err, sspscope.ErrSSPNotFound)
	_, err = s.svc.AddEntry(missing, labels("repo", "a"), nil)
	s.ErrorIs(err, sspscope.ErrSSPNotFound)
	_, err = s.svc.AddException(missing, labels("repo", "a"), nil)
	s.ErrorIs(err, sspscope.ErrSSPNotFound)
	s.ErrorIs(s.svc.SetAutoFilters(missing, true), sspscope.ErrSSPNotFound)
}

func (s *ScopeServiceIntegrationSuite) TestAddEntryNormalizesAndRejectsDuplicates() {
	actor := uuid.New()
	entry, err := s.svc.AddEntry(s.sspID, labels("Region", " eu-west-1", "aws_account", "111"), &actor)
	s.Require().NoError(err)
	s.Equal(relational.ScopeEntryOriginManual, entry.Origin)
	s.Equal(labels("aws_account", "111", "region", "eu-west-1"), []relational.ScopeLabel(entry.Labels))
	s.Equal(&actor, entry.CreatedBy)

	_, err = s.svc.AddEntry(s.sspID, labels("AWS_ACCOUNT", "111", "region", "EU-WEST-1"), nil)
	s.ErrorIs(err, sspscope.ErrDuplicateEntry, "entries matching the same evidence collide")

	_, err = s.svc.AddEntry(s.sspID, labels("repo", ""), nil)
	s.ErrorIs(err, sspscope.ErrInvalidLabels)

	// Another SSP may hold the same entry.
	_, err = s.svc.AddEntry(s.createSSP(), labels("aws_account", "111", "region", "eu-west-1"), nil)
	s.NoError(err)
}

func (s *ScopeServiceIntegrationSuite) TestExceptionRemovesEntry() {
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	_, err = s.svc.AddEntry(s.sspID, labels("repo", "b"), nil)
	s.Require().NoError(err)

	exception, err := s.svc.AddException(s.sspID, labels("REPO", "A"), nil)
	s.Require().NoError(err)

	scope, err := s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.Require().Len(scope.Entries, 1)
	s.Equal("b", scope.Entries[0].Labels[0].Value)
	s.Require().Len(scope.Exceptions, 1)
	s.Equal(*exception.ID, *scope.Exceptions[0].ID)

	_, err = s.svc.AddException(s.sspID, labels("repo", "a"), nil)
	s.ErrorIs(err, sspscope.ErrDuplicateException)

	// An exception without a matching entry is allowed: it pre-empts a template.
	_, err = s.svc.AddException(s.sspID, labels("repo", "never-seen"), nil)
	s.NoError(err)
}

func (s *ScopeServiceIntegrationSuite) TestEntryRemovesException() {
	_, err := s.svc.AddException(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	_, err = s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)

	scope, err := s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.Len(scope.Entries, 1)
	s.Empty(scope.Exceptions)
}

func (s *ScopeServiceIntegrationSuite) TestDeleteExceptionDoesNotRestoreEntry() {
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	exception, err := s.svc.AddException(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)

	s.Require().NoError(s.svc.DeleteException(s.sspID, *exception.ID))
	scope, err := s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.Empty(scope.Entries)
	s.Empty(scope.Exceptions)

	s.ErrorIs(s.svc.DeleteException(s.sspID, *exception.ID), sspscope.ErrExceptionNotFound)

	other, err := s.svc.AddException(s.sspID, labels("repo", "b"), nil)
	s.Require().NoError(err)
	s.ErrorIs(s.svc.DeleteException(s.createSSP(), *other.ID), sspscope.ErrExceptionNotFound, "exceptions are SSP-bound")
}

func (s *ScopeServiceIntegrationSuite) TestSetAutoFilters() {
	s.Require().NoError(s.svc.SetAutoFilters(s.sspID, true))
	scope, err := s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.True(scope.AutoFilters)

	s.Require().NoError(s.svc.SetAutoFilters(s.sspID, false))
	scope, err = s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.False(scope.AutoFilters)
}

func (s *ScopeServiceIntegrationSuite) TestLearnSkipsDuplicatesAndExceptions() {
	templateID := uuid.New()
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "manual"), nil)
	s.Require().NoError(err)
	_, err = s.svc.AddException(s.sspID, labels("repo", "excluded"), nil)
	s.Require().NoError(err)

	tooLong := labels("image", strings.Repeat("x", sspscope.MaxLabelPartLength+1))
	var result sspscope.LearnResult
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = s.svc.Learn(tx, s.sspID, templateID, [][]relational.ScopeLabel{
			labels("repo", "MANUAL"),
			labels("repo", "excluded"),
			tooLong,
			labels("repo", "new"),
			labels("Repo", "new"),
			labels("aws_account", "111", "region", "eu-west-1"),
		})
		return err
	})
	s.Require().NoError(err, "an invalid entry does not fail the others")
	s.Require().Len(result.Skipped, 1)
	s.Equal(tooLong, result.Skipped[0].Labels)
	s.Contains(result.Skipped[0].Reason, "at most")
	added := result.Added
	s.Require().Len(added, 2)
	for _, entry := range added {
		s.Equal(relational.ScopeEntryOriginTemplate, entry.Origin)
		s.Equal(&templateID, entry.TemplateID)
	}

	scope, err := s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.Len(scope.Entries, 3)
	s.Len(scope.Exceptions, 1)

	// Learning again adds nothing and never removes entries.
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = s.svc.Learn(tx, s.sspID, templateID, [][]relational.ScopeLabel{labels("repo", "new")})
		return err
	})
	s.Require().NoError(err)
	s.Empty(result.Added)
	scope, err = s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.Len(scope.Entries, 3)
}

func (s *ScopeServiceIntegrationSuite) TestLimits() {
	entries := make([][]relational.ScopeLabel, 0, sspscope.MaxEntries)
	for i := 0; i < sspscope.MaxEntries; i++ {
		entries = append(entries, labels("repo", uuid.NewString()))
	}
	s.Require().NoError(s.DB.Transaction(func(tx *gorm.DB) error {
		_, err := s.svc.Learn(tx, s.sspID, uuid.New(), entries)
		return err
	}))
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "one-too-many"), nil)
	s.ErrorIs(err, sspscope.ErrLimitExceeded)

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		_, err := s.svc.Learn(tx, s.sspID, uuid.New(), [][]relational.ScopeLabel{labels("repo", "one-too-many")})
		return err
	})
	s.ErrorIs(err, sspscope.ErrLimitExceeded)
}

func (s *ScopeServiceIntegrationSuite) TestFilterFollowsEntries() {
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	f, ok, err := s.svc.Filter(s.sspID)
	s.Require().NoError(err)
	s.Require().True(ok)
	s.Require().NotNil(f.Scope.Condition)
	s.Equal("repo", f.Scope.Condition.Label)
	s.Equal("a", f.Scope.Condition.Value)
}

func (s *ScopeServiceIntegrationSuite) TestConcurrentWritesSerialize() {
	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = s.svc.AddEntry(s.sspID, labels("repo", "same"), nil)
		}(i)
	}
	wg.Wait()
	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		s.ErrorIs(err, sspscope.ErrDuplicateEntry)
	}
	s.Equal(1, succeeded)
}

func (s *ScopeServiceIntegrationSuite) TestSSPDeleteCascades() {
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	_, err = s.svc.AddException(s.sspID, labels("repo", "b"), nil)
	s.Require().NoError(err)
	s.Require().NoError(s.svc.SetAutoFilters(s.sspID, true))

	s.Require().NoError(s.DB.Delete(&relational.SystemSecurityPlan{}, "id = ?", s.sspID).Error)
	for _, model := range []any{&relational.SSPScopeEntry{}, &relational.SSPScopeException{}, &relational.SSPScopeSettings{}} {
		var count int64
		s.Require().NoError(s.DB.Model(model).Where("ssp_id = ?", s.sspID).Count(&count).Error)
		s.Zero(count)
	}
}

func (s *ScopeServiceIntegrationSuite) followingFilter(base labelfilter.Filter) uuid.UUID {
	var id uuid.UUID
	s.Require().NoError(s.DB.Transaction(func(tx *gorm.DB) error {
		effective, err := sspscope.FollowScope(tx, s.sspID, base)
		if err != nil {
			return err
		}
		scopeBase := datatypes.NewJSONType(base)
		filter := relational.Filter{Name: "follows", SSPID: &s.sspID, ScopeBase: &scopeBase, Filter: datatypes.NewJSONType(effective)}
		if err := tx.Create(&filter).Error; err != nil {
			return err
		}
		id = *filter.ID
		return nil
	}))
	return id
}

// matches reports which of the label sets the stored filter matches.
func (s *ScopeServiceIntegrationSuite) storedMatches(filterID uuid.UUID, sets ...map[string][]string) []int {
	var filter relational.Filter
	s.Require().NoError(s.DB.First(&filter, "id = ?", filterID).Error)
	got, err := labelfilter.NewIndex(sets).Match(filter.Filter.Data().Scope)
	s.Require().NoError(err)
	return got
}

func (s *ScopeServiceIntegrationSuite) TestFollowScopeNeedsEntries() {
	_, err := sspscope.FollowScope(s.DB, s.sspID, labelfilter.Filter{})
	s.ErrorIs(err, sspscope.ErrScopeEmpty)
	_, err = sspscope.FollowScope(s.DB, uuid.New(), labelfilter.Filter{})
	s.ErrorIs(err, sspscope.ErrSSPNotFound)
}

func (s *ScopeServiceIntegrationSuite) TestFollowingFiltersAreRewrittenOnScopeChange() {
	prodA := map[string][]string{"env": {"prod"}, "repo": {"a"}}
	prodB := map[string][]string{"env": {"prod"}, "repo": {"b"}}
	devA := map[string][]string{"env": {"dev"}, "repo": {"a"}}

	_, err := s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	prod := labelfilter.Filter{Scope: &labelfilter.Scope{Condition: &labelfilter.Condition{Label: "env", Operator: "=", Value: "prod"}}}
	id := s.followingFilter(prod)
	unrelated := relational.Filter{Name: "plain", SSPID: &s.sspID, Filter: datatypes.NewJSONType(prod)}
	s.Require().NoError(s.DB.Create(&unrelated).Error)

	s.Equal([]int{0}, s.storedMatches(id, prodA, prodB, devA))

	_, err = s.svc.AddEntry(s.sspID, labels("repo", "b"), nil)
	s.Require().NoError(err)
	s.Equal([]int{0, 1}, s.storedMatches(id, prodA, prodB, devA), "adding an entry widens the filter")

	_, err = s.svc.AddException(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	s.Equal([]int{1}, s.storedMatches(id, prodA, prodB, devA), "an exception narrows it")

	s.Require().NoError(s.DB.Transaction(func(tx *gorm.DB) error {
		_, err := s.svc.Learn(tx, s.sspID, uuid.New(), [][]relational.ScopeLabel{labels("env", "dev")})
		return err
	}))
	s.Equal([]int{1}, s.storedMatches(id, prodA, prodB, devA), "the base still applies to learned entries")
	s.Equal([]int{0, 1}, s.storedMatches(*unrelated.ID, prodA, prodB, devA), "filters that do not follow are untouched")
}

func (s *ScopeServiceIntegrationSuite) TestExceptionCannotEmptyScopeWhileFiltersFollow() {
	_, err := s.svc.AddEntry(s.sspID, labels("repo", "a"), nil)
	s.Require().NoError(err)
	s.followingFilter(labelfilter.Filter{})
	s.followingFilter(labelfilter.Filter{})

	_, err = s.svc.AddException(s.sspID, labels("repo", "a"), nil)
	s.Require().ErrorIs(err, sspscope.ErrScopeWouldEmpty)
	var wouldEmpty *sspscope.ScopeWouldEmptyError
	s.Require().ErrorAs(err, &wouldEmpty)
	s.EqualValues(2, wouldEmpty.Following)
	s.Contains(err.Error(), "2 filter(s)")

	scope, err := s.svc.Get(s.sspID)
	s.Require().NoError(err)
	s.Len(scope.Entries, 1, "nothing changed")
	s.Empty(scope.Exceptions)

	// Another entry's exception, or one with a second entry left, is fine.
	_, err = s.svc.AddException(s.sspID, labels("repo", "other"), nil)
	s.NoError(err)
	_, err = s.svc.AddEntry(s.sspID, labels("repo", "b"), nil)
	s.Require().NoError(err)
	_, err = s.svc.AddException(s.sspID, labels("repo", "a"), nil)
	s.NoError(err)
}
