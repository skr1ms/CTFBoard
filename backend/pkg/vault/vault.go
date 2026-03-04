package vault

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	vault "github.com/hashicorp/vault/api"
)

type Client struct {
	client    *vault.Client
	mountPath string
}

func New(
	addr string,
	token string,
) (*Client, error) {
	return NewWithMount(addr, token, "secret")
}

func NewWithMount(
	addr string,
	token string,
	mountPath string,
) (*Client, error) {
	config := vault.DefaultConfig()
	config.Address = addr

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(token)

	return &Client{
		client:    client,
		mountPath: mountPath,
	}, nil
}

func NewFromEnv() (*Client, error) {
	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		return nil, fmt.Errorf("VAULT_ADDR environment variable is not set")
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable is not set")
	}

	mountPath := os.Getenv("VAULT_MOUNT_PATH")
	if mountPath == "" {
		mountPath = "secret"
	}

	return NewWithMount(addr, token, mountPath)
}

func (c *Client) GetSecret(ctx context.Context, secretPath string) (map[string]any, error) {
	kv := c.client.KVv2(c.mountPath)

	var data map[string]any
	operation := func() error {
		secret, err := kv.Get(ctx, secretPath)
		if err != nil {
			if isVaultPermanentError(err) {
				return backoff.Permanent(fmt.Errorf("failed to read secret from vault: %w", err))
			}
			return fmt.Errorf("failed to read secret from vault: %w", err)
		}
		if secret == nil || secret.Data == nil {
			return backoff.Permanent(fmt.Errorf("secret not found at path: %s/%s", c.mountPath, secretPath))
		}
		data = secret.Data
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 30 * time.Second
	bo.InitialInterval = 500 * time.Millisecond

	if err := backoff.Retry(operation, backoff.WithContext(bo, ctx)); err != nil {
		return nil, err
	}
	return data, nil
}

func isVaultPermanentError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, code := range []string{"403", "404", "400"} {
		if strings.Contains(msg, "* "+code) || strings.Contains(msg, code+" ") {
			return true
		}
	}
	var re *vault.ResponseError
	if asErr := asVaultResponseError(err, &re); asErr && re != nil {
		return re.StatusCode == http.StatusForbidden ||
			re.StatusCode == http.StatusUnauthorized ||
			re.StatusCode == http.StatusNotFound ||
			re.StatusCode == http.StatusBadRequest
	}
	return false
}

func asVaultResponseError(err error, target **vault.ResponseError) bool {
	if re, ok := err.(*vault.ResponseError); ok { //nolint:errorlint
		*target = re
		return true
	}
	return false
}

func (c *Client) GetString(ctx context.Context, secretPath, key string) (string, error) {
	data, err := c.GetSecret(ctx, secretPath)
	if err != nil {
		return "", err
	}

	value, ok := data[key].(string)
	if !ok {
		return "", fmt.Errorf("key %s not found or not a string in path %s/%s", key, c.mountPath, secretPath)
	}

	return value, nil
}
