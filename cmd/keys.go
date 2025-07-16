package cmd

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/compliance-framework/configuration-service/internal/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"log"
	"os"
	"path/filepath"
)

func newGenerateKeysCMD() *cobra.Command {
	keys := &cobra.Command{
		Use:   "generate-keys",
		Short: "Generate new JWT public and private keys",
		Run:   generateKeys,
	}

	keys.Flags().StringP("output", "o", "", "Output directory to store PEM encoded files")
	err := keys.MarkFlagRequired("output")
	if err != nil {
		panic(err)
	}

	return keys
}

func generateKeys(cmd *cobra.Command, args []string) {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Can't initialize zap logger: %v", err)
	}
	logger := zapLogger.Sugar()
	defer zapLogger.Sync() // flushes buffer, if any

	jwtPrivateKey, jwtPublicKey, err := config.GenerateKeyPair(2048)
	if err != nil {
		logger.Fatalw("Failed to generate RSA key pair", "error", err)
	}

	privatePem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(jwtPrivateKey),
		},
	)

	pubData, err := x509.MarshalPKIXPublicKey(jwtPublicKey)
	if err != nil {
		logger.Fatalw("Failed to marshal public key", "error", err)
	}
	publicPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubData,
		},
	)

	// Determine output directory from flag
	outputDir, err := cmd.Flags().GetString("output")
	if err != nil {
		logger.Fatalw("Failed to get output directory flag", "error", err)
	}
	// Create directory if it doesn't exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			logger.Fatalw("Failed to create output directory", "dir", outputDir, "error", err)
		}
	}
	// Write private key
	privatePath := filepath.Join(outputDir, "private.pem")
	if err := os.WriteFile(privatePath, privatePem, 0600); err != nil {
		logger.Fatalw("Failed to write private key file", "path", privatePath, "error", err)
	}
	// Write public key
	publicPath := filepath.Join(outputDir, "public.pem")
	if err := os.WriteFile(publicPath, publicPem, 0644); err != nil {
		logger.Fatalw("Failed to write public key file", "path", publicPath, "error", err)
	}
	fmt.Printf("Keys generated and saved to:\n  %s\n  %s\n", privatePath, publicPath)
}
