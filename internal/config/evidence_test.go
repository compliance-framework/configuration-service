package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestLoadEvidenceSubjectConfigDefaults(t *testing.T) {
	cfg, err := LoadEvidenceSubjectConfigFromViper(viper.New())

	require.NoError(t, err)
	require.Equal(t, EvidenceRequireSubjectOff, cfg.RequireSubject)
	require.False(t, cfg.ManualRequireSubject)
}

func TestLoadEvidenceSubjectConfigReadsFlags(t *testing.T) {
	v := viper.New()
	v.Set("evidence_require_subject", " Enforce ")
	v.Set("manual_evidence_require_subject", "true")

	cfg, err := LoadEvidenceSubjectConfigFromViper(v)

	require.NoError(t, err)
	require.Equal(t, EvidenceRequireSubjectEnforce, cfg.RequireSubject)
	require.True(t, cfg.ManualRequireSubject)
}

func TestLoadEvidenceSubjectConfigRejectsUnknownMode(t *testing.T) {
	v := viper.New()
	v.Set("evidence_require_subject", "strict")

	cfg, err := LoadEvidenceSubjectConfigFromViper(v)

	require.Nil(t, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "CCF_EVIDENCE_REQUIRE_SUBJECT")
}

func TestLoadEvidenceBatchConfigDefaults(t *testing.T) {
	cfg, err := LoadEvidenceBatchConfigFromViper(viper.New())

	require.NoError(t, err)
	require.Equal(t, &EvidenceBatchConfig{
		MaxItems:  DefaultEvidenceBatchMaxItems,
		ChunkSize: DefaultEvidenceBatchChunkSize,
		Workers:   DefaultEvidenceBatchWorkers,
	}, cfg)
}

func TestLoadEvidenceBatchConfigReadsSettings(t *testing.T) {
	v := viper.New()
	v.Set("evidence_batch_max_items", "5000")
	v.Set("evidence_batch_chunk_size", "250")
	v.Set("evidence_batch_workers", "8")

	cfg, err := LoadEvidenceBatchConfigFromViper(v)

	require.NoError(t, err)
	require.Equal(t, &EvidenceBatchConfig{MaxItems: 5000, ChunkSize: 250, Workers: 8}, cfg)
}

func TestLoadEvidenceBatchConfigRejectsNonPositive(t *testing.T) {
	for _, key := range []string{"evidence_batch_max_items", "evidence_batch_chunk_size", "evidence_batch_workers"} {
		t.Run(key, func(t *testing.T) {
			v := viper.New()
			v.Set(key, "0")

			cfg, err := LoadEvidenceBatchConfigFromViper(v)

			require.Nil(t, cfg)
			require.Error(t, err)
			require.Contains(t, err.Error(), "CCF_EVIDENCE_BATCH_")
		})
	}
}
