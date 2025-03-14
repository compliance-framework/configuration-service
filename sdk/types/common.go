package types

type Link struct {
	Href             string `json:"href" yaml:"href"`
	MediaType        string `json:"media-type,omitempty" yaml:"media-type,omitempty"`
	Rel              string `json:"rel,omitempty" yaml:"rel,omitempty"`
	ResourceFragment string `json:"resource-fragment,omitempty" yaml:"resource-fragment,omitempty"`
	Text             string `json:"text,omitempty" yaml:"text,omitempty"`
}

type Property struct {
	Class   string `json:"class,omitempty" yaml:"class,omitempty"`
	Group   string `json:"group,omitempty" yaml:"group,omitempty"`
	Name    string `json:"name" yaml:"name"`
	Ns      string `json:"ns,omitempty" yaml:"ns,omitempty"`
	Remarks string `json:"remarks,omitempty" yaml:"remarks,omitempty"`
	UUID    string `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	Value   string `json:"value" yaml:"value"`
}

type BackMatter struct {
	Resources *[]Resource `json:"resources,omitempty" yaml:"resources,omitempty"`
}

type Base64 struct {
	Filename  string `json:"filename,omitempty" yaml:"filename,omitempty"`
	MediaType string `json:"media-type,omitempty" yaml:"media-type,omitempty"`
	Value     string `json:"value" yaml:"value"`
}

type Citation struct {
	Links *[]Link     `json:"links,omitempty" yaml:"links,omitempty"`
	Props *[]Property `json:"props,omitempty" yaml:"props,omitempty"`
	Text  string      `json:"text" yaml:"text"`
}

type ResourceLink struct {
	Hashes    *[]Hash `json:"hashes,omitempty" yaml:"hashes,omitempty"`
	Href      string  `json:"href" yaml:"href"`
	MediaType string  `json:"media-type,omitempty" yaml:"media-type,omitempty"`
}

type DocumentId struct {
	Identifier string `json:"identifier" yaml:"identifier"`
	Scheme     string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
}

type Hash struct {
	Algorithm string `json:"algorithm" yaml:"algorithm"`
	Value     string `json:"value" yaml:"value"`
}

type Resource struct {
	Base64      *Base64         `json:"base64,omitempty" yaml:"base64,omitempty"`
	Citation    *Citation       `json:"citation,omitempty" yaml:"citation,omitempty"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	DocumentIds *[]DocumentId   `json:"document-ids,omitempty" yaml:"document-ids,omitempty"`
	Props       *[]Property     `json:"props,omitempty" yaml:"props,omitempty"`
	Remarks     string          `json:"remarks,omitempty" yaml:"remarks,omitempty"`
	Rlinks      *[]ResourceLink `json:"rlinks,omitempty" yaml:"rlinks,omitempty"`
	Title       string          `json:"title,omitempty" yaml:"title,omitempty"`
	UUID        string          `json:"uuid" yaml:"uuid"`
}
