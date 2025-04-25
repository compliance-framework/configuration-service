package relational

import (
	"encoding/json"
	"fmt"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPart_OscalMarshalling(t *testing.T) {
	oscalPart := oscaltypes113.Part{
		ID:    "a id",
		Ns:    "a ns",
		Name:  "a name",
		Title: "a title",
		Class: "a class",
		Prose: "some prose",
		Links: &[]oscaltypes113.Link{
			{
				Href:      "http://one",
				MediaType: "related",
				Text:      "One Link",
			},
			{
				Href:      "http://two",
				MediaType: "related",
				Text:      "Two Link",
			},
		},
		Parts: &[]oscaltypes113.Part{
			{
				ID:    "sub id",
				Ns:    "sub ns",
				Name:  "sub name",
				Title: "sub title",
				Class: "sub class",
				Prose: "some sub prose",
				Links: &[]oscaltypes113.Link{
					{
						Href:      "http://one",
						MediaType: "related",
						Text:      "One Link",
					},
					{
						Href:      "http://two",
						MediaType: "related",
						Text:      "Two Link",
					},
				},
				Props: &[]oscaltypes113.Property{
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
			},
		},
		Props: &[]oscaltypes113.Property{
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
	}
	inputJson, err := json.Marshal(oscalPart)

	part := &Part{}
	part.UnmarshalOscal(oscalPart)
	output := part.MarshalOscal()

	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)

	assert.JSONEq(t, string(inputJson), string(outputJson))
	fmt.Println(string(inputJson))
	fmt.Println(string(outputJson))
}
