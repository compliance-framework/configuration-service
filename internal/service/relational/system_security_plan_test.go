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

func TestStatement_OscalMarshalling(t *testing.T) {
	oscalStmt := oscalTypes_1_1_3.Statement{
		UUID:        uuid.New().String(),
		StatementId: "stmt-1",
		Props: &[]oscalTypes_1_1_3.Property{
			{Name: "prop1", Value: "val1"},
		},
		Links: &[]oscalTypes_1_1_3.Link{
			{Href: "http://link", MediaType: "mt", Text: "text"},
		},
		ResponsibleRoles: &[]oscalTypes_1_1_3.ResponsibleRole{
			{RoleId: "role-1", Remarks: "r1"},
		},
		Remarks: "remarks1",
	}
	inputJson, err := json.Marshal(oscalStmt)
	assert.NoError(t, err)

	stmt := &Statement{}
	stmt.UnmarshalOscal(oscalStmt)
	output := stmt.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestExport_OscalMarshalling(t *testing.T) {
	oscalExp := oscalTypes_1_1_3.Export{
		Description: "exp-desc",
		Props: &[]oscalTypes_1_1_3.Property{
			{Name: "p", Value: "v"},
		},
		Links: &[]oscalTypes_1_1_3.Link{
			{Href: "h", MediaType: "m", Text: "t"},
		},
		Remarks: "exp-remarks",
		Provided: &[]oscalTypes_1_1_3.ProvidedControlImplementation{
			{
				UUID:        uuid.New().String(),
				Description: "prov-desc",
			},
		},
		Responsibilities: &[]oscalTypes_1_1_3.ControlImplementationResponsibility{
			{
				UUID:        uuid.New().String(),
				Description: "res-desc",
			},
		},
	}
	inputJson, err := json.Marshal(oscalExp)
	assert.NoError(t, err)

	exp := &Export{}
	exp.UnmarshalOscal(oscalExp)
	output := exp.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestByComponent_OscalMarshalling(t *testing.T) {
	oscalBC := oscalTypes_1_1_3.ByComponent{
		UUID:          uuid.New().String(),
		ComponentUuid: uuid.New().String(),
		Description:   "comp-desc",
		Remarks:       "bc-remarks",
	}
	inputJson, err := json.Marshal(oscalBC)
	assert.NoError(t, err)

	bc := &ByComponent{}
	bc.UnmarshalOscal(oscalBC)
	output := bc.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestImplementedRequirement_OscalMarshalling(t *testing.T) {
	oscalReq := oscalTypes_1_1_3.ImplementedRequirement{
		UUID:      uuid.New().String(),
		ControlId: "ctrl-1",
		Remarks:   "req-remarks",
	}
	inputJson, err := json.Marshal(oscalReq)
	assert.NoError(t, err)

	ir := &ImplementedRequirement{}
	ir.UnmarshalOscal(oscalReq)
	output := ir.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestControlImplementation_OscalMarshalling(t *testing.T) {
	oscalCI := oscalTypes_1_1_3.ControlImplementation{
		Description: "ci-desc",
		SetParameters: &[]oscalTypes_1_1_3.SetParameter{
			{ParamId: "param-name", Values: []string{"param-value"}},
		},
		ImplementedRequirements: []oscalTypes_1_1_3.ImplementedRequirement{
			{
				UUID:      uuid.New().String(),
				ControlId: "ctrl-1",
				Remarks:   "req-remarks",
			},
		},
	}
	inputJson, err := json.Marshal(oscalCI)
	assert.NoError(t, err)

	ci := &ControlImplementation{}
	ci.UnmarshalOscal(oscalCI)
	output := ci.MarshalOscal()
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
