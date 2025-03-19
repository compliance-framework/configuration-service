package sdk

import (
	"context"
	"fmt"
	"github.com/compliance-framework/configuration-service/sdk/types"
	"net/http"
)

type observationsAndFindingsClient struct {
	httpClient *http.Client
	config     *Config
}

func (r *observationsAndFindingsClient) Create(ctx context.Context, observations []types.Observation, findings []types.Finding) error {
	// Not yet implemented
	fmt.Println("Creating observations and findings")
	fmt.Println(observations)
	fmt.Println(findings)

	for _, observation := range observations {
		fmt.Println("########################")
		fmt.Println("Observation:")
		fmt.Println("Title:", observation.Title)
		fmt.Println("Description:", observation.Description)
		fmt.Println("Remarks:", observation.Remarks)
		fmt.Println("Collected:", observation.Collected.String())
		fmt.Println("Expires:", observation.Expires.String())

		for _, i := range *observation.RelevantEvidence {
			fmt.Println("###### Evidence: ")
			fmt.Println("Description:", i.Description)
			fmt.Println("Remarks:", i.Remarks)
		}

		for _, i := range *observation.Components {
			fmt.Println("###### Component: ")
			fmt.Println("Identifier:", i.Identifier)
			fmt.Println("Href:", i.Href)
		}

		for _, i := range *observation.Subjects {
			fmt.Println("###### Subject: ")
			fmt.Println("Title:", i.Title)
			fmt.Println("Remarks:", i.Remarks)
			fmt.Println("Type:", i.Type)
			for k, v := range i.Attributes {
				fmt.Println("Attribute:", k, ":", v)
			}
		}

		for _, i := range *observation.Origins {
			fmt.Println("###### Origin: ")
			for _, k := range i.Actors {
				fmt.Println("UUID:", k.UUID)
				fmt.Println("Type:", k.Type)
				fmt.Println("Title:", k.Title)
			}
		}

		for _, i := range *observation.Activities {
			fmt.Println("###### Activity: ")
			fmt.Println("UUID:", i.UUID)
			fmt.Println("Title:", i.Title)
			fmt.Println("Description:", i.Description)
			fmt.Println("Remarks:", i.Remarks)
			for _, k := range *i.Steps {
				fmt.Println("########### Step: ")
				fmt.Println("UUID:", k.UUID)
				fmt.Println("Title:", k.Title)
				fmt.Println("Description:", k.Description)
				fmt.Println("Remarks:", k.Remarks)
			}
		}

		fmt.Println("########################")
	}
	//reqBody, _ := json.Marshal(observations)
	//req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/assessment-results", r.config.BaseURL), bytes.NewReader(reqBody))
	//if err != nil {
	//	return nil, err
	//}
	//req.Header.Set("Content-Type", "application/json")
	//response, err := r.httpClient.Do(req)
	//if err != nil {
	//	return nil, err
	//}
	//defer response.Body.Close()
	//
	//if response.StatusCode != http.StatusCreated {
	//	return nil, fmt.Errorf("unexpected api response status code: %d", response.StatusCode)
	//}
	//
	//bodyBytes, err := io.ReadAll(response.Body)
	//if err != nil {
	//	panic(err)
	//}
	//err = json.Unmarshal(bodyBytes, result)
	//if err != nil {
	//	return nil, err
	//}
	//return result, nil
	return nil
}
