package relational

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestComponentDefinition_MarshalUnmarshalOscal tests the marshaling and unmarshaling of ComponentDefinition
// to and from OSCAL format. It ensures that the conversion process preserves all fields and structure.
func TestComponentDefinition_MarshalUnmarshalOscal(t *testing.T) {
	tests := []struct {
		name string
		data oscalTypes_1_1_3.ComponentDefinition
	}{
		{
			name: "minimal fields",
			data: oscalTypes_1_1_3.ComponentDefinition{
				UUID: uuid.New().String(),
				Metadata: oscalTypes_1_1_3.Metadata{
					Title:        "Minimal Metadata",
					LastModified: time.Now(),
					Version:      "1.0.0",
					OscalVersion: "1.1.3",
				},
			},
		},
		{
			name: "all fields set",
			data: oscalTypes_1_1_3.ComponentDefinition{
				UUID: uuid.New().String(),
				Metadata: oscalTypes_1_1_3.Metadata{
					Title:        "Full Metadata",
					LastModified: time.Now(),
					Version:      "2.0.0",
					OscalVersion: "1.1.3",
				},
				ImportComponentDefinitions: &[]oscalTypes_1_1_3.ImportComponentDefinition{
					{Href: "#import-1"},
				},
				Components: &[]oscalTypes_1_1_3.DefinedComponent{
					{
						UUID:        uuid.New().String(),
						Type:        "service",
						Title:       "Component 1",
						Description: "A component",
					},
				},
				Capabilities: &[]oscalTypes_1_1_3.Capability{
					{
						UUID:        uuid.New().String(),
						Name:        "capability-1",
						Description: "A capability",
					},
				},
				BackMatter: &oscalTypes_1_1_3.BackMatter{
					Resources: &[]oscalTypes_1_1_3.Resource{
						{
							UUID:        uuid.New().String(),
							Title:       "Resource 1",
							Description: "A resource",
						},
					},
				},
			},
		},
		{
			name: "only back-matter",
			data: oscalTypes_1_1_3.ComponentDefinition{
				UUID: uuid.New().String(),
				Metadata: oscalTypes_1_1_3.Metadata{
					Title:        "BackMatter Only",
					LastModified: time.Now(),
					Version:      "1.0.1",
					OscalVersion: "1.1.3",
				},
				BackMatter: &oscalTypes_1_1_3.BackMatter{
					Resources: &[]oscalTypes_1_1_3.Resource{
						{
							UUID:        uuid.New().String(),
							Title:       "Resource Only",
							Description: "Resource only description",
						},
					},
				},
			},
		},
		{
			name: "empty optional fields omitted",
			data: oscalTypes_1_1_3.ComponentDefinition{
				UUID: uuid.New().String(),
				Metadata: oscalTypes_1_1_3.Metadata{
					Title:        "Empty Optional",
					LastModified: time.Now(),
					Version:      "1.0.2",
					OscalVersion: "1.1.3",
				},
			},
		},
		{
			name: "nil back-matter",
			data: oscalTypes_1_1_3.ComponentDefinition{
				UUID: uuid.New().String(),
				Metadata: oscalTypes_1_1_3.Metadata{
					Title:        "Nil BackMatter",
					LastModified: time.Now(),
					Version:      "1.0.3",
					OscalVersion: "1.1.3",
				},
				BackMatter: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJson, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			cd := &ComponentDefinition{}
			assert.NotPanics(t, func() {
				cd.UnmarshalOscal(tt.data)
			})
			output := cd.MarshalOscal()
			outputJson, err := json.Marshal(output)
			assert.NoError(t, err)

			assert.JSONEq(t, string(inputJson), string(outputJson))
		})
	}
}

// TestDefinedComponent_MarshalUnmarshalOscal tests the marshaling and unmarshaling of DefinedComponent
// to and from OSCAL format. It ensures that the conversion process preserves all fields and structure.
func TestDefinedComponent_MarshalUnmarshalOscal(t *testing.T) {
	tests := []struct {
		name string
		data oscalTypes_1_1_3.DefinedComponent
	}{
		{
			name: "minimal fields",
			data: oscalTypes_1_1_3.DefinedComponent{
				UUID:        uuid.New().String(),
				Type:        "service",
				Title:       "Minimal Component",
				Description: "A minimal component definition",
			},
		},
		{
			name: "all fields set",
			data: oscalTypes_1_1_3.DefinedComponent{
				UUID:        uuid.New().String(),
				Type:        "service",
				Title:       "Full Component",
				Description: "A full component definition",
				Purpose:     "Test purpose",
				Remarks:     "Some remarks",
				Protocols: &[]oscalTypes_1_1_3.Protocol{
					{
						UUID: uuid.New().String(),
						Name: "https",
					},
				},
				Links: &[]oscalTypes_1_1_3.Link{
					{Href: "http://example.com", Text: "Example"},
				},
				Props: &[]oscalTypes_1_1_3.Property{
					{Name: "env", Value: "prod"},
				},
				ResponsibleRoles: &[]oscalTypes_1_1_3.ResponsibleRole{
					{RoleId: "admin"},
				},
				ControlImplementations: &[]oscalTypes_1_1_3.ControlImplementationSet{
					{
						UUID:        uuid.New().String(),
						Source:      "source-1",
						Description: "desc",
						ImplementedRequirements: []oscalTypes_1_1_3.ImplementedRequirementControlImplementation{
							{
								UUID:        uuid.New().String(),
								ControlId:   "ac-1",
								Description: "req desc",
							},
						},
					},
				},
			},
		},
		{
			name: "only optional fields",
			data: oscalTypes_1_1_3.DefinedComponent{
				UUID:        uuid.New().String(),
				Type:        "service",
				Title:       "Optional Fields Component",
				Description: "Component with only optional fields set",
				Purpose:     "Optional purpose",
				Remarks:     "Optional remarks",
			},
		},
		{
			name: "empty optional fields",
			data: oscalTypes_1_1_3.DefinedComponent{
				UUID:        uuid.New().String(),
				Type:        "service",
				Title:       "Empty Optional Fields",
				Description: "Component with empty optional fields",
				Purpose:     "",
				Remarks:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJson, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			dc := &DefinedComponent{}
			dc.UnmarshalOscal(tt.data)
			output := dc.MarshalOscal()
			outputJson, err := json.Marshal(output)
			assert.NoError(t, err)

			assert.JSONEq(t, string(inputJson), string(outputJson))
		})
	}
}

// TestProtocol_MarshalUnmarshalOscal tests the marshaling and unmarshaling of Protocol
// to and from OSCAL format, verifying that all fields are correctly handled.
func TestProtocol_MarshalUnmarshalOscal(t *testing.T) {
	tests := []struct {
		name string
		data oscalTypes_1_1_3.Protocol
	}{
		{
			name: "minimal fields",
			data: oscalTypes_1_1_3.Protocol{
				UUID: uuid.New().String(),
			},
		},
		{
			name: "all fields set",
			data: oscalTypes_1_1_3.Protocol{
				UUID:  uuid.New().String(),
				Name:  "https",
				Title: "Hypertext Transfer Protocol Secure",
				PortRanges: &[]oscalTypes_1_1_3.PortRange{
					{
						Start: 443,
						End:   443,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJson, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			p := &Protocol{}
			p.UnmarshalOscal(tt.data)
			output := p.MarshalOscal()
			outputJson, err := json.Marshal(output)
			assert.NoError(t, err)

			assert.JSONEq(t, string(inputJson), string(outputJson))
		})
	}
}

// TestSetParameter_MarshalUnmarshalOscal tests the marshaling and unmarshaling of SetParameter
// to and from OSCAL format, ensuring all fields and edge cases are covered.
func TestSetParameter_MarshalUnmarshalOscal(t *testing.T) {
	tests := []struct {
		name string
		data oscalTypes_1_1_3.SetParameter
	}{
		{
			name: "minimal fields",
			data: oscalTypes_1_1_3.SetParameter{
				ParamId: "test-param-minimal",
				Values:  []string{"val1"},
			},
		},
		{
			name: "all fields set",
			data: oscalTypes_1_1_3.SetParameter{
				ParamId: "test-param-all",
				Values:  []string{"val1", "val2"},
				Remarks: "All fields remarks",
			},
		},
		{
			name: "only remarks",
			data: oscalTypes_1_1_3.SetParameter{
				ParamId: "test-param-remarks",
				Values:  []string{"val3"},
				Remarks: "Only remarks field set",
			},
		},
		{
			name: "only values",
			data: oscalTypes_1_1_3.SetParameter{
				ParamId: "test-param-values",
				Values:  []string{"val4", "val5"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJson, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			sp := &SetParameter{}
			sp.UnmarshalOscal(tt.data)
			output := sp.MarshalOscal()
			outputJson, err := json.Marshal(output)
			assert.NoError(t, err)

			assert.JSONEq(t, string(inputJson), string(outputJson))
		})
	}
}

// TestControlImplementationSet_MarshalUnmarshalOscal tests the marshaling and unmarshaling of ControlImplementationSet
// to and from OSCAL format, verifying correct handling of implemented requirements and all fields.
func TestControlImplementationSet_MarshalUnmarshalOscal(t *testing.T) {
	tests := []struct {
		name string
		data oscalTypes_1_1_3.ControlImplementationSet
	}{
		{
			name: "minimal fields",
			data: oscalTypes_1_1_3.ControlImplementationSet{
				UUID:        uuid.New().String(),
				Source:      "source-1",
				Description: "minimal description",
				ImplementedRequirements: []oscalTypes_1_1_3.ImplementedRequirementControlImplementation{
					{
						UUID:        uuid.New().String(),
						ControlId:   "control-1",
						Description: "req description",
					},
				},
			},
		},
		{
			name: "all fields set",
			data: oscalTypes_1_1_3.ControlImplementationSet{
				UUID:        uuid.New().String(),
				Source:      "source-2",
				Description: "full description",
				Props: &[]oscalTypes_1_1_3.Property{
					{Name: "prop-name", Value: "prop-value"},
				},
				Links: &[]oscalTypes_1_1_3.Link{
					{Href: "http://test-link", MediaType: "application/json", Text: "Test Link"},
				},
				SetParameters: &[]oscalTypes_1_1_3.SetParameter{
					{ParamId: "param-1", Values: []string{"value1"}},
				},
				ImplementedRequirements: []oscalTypes_1_1_3.ImplementedRequirementControlImplementation{
					{
						UUID:        uuid.New().String(),
						ControlId:   "control-2",
						Description: "req description",
						Props: &[]oscalTypes_1_1_3.Property{
							{Name: "req-prop", Value: "req-value"},
						},
					},
				},
			},
		},
		{
			name: "multiple implemented requirements",
			data: oscalTypes_1_1_3.ControlImplementationSet{
				UUID:        uuid.New().String(),
				Source:      "source-3",
				Description: "multiple requirements",
				ImplementedRequirements: []oscalTypes_1_1_3.ImplementedRequirementControlImplementation{
					{
						UUID:        uuid.New().String(),
						ControlId:   "control-3",
						Description: "first requirement",
					},
					{
						UUID:        uuid.New().String(),
						ControlId:   "control-4",
						Description: "second requirement",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJson, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			cis := &ControlImplementationSet{}
			cis.UnmarshalOscal(tt.data)
			output := cis.MarshalOscal()
			outputJson, err := json.Marshal(output)
			assert.NoError(t, err)

			assert.JSONEq(t, string(inputJson), string(outputJson))
		})
	}
}

// TestImplementedRequirementControlImplementation_MarshalUnmarshalOscal tests the marshaling and unmarshaling of ImplementedRequirementControlImplementation
// to and from OSCAL format, ensuring all nested and optional fields are preserved.
func TestImplementedRequirementControlImplementation_MarshalUnmarshalOscal(t *testing.T) {
	tests := []struct {
		name string
		data oscalTypes_1_1_3.ImplementedRequirementControlImplementation
	}{
		{
			name: "minimal fields",
			data: oscalTypes_1_1_3.ImplementedRequirementControlImplementation{
				UUID:        uuid.New().String(),
				ControlId:   "control-1",
				Description: "minimal description",
			},
		},
		{
			name: "all fields set",
			data: oscalTypes_1_1_3.ImplementedRequirementControlImplementation{
				UUID:        uuid.New().String(),
				ControlId:   "control-2",
				Description: "full description",
				Remarks:     "test remarks",
				Props: &[]oscalTypes_1_1_3.Property{
					{Name: "prop-name", Value: "prop-value"},
				},
				Links: &[]oscalTypes_1_1_3.Link{
					{Href: "http://test-link", MediaType: "application/json", Text: "Test Link"},
				},
				SetParameters: &[]oscalTypes_1_1_3.SetParameter{
					{ParamId: "param-1", Values: []string{"value1"}},
				},
				ResponsibleRoles: &[]oscalTypes_1_1_3.ResponsibleRole{
					{RoleId: "role-1", Remarks: "role remarks"},
				},
				Statements: &[]oscalTypes_1_1_3.ControlStatementImplementation{
					{
						UUID:        uuid.New().String(),
						StatementId: "statement-1",
						Description: "statement description",
					},
				},
			},
		},
		{
			name: "with only statements",
			data: oscalTypes_1_1_3.ImplementedRequirementControlImplementation{
				UUID:        uuid.New().String(),
				ControlId:   "control-3",
				Description: "description with statements",
				Statements: &[]oscalTypes_1_1_3.ControlStatementImplementation{
					{
						UUID:        uuid.New().String(),
						StatementId: "statement-2",
						Description: "nested statement",
						Props: &[]oscalTypes_1_1_3.Property{
							{Name: "nested-prop", Value: "nested-value"},
						},
					},
				},
			},
		},
		{
			name: "with only set parameters",
			data: oscalTypes_1_1_3.ImplementedRequirementControlImplementation{
				UUID:        uuid.New().String(),
				ControlId:   "control-4",
				Description: "description with params",
				SetParameters: &[]oscalTypes_1_1_3.SetParameter{
					{ParamId: "param-2", Values: []string{"value2", "value3"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJson, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			irci := &ImplementedRequirementControlImplementation{}
			irci.UnmarshalOscal(tt.data)
			output := irci.MarshalOscal()
			outputJson, err := json.Marshal(output)
			assert.NoError(t, err)

			assert.JSONEq(t, string(inputJson), string(outputJson))
		})
	}
}

// TestControlStatementImplementation_MarshalUnmarshalOscal tests the marshaling and unmarshaling of ControlStatementImplementation
// to and from OSCAL format, verifying correct handling of responsible roles, properties, and links.
func TestControlStatementImplementation_MarshalUnmarshalOscal(t *testing.T) {
	tests := []struct {
		name string
		data oscalTypes_1_1_3.ControlStatementImplementation
	}{
		{
			name: "minimal fields",
			data: oscalTypes_1_1_3.ControlStatementImplementation{
				UUID:        uuid.New().String(),
				StatementId: "statement-1",
				Description: "desc",
			},
		},
		{
			name: "all fields set",
			data: oscalTypes_1_1_3.ControlStatementImplementation{
				UUID:        uuid.New().String(),
				StatementId: "statement-2",
				Description: "desc2",
				Remarks:     "remarks",
				Props: &[]oscalTypes_1_1_3.Property{
					{Name: "p", Value: "v"},
				},
				Links: &[]oscalTypes_1_1_3.Link{
					{Href: "http://link", MediaType: "application/json", Text: "Link"},
				},
				ResponsibleRoles: &[]oscalTypes_1_1_3.ResponsibleRole{
					{
						RoleId:     "role-1",
						Remarks:    "role remarks",
						PartyUuids: &[]string{uuid.New().String()},
						Links: &[]oscalTypes_1_1_3.Link{
							{Href: "http://role-link", MediaType: "application/json", Text: "Role Link"},
						},
						Props: &[]oscalTypes_1_1_3.Property{
							{Name: "role-prop-name", Value: "role-prop-value"},
						},
					},
				},
			},
		},
		{
			name: "with only responsible roles",
			data: oscalTypes_1_1_3.ControlStatementImplementation{
				UUID:        uuid.New().String(),
				StatementId: "statement-3",
				Description: "desc3",
				ResponsibleRoles: &[]oscalTypes_1_1_3.ResponsibleRole{
					{RoleId: "role-2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJson, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			csi := &ControlStatementImplementation{}
			csi.UnmarshalOscal(tt.data)
			output := csi.MarshalOscal()
			outputJson, err := json.Marshal(output)
			assert.NoError(t, err)

			assert.JSONEq(t, string(inputJson), string(outputJson))
		})
	}
}

// TestResponsibleRole_MarshalUnmarshalOscal tests the marshaling and unmarshaling of ResponsibleRole
// to and from OSCAL format, ensuring all fields and nested structures are handled.
func TestResponsibleRole_MarshalUnmarshalOscal(t *testing.T) {
	osc := oscalTypes_1_1_3.ResponsibleRole{
		RoleId:     "test-role",
		Remarks:    "test remarks",
		PartyUuids: &[]string{"a6ecb154-014c-45a5-8617-96d2d1890941", "a6ecb154-014c-45a5-8617-96d2d1890941"},
		Links: &[]oscalTypes_1_1_3.Link{
			{Href: "http://example.com", Text: "example link"},
		},
		Props: &[]oscalTypes_1_1_3.Property{
			{Name: "prop1", Value: "val1"},
		},
	}
	inputJson, err := json.Marshal(osc)
	assert.NoError(t, err)

	rr := &ResponsibleRole{}
	rr.UnmarshalOscal(osc)
	output := rr.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)

	assert.JSONEq(t, string(inputJson), string(outputJson))
}

// TestImportComponentDefinition_MarshalUnmarshalOscal tests the marshaling and unmarshaling of ImportComponentDefinition
// to and from OSCAL format, verifying correct conversion of the Href field.
func TestImportComponentDefinition_MarshalUnmarshalOscal(t *testing.T) {
	osc := oscalTypes_1_1_3.ImportComponentDefinition{
		Href: "#000000-1111-2222-333333333333",
	}
	inputJson, err := json.Marshal(osc)
	assert.NoError(t, err)

	icd := &ImportComponentDefinition{}
	icd.UnmarshalOscal(osc)
	output := icd.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)

	assert.JSONEq(t, string(inputJson), string(outputJson))
}

// TestIncorporatesComponents_MarshalUnmarshalOscal tests the marshaling and unmarshaling of IncorporatesComponents
// to and from OSCAL format, ensuring all fields are correctly converted.
func TestIncorporatesComponents_MarshalUnmarshalOscal(t *testing.T) {
	osc := oscalTypes_1_1_3.IncorporatesComponent{
		ComponentUuid: uuid.New().String(),
		Description:   "desc",
	}
	inputJson, err := json.Marshal(osc)
	assert.NoError(t, err)

	ic := &IncorporatesComponents{}
	ic.UnmarshalOscal(osc)
	output := ic.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)

	assert.JSONEq(t, string(inputJson), string(outputJson))
}

// Full test using the full component definition with the file testdata/sp800-53-component.json
func TestFileComponentDefinition_MarshalUnmarshalOscal(t *testing.T) {
	t.Run("Full Component Definition", func(t *testing.T) {
		// This test ensures that a FULL component definition can be unmarshalled, and re-marshalled, producing the same JSON object.
		// This proves our entire schema for a Component Definition works correctly.
		f, err := os.Open("../../../testdata/sp800-53-component-aws.json")
		assert.NoError(t, err)
		defer f.Close()

		// Decode the JSON into a ComponentDefinition
		embed := struct {
			CD oscalTypes_1_1_3.ComponentDefinition `json:"component-definition"`
		}{}
		err = json.NewDecoder(f).Decode(&embed)
		assert.NoError(t, err)

		inputJson, err := json.Marshal(embed.CD)
		assert.NoError(t, err)

		componentDef := &ComponentDefinition{}
		componentDef.UnmarshalOscal(embed.CD)
		output := componentDef.MarshalOscal()
		outputJson, err := json.Marshal(output)
		assert.NoError(t, err)
		assert.JSONEq(t, string(inputJson), string(outputJson))
	})
}
