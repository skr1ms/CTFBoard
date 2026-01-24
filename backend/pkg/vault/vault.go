package vault

import (
	"context"
	"fmt"
	"os"

	vault "github.com/hashicorp/vault/api"
)

type Client struct {
	client    *vault.Client
	mountPath string
}

func New(addr, token string) (*Client, error) {
	return NewWithMount(addr, token, "secret")
}

func NewWithMount(addr, token, mountPath string) (*Client, error) {
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

func (c *Client) GetSecret(secretPath string) (map[string]any, error) {
	kv := c.client.KVv2(c.mountPath)
	secret, err := kv.Get(context.Background(), secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found at path: %s/%s", c.mountPath, secretPath)
	}

	return secret.Data, nil
}

func (c *Client) GetString(secretPath, key string) (string, error) {
	data, err := c.GetSecret(secretPath)
	if err != nil {
		return "", err
	}

	value, ok := data[key].(string)
	if !ok {
		return "", fmt.Errorf("key %s not found or not a string in path %s/%s", key, c.mountPath, secretPath)
	}

	return value, nil
}
