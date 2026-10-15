//go:build integration

// Package bench holds opt-in benchmarks that run the API against a real, resource-limited
// Postgres. They are skipped unless their CCF_BENCH_* switch is set.
package bench

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/api/handler"
	authhandler "github.com/compliance-framework/api/internal/api/handler/auth"
	"github.com/compliance-framework/api/internal/config"
	"github.com/compliance-framework/api/internal/service/email"
	"github.com/compliance-framework/api/internal/service/relational"
	evidencesvc "github.com/compliance-framework/api/internal/service/relational/evidence"
	riskrel "github.com/compliance-framework/api/internal/service/relational/risks"
	"github.com/compliance-framework/api/internal/service/relational/templates"
	"github.com/compliance-framework/api/internal/service/worker"
	"github.com/compliance-framework/api/internal/tests"
	"github.com/compliance-framework/api/sdk"
	"github.com/compliance-framework/api/sdk/types"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgresContainers "github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// benchSettings are the benchmark's knobs, read from CCF_BENCH_* variables. The batch
// limits themselves come from the API's own CCF_EVIDENCE_BATCH_* variables.
type benchSettings struct {
	evidence          int
	dbCPUs            float64
	dbMemoryMB        int64
	apiCPUs           int
	apiDBConns        int
	singleConcurrency int
	// latency is added before every request, standing in for the network between an
	// agent and the API (loopback has none).
	latency time.Duration
	output  string
}

func loadBenchSettings(t *testing.T) benchSettings {
	t.Helper()
	return benchSettings{
		evidence:          envInt(t, "CCF_BENCH_EVIDENCE_COUNT", 5000),
		dbCPUs:            envFloat(t, "CCF_BENCH_DB_CPUS", 1),
		dbMemoryMB:        int64(envInt(t, "CCF_BENCH_DB_MEMORY_MB", 1024)),
		apiCPUs:           envInt(t, "CCF_BENCH_API_CPUS", 2),
		apiDBConns:        envInt(t, "CCF_BENCH_API_DB_CONNS", 10),
		singleConcurrency: envInt(t, "CCF_BENCH_SINGLE_CONCURRENCY", 1),
		latency:           time.Duration(envIntOrZero(t, "CCF_BENCH_LATENCY_MS")) * time.Millisecond,
		output:            os.Getenv("CCF_BENCH_OUTPUT"),
	}
}

func envInt(t *testing.T, name string, fallback int) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	require.NoError(t, err, name)
	require.Positive(t, value, name)
	return value
}

func envIntOrZero(t *testing.T, name string) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	require.NoError(t, err, name)
	require.GreaterOrEqual(t, value, 0, name)
	return value
}

func envFloat(t *testing.T, name string, fallback float64) float64 {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	require.NoError(t, err, name)
	require.Positive(t, value, name)
	return value
}

// noProfileControls satisfies the worker service's profile resolver; the benchmark never
// changes a profile binding.
type noProfileControls struct{}

func (noProfileControls) ResolveProfileControlKeys(context.Context, uuid.UUID) ([]riskrel.ControlKey, error) {
	return nil, nil
}

// benchEnv is one API, with its worker service, over one resource-limited Postgres.
type benchEnv struct {
	db       *gorm.DB
	migrator *tests.TestMigrator
	baseURL  string
	latency  time.Duration
}

// delayedTransport waits before each request, simulating network latency.
type delayedTransport struct {
	delay time.Duration
	next  http.RoundTripper
}

func (d delayedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(d.delay)
	return d.next.RoundTrip(req)
}

// TestEvidenceIngestBenchmark stores the same evidence through POST /api/evidence (one
// request per evidence, as agents send it today) and through POST /api/evidence/batches,
// against the same API and the same resource-limited Postgres, and reports both times.
//
// Opt in with CCF_BENCH_EVIDENCE_INGEST=1 (make bench-evidence-ingest); see
// docs/evidence-batches.md for the knobs.
func TestEvidenceIngestBenchmark(t *testing.T) {
	if os.Getenv("CCF_BENCH_EVIDENCE_INGEST") == "" {
		t.Skip("set CCF_BENCH_EVIDENCE_INGEST=1 to run the evidence ingest benchmark")
	}
	settings := loadBenchSettings(t)
	env := startBenchEnv(t, settings)

	single := env.run(t, "single", func(ctx context.Context, client *sdk.Client, evidence []types.Evidence) {
		sendOneByOne(ctx, t, client, evidence, settings.singleConcurrency)
	}, settings.evidence)

	batchCfg := env.batchConfig(t)
	batched := env.run(t, "batch", func(ctx context.Context, client *sdk.Client, evidence []types.Evidence) {
		sendInBatches(ctx, t, client, evidence, batchCfg.MaxItems)
	}, settings.evidence)

	report := fmt.Sprintf(`Evidence ingest benchmark (%d evidence)

Postgres: %.2f CPUs, %d MB. API: GOMAXPROCS %d, %d DB connections.
Added latency per request: %s.
POST /api/evidence: %d concurrent client(s).
POST /api/evidence/batches: %d evidence per batch, chunks of %d, %d chunk workers.

| Mode   | Stored | Wall time | Evidence/s |
|--------|--------|-----------|------------|
| single | %6d | %9s | %10.1f |
| batch  | %6d | %9s | %10.1f |

Batch speed-up: %.1fx
`,
		settings.evidence, settings.dbCPUs, settings.dbMemoryMB, settings.apiCPUs, settings.apiDBConns,
		settings.latency, settings.singleConcurrency, batchCfg.MaxItems, batchCfg.ChunkSize, batchCfg.Workers,
		single.stored, single.elapsed.Round(time.Millisecond), single.rate(),
		batched.stored, batched.elapsed.Round(time.Millisecond), batched.rate(),
		float64(single.elapsed)/float64(batched.elapsed))
	t.Log("\n" + report)
	if settings.output != "" {
		require.NoError(t, os.WriteFile(settings.output, []byte(report), 0o644))
	}
}

type benchResult struct {
	stored  int64
	elapsed time.Duration
}

func (r benchResult) rate() float64 {
	return float64(r.stored) / r.elapsed.Seconds()
}

// run stores count evidence with send on a fresh database and times it, from the first
// request until every evidence is stored.
func (e *benchEnv) run(t *testing.T, mode string, send func(context.Context, *sdk.Client, []types.Evidence), count int) benchResult {
	t.Helper()
	e.reset(t)
	client := e.agentClient(t)
	evidence := benchEvidence(count)
	ctx := context.Background()

	start := time.Now()
	send(ctx, client, evidence)
	elapsed := time.Since(start)

	var stored int64
	require.NoError(t, e.db.Model(&relational.Evidence{}).Count(&stored).Error)
	require.EqualValues(t, count, stored, "%s: every evidence is stored", mode)
	t.Logf("%s: %d evidence in %s", mode, stored, elapsed)
	return benchResult{stored: stored, elapsed: elapsed}
}

// sendOneByOne posts each evidence with concurrency senders, stopping at the first error.
func sendOneByOne(ctx context.Context, t *testing.T, client *sdk.Client, evidence []types.Evidence, concurrency int) {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	work := make(chan types.Evidence)
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range work {
				if err := client.Evidence.Create(ctx, item); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					cancel()
					return
				}
			}
		}()
	}

produce:
	for _, item := range evidence {
		select {
		case <-ctx.Done():
			break produce
		case work <- item:
		}
	}
	close(work)
	wg.Wait()
	require.NoError(t, errors.Join(errs...))
}

func sendInBatches(ctx context.Context, t *testing.T, client *sdk.Client, evidence []types.Evidence, size int) {
	t.Helper()
	var ids []uuid.UUID
	for first := 0; first < len(evidence); first += size {
		last := min(first+size, len(evidence))
		batch, err := client.Evidence.CreateBatch(ctx, evidence[first:last])
		require.NoError(t, err)
		ids = append(ids, batch.ID)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	for _, id := range ids {
		batch, err := client.Evidence.WaitForBatch(waitCtx, id, 250*time.Millisecond)
		require.NoError(t, err)
		require.Zero(t, batch.Failed, "batch %s: %+v", id, firstFailure(batch))
	}
}

func firstFailure(batch *types.EvidenceBatch) *types.EvidenceBatchItem {
	for i := range batch.Items {
		if batch.Items[i].Status == types.EvidenceBatchItemFailed {
			return &batch.Items[i]
		}
	}
	return nil
}

// benchEvidence is count pieces of evidence shaped like an agent's: plugin labels, a
// component, an inventory item it implements, a legacy subject, and an activity.
func benchEvidence(count int) []types.Evidence {
	now := time.Now().UTC()
	evidence := make([]types.Evidence, 0, count)
	for i := 0; i < count; i++ {
		instance := fmt.Sprintf("web-server/ec2/i-%06d", i)
		state := "satisfied"
		if i%7 == 0 {
			state = "not-satisfied"
		}
		evidence = append(evidence, types.Evidence{
			UUID:        uuid.NewSHA1(uuid.NameSpaceURL, []byte(instance)),
			Title:       fmt.Sprintf("EC2 instance %d has IMDSv2 enforced", i),
			Description: "Checks the instance metadata service options.",
			Start:       now.Add(-2 * time.Minute),
			End:         now.Add(-time.Minute),
			Labels: map[string]string{
				"_plugin":  "bench-aws-ec2",
				"_policy":  fmt.Sprintf("ec2-policy-%d", i%20),
				"provider": "aws",
				"service":  "ec2",
				"region":   fmt.Sprintf("eu-west-%d", i%3+1),
				"account":  fmt.Sprintf("acct-%d", i%5),
			},
			Props: []types.Property{
				{Name: "instance-type", Value: "t3.micro"},
				{Name: "imds", Value: "v2"},
			},
			Activities: []types.Activity{{
				UUID:  uuid.New(),
				Title: "Evaluate instance configuration",
				Steps: []types.Step{
					{UUID: uuid.New(), Title: "Describe the instance"},
					{UUID: uuid.New(), Title: "Evaluate the policy"},
				},
			}},
			Components: []types.Component{{
				Identifier: "components/common/aws-ec2",
				Type:       "service",
				Title:      "Amazon EC2",
			}},
			InventoryItems: []types.InventoryItem{{
				Identifier:            instance,
				Type:                  "web-server",
				Title:                 fmt.Sprintf("Instance %d", i),
				ImplementedComponents: []types.ComponentIdentifier{{Identifier: "components/common/aws-ec2"}},
			}},
			Subjects: []types.Subject{{Identifier: instance, Type: "inventory-item"}},
			Status:   types.ObjectiveStatus{State: state},
		})
	}
	return evidence
}

func startBenchEnv(t *testing.T, settings benchSettings) *benchEnv {
	t.Helper()
	ctx := context.Background()

	previous := runtime.GOMAXPROCS(settings.apiCPUs)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	pg, err := postgresContainers.Run(ctx, "postgres:17.5",
		postgresContainers.WithDatabase("ccf"),
		postgresContainers.WithUsername("postgres"),
		postgresContainers.WithPassword("postgres"),
		postgresContainers.BasicWaitStrategies(),
		testcontainers.WithHostConfigModifier(func(hc *container.HostConfig) {
			hc.NanoCPUs = int64(settings.dbCPUs * 1e9)
			hc.Memory = settings.dbMemoryMB * 1024 * 1024
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(pg) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable", "application_name=ccf-bench")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(settings.apiDBConns)
	sqlDB.SetMaxIdleConns(settings.apiDBConns)

	migrator := tests.NewTestMigrator(db)
	require.NoError(t, migrator.Up())
	require.NoError(t, migrator.CreateUser())

	cfg := &config.Config{EvidenceDefaultExpiryMonths: 1}
	cfg.JWTPrivateKey, cfg.JWTPublicKey, err = config.GenerateKeyPair(2048)
	require.NoError(t, err)
	cfg.StrictDisablePublicAgentEndpoints = true
	cfg.APIAllowedOrigins = []string{"*"}
	cfg.EvidenceBatch, err = config.LoadEvidenceBatchConfigFromViper(benchViper())
	require.NoError(t, err)

	log := zap.NewNop().Sugar()
	emailService, err := email.NewService(nil, log)
	require.NoError(t, err)
	workerService, err := worker.NewServiceWithDigest(config.DefaultWorkerConfig(), db, emailService, nil, cfg, noProfileControls{}, log)
	require.NoError(t, err)
	require.NoError(t, workerService.Migrate(ctx))
	require.NoError(t, workerService.Start(ctx))
	t.Cleanup(func() { _ = workerService.Stop(context.Background()) })

	server := api.NewServer(ctx, log, cfg, api.NewMetricsHandler(ctx, log))
	handler.RegisterHandlers(server, log, db, cfg, &handler.APIServices{
		EvidenceService: evidencesvc.NewEvidenceService(db, log, cfg, workerService,
			evidencesvc.WithComponentDefinitionResolver(templates.NewSubjectTemplateService(db))),
		RiskEnqueuer:       workerService,
		EvidenceBatchQueue: workerService,
	})
	authhandler.RegisterHandlers(server, log, db, cfg, nil, nil, nil)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	httpServer := &http.Server{Handler: server.E(), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = httpServer.Serve(listener) }()
	t.Cleanup(func() { _ = httpServer.Close() })

	return &benchEnv{db: db, migrator: migrator, baseURL: "http://" + listener.Addr().String(), latency: settings.latency}
}

// benchViper reads the API's CCF_EVIDENCE_BATCH_* variables, as the API does.
func benchViper() *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix("ccf")
	v.AutomaticEnv()
	return v
}

func (e *benchEnv) batchConfig(t *testing.T) *config.EvidenceBatchConfig {
	t.Helper()
	cfg, err := config.LoadEvidenceBatchConfigFromViper(benchViper())
	require.NoError(t, err)
	return cfg
}

// reset waits for the background jobs of the previous run (risk processing) to finish,
// then empties the database, so every run starts from the same state.
func (e *benchEnv) reset(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Minute)
	for {
		var active int64
		require.NoError(t, e.db.Raw(`SELECT count(*) FROM river_job WHERE state IN ('available', 'running', 'retryable', 'scheduled', 'pending')`).Scan(&active).Error)
		if active == 0 {
			break
		}
		require.True(t, time.Now().Before(deadline), "%d background jobs still running", active)
		time.Sleep(time.Second)
	}
	require.NoError(t, e.db.Exec(`DELETE FROM river_job`).Error)
	require.NoError(t, e.migrator.Refresh())
}

// agentClient returns an SDK client authenticated as a new agent.
func (e *benchEnv) agentClient(t *testing.T) *sdk.Client {
	t.Helper()
	agent := &relational.Agent{Name: "bench-agent-" + uuid.NewString(), IsActive: true}
	require.NoError(t, e.db.Create(agent).Error)
	secret := uuid.NewString()
	key := &relational.AgentServiceAccountKey{AgentID: agent.ID, ClientID: uuid.NewString()}
	require.NoError(t, key.SetSecret(secret))
	require.NoError(t, e.db.Create(key).Error)

	var transport http.RoundTripper = http.DefaultTransport
	if e.latency > 0 {
		transport = delayedTransport{delay: e.latency, next: transport}
	}
	return sdk.NewClient(&http.Client{Timeout: 5 * time.Minute, Transport: transport}, &sdk.Config{
		BaseURL:   e.baseURL,
		AgentAuth: &sdk.AgentAuthConfig{ClientID: key.ClientID, ClientSecret: secret},
	})
}
