package sspscope

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ReconcileAutoFilters creates the SSP's missing automatic per-policy filters: one
// scope-following filter per _policy value of its in-scope latest evidence, with no
// controls or components. It never deletes or renames automatic filters, and never
// recreates one that was deleted. It is a no-op when automatic filters are off or the
// scope is empty. It returns how many filters it created.
func (s *Service) ReconcileAutoFilters(sspID uuid.UUID) (int, error) {
	created := 0
	err := s.write(sspID, func(tx *gorm.DB) error {
		var err error
		created, err = reconcileAutoFilters(tx, sspID)
		return err
	})
	return created, err
}

// ReconcileAllAutoFilters reconciles every SSP with automatic filters on. An SSP that
// fails does not stop the others; the errors are joined.
func (s *Service) ReconcileAllAutoFilters(ctx context.Context) (int, error) {
	var sspIDs []uuid.UUID
	if err := s.db.WithContext(ctx).Model(&relational.SSPScopeSettings{}).
		Where("auto_filters = ?", true).Order("ssp_id").Pluck("ssp_id", &sspIDs).Error; err != nil {
		return 0, err
	}
	scoped := &Service{db: s.db.WithContext(ctx)}
	created := 0
	var errs []error
	for _, sspID := range sspIDs {
		if err := ctx.Err(); err != nil {
			return created, err
		}
		n, err := scoped.ReconcileAutoFilters(sspID)
		created += n
		if err != nil && !errors.Is(err, ErrSSPNotFound) {
			errs = append(errs, fmt.Errorf("ssp %s: %w", sspID, err))
		}
	}
	return created, errors.Join(errs...)
}

// reconcileAutoFilters does ReconcileAutoFilters inside tx, which must hold the SSP's
// scope lock.
func reconcileAutoFilters(tx *gorm.DB, sspID uuid.UUID) (int, error) {
	var enabled int64
	if err := tx.Model(&relational.SSPScopeSettings{}).Where("ssp_id = ? AND auto_filters = ?", sspID, true).Count(&enabled).Error; err != nil {
		return 0, err
	}
	if enabled == 0 {
		return 0, nil
	}
	scope, ok, err := scopeFilter(tx, sspID)
	if err != nil || !ok {
		return 0, err
	}
	policies, err := inScopePolicies(tx, scope)
	if err != nil {
		return 0, err
	}
	var recorded []string
	if err := tx.Model(&relational.SSPScopePolicyFilter{}).Where("ssp_id = ?", sspID).Pluck("policy", &recorded).Error; err != nil {
		return 0, err
	}
	planned := planAutoFilters(policies, recorded)
	for _, policy := range planned {
		base := labelfilter.Filter{Scope: &labelfilter.Scope{Condition: &labelfilter.Condition{
			Label:    relational.EvidencePolicyLabel,
			Operator: "=",
			Value:    policy,
		}}}
		scopeBase := datatypes.NewJSONType(base)
		filter := relational.Filter{
			Name:      policy,
			SSPID:     &sspID,
			Filter:    datatypes.NewJSONType(And(base, scope)),
			ScopeBase: &scopeBase,
			Origin:    relational.FilterOriginScopePolicy,
		}
		if err := tx.Create(&filter).Error; err != nil {
			return 0, err
		}
		if err := tx.Create(&relational.SSPScopePolicyFilter{SSPID: sspID, Policy: policy, FilterID: filter.ID}).Error; err != nil {
			return 0, err
		}
	}
	return len(planned), nil
}

// planAutoFilters returns the policies that need a filter: those found in evidence with
// no recorded filter. Label values match case-insensitively, so policies are compared
// case-insensitively and the first spelling found wins.
func planAutoFilters(found, recorded []string) []string {
	seen := make(map[string]struct{}, len(recorded)+len(found))
	for _, policy := range recorded {
		seen[strings.ToLower(policy)] = struct{}{}
	}
	out := []string{}
	for _, policy := range found {
		key := strings.ToLower(policy)
		if strings.TrimSpace(policy) == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, policy)
	}
	return out
}

// inScopePolicies returns the distinct _policy values of the in-scope latest evidence,
// in ascending order.
func inScopePolicies(tx *gorm.DB, scope labelfilter.Filter) ([]string, error) {
	latest := relational.LatestEvidenceQuery(tx, relational.LatestEvidenceOptions{})
	query, err := relational.GetEvidenceSearchByFilterQuery(latest, tx, scope)
	if err != nil {
		return nil, err
	}
	var policies []string
	err = query.
		Joins("JOIN evidence_labels el ON el.evidence_id = l.id").
		Where("lower(el.labels_name) = ?", relational.EvidencePolicyLabel).
		Distinct().
		Order("el.labels_value ASC").
		Pluck("el.labels_value", &policies).Error
	return policies, err
}
