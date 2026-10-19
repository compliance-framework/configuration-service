package ssptemplates

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	"github.com/compliance-framework/api/internal/service/relational/sspservice"
	"github.com/compliance-framework/api/internal/service/relational/templates"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	// MaxNewSSPsPerRun caps the SSPs one run of a template creates; the next run creates
	// the rest.
	MaxNewSSPsPerRun = 100
	// maxSkippedSample is how many skipped entries an instance keeps.
	maxSkippedSample = 5
)

// sspIDNamespace seeds the ids of the SSPs templates create, so two concurrent runs
// cannot create two SSPs for the same key values.
var sspIDNamespace = uuid.MustParse("0bc0400e-f864-41c0-aad4-2a84a628bb2e")

// SSPID is the id of the SSP the template creates for the key values with keyHash.
func SSPID(templateID uuid.UUID, keyHash string) uuid.UUID {
	return uuid.NewSHA1(sspIDNamespace, append(templateID[:], keyHash...))
}

// RunResult counts what one discovery run of a template did.
type RunResult struct {
	Created int // SSPs created
	Updated int // existing SSPs whose instance was refreshed and scope learned
	Failed  int // groups whose apply failed
}

// Discovery applies SSP templates: it creates their SSPs and keeps their scopes up to
// date. Each group is applied in its own transaction holding the template's lock, so a
// periodic and a manual run do not interleave.
type Discovery struct {
	db    *gorm.DB
	ssps  *sspservice.Service
	scope *sspscope.Service
}

// NewDiscovery returns a Discovery attaching profiles with resolveControls.
func NewDiscovery(db *gorm.DB, resolveControls sspservice.ControlResolver) *Discovery {
	return &Discovery{db: db, ssps: sspservice.NewService(resolveControls), scope: sspscope.NewService(db)}
}

// RunAll runs every enabled template over one load of the latest evidence, going on
// past a failing template.
func (d *Discovery) RunAll(ctx context.Context) error {
	db := d.db.WithContext(ctx)
	var ids []uuid.UUID
	if err := db.Model(&relational.SSPTemplate{}).Where("enabled").Order("created_at").Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	streams, err := relational.LoadLatestEvidenceStreams(db, nil)
	if err != nil {
		return err
	}
	var errs []error
	for _, id := range ids {
		if _, err := d.run(db, id, streams); err != nil {
			errs = append(errs, fmt.Errorf("SSP template %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

// Run applies the template to the latest evidence. A deleted or disabled template is
// skipped. A group that fails is recorded on its instance (when it has one) and
// reported in the error; the other groups are still applied.
func (d *Discovery) Run(ctx context.Context, templateID uuid.UUID) (RunResult, error) {
	db := d.db.WithContext(ctx)
	streams, err := relational.LoadLatestEvidenceStreams(db, nil)
	if err != nil {
		return RunResult{}, err
	}
	return d.run(db, templateID, streams)
}

func (d *Discovery) run(db *gorm.DB, templateID uuid.UUID, streams []relational.LatestEvidenceStream) (RunResult, error) {
	tmpl, err := get(db, templateID)
	if errors.Is(err, ErrNotFound) {
		return RunResult{}, nil
	}
	if err != nil {
		return RunResult{}, err
	}
	if !tmpl.Enabled {
		return RunResult{}, nil
	}
	var instances []relational.SSPTemplateInstance
	if err := db.Where("template_id = ?", templateID).Find(&instances).Error; err != nil {
		return RunResult{}, err
	}
	entries, exceptions, err := scopeHashes(db, instances)
	if err != nil {
		return RunResult{}, err
	}

	result := RunResult{}
	var errs []error
	for _, planned := range BuildPlan(tmpl.SSPTemplate, streams, instances, entries, exceptions).SSPs {
		if planned.Exists && planned.SSPID == nil {
			continue // the SSP was deleted: never recreate it
		}
		if !planned.Exists && result.Created >= MaxNewSSPsPerRun {
			continue
		}
		outcome, err := d.apply(db, tmpl.SSPTemplate, planned)
		if err != nil {
			result.Failed++
			errs = append(errs, fmt.Errorf("key values %v: %w", planned.KeyValues, err))
			if planned.Exists {
				errs = append(errs, setLastError(db, templateID, planned.KeyHash, err))
			}
			continue
		}
		switch outcome {
		case outcomeCreated:
			result.Created++
		case outcomeUpdated:
			result.Updated++
		}
	}
	if err := db.Model(&relational.SSPTemplate{}).Where("id = ?", templateID).UpdateColumn("last_run_at", time.Now()).Error; err != nil {
		errs = append(errs, err)
	}
	return result, errors.Join(errs...)
}

// outcome is what apply did with a group.
type outcome int

const (
	outcomeSkipped outcome = iota // the SSP was deleted since the plan was made
	outcomeCreated
	outcomeUpdated
)

// apply creates or refreshes one group's SSP and learns its scope, then attaches the
// template's profiles to a new SSP (or retries those that failed).
func (d *Discovery) apply(db *gorm.DB, tmpl relational.SSPTemplate, planned PlannedSSP) (outcome, error) {
	var instance relational.SSPTemplateInstance
	result := outcomeUpdated
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockTemplate(tx, *tmpl.ID); err != nil {
			return err
		}
		found := tx.Where("template_id = ? AND key_hash = ?", tmpl.ID, planned.KeyHash).Limit(1).Find(&instance)
		if found.Error != nil {
			return found.Error
		}
		now := time.Now()
		switch {
		case found.RowsAffected == 0:
			ssp, err := newSSP(tmpl, planned)
			if err != nil {
				return err
			}
			stored, err := d.ssps.Create(tx, ssp)
			if err != nil {
				return err
			}
			instance = relational.SSPTemplateInstance{
				TemplateID: *tmpl.ID, KeyHash: planned.KeyHash, KeyValues: keyValuesMap(planned.KeyValues),
				SSPID: stored.ID, CreatedAt: now, ProfilesPending: len(tmpl.ProfileIDs) > 0,
			}
			if err := tx.Create(&instance).Error; err != nil {
				return err
			}
			result = outcomeCreated
		case instance.SSPID == nil:
			result = outcomeSkipped
			return nil
		}

		skipped := append([]relational.SkippedScopeEntry{}, planned.InvalidEntries...)
		if len(planned.Learn) > 0 {
			learned, err := d.scope.Learn(tx, *instance.SSPID, *tmpl.ID, planned.Learn)
			if err != nil {
				return err
			}
			for _, entry := range learned.Skipped {
				skipped = append(skipped, relational.SkippedScopeEntry{Labels: entry.Labels, Reason: entry.Reason})
			}
		}
		instance.LastSeenAt = now
		instance.SkippedEntries = len(skipped)
		instance.SkippedSample = skipped[:min(len(skipped), maxSkippedSample)]
		updates := map[string]any{"last_seen_at": now, "skipped_entries": instance.SkippedEntries, "skipped_sample": instance.SkippedSample}
		if !instance.ProfilesPending {
			updates["last_error"] = nil
		}
		return tx.Model(&instance).Updates(updates).Error
	})
	if err != nil || result == outcomeSkipped || !instance.ProfilesPending {
		return result, err
	}
	return result, d.attachProfiles(db, tmpl, instance)
}

// attachProfiles attaches the template's profiles to the instance's SSP, skipping those
// already bound. A failure does not undo the SSP: it stays in LastError and
// ProfilesPending, and the next run retries.
func (d *Discovery) attachProfiles(db *gorm.DB, tmpl relational.SSPTemplate, instance relational.SSPTemplateInstance) error {
	var failures []string
	for _, profileID := range tmpl.ProfileIDs {
		err := d.ssps.AttachProfile(db, *instance.SSPID, profileID)
		if err != nil && !errors.Is(err, sspservice.ErrProfileAlreadyBound) {
			failures = append(failures, fmt.Sprintf("attach profile %s: %v", profileID, err))
		}
	}
	var lastError *string
	if len(failures) > 0 {
		message := strings.Join(failures, "; ")
		lastError = &message
	}
	return db.Model(&instance).Updates(map[string]any{"profiles_pending": len(failures) > 0, "last_error": lastError}).Error
}

// newSSP builds the SSP for a group: title, description and props rendered from its key
// values, a prop naming the template, status under-development and nothing else, as the
// UI's "new SSP" form leaves it.
func newSSP(tmpl relational.SSPTemplate, planned PlannedSSP) (oscalTypes_1_1_3.SystemSecurityPlan, error) {
	description := planned.Title
	if tmpl.DescriptionTemplate != nil {
		rendered, err := templates.RenderTemplate(*tmpl.DescriptionTemplate, planned.KeyValues)
		if err != nil {
			return oscalTypes_1_1_3.SystemSecurityPlan{}, err
		}
		if strings.TrimSpace(rendered) != "" {
			description = rendered
		}
	}
	props := make([]oscalTypes_1_1_3.Property, 0, len(tmpl.Props)+1)
	for _, prop := range tmpl.Props {
		value, err := templates.RenderTemplate(prop.Value, planned.KeyValues)
		if err != nil {
			return oscalTypes_1_1_3.SystemSecurityPlan{}, err
		}
		if value != "" { // OSCAL props need a value
			props = append(props, oscalTypes_1_1_3.Property{Name: prop.Name, Value: value})
		}
	}
	props = append(props, oscalTypes_1_1_3.Property{Name: "ssp-template", Ns: relational.CCFOSCALNamespace, Value: tmpl.ID.String()})
	return oscalTypes_1_1_3.SystemSecurityPlan{
		UUID:     SSPID(*tmpl.ID, planned.KeyHash).String(),
		Metadata: oscalTypes_1_1_3.Metadata{Title: planned.Title, Version: "1.0", Props: &props},
		SystemCharacteristics: oscalTypes_1_1_3.SystemCharacteristics{
			SystemName:  planned.Title,
			Description: description,
			Status:      oscalTypes_1_1_3.Status{State: "under-development"},
		},
	}, nil
}

func setLastError(db *gorm.DB, templateID uuid.UUID, keyHash string, cause error) error {
	message := cause.Error()
	return db.Model(&relational.SSPTemplateInstance{}).Where("template_id = ? AND key_hash = ?", templateID, keyHash).
		Update("last_error", &message).Error
}

func keyValuesMap(keyValues map[string]string) datatypes.JSONMap {
	out := make(datatypes.JSONMap, len(keyValues))
	for key, value := range keyValues {
		out[key] = value
	}
	return out
}

// lockTemplate takes the template's transaction-scoped advisory lock (postgres only).
func lockTemplate(tx *gorm.DB, templateID uuid.UUID) error {
	if tx.Name() != "postgres" {
		return nil
	}
	sum := sha256.Sum256([]byte("ssp-template:" + templateID.String()))
	key1 := int32(binary.BigEndian.Uint32(sum[0:4]))
	key2 := int32(binary.BigEndian.Uint32(sum[4:8]))
	return tx.Exec("SELECT pg_advisory_xact_lock(?, ?)", key1, key2).Error
}
