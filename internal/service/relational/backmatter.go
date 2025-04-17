package relational

import (
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type BackMatter struct {
	UUIDModel
	ParentID   *string
	ParentType *string
	Resources  []BackMatterResource `json:"resources"`
}

func (b *BackMatter) UnmarshalOscal(resource oscaltypes113.BackMatter) *BackMatter {
	*b = BackMatter{
		Resources: ConvertList(resource.Resources, func(res oscaltypes113.Resource) BackMatterResource {
			bm := BackMatterResource{}
			bm.UnmarshalOscal(res)
			return bm
		}),
	}
	return b
}

type BackMatterResource struct {
	UUIDModel                                      // required
	BackMatterID uuid.UUID                         `json:"back-matter-id"`
	Title        *string                           `json:"title"`
	Description  *string                           `json:"description"`
	Remarks      *string                           `json:"remarks"`
	Citation     *datatypes.JSONType[Citation]     `json:"citation"`
	Base64       *datatypes.JSONType[Base64]       `json:"base64"`
	Props        datatypes.JSONSlice[Prop]         `json:"props"`
	DocumentIDs  datatypes.JSONSlice[DocumentID]   `json:"document-ids"`
	RLinks       datatypes.JSONSlice[ResourceLink] `json:"rlinks"`
}

func (c *BackMatterResource) UnmarshalOscal(resource oscaltypes113.Resource) *BackMatterResource {
	id := uuid.MustParse(resource.UUID)

	*c = BackMatterResource{
		UUIDModel: UUIDModel{
			ID: &id,
		},
		Title:       &resource.Title,
		Description: &resource.Description,
		Remarks:     &resource.Remarks,

		Props: ConvertList(resource.Props, func(property oscaltypes113.Property) Prop {
			prop := Prop{}
			prop.UnmarshalOscal(property)
			return prop
		}),
		DocumentIDs: ConvertList(resource.DocumentIds, func(doc oscaltypes113.DocumentId) DocumentID {
			d := DocumentID{}
			d.UnmarshalOscal(doc)
			return d
		}),
		RLinks: ConvertList(resource.Rlinks, func(olink oscaltypes113.ResourceLink) ResourceLink {
			r := ResourceLink{}
			r.UnmarshalOscal(olink)
			return r
		}),
	}

	if resource.Citation != nil {
		citation := Citation{}
		citation.UnmarshalOscal(*resource.Citation)
		jcitation := datatypes.NewJSONType[Citation](citation)
		c.Citation = &jcitation
	}

	if resource.Base64 != nil {
		base64 := Base64{}
		base64.UnmarshalOscal(*resource.Base64)
		jbase64 := datatypes.NewJSONType[Base64](base64)
		c.Base64 = &jbase64
	}

	return c
}

type Citation struct {
	Text  string `json:"text"` // required
	Props []Prop `json:"props"`
	Links []Link `json:"links"`
}

func (c *Citation) UnmarshalOscal(cit oscaltypes113.Citation) *Citation {
	*c = Citation{
		Text: cit.Text,
		Props: ConvertList(cit.Props, func(property oscaltypes113.Property) Prop {
			prop := Prop{}
			prop.UnmarshalOscal(property)
			return prop
		}),
		Links: ConvertList(cit.Links, func(olink oscaltypes113.Link) Link {
			link := Link{}
			link.UnmarshalOscal(olink)
			return link
		}),
	}
	return c
}

type HashAlgorithm string

const (
	HashAlgorithmSHA_224  HashAlgorithm = "SHA-224"
	HashAlgorithmSHA_256  HashAlgorithm = "SHA-256"
	HashAlgorithmSHA_384  HashAlgorithm = "SHA-384"
	HashAlgorithmSHA_512  HashAlgorithm = "SHA-512"
	HashAlgorithmSHA3_224 HashAlgorithm = "SHA3-224"
	HashAlgorithmSHA3_256 HashAlgorithm = "SHA3-256"
	HashAlgorithmSHA3_384 HashAlgorithm = "SHA3-384"
	HashAlgorithmSHA3_512 HashAlgorithm = "SHA3-512"
)

type Hash struct {
	Algorithm HashAlgorithm `json:"algorithm"` // required
	Value     string        `json:"value"`     // required
}

func (h *Hash) UnmarshalOscal(hash oscaltypes113.Hash) *Hash {
	*h = Hash{
		Algorithm: HashAlgorithm(hash.Algorithm),
		Value:     hash.Value,
	}
	return h
}

type ResourceLink struct {
	Href      string `json:"href"` // required
	MediaType string `json:"media-type"`
	Hashes    []Hash `json:"hashes"`
}

func (r *ResourceLink) UnmarshalOscal(orlink oscaltypes113.ResourceLink) {
	*r = ResourceLink{
		Href:      orlink.Href,
		MediaType: orlink.MediaType,
		Hashes: ConvertList(orlink.Hashes, func(ohash oscaltypes113.Hash) Hash {
			hash := Hash{}
			hash.UnmarshalOscal(ohash)
			return hash
		}),
	}
}

type Base64 struct {
	Filename  string `json:"filename"`
	MediaType string `json:"media-type"`
	Value     string `json:"value"` // required
}

func (b *Base64) UnmarshalOscal(base oscaltypes113.Base64) *Base64 {
	*b = Base64{
		Filename:  base.Filename,
		MediaType: base.MediaType,
		Value:     base.Value,
	}
	return b
}
