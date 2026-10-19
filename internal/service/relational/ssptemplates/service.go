package ssptemplates

import (
	"errors"
	"slices"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrNotFound is returned when the template does not exist.
var ErrNotFound = errors.New("SSP template not found")

// Service reads and writes SSP templates.
type Service struct {
	db *gorm.DB
}

// NewService returns a Service over db.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Listed is a template with the number of SSPs it created that still exist.
type Listed struct {
	relational.SSPTemplate
	SSPCount int64
}

// withSSPCount selects a template and its Listed.SSPCount.
const withSSPCount = `ssp_templates.*, (SELECT COUNT(*) FROM ssp_template_instances i
	WHERE i.template_id = ssp_templates.id AND i.ssp_id IS NOT NULL) AS ssp_count`

// List returns a page of templates, newest first, and the total count.
func (s *Service) List(limit, offset int) ([]Listed, int64, error) {
	var total int64
	if err := s.db.Model(&relational.SSPTemplate{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []Listed{}
	if err := s.db.Model(&relational.SSPTemplate{}).Select(withSSPCount).Order("created_at DESC, id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Get returns the template or ErrNotFound.
func (s *Service) Get(id uuid.UUID) (Listed, error) {
	return get(s.db, id)
}

// Create validates in and stores it as a new template.
func (s *Service) Create(in Input) (Listed, error) {
	tmpl, err := Build(in)
	if err != nil {
		return Listed{}, err
	}
	if err := checkProfiles(s.db, tmpl.ProfileIDs); err != nil {
		return Listed{}, err
	}
	if err := s.db.Create(&tmpl).Error; err != nil {
		return Listed{}, err
	}
	return Listed{SSPTemplate: tmpl}, nil
}

// Update replaces the template's settings with in. Its keys cannot change: a template
// with other keys would describe other SSPs, so it must be a new template.
func (s *Service) Update(id uuid.UUID, in Input) (Listed, error) {
	tmpl, err := Build(in)
	if err != nil {
		return Listed{}, err
	}
	var existing Listed
	err = s.db.Transaction(func(tx *gorm.DB) error {
		existing, err = get(tx.Clauses(clause.Locking{Strength: "UPDATE"}), id)
		if err != nil {
			return err
		}
		if !slices.Equal(existing.Keys, tmpl.Keys) {
			return invalid("keys cannot change after creation; create a new template instead")
		}
		if err := checkProfiles(tx, tmpl.ProfileIDs); err != nil {
			return err
		}
		tmpl.ID = existing.ID
		tmpl.CreatedAt = existing.CreatedAt
		tmpl.LastRunAt = existing.LastRunAt
		return tx.Select("*").Save(&tmpl).Error
	})
	if err != nil {
		return Listed{}, err
	}
	return Listed{SSPTemplate: tmpl, SSPCount: existing.SSPCount}, nil
}

// Delete deletes the template and its instance records. The SSPs it created and their
// scope entries are kept.
func (s *Service) Delete(id uuid.UUID) error {
	res := s.db.Where("id = ?", id).Delete(&relational.SSPTemplate{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func get(db *gorm.DB, id uuid.UUID) (Listed, error) {
	var tmpl Listed
	if err := db.Model(&relational.SSPTemplate{}).Select(withSSPCount).Where("ssp_templates.id = ?", id).Take(&tmpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Listed{}, ErrNotFound
		}
		return Listed{}, err
	}
	return tmpl, nil
}

// checkProfiles fails with ErrInvalidTemplate when a profile id does not exist.
func checkProfiles(db *gorm.DB, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	var found []uuid.UUID
	if err := db.Model(&relational.Profile{}).Where("id IN ?", ids).Pluck("id", &found).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if !slices.Contains(found, id) {
			return invalid("profile %s does not exist", id)
		}
	}
	return nil
}

// Preview plans the template in against the latest evidence, writing nothing. id, when
// set, names the saved template whose instances say which SSPs already exist.
func (s *Service) Preview(id *uuid.UUID, in Input) (Plan, error) {
	tmpl, err := Build(in)
	if err != nil {
		return Plan{}, err
	}
	var instances []relational.SSPTemplateInstance
	if id != nil {
		if _, err := get(s.db, *id); err != nil {
			return Plan{}, err
		}
		if err := s.db.Where("template_id = ?", *id).Find(&instances).Error; err != nil {
			return Plan{}, err
		}
	}
	streams, err := relational.LoadLatestEvidenceStreams(s.db, nil)
	if err != nil {
		return Plan{}, err
	}
	entries, exceptions, err := scopeHashes(s.db, instances)
	if err != nil {
		return Plan{}, err
	}
	return BuildPlan(tmpl, streams, instances, entries, exceptions), nil
}

// scopeHashes loads the MatchHashes of the scope entries and exceptions of the
// instances' SSPs.
func scopeHashes(db *gorm.DB, instances []relational.SSPTemplateInstance) (entries, exceptions HashesBySSP, err error) {
	sspIDs := make([]uuid.UUID, 0, len(instances))
	for _, instance := range instances {
		if instance.SSPID != nil {
			sspIDs = append(sspIDs, *instance.SSPID)
		}
	}
	entries, exceptions = HashesBySSP{}, HashesBySSP{}
	if len(sspIDs) == 0 {
		return entries, exceptions, nil
	}
	for model, into := range map[any]HashesBySSP{&relational.SSPScopeEntry{}: entries, &relational.SSPScopeException{}: exceptions} {
		var rows []struct {
			SSPID     uuid.UUID
			MatchHash string
		}
		if err := db.Model(model).Select("ssp_id, match_hash").Where("ssp_id IN ?", sspIDs).Scan(&rows).Error; err != nil {
			return nil, nil, err
		}
		for _, row := range rows {
			if into[row.SSPID] == nil {
				into[row.SSPID] = map[string]struct{}{}
			}
			into[row.SSPID][row.MatchHash] = struct{}{}
		}
	}
	return entries, exceptions, nil
}
