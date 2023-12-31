package domain

type Control struct {
	Uuid Uuid `json:"uuid"`

	Props []Property `json:"props,omitempty"`
	Links []Link     `json:"links,omitempty"`
	Parts []Part     `json:"parts,omitempty"`

	Class    string      `json:"class"`
	Title    string      `json:"title"`
	Params   []Parameter `json:"params"`
	Controls []Uuid      `json:"controlUuids"` // Reference to controls
}
