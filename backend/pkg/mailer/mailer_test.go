package mailer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Success(t *testing.T) {
	m := New(Config{APIKey: "key", FromEmail: "a@b.c", FromName: "CTF"})
	require.NotNil(t, m)
	assert.NotNil(t, m.client)
}

func TestResendMailer_Send_Error(t *testing.T) {
	m := New(Config{APIKey: "re_skip", FromEmail: "a@b.c"})
	require.NotNil(t, m)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.Send(ctx, Message{To: "to@example.com", Subject: "s", Body: "b", IsHTML: false})
	require.Error(t, err)
}
