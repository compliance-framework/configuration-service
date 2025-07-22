//go:build integration

package tests

import (
	"github.com/compliance-framework/api/internal/service"
	"github.com/compliance-framework/api/internal/service/relational"
	"gorm.io/gorm"
)

type TestMigrator struct {
	db *gorm.DB
}

func NewTestMigrator(db *gorm.DB) *TestMigrator {
	return &TestMigrator{
		db: db,
	}
}

func (t *TestMigrator) Refresh() error {
	err := t.Down()
	if err != nil {
		return err
	}

	err = t.Up()
	if err != nil {
		return err
	}
	err = t.CreateUser()
	if err != nil {
		return err
	}

	return nil
}

func (t *TestMigrator) Up() error {
	return t.db.AutoMigrate(
		&relational.ResponsiblePartyParties{},
		&relational.Location{},
		&relational.Party{},
		&relational.BackMatterResource{},
		&relational.BackMatter{},
		&relational.Role{},
		&relational.Revision{},
		&relational.ResponsibleParty{},
		&relational.ResponsibleRole{},
		&relational.Action{},
		&relational.Metadata{},
		&relational.Group{},
		&relational.Control{},
		&relational.Catalog{},
		&relational.ControlStatementImplementation{},
		&relational.ImplementedRequirementControlImplementation{},
		&relational.ControlImplementationSet{},
		&relational.ComponentDefinition{},
		&relational.Capability{},
		&relational.DefinedComponent{},
		&relational.Diagram{},
		&relational.DataFlow{},
		&relational.NetworkArchitecture{},
		&relational.AuthorizationBoundary{},
		&relational.InformationType{},
		&relational.SystemInformation{},
		&relational.SystemCharacteristics{},
		&relational.AuthorizedPrivilege{},
		&relational.SystemUser{},
		&relational.LeveragedAuthorization{},
		&relational.SystemComponent{},
		&relational.ImplementedComponent{},
		&relational.InventoryItem{},
		&relational.SystemImplementation{},
		&relational.ControlImplementationResponsibility{},
		&relational.ProvidedControlImplementation{},
		&relational.SatisfiedControlImplementationResponsibility{},
		&relational.Export{},
		&relational.InheritedControlImplementation{},
		&relational.ByComponent{},
		&relational.Statement{},
		&relational.ImplementedRequirement{},
		&relational.ControlImplementation{},
		&relational.SystemSecurityPlan{},
		&relational.AuthorizationBoundary{},
		&relational.NetworkArchitecture{},
		&relational.DataFlow{},
		&relational.Diagram{},

		&relational.AssessmentPlan{},
		&relational.TermsAndConditions{},
		&relational.AssessmentPart{},
		&relational.LocalDefinitions{},
		&relational.LocalObjective{},
		&relational.Task{},
		&relational.TaskDependency{},
		&relational.AssessmentAsset{},
		&relational.AssessmentPlatform{},
		&relational.UsesComponent{},
		&relational.AssessmentSubject{},
		&relational.SelectSubjectById{},
		&relational.AssociatedActivity{},
		&relational.Activity{},
		&relational.Step{},
		&relational.ReviewedControls{},
		&relational.ControlSelection{},
		&relational.ControlObjectiveSelection{},
		&relational.SelectObjectiveById{},

		// POAM entities
		&relational.PlanOfActionAndMilestones{},
		&relational.PlanOfActionAndMilestonesLocalDefinitions{},
		&relational.PoamItem{},
		&relational.Risk{},
		&relational.Observation{},
		&relational.Finding{},

		&relational.Profile{},
		&relational.Import{},
		&relational.Merge{},
		&relational.Modify{},
		&relational.ParameterSetting{},
		&relational.Alteration{},
		&relational.Addition{},
		&relational.SelectControlById{},
		&relational.ResponsibleRole{},
		&relational.AssessmentResult{},
		&relational.Activity{},
		&relational.Step{},
		&relational.Task{},
		&relational.AssessedControlsSelectControlById{},
		&relational.Result{},
		&relational.AssessmentLog{},
		&relational.AssessmentLogEntry{},
		&relational.User{},

		&service.Heartbeat{},
		&relational.Evidence{},
		&relational.Labels{},
		&relational.SelectSubjectById{},
		&relational.Filter{},
		&relational.Step{},
	)
}

func (t *TestMigrator) Down() error {
	return t.db.Migrator().DropTable(
		&relational.Location{},
		&relational.Party{},
		&relational.BackMatterResource{},
		&relational.BackMatter{},
		&relational.Role{},
		&relational.Revision{},
		&relational.Control{},
		&relational.Group{},
		&relational.ResponsibleParty{},
		&relational.Action{},
		&relational.Metadata{},
		&relational.Catalog{},
		&relational.ControlStatementImplementation{},
		&relational.ImplementedRequirementControlImplementation{},
		&relational.ControlImplementationSet{},
		&relational.ComponentDefinition{},
		&relational.Capability{},
		&relational.DefinedComponent{},
		&relational.Diagram{},
		&relational.DataFlow{},
		&relational.NetworkArchitecture{},
		&relational.AuthorizationBoundary{},
		&relational.InformationType{},
		&relational.SystemInformation{},
		&relational.SystemCharacteristics{},
		&relational.AuthorizedPrivilege{},
		&relational.AuthorizationBoundary{},
		&relational.NetworkArchitecture{},
		&relational.DataFlow{},
		&relational.SystemUser{},
		&relational.LeveragedAuthorization{},
		&relational.SystemComponent{},
		&relational.ImplementedComponent{},
		&relational.InventoryItem{},
		&relational.SystemImplementation{},
		&relational.ControlImplementationResponsibility{},
		&relational.ProvidedControlImplementation{},
		&relational.SatisfiedControlImplementationResponsibility{},
		&relational.Export{},
		&relational.InheritedControlImplementation{},
		&relational.ByComponent{},
		&relational.Statement{},
		&relational.ImplementedRequirement{},
		&relational.ControlImplementation{},
		&relational.SystemSecurityPlan{},
		"metadata_responsible_parties",
		"party_locations",
		"party_member_of_organisations",
		"responsible_party_parties",
		"action_responsible_parties",
		"capability_control_implementation_sets",
		"defined_components_control_implementation_sets",
		&relational.AssessmentPlan{},
		&relational.TermsAndConditions{},
		&relational.AssessmentPart{},
		&relational.LocalDefinitions{},
		&relational.LocalObjective{},
		&relational.Task{},
		&relational.TaskDependency{},
		&relational.AssessmentAsset{},
		&relational.AssessmentPlatform{},
		&relational.UsesComponent{},
		&relational.AssessmentSubject{},
		&relational.SelectSubjectById{},
		&relational.AssociatedActivity{},
		&relational.Activity{},
		&relational.Step{},
		&relational.ReviewedControls{},
		&relational.ControlSelection{},
		&relational.ControlObjectiveSelection{},
		&relational.SelectObjectiveById{},
		"assessment_asset_components",
		"assessment_plan_assessment_subjects",
		"associated_activity_subjects",
		"local_definition_activities",
		"local_definition_components",
		"local_definition_inventory_items",
		"local_definition_objectives",
		"local_definition_users",
		"metadata_parties",
		"metadata_roles",
		"metadata_locations",
		"responsible_role_parties",
		"responsible_roles",
		"task_subjects",
		"task_tasks",
		"uses_component_responsible_parties",
		"result_observations",
		"result_findings",
		"result_risks",
		"control_selection_assessed_controls_included",
		"control_selection_assessed_controls_excluded",
		&relational.Profile{},
		&relational.Import{},
		&relational.Merge{},
		&relational.Modify{},
		&relational.ParameterSetting{},
		&relational.Alteration{},
		&relational.Addition{},
		&relational.SelectControlById{},
		&relational.AssessmentResult{},
		&relational.Activity{},
		&relational.Step{},
		&relational.Task{},
		&relational.AssessedControlsSelectControlById{},
		&relational.Result{},
		&relational.AssessmentLog{},
		&relational.AssessmentLogEntry{},
		"assessed_controls_select_control_by_id_statements",

		&relational.PlanOfActionAndMilestones{},
		&relational.PlanOfActionAndMilestonesLocalDefinitions{},
		&relational.PoamItem{},
		&relational.Risk{},
		&relational.Observation{},
		&relational.Finding{},
		"finding_related_observations",
		"finding_related_risks",
		"poam_item_related_observations",
		"poam_item_related_findings",
		"poam_item_related_risks",
		"poam_observations",
		"poam_findings",
		"poam_risks",

		&relational.User{},

		&service.Heartbeat{},
		&relational.Evidence{},
		"evidence_activities",
		"evidence_components",
		"evidence_inventory_items",
		"evidence_labels",
		"evidence_subjects",
		&relational.Labels{},
		&relational.Filter{},
	)
}

func (t *TestMigrator) CreateUser() error {
	user := &relational.User{
		Email:     "dummy@example.com",
		FirstName: "Dummy",
		LastName:  "User",
	}
	user.SetPassword("Pa55w0rd")
	return t.db.Create(user).Error
}
