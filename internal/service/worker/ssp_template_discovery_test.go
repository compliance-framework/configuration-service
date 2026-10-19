package worker

import (
	"context"
	"testing"

	"github.com/compliance-framework/api/internal/config"
	riskrel "github.com/compliance-framework/api/internal/service/relational/risks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type staticControlResolver []riskrel.ControlKey

func (r staticControlResolver) ResolveProfileControlKeys(context.Context, uuid.UUID) ([]riskrel.ControlKey, error) {
	return r, nil
}

func TestSSPTemplateDiscoverySchedule(t *testing.T) {
	require.Equal(t, config.DefaultSSPTemplateDiscoverySchedule, sspTemplateDiscoverySchedule(nil))
	require.Equal(t, config.DefaultSSPTemplateDiscoverySchedule, sspTemplateDiscoverySchedule(&config.Config{}))
	require.Equal(t, "0 0 * * * *", sspTemplateDiscoverySchedule(&config.Config{SSPTemplate: &config.SSPTemplateConfig{DiscoverySchedule: "0 0 * * * *"}}))
	// An invalid schedule falls back to the default instead of failing start-up.
	require.NotNil(t, NewSSPTemplateDiscoveryPeriodicJob("not a cron", zap.NewNop().Sugar()))
}

func TestControlIDResolverDedupesCaseInsensitively(t *testing.T) {
	resolve := controlIDResolver(context.Background(), staticControlResolver{
		{CatalogID: "a", ControlID: "AC-1"}, {CatalogID: "b", ControlID: "ac-1"}, {CatalogID: "a", ControlID: ""}, {CatalogID: "a", ControlID: "AC-2"},
	})
	ids, err := resolve(uuid.New())
	require.NoError(t, err)
	require.Equal(t, []string{"AC-1", "AC-2"}, ids)
}

func TestEnqueueSSPTemplateDiscoveryNeedsTheWorker(t *testing.T) {
	disabled := &Service{config: &config.WorkerConfig{Enabled: false}}
	require.Error(t, disabled.EnqueueSSPTemplateDiscovery(context.Background(), uuid.New()), "a run that would never happen is not accepted")
}
