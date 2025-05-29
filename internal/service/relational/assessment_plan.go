package relational

import (
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"gorm.io/datatypes"
)

type AssessmentPlan struct {
	UUIDModel
	Metadata   Metadata    `gorm:"polymorphic:Parent;"`
	BackMatter *BackMatter `gorm:"polymorphic:Parent;"`
	ImportSSP  datatypes.JSONType[ImportSsp]

	/**
	"local-definitions": {
	  "title": "Local Definitions",
	  "description": "Used to define data objects that are used in the assessment plan, that do not appear in the referenced SSP.",
	  "type": "object",
	  "properties": {
		"components": {
		  "type": "array",
		  "minItems": 1,
		  "items": {
			"$ref": "#/definitions/oscal-complete-oscal-implementation-common:system-component"
		  }
		},
		"inventory-items": {
		  "type": "array",
		  "minItems": 1,
		  "items": {
			"$ref": "#/definitions/oscal-complete-oscal-implementation-common:inventory-item"
		  }
		},
		"users": {
		  "type": "array",
		  "minItems": 1,
		  "items": {
			"$ref": "#/definitions/oscal-complete-oscal-implementation-common:system-user"
		  }
		},
		"objectives-and-methods": {
		  "type": "array",
		  "minItems": 1,
		  "items": {
			"$ref": "#/definitions/oscal-complete-oscal-assessment-common:local-objective"
		  }
		},
		"activities": {
		  "type": "array",
		  "minItems": 1,
		  "items": {
			"$ref": "#/definitions/oscal-complete-oscal-assessment-common:activity"
		  }
		},
		"remarks": {
		  "$ref": "#/definitions/oscal-complete-oscal-metadata:remarks"
		}
	  },
	  "additionalProperties": false
	},
	"terms-and-conditions": {
	  "title": "Assessment Plan Terms and Conditions",
	  "description": "Used to define various terms and conditions under which an assessment, described by the plan, can be performed. Each child part defines a different type of term or condition.",
	  "type": "object",
	  "properties": {
		"parts": {
		  "type": "array",
		  "minItems": 1,
		  "items": {
			"$ref": "#/definitions/oscal-complete-oscal-assessment-common:assessment-part"
		  }
		}
	  },
	  "additionalProperties": false
	},
	"reviewed-controls": {
	  "$ref": "#/definitions/oscal-complete-oscal-assessment-common:reviewed-controls"
	},
	"assessment-subjects": {
	  "type": "array",
	  "minItems": 1,
	  "items": {
		"$ref": "#/definitions/oscal-complete-oscal-assessment-common:assessment-subject"
	  }
	},
	"assessment-assets": {
	  "$ref": "#/definitions/oscal-complete-oscal-assessment-common:assessment-assets"
	},
	"tasks": {
	  "type": "array",
	  "minItems": 1,
	  "items": {
		"$ref": "#/definitions/oscal-complete-oscal-assessment-common:task"
	  }
	},
	"back-matter": {
	  "$ref": "#/definitions/oscal-complete-oscal-metadata:back-matter"
	}
	*/
}

type ImportSsp oscalTypes_1_1_3.ImportSsp

func (i *ImportSsp) UnmarshalOscal(oip oscalTypes_1_1_3.ImportSsp) *ImportSsp {
	*i = ImportSsp(oip)
	return i
}

func (i *ImportSsp) MarshalOscal() *oscalTypes_1_1_3.ImportSsp {
	p := oscalTypes_1_1_3.ImportSsp(*i)
	return &p
}

type Task struct {
	UUIDModel

	Type        string // required: [ milestone | action ]
	Title       string // required
	Description *string
	Props       datatypes.JSONSlice[Prop] `json:"props"`
	Links       datatypes.JSONSlice[Link] `json:"links"`

	Dependencies []TaskDependency // Different struct, as each dependency can have additional remarks
	Tasks        []Task           // Sub tasks

	/**
	"timing": {
	  "title": "Event Timing",
	  "description": "The timing under which the task is intended to occur.",
	  "type": "object",
	  "properties": {
		"on-date": {
		  "title": "On Date Condition",
		  "description": "The task is intended to occur on the specified date.",
		  "type": "object",
		  "properties": {
			"date": {
			  "title": "On Date Condition",
			  "description": "The task must occur on the specified date.",
			  "$ref": "#/definitions/DateTimeWithTimezoneDatatype"
			}
		  },
		  "required": [
			"date"
		  ],
		  "additionalProperties": false
		},
		"within-date-range": {
		  "title": "On Date Range Condition",
		  "description": "The task is intended to occur within the specified date range.",
		  "type": "object",
		  "properties": {
			"start": {
			  "title": "Start Date Condition",
			  "description": "The task must occur on or after the specified date.",
			  "$ref": "#/definitions/DateTimeWithTimezoneDatatype"
			},
			"end": {
			  "title": "End Date Condition",
			  "description": "The task must occur on or before the specified date.",
			  "$ref": "#/definitions/DateTimeWithTimezoneDatatype"
			}
		  },
		  "required": [
			"start",
			"end"
		  ],
		  "additionalProperties": false
		},
		"at-frequency": {
		  "title": "Frequency Condition",
		  "description": "The task is intended to occur at the specified frequency.",
		  "type": "object",
		  "properties": {
			"period": {
			  "title": "Period",
			  "description": "The task must occur after the specified period has elapsed.",
			  "$ref": "#/definitions/PositiveIntegerDatatype"
			},
			"unit": {
			  "title": "Time Unit",
			  "description": "The unit of time for the period.",
			  "allOf": [
				{
				  "$ref": "#/definitions/StringDatatype"
				},
				{
				  "enum": [
					"seconds",
					"minutes",
					"hours",
					"days",
					"months",
					"years"
				  ]
				}
			  ]
			}
		  },
		  "required": [
			"period",
			"unit"
		  ],
		  "additionalProperties": false
		}
	  },
	  "additionalProperties": false
	},

	"associated-activities": {
	  "type": "array",
	  "minItems": 1,
	  "items": {
		"title": "Associated Activity",
		"description": "Identifies an individual activity to be performed as part of a task.",
		"type": "object",
		"properties": {
		  "activity-uuid": {
			"title": "Activity Universally Unique Identifier Reference",
			"description": "A machine-oriented identifier reference to an activity defined in the list of activities.",
			"$ref": "#/definitions/UUIDDatatype"
		  },
		  "props": {
			"type": "array",
			"minItems": 1,
			"items": {
			  "$ref": "#/definitions/oscal-complete-oscal-metadata:property"
			}
		  },
		  "links": {
			"type": "array",
			"minItems": 1,
			"items": {
			  "$ref": "#/definitions/oscal-complete-oscal-metadata:link"
			}
		  },
		  "responsible-roles": {
			"type": "array",
			"minItems": 1,
			"items": {
			  "$ref": "#/definitions/oscal-complete-oscal-metadata:responsible-role"
			}
		  },
		  "subjects": {
			"type": "array",
			"minItems": 1,
			"items": {
			  "$ref": "#/definitions/oscal-complete-oscal-assessment-common:assessment-subject"
			}
		  },
		  "remarks": {
			"$ref": "#/definitions/oscal-complete-oscal-metadata:remarks"
		  }
		},
		"required": [
		  "activity-uuid",
		  "subjects"
		],
		"additionalProperties": false
	  }
	},
	"subjects": {
	  "type": "array",
	  "minItems": 1,
	  "items": {
		"$ref": "#/definitions/oscal-complete-oscal-assessment-common:assessment-subject"
	  }
	},
	"responsible-roles": {
	  "type": "array",
	  "minItems": 1,
	  "items": {
		"$ref": "#/definitions/oscal-complete-oscal-metadata:responsible-role"
	  }
	},
	"remarks": {
	  "$ref": "#/definitions/oscal-complete-oscal-metadata:remarks"
	}
	*/
}

type AssessmentSubject struct {
	// Assessment Subject is a loose reference to some subject.
	// A subject can be a Component, InventoryItem, Location, Party, User, Resource.
	// In our struct we don't store the type, but rather have relations to each of these, and when marhsalling and unmarshalling,
	// setting the type to what we know it is.
	UUIDModel

	Description *string
	Props       datatypes.JSONSlice[Prop]
	Links       datatypes.JSONSlice[Link]

	//"component",
	//"inventory-item",
	//"location",
	//"party",
	//"user"

	IncludeAll      datatypes.JSONType[*IncludeAll]
	IncludeSubjects []SelectSubjectById
	ExcludeSubjects []SelectSubjectById

	/**
	"required": [
		"type"
	  ],
	"properties": {
		"type": {
		  "title": "Subject Type",
		  "description": "Indicates the type of assessment subject, such as a component, inventory, item, location, or party represented by this selection statement.",
		  "anyOf": [
			{
			  "$ref": "#/definitions/TokenDatatype"
			},
			{
			  "enum": [
				"component",
				"inventory-item",
				"location",
				"party",
				"user"
			  ]
			}
		  ]
		},

		"include-all": {
		  "$ref": "#/definitions/oscal-complete-oscal-control-common:include-all"
		},
		"include-subjects": {
		  "type": "array",
		  "minItems": 1,
		  "items": {
			"$ref": "#/definitions/oscal-complete-oscal-assessment-common:select-subject-by-id"
		  }
		},
		"exclude-subjects": {
		  "type": "array",
		  "minItems": 1,
		  "items": {
			"$ref": "#/definitions/oscal-complete-oscal-assessment-common:select-subject-by-id"
		  }
		},
		"remarks": {
		  "$ref": "#/definitions/oscal-complete-oscal-metadata:remarks"
		}
	  },

	*/
}

type SelectSubjectById struct {
	Remarks *string
	Subject AssessmentSubject
	Props   datatypes.JSONSlice[Prop]
	Links   datatypes.JSONSlice[Link]
}

type AssociatedActivity struct {
	UUIDModel
	Remarks *string

	Activity         Activity // required
	Props            datatypes.JSONSlice[Prop]
	Links            datatypes.JSONSlice[Link]
	ResponsibleRoles datatypes.JSONSlice[ResponsibleRole]

	/**
	"properties": {
	  "subjects": {
		"type": "array",
		"minItems": 1,
		"items": {
		  "$ref": "#/definitions/oscal-complete-oscal-assessment-common:assessment-subject"
		}
	  },
	  "remarks": {
		"$ref": "#/definitions/oscal-complete-oscal-metadata:remarks"
	  }
	},
	"required": [
	  "activity-uuid",
	  "subjects"
	],
	*/
}

type TaskDependency struct {
	Task    Task
	Remarks *string
}

type Activity struct {
	UUIDModel
	Title       *string
	Description string  // required
	Remarks     *string // required

	Props datatypes.JSONSlice[Prop] `json:"props"`
	Links datatypes.JSONSlice[Link] `json:"links"`
	Steps []Step

	RelatedControls  ReviewedControls
	ResponsibleRoles datatypes.JSONSlice[ResponsibleRole]
}

type Step struct {
	UUIDModel
	Title       *string
	Description string // required
	Remarks     *string

	Props datatypes.JSONSlice[Prop] `json:"props"`
	Links datatypes.JSONSlice[Link] `json:"links"`

	ResponsibleRoles datatypes.JSONSlice[ResponsibleRole]
	ReviewedControls ReviewedControls
}

type ReviewedControls struct {
	Description                *string
	Remarks                    *string
	Props                      datatypes.JSONSlice[Prop]
	Links                      datatypes.JSONSlice[Link]
	ControlSelections          []ControlSelection // required
	ControlObjectiveSelections []ControlObjectiveSelection
}

type ControlSelection struct {
	UUIDModel
	Description *string
	Remarks     *string
	Props       datatypes.JSONSlice[Prop]
	Links       datatypes.JSONSlice[Link]

	IncludeAll      datatypes.JSONType[*IncludeAll]
	IncludeControls []SelectControlById `gorm:"Polymorphic:Parent;polymorphicValue:included"`
	ExcludeControls []SelectControlById `gorm:"Polymorphic:Parent;polymorphicValue:excluded"`
}

type ControlObjectiveSelection struct {
	UUIDModel
	Description *string
	Remarks     *string
	Props       datatypes.JSONSlice[Prop]
	Links       datatypes.JSONSlice[Link]

	IncludeAll        datatypes.JSONType[*IncludeAll]
	IncludeObjectives []SelectObjectiveById `gorm:"Polymorphic:Parent;polymorphicValue:included"`
	ExcludeObjectives []SelectObjectiveById `gorm:"Polymorphic:Parent;polymorphicValue:excluded"`
}

type SelectObjectiveById struct { // We should figure out what this looks like for real, because this references objectives hidden in `part`s of a control
	ObjectiveID string // required
}
