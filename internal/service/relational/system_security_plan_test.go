package relational

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
)

func TestSatisfiedControlImplementationResponsibilityUnmarshal(t *testing.T) {
	data := oscalTypes_1_1_3.SatisfiedControlImplementationResponsibility{
		UUID:               uuid.New().String(),
		ResponsibilityUuid: uuid.New().String(),
		Description:        "Test description",
		Remarks:            "Test remarks",
		Props: &[]oscalTypes_1_1_3.Property{
			{
				Class:   "prop class",
				Group:   "prop group",
				Name:    "prop name",
				Ns:      "prop ns",
				Remarks: "prop remarks",
				UUID:    uuid.New().String(),
				Value:   "prop value",
			},
		},
		Links: &[]oscalTypes_1_1_3.Link{
			{
				Href:      "http://one",
				MediaType: "related",
				Text:      "One Link",
			},
		},
		ResponsibleRoles: &[]oscalTypes_1_1_3.ResponsibleRole{
			{
				RoleId:     "role-1",
				Remarks:    "Test role remarks",
				PartyUuids: &[]string{uuid.New().String()},
				Links: &[]oscalTypes_1_1_3.Link{
					{
						Href:      "http://role-link",
						MediaType: "application/json",
						Text:      "Role Link",
					},
				},
				Props: &[]oscalTypes_1_1_3.Property{
					{
						Name:  "role-prop-name",
						Value: "role-prop-value",
					},
				},
			},
		},
	}

	inputJson, err := json.Marshal(data)
	assert.NoError(t, err)

	ci := &SatisfiedControlImplementationResponsibility{}
	ci.UnmarshalOscal(data)
	output := ci.MarshalOscal()
	outputJson, err := json.Marshal(output)

	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestControlImplementationResponsibilityUnmarshal(t *testing.T) {
	data := oscalTypes_1_1_3.ControlImplementationResponsibility{
		UUID:         uuid.New().String(),
		ProvidedUuid: uuid.New().String(),
		Description:  "Test description",
		Remarks:      "Test remarks",
		Props: &[]oscalTypes_1_1_3.Property{
			{
				Class:   "prop class",
				Group:   "prop group",
				Name:    "prop name",
				Ns:      "prop ns",
				Remarks: "prop remarks",
				UUID:    uuid.New().String(),
				Value:   "prop value",
			},
		},
		Links: &[]oscalTypes_1_1_3.Link{
			{
				Href:      "http://one",
				MediaType: "related",
				Text:      "One Link",
			},
		},
		ResponsibleRoles: &[]oscalTypes_1_1_3.ResponsibleRole{
			{
				RoleId:     "role-1",
				Remarks:    "Test role remarks",
				PartyUuids: &[]string{uuid.New().String()},
			},
		},
	}

	inputJson, err := json.Marshal(data)
	assert.NoError(t, err)

	ci := &ControlImplementationResponsibility{}
	ci.UnmarshalOscal(data)
	output := ci.MarshalOscal()
	outputJson, err := json.Marshal(output)

	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}
