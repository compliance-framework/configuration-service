package evidence

import (
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/authn"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func agentSignerForTest() *SignerContext {
	agentID := uuid.New()
	keyID := uuid.New()
	now := time.Now().Truncate(time.Second)
	return NewAgentSignerContext(
		&authn.AgentClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "agent-subject",
				Issuer:    "ccf",
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			},
			TokenKind:    authn.TokenKindAgent,
			AgentID:      agentID.String(),
			CredentialID: keyID.String(),
			AuthMethod:   "key",
		},
		&relational.Agent{UUIDModel: relational.UUIDModel{ID: &agentID}, Name: "collector"},
		&relational.AgentServiceAccountKey{UUIDModel: relational.UUIDModel{ID: &keyID}},
	)
}

func TestBatchSignerRoundTripKeepsWhatSigningReads(t *testing.T) {
	acceptedAt := time.Now().UTC().Truncate(time.Microsecond)
	for name, signer := range map[string]*SignerContext{
		"agent": agentSignerForTest(),
		"user":  userSignerForUnitTest(),
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := encodeBatchSigner(signer)
			require.NoError(t, err)

			decoded, err := decodeBatchSigner(encoded, acceptedAt)
			require.NoError(t, err)

			require.Equal(t, signer.signerSnapshot(), decoded.signerSnapshot())
			require.Equal(t, signer.claimsSnapshot(), decoded.claimsSnapshot())
			require.Equal(t, BatchSubmitter(signer), BatchSubmitter(decoded))
			require.Equal(t, acceptedAt, decoded.AcceptedAt)
		})
	}
}

func userSignerForUnitTest() *SignerContext {
	now := time.Now().Truncate(time.Second)
	return NewUserSignerContextFromClaims(&authn.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user@example.com",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
		TokenKind:  authn.TokenKindUser,
		GivenName:  "Ada",
		FamilyName: "Lovelace",
	})
}

func TestBatchSubmitter(t *testing.T) {
	agent := agentSignerForTest()
	require.Equal(t, "agent:"+agent.Agent.Agent.ID.String(), BatchSubmitter(agent))
	require.Equal(t, "user:user@example.com", BatchSubmitter(userSignerForUnitTest()))

	// An agent known only by its claims is still identified.
	claimsOnly := NewAgentSignerContext(&authn.AgentClaims{AgentID: "a-1"}, nil, nil)
	require.Equal(t, "agent:a-1", BatchSubmitter(claimsOnly))

	require.Empty(t, BatchSubmitter(nil))
	require.Empty(t, BatchSubmitter(&SignerContext{}))
	require.Empty(t, BatchSubmitter(NewUserSignerContextFromClaims(&authn.UserClaims{})))
}

func TestSignedAtUsesAcceptedAt(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	clock := func() time.Time { return now }

	require.Equal(t, now, signedAt(&SignerContext{}, clock))

	accepted := now.Add(-time.Hour).In(time.FixedZone("x", 3600))
	require.Equal(t, accepted.UTC(), signedAt(&SignerContext{AcceptedAt: accepted}, clock))
}
