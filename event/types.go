package event

type TopicType string

const (
	TopicTypePlan   TopicType = "runtime.configuration"
	TopicTypeResult TopicType = "job.result"
)

type Subscriber[T any] func(topic TopicType) (chan T, error)
type Publisher func(msg interface{}, topic TopicType) error

type PlanEvent struct {
	// Type holds the type of the event: created / updated / deleted
	Type             string `yaml:"type" json:"type"`
	JobSpecification `yaml:"data" json:"data"`
}

type ResultEvent struct{}
