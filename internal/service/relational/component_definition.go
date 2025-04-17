package relational

type ComponentDefinition struct {
	UUIDModel
	Metadata   Metadata   `json:"metadata" gorm:"polymorphic:Parent;"`
	BackMatter BackMatter `json:"back-matter" gorm:"polymorphic:Parent;"`

	//oscaltypes113.ComponentDefinition
}
