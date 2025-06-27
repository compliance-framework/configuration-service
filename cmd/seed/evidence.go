package seed

import (
	"fmt"
	"github.com/compliance-framework/configuration-service/internal"
	"github.com/compliance-framework/configuration-service/internal/service/relational"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
	"log"
	"sync"
	"time"

	"github.com/compliance-framework/configuration-service/internal/config"
	"github.com/compliance-framework/configuration-service/internal/service"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newEvidenceCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evidence",
		Short: "Generates evidence",
		Run:   generateEvidence,
	}

	cmd.Flags().CountP("amount", "a", "Amount of evidences")
	cmd.Flags().CountP("beats", "b", "Amount of beats per evidence")

	return cmd
}

func generateEvidence(cmd *cobra.Command, args []string) {
	var err error

	amount := 10
	if cmd.Flags().Changed("amount") {
		amount, err = cmd.Flags().GetCount("amount")
		if err != nil {
			log.Fatal(err)
		}
	}

	beats := 5 // each minute of a day
	if cmd.Flags().Changed("beats") {
		beats, err = cmd.Flags().GetCount("beats")
		if err != nil {
			log.Fatal(err)
		}
	}

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Can't initialize zap logger: %v", err)
	}
	sugar := zapLogger.Sugar()
	defer zapLogger.Sync() // flushes buffer, if any

	cmdConfig := config.NewConfig(sugar)
	db, err := service.ConnectSQLDb(cmdConfig, sugar)
	if err != nil {
		panic("failed to connect database")
	}

	bar := progressbar.Default(int64(amount * beats))
	defer bar.Close()

	var wg sync.WaitGroup
	for i := range amount {
		wg.Add(1)
		go func() {
			defer wg.Done()

			evidenceId := uuid.New()

			components := []relational.SystemComponent{
				{
					UUIDModel: relational.UUIDModel{
						ID: internal.Pointer(uuid.New()),
					},
					Type:  "component",
					Title: fmt.Sprintf("Some component - %d", i),
				},
				{
					UUIDModel: relational.UUIDModel{
						ID: internal.Pointer(uuid.New()),
					},
					Type:  "component",
					Title: fmt.Sprintf("Some other component - %d", i),
				},
			}
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&components)

			inventoryItems := []relational.InventoryItem{
				{
					UUIDModel: relational.UUIDModel{
						ID: internal.Pointer(uuid.New()),
					},
					Description: fmt.Sprintf("EC2 Instance - %d", i),
					ImplementedComponents: []relational.ImplementedComponent{
						{ComponentID: *components[0].ID},
						{ComponentID: *components[1].ID},
					},
				},
			}
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&inventoryItems)

			activities := []relational.Activity{
				{
					Title: internal.Pointer("Activity - 1"),
					Steps: []relational.Step{
						{
							Title: internal.Pointer("Activity - 1 - Step - 1"),
						},
						{
							Title: internal.Pointer("Activity - 1 - Step - 2"),
						},
					},
				},
				{
					Title: internal.Pointer("Activity - 2"),
					Steps: []relational.Step{
						{
							Title: internal.Pointer("Activity - 2 - Step - 1"),
						},
						{
							Title: internal.Pointer("Activity - 2 - Step - 2"),
						},
					},
				},
			}
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&activities)

			subjects := []relational.AssessmentSubject{
				{
					Type: "component",
					IncludeSubjects: []relational.SelectSubjectById{
						{
							SubjectUUID: *components[0].ID,
						},
					},
				},
				{
					Type: "component",
					IncludeSubjects: []relational.SelectSubjectById{
						{
							SubjectUUID: *components[1].ID,
						},
					},
				},
			}
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&subjects)

			labels := []relational.Labels{
				{
					Name:  "provider",
					Value: "AWS",
				},
				{
					Name:  "service",
					Value: "EC2",
				},
				{
					Name:  "instance",
					Value: fmt.Sprintf("i-%d", i),
				},
			}
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&labels)

			for b := range beats {
				err = bar.Add(1)
				if err != nil {
					log.Fatal(err)
				}

				evidence := relational.Evidence{
					UUID:  evidenceId,
					Title: internal.Pointer(fmt.Sprintf("Evidence %d", b)),
					Start: time.Now().Add(-(time.Hour + (time.Duration(b) * time.Minute))),
					End:   time.Now().Add(-(time.Hour + time.Minute + (time.Duration(b) * time.Minute))),
					Status: datatypes.NewJSONType(oscalTypes_1_1_3.ObjectiveStatus{
						Reason: "pass",
						State:  "satisfied",
					}),
				}

				if err := db.Create(&evidence).Error; err != nil {
					panic(err)
				}

				if err = db.Model(&evidence).Association("Activities").Append(activities); err != nil {
					panic(err)
				}

				if err = db.Model(&evidence).Association("InventoryItems").Append(inventoryItems); err != nil {
					panic(err)
				}

				if err = db.Model(&evidence).Association("Components").Append(components); err != nil {
					panic(err)
				}

				if err = db.Model(&evidence).Association("Subjects").Append(subjects); err != nil {
					panic(err)
				}

				if err = db.Model(&evidence).Association("Labels").Append(labels); err != nil {
					panic(err)
				}
			}
		}()
	}
	wg.Wait()
}
