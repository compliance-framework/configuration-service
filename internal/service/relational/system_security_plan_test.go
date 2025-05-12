package relational

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

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

func TestInheritedControlImplementationUnmarshal(t *testing.T) {
	data := oscalTypes_1_1_3.InheritedControlImplementation{
		UUID:         uuid.New().String(),
		ProvidedUuid: uuid.New().String(),
		Description:  "Test description",
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

	ici := &InheritedControlImplementation{}
	ici.UnmarshalOscal(data)
	output := ici.MarshalOscal()
	outputJson, err := json.Marshal(output)

	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestSystemSecurityPlan_OscalMarshalling(t *testing.T) {
	t.Run("Full FedRamp SSP", func(t *testing.T) {
		// SP800-53 ensures that a FULL catalog can be unmarshalled, and re-marshalled, producing the same JSON object.
		// This proves our entire schema for a Catalog works correctly.
		f, err := os.Open("./testdata/full_ssp.json")
		assert.NoError(t, err)
		defer f.Close()

		embed := struct {
			SSP oscalTypes_1_1_3.SystemSecurityPlan `json:"system-security-plan"`
		}{}
		err = json.NewDecoder(f).Decode(&embed)
		assert.NoError(t, err)

		inputJson, err := json.Marshal(embed.SSP)
		assert.NoError(t, err)

		ssp := &SystemSecurityPlan{}
		// Use a random UUID for the catalogId parameter
		ssp.UnmarshalOscal(embed.SSP)
		output := ssp.MarshalOscal()
		outputJson, err := json.Marshal(output)
		assert.NoError(t, err)
		assert.JSONEq(t, string(inputJson), string(outputJson))
	})
}
