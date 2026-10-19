package ssptemplates

import (
	"errors"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrTemplateDisabled is returned when a disabled template is asked to run.
	ErrTemplateDisabled = errors.New("SSP template is disabled")
	// ErrInstanceNotFound is returned when the template has no instance with the key hash.
	ErrInstanceNotFound = errors.New("SSP template instance not found")
	// ErrSSPExists is returned when forgetting an instance whose SSP still exists.
	ErrSSPExists = errors.New("the SSP still exists: only an instance whose SSP was deleted can be forgotten")
)

// Instance is an SSP a template created, with the SSP's current title (nil once the SSP
// was deleted).
type Instance struct {
	relational.SSPTemplateInstance
	Title *string
}

// CheckRunnable returns ErrNotFound when the template does not exist and
// ErrTemplateDisabled when it is disabled.
func (s *Service) CheckRunnable(id uuid.UUID) error {
	tmpl, err := get(s.db, id)
	if err != nil {
		return err
	}
	if !tmpl.Enabled {
		return ErrTemplateDisabled
	}
	return nil
}

// Instances returns a page of the template's instances, newest first, and their total.
func (s *Service) Instances(templateID uuid.UUID, limit, offset int) ([]Instance, int64, error) {
	if _, err := get(s.db, templateID); err != nil {
		return nil, 0, err
	}
	query := s.db.Model(&relational.SSPTemplateInstance{}).Where("template_id = ?", templateID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []relational.SSPTemplateInstance
	if err := query.Order("created_at DESC, key_hash ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	sspIDs := []uuid.UUID{}
	for _, row := range rows {
		if row.SSPID != nil {
			sspIDs = append(sspIDs, *row.SSPID)
		}
	}
	titles := map[uuid.UUID]string{}
	if len(sspIDs) > 0 {
		var ssps []relational.SystemSecurityPlan
		if err := s.db.Preload("Metadata").Where("id IN ?", sspIDs).Find(&ssps).Error; err != nil {
			return nil, 0, err
		}
		for _, ssp := range ssps {
			titles[*ssp.ID] = ssp.Metadata.Title
		}
	}
	instances := make([]Instance, 0, len(rows))
	for _, row := range rows {
		instance := Instance{SSPTemplateInstance: row}
		if row.SSPID != nil {
			if title, ok := titles[*row.SSPID]; ok {
				instance.Title = &title
			}
		}
		instances = append(instances, instance)
	}
	return instances, total, nil
}

// Forget deletes the instance with keyHash, so discovery creates its SSP again. Only an
// instance whose SSP was deleted can be forgotten (ErrSSPExists otherwise).
func (s *Service) Forget(templateID uuid.UUID, keyHash string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var instance relational.SSPTemplateInstance
		found := tx.Where("template_id = ? AND key_hash = ?", templateID, keyHash).Limit(1).Find(&instance)
		if found.Error != nil {
			return found.Error
		}
		if found.RowsAffected == 0 {
			return ErrInstanceNotFound
		}
		if instance.SSPID != nil {
			return ErrSSPExists
		}
		return tx.Where("template_id = ? AND key_hash = ? AND ssp_id IS NULL", templateID, keyHash).
			Delete(&relational.SSPTemplateInstance{}).Error
	})
}
