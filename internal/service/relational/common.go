package relational

import (
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Prop oscaltypes113.Property
type Props []*Prop

type Link oscaltypes113.Link
type Links []*Link

type UUIDModel struct {
	ID *uuid.UUID `json:"id" gorm:"type:uuid;primary_key;"`
}

func (u *UUIDModel) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == nil {
		id := uuid.New()
		u.ID = &id
	}
	return
}
