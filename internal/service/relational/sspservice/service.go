// Package sspservice creates System Security Plans and binds profiles to them. The SSP
// HTTP handlers and SSP template discovery both go through it, so an SSP is built the
// same way whoever creates it.
package sspservice

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/defenseunicorns/go-oscal/src/pkg/versioning"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrSSPNotFound is returned when the SSP does not exist.
	ErrSSPNotFound = errors.New("SSP not found")
	// ErrProfileNotFound is returned when the profile does not exist.
	ErrProfileNotFound = errors.New("profile not found")
	// ErrProfileAlreadyBound is returned when the profile is already bound to the SSP.
	ErrProfileAlreadyBound = errors.New("profile is already bound to this SSP")
	// ErrNoResolvedControls is returned when the profile resolves to no controls.
	ErrNoResolvedControls = errors.New("no controls were resolved from the selected profile")
)

// ControlResolver returns the catalog-canonical control IDs a profile selects,
// de-duplicated.
type ControlResolver func(profileID uuid.UUID) ([]string, error)

// Service creates SSPs and binds profiles to them. Every method takes the *gorm.DB to
// run on, so callers can pass a transaction.
type Service struct {
	resolveControls ControlResolver
}

// NewService returns a Service resolving profile controls with resolveControls.
func NewService(resolveControls ControlResolver) *Service {
	return &Service{resolveControls: resolveControls}
}

// Create stores ssp, stamping its last-modified time and OSCAL version. Callers
// validate ssp first.
func (s *Service) Create(db *gorm.DB, ssp oscalTypes_1_1_3.SystemSecurityPlan) (*relational.SystemSecurityPlan, error) {
	now := time.Now()
	relSSP := &relational.SystemSecurityPlan{}
	relSSP.UnmarshalOscal(ssp)
	relSSP.Metadata.LastModified = &now
	relSSP.Metadata.OscalVersion = versioning.GetLatestSupportedVersion()
	if err := db.Create(relSSP).Error; err != nil {
		return nil, err
	}
	return relSSP, nil
}

// AttachProfile binds the profile to the SSP and creates an implemented requirement for
// each of the profile's controls the SSP does not implement yet (matched
// case-insensitively, stored with the catalog's casing). The writes run in one
// transaction (a savepoint when db is already one). It fails with ErrSSPNotFound,
// ErrProfileNotFound, ErrProfileAlreadyBound or ErrNoResolvedControls.
func (s *Service) AttachProfile(db *gorm.DB, sspID, profileID uuid.UUID) error {
	var ssp relational.SystemSecurityPlan
	if err := db.Preload("ControlImplementation").First(&ssp, "id = ?", sspID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSSPNotFound
		}
		return err
	}
	var profile relational.Profile
	if err := db.First(&profile, "id = ?", profileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProfileNotFound
		}
		return err
	}

	var duplicateBinding bool
	err := db.Transaction(func(tx *gorm.DB) error {
		// Insert join-table row idempotently; ON CONFLICT DO NOTHING avoids a TOCTOU race.
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relational.SSPProfile{
			SystemSecurityPlanID: sspID,
			ProfileID:            profileID,
		})
		if result.Error != nil {
			return fmt.Errorf("failed to insert ssp_profiles row: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			duplicateBinding = true
			return nil
		}

		controlIDs, err := s.resolveControls(profileID)
		if err != nil {
			return fmt.Errorf("failed to resolve control IDs for profile: %w", err)
		}
		if len(controlIDs) == 0 {
			return ErrNoResolvedControls
		}

		// Ensure ControlImplementation exists
		if ssp.ControlImplementation.ID == nil {
			newID := uuid.New()
			ssp.ControlImplementation = relational.ControlImplementation{
				UUIDModel:            relational.UUIDModel{ID: &newID},
				Description:          "Control implementation",
				SystemSecurityPlanId: *ssp.ID,
			}
			if err := tx.Create(&ssp.ControlImplementation).Error; err != nil {
				return err
			}
		}

		// Add any new ImplementedRequirements for the profile's controls
		var existingControlIDs []string
		if err := tx.Model(&relational.ImplementedRequirement{}).
			Where("control_implementation_id = ?", ssp.ControlImplementation.ID).
			Pluck("control_id", &existingControlIDs).Error; err != nil {
			return err
		}

		// Control IDs compare case-insensitively; the lowered form is only a key.
		existingMap := make(map[string]bool, len(existingControlIDs))
		for _, id := range existingControlIDs {
			existingMap[strings.ToLower(id)] = true
		}

		var newReqs []relational.ImplementedRequirement
		for _, controlID := range controlIDs {
			// Dedupe case-insensitively, but persist the canonical casing.
			if !existingMap[strings.ToLower(controlID)] {
				newUUID := uuid.New()
				newReqs = append(newReqs, relational.ImplementedRequirement{
					UUIDModel:               relational.UUIDModel{ID: &newUUID},
					ControlImplementationId: *ssp.ControlImplementation.ID,
					ControlId:               controlID,
				})
			}
		}

		if len(newReqs) > 0 {
			if err := tx.Create(&newReqs).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	if duplicateBinding {
		return ErrProfileAlreadyBound
	}
	return nil
}
