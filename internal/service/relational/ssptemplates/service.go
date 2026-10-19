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
	// Select("*") so a false Enabled is written rather than replaced by the column default.
	if err := s.db.Select("*").Create(&tmpl).Error; err != nil {
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
