package runtime

import (
	"context"
	"fmt"
	oscaltypes113 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"time"

	"github.com/compliance-framework/configuration-service/domain"
	"github.com/compliance-framework/configuration-service/event"
	"github.com/compliance-framework/configuration-service/service"
)

type Processor struct {
	planService   *service.PlanService
	resultService *service.ResultsService
	sub           event.Subscriber[ExecutionResult]
}

func NewProcessor(s event.Subscriber[ExecutionResult], planService *service.PlanService, resultService *service.ResultsService) *Processor {
	return &Processor{
		sub:           s,
		planService:   planService,
		resultService: resultService,
	}
}

func (r *Processor) Listen() {
	// TODO This whole method needs better error handling, and a logger to properly log errors when they happen.
	ch, err := r.sub(event.TopicTypeResult)
	if err != nil {
		panic(err)
	}

	go func() {
		for msg := range ch {
			fmt.Printf("Received message: %v\n", msg)

			// TODO: Create an actor for the runtime that publishes the events to store it as the origin
			// TODO: Handle execution status

			subject := msg.Subject

			err := r.planService.SaveSubject(subject)
			if err != nil {
				return
			}

			// TODO: Start and End times should arrive from the runtime inside the message
			theTime := time.Now()
			theId := uuid.New()
			result := domain.Result{
				UUID:     &theId,
				StreamID: msg.StreamId,
				Labels:   msg.Labels,
				Result: oscaltypes113.Result{
					Title:         msg.Title,
					Observations:  &msg.Observations,
					Risks:         &msg.Risks,
					Findings:      &msg.Findings,
					AssessmentLog: &msg.Logs,
					Start:         theTime,
					End:           &theTime,
				},
			}

			fmt.Printf("Plumbed message: %v\n", msg)

			err = r.resultService.Create(context.TODO(), &result)
			if err != nil {
				return
			}
		}
	}()
}
