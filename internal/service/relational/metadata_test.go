package relational

import (
	"encoding/json"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRole_OscalMarshalling(t *testing.T) {
	oscalRole := oscaltypes113.Role{
		ID:          "role-id",
		Title:       "role-title",
		ShortName:   "rshort",
		Description: "rdesc",
		Remarks:     "rremarks",
	}
	// add a prop and link
	oscalRole.Props = &[]oscaltypes113.Property{
		{
			Class:   "pc",
			Group:   "pg",
			Name:    "pn",
			Ns:      "pns",
			Remarks: "pr",
			UUID:    uuid.New().String(),
			Value:   "pv",
		},
	}
	oscalRole.Links = &[]oscaltypes113.Link{
		{
			Href:      "http://link",
			MediaType: "mt",
			Text:      "txt",
		},
	}
	inputJson, err := json.Marshal(oscalRole)
	assert.NoError(t, err)

	r := &Role{}
	r.UnmarshalOscal(oscalRole)
	output := r.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestLocation_OscalMarshalling(t *testing.T) {
	oscalLoc := oscaltypes113.Location{
		UUID:           uuid.New().String(),
		EmailAddresses: &[]string{"email@example.com"},
		TelephoneNumbers: &[]oscaltypes113.TelephoneNumber{
			{Type: "mobile", Number: "12345"},
		},
		Urls: &[]string{"http://example.com"},
		Props: &[]oscaltypes113.Property{
			{
				Class:   "pc",
				Group:   "pg",
				Name:    "pn",
				Ns:      "pns",
				Remarks: "pr",
				UUID:    uuid.New().String(),
				Value:   "pv",
			},
		},
		Links: &[]oscaltypes113.Link{
			{
				Href:      "http://link",
				MediaType: "mt",
				Text:      "txt",
			},
		},
		Remarks: "location remarks",
	}
	inputJson, err := json.Marshal(oscalLoc)
	assert.NoError(t, err)

	loc := &Location{}
	loc.UnmarshalOscal(oscalLoc)
	output := loc.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestRevision_OscalMarshalling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	oscalRev := oscaltypes113.RevisionHistoryEntry{
		Title:        "rev-title",
		Published:    &now,
		LastModified: &now,
		Version:      "v1",
		OscalVersion: "1.1.3",
		Props: &[]oscaltypes113.Property{
			{
				Class:   "pc",
				Group:   "pg",
				Name:    "pn",
				Ns:      "pns",
				Remarks: "pr",
				UUID:    uuid.New().String(),
				Value:   "pv",
			},
		},
		Links: &[]oscaltypes113.Link{
			{
				Href:      "http://link",
				MediaType: "mt",
				Text:      "txt",
			},
		},
		Remarks: "rev-remarks",
	}
	inputJson, err := json.Marshal(oscalRev)
	assert.NoError(t, err)

	r := &Revision{}
	r.UnmarshalOscal(oscalRev)
	output := r.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestPartyExternalID_OscalMarshalling(t *testing.T) {
	oscalPEI := oscaltypes113.PartyExternalIdentifier{
		ID:     "id123",
		Scheme: "http://orcid.org/",
	}
	inputJson, err := json.Marshal(oscalPEI)
	assert.NoError(t, err)

	pei := &PartyExternalID{}
	pei.UnmarshalOscal(oscalPEI)
	output := pei.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}

func TestParty_OscalMarshalling(t *testing.T) {
	oscalParty := oscaltypes113.Party{
		UUID:             uuid.New().String(),
		Type:             "person",
		Name:             "TestName",
		ShortName:        "TestShort",
		ExternalIds:      &[]oscaltypes113.PartyExternalIdentifier{{ID: "eid", Scheme: "http://orcid.org/"}},
		Props:            &[]oscaltypes113.Property{{Class: "pc", Group: "pg", Name: "pn", Ns: "pns", Remarks: "pr", UUID: uuid.New().String(), Value: "pv"}},
		Links:            &[]oscaltypes113.Link{{Href: "http://link", MediaType: "mt", Text: "txt"}},
		EmailAddresses:   &[]string{"e1", "e2"},
		TelephoneNumbers: &[]oscaltypes113.TelephoneNumber{{Type: "mobile", Number: "12345"}},
		Addresses:        &[]oscaltypes113.Address{{Type: "home", AddrLines: &[]string{"l1"}, City: "city", State: "state", PostalCode: "pc", Country: "country"}},
		Remarks:          "remarks",
	}
	inputJson, err := json.Marshal(oscalParty)
	assert.NoError(t, err)

	p := &Party{}
	p.UnmarshalOscal(oscalParty)
	output := p.MarshalOscal()
	outputJson, err := json.Marshal(output)
	assert.NoError(t, err)
	assert.JSONEq(t, string(inputJson), string(outputJson))
}
