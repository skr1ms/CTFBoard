package vault

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	vault "github.com/hashicorp/vault/api"
)

const (
	vaultRetryMaxElapsed      = 30 * time.Second
	vaultRetryInitialInterval = 500 * time.Millisecond
)

// Client wraps the HashiCorp Vault API client with a configured KV v2 mount path.
type Client struct {
	client    *vault.Client
	mountPath string
}

// New creates a Client pointed at addr, authenticated with token, using the default "secret" KV mount.
func New(
	addr string,
	token string,
) (*Client, error) {
	return NewWithMount(addr, token, "secret")
}

// NewWithMount creates a Client with an explicit KV v2 mount path.
func NewWithMount(
	addr string,
	token string,
	mountPath string,
) (*Client, error) {
	config := vault.DefaultConfig()
	config.Address = addr

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("vault.NewWithMount: %w", err)
	}

	client.SetToken(token)

	return &Client{
		client:    client,
		mountPath: mountPath,
	}, nil
}

// GetSecret reads all key-value pairs from a KV v2 secret at secretPath, retrying on transient errors.
func (c *Client) GetSecret(ctx context.Context, secretPath string) (map[string]any, error) {
	kv := c.client.KVv2(c.mountPath)

	var data map[string]any

	operation := func() error {
		secret, err := kv.Get(ctx, secretPath)
		if err != nil {
			if isVaultPermanentError(err) {
				return backoff.Permanent(fmt.Errorf("vault.GetSecret: %w", err))
			}

			return fmt.Errorf("vault.GetSecret: %w", err)
		}

		if secret == nil || secret.Data == nil {
			return backoff.Permanent(fmt.Errorf("vault.GetSecret: secret not found at path: %s/%s", c.mountPath, secretPath))
		}

		data = secret.Data

		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = vaultRetryMaxElapsed
	bo.InitialInterval = vaultRetryInitialInterval

	err := backoff.Retry(operation, backoff.WithContext(bo, ctx))
	if err != nil {
		return nil, err
	}

	return data, nil
}

// isVaultPermanentError returns true for errors that should not be retried.
// Uses two strategies: substring matching on status code strings in the error message,
// and errors.As on *vault.ResponseError to check the HTTP status code directly.
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
	var re *vault.ResponseError
	if errors.As(err, &re) {
		*target = re

		return true
	}

	return false
}

// GetString retrieves a single string value identified by key from the secret at secretPath.
func (c *Client) GetString(ctx context.Context, secretPath, key string) (string, error) {
	data, err := c.GetSecret(ctx, secretPath)
	if err != nil {
		return "", err
	}

	value, ok := data[key].(string)
	if !ok {
		return "", fmt.Errorf("vault.GetString: key %s not found or not a string in path %s/%s", key, c.mountPath, secretPath)
	}

	return value, nil
}
