package credentials

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/glebarez/sqlite"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/server/options/encryptionconfig"
	"k8s.io/apiserver/pkg/storage/value"
)

// uid is here to fulfill the value.Context interface for the transformer.
// This is similar to authenticatedDataString from the k8s apiserver's storage interface
// for etcd: https://github.com/kubernetes/kubernetes/blob/a42f4f61c2c46553bfe338eefe9e81818c7360b4/staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.go#L63
type uid string

func (u uid) AuthenticatedData() []byte {
	return []byte(u)
}

var groupResource = schema.GroupResource{
	Group:    "", // deliberately left empty
	Resource: "credentials",
}

type DBStore struct {
	credCtxs    []string
	cfg         *config.CLIConfig
	db          *gorm.DB
	transformer value.Transformer
}

// GptscriptCredential is the struct we use to represent credentials in the database.
type GptscriptCredential struct {
	// We aren't using gorm.Model because we don't want a DeletedAt field.
	// We want records to be fully deleted from the database.
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// We set up an extra index here to enforce a unique constraint on context+name.
	Context string `gorm:"index:contextname,unique"`
	Name    string `gorm:"index:contextname,unique"`

	Type, Env, RefreshToken string
	Ephemeral               bool
	ExpiresAt               *time.Time
}

func credToDBCred(cred Credential) (GptscriptCredential, error) {
	envJSON, err := json.Marshal(cred.Env)
	if err != nil {
		return GptscriptCredential{}, fmt.Errorf("failed to marshal env: %w", err)
	}

	return GptscriptCredential{
		Context:      cred.Context,
		Name:         cred.ToolName,
		Type:         string(cred.Type),
		Env:          string(envJSON),
		Ephemeral:    cred.Ephemeral,
		ExpiresAt:    cred.ExpiresAt,
		RefreshToken: cred.RefreshToken,
	}, nil
}

func dbCredToCred(dbCred GptscriptCredential) (Credential, error) {
	var env map[string]string
	if err := json.Unmarshal([]byte(dbCred.Env), &env); err != nil {
		return Credential{}, fmt.Errorf("failed to unmarshal env: %w", err)
	}

	return Credential{
		Context:      dbCred.Context,
		ToolName:     dbCred.Name,
		Type:         CredentialType(dbCred.Type),
		Env:          env,
		Ephemeral:    dbCred.Ephemeral,
		ExpiresAt:    dbCred.ExpiresAt,
		RefreshToken: dbCred.RefreshToken,
	}, nil
}

func NewDBStore(ctx context.Context, cfg *config.CLIConfig, credCtxs []string) (*DBStore, error) {
	store := DBStore{
		credCtxs: credCtxs,
		cfg:      cfg,
	}

	encryptionConf, err := readEncryptionConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read encryption config: %w", err)
	} else if encryptionConf != nil {
		// The transformer that we get from the encryption configuration is the interface we use for encryption and decryption.
		transformer, exists := encryptionConf.Transformers[groupResource]
		if !exists {
			return nil, fmt.Errorf("failed to find encryption transformer for %s", groupResource.String())
		}
		store.transformer = transformer
	}

	var dbPath string
	if os.Getenv("GPTSCRIPT_SQLITE_FILE") != "" {
		dbPath = os.Getenv("GPTSCRIPT_SQLITE_FILE")
	} else {
		dbPath, err = xdg.ConfigFile("gptscript/credentials.db")
		if err != nil {
			return nil, fmt.Errorf("failed to get credentials db path: %w", err)
		}
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.AutoMigrate(&GptscriptCredential{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate GptscriptCredential: %w", err)
	}

	store.db = db

	return &store, nil
}

func readEncryptionConfig(ctx context.Context) (*encryptionconfig.EncryptionConfiguration, error) {
	encryptionConfigPath := os.Getenv("GPTSCRIPT_ENCRYPTION_CONFIG_FILE")
	if encryptionConfigPath == "" {
		var err error
		if encryptionConfigPath, err = xdg.ConfigFile("gptscript/encryptionconfig.yaml"); err != nil {
			return nil, fmt.Errorf("failed to read encryption config from standard location: %w", err)
		}
	}

	if _, err := os.Stat(encryptionConfigPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to stat encryption config file: %w", err)
	}

	// Use k8s libraries to load the encryption config from the file:
	return encryptionconfig.LoadEncryptionConfig(ctx, encryptionConfigPath, false, "gptscript")
}

func (d *DBStore) encryptCred(ctx context.Context, cred GptscriptCredential) (GptscriptCredential, error) {
	if d.transformer == nil {
		return cred, nil
	}

	// Encrypt the environment variables
	envBytes := []byte(cred.Env)
	encryptedEnvBytes, err := d.transformer.TransformToStorage(ctx, envBytes, uid(cred.Name+cred.Context))
	if err != nil {
		return GptscriptCredential{}, fmt.Errorf("failed to encrypt env: %w", err)
	}
	cred.Env = fmt.Sprintf("{\"e\": %q}", base64.StdEncoding.EncodeToString(encryptedEnvBytes))

	// Encrypt the refresh token
	if cred.RefreshToken != "" {
		refreshTokenBytes := []byte(cred.RefreshToken)
		encryptedRefreshTokenBytes, err := d.transformer.TransformToStorage(ctx, refreshTokenBytes, uid(cred.Name+cred.Context))
		if err != nil {
			return GptscriptCredential{}, fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
		cred.RefreshToken = fmt.Sprintf("{\"e\": %q}", base64.StdEncoding.EncodeToString(encryptedRefreshTokenBytes))
	}

	return cred, nil
}

func (d *DBStore) decryptCred(ctx context.Context, cred GptscriptCredential) (GptscriptCredential, error) {
	if d.transformer == nil {
		return cred, nil
	}

	var envMap map[string]string
	if err := json.Unmarshal([]byte(cred.Env), &envMap); err == nil {
		if encryptedEnvB64, exists := envMap["e"]; exists && len(envMap) == 1 {
			encryptedEnvBytes, err := base64.StdEncoding.DecodeString(encryptedEnvB64)
			if err != nil {
				return GptscriptCredential{}, fmt.Errorf("failed to decode env: %w", err)
			}

			envBytes, _, err := d.transformer.TransformFromStorage(ctx, encryptedEnvBytes, uid(cred.Name+cred.Context))
			if err != nil {
				return GptscriptCredential{}, fmt.Errorf("failed to decrypt env: %w", err)
			}
			cred.Env = string(envBytes)
		}
	}

	var refreshTokenMap map[string]string
	if err := json.Unmarshal([]byte(cred.RefreshToken), &refreshTokenMap); err == nil {
		if encryptedRefreshTokenB64, exists := refreshTokenMap["e"]; exists && len(refreshTokenMap) == 1 {
			encryptedRefreshTokenBytes, err := base64.StdEncoding.DecodeString(encryptedRefreshTokenB64)
			if err != nil {
				return GptscriptCredential{}, fmt.Errorf("failed to decode refresh token: %w", err)
			}

			refreshTokenBytes, _, err := d.transformer.TransformFromStorage(ctx, encryptedRefreshTokenBytes, uid(cred.Name+cred.Context))
			if err != nil {
				return GptscriptCredential{}, fmt.Errorf("failed to decrypt refresh token: %w", err)
			}
			cred.RefreshToken = string(refreshTokenBytes)
		}
	}

	return cred, nil
}

func (d *DBStore) Get(ctx context.Context, toolName string) (*Credential, bool, error) {
	var (
		dbCred GptscriptCredential
		found  bool
	)
	for _, credCtx := range d.credCtxs {
		if err := d.db.WithContext(ctx).Where("context = ? AND name = ?", credCtx, toolName).First(&dbCred).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, false, fmt.Errorf("failed to query for credential: %w", err)
		}
		found = true
		break
	}

	if !found {
		return nil, false, nil
	}

	dbCred, err := d.decryptCred(ctx, dbCred)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	cred, err := dbCredToCred(dbCred)
	if err != nil {
		return nil, false, fmt.Errorf("failed to convert GptscriptCredential to Credential: %w", err)
	}

	return &cred, true, nil
}

func (d *DBStore) Add(ctx context.Context, cred Credential) error {
	cred.Context = first(d.credCtxs)

	dbCred, err := credToDBCred(cred)
	if err != nil {
		return fmt.Errorf("failed to convert credential to GptscriptCredential: %w", err)
	}

	dbCred, err = d.encryptCred(ctx, dbCred)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential: %w", err)
	}

	if err := d.db.WithContext(ctx).Create(&dbCred).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("credential with name [%s] and context [%s] already exists", cred.ToolName, cred.Context)
		}
		return fmt.Errorf("failed to insert credential into database: %w", err)
	}
	return nil
}

func (d *DBStore) Remove(ctx context.Context, toolName string) error {
	first := first(d.credCtxs)
	if len(d.credCtxs) > 1 || first == AllCredentialContexts {
		return fmt.Errorf("error: credential deletion is not supported when multiple credential contexts are provided")
	}

	var cred GptscriptCredential
	if err := d.db.WithContext(ctx).Where("context = ? AND name = ?", first, toolName).First(&cred).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("credential with name [%s] not found", toolName)
		}
		return fmt.Errorf("failed to query for credential: %w", err)
	}

	if err := d.db.WithContext(ctx).Where("context = ? AND name = ?", cred.Context, cred.Name).Delete(&GptscriptCredential{}).Error; err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	return nil
}

func (d *DBStore) List(ctx context.Context) ([]Credential, error) {
	var (
		dbCreds []GptscriptCredential
		err     error
	)
	if err = d.db.WithContext(ctx).Where("context = ?", first(d.credCtxs)).Find(&dbCreds).Error; err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	var credentials []Credential
	for _, dbCred := range dbCreds {
		dbCred, err = d.decryptCred(ctx, dbCred)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credential: %w", err)
		}

		credential, err := dbCredToCred(dbCred)
		if err != nil {
			return nil, fmt.Errorf("failed to convert GptscriptCredential to Credential: %w", err)
		}

		credentials = append(credentials, credential)
	}

	return credentials, nil
}
