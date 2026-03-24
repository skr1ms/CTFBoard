package mailer_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	mailerMock "github.com/TakuyaYagam1/AstroCTFb/pkg/mailer/mock"
)

func TestAsyncMailer(t *testing.T) {
	t.Parallel()
	mockMailer := mailerMock.NewMockMailer(t)
	l, err := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)
	asyncMailer := mailer.NewAsyncMailer(mockMailer, 10, 1, l)
	asyncMailer.Start()
	defer asyncMailer.Stop()

	msg := mailer.Message{To: "test@example.com", Subject: "Test", Body: "Body"}
	done := make(chan struct{})

	mockMailer.On("Send", mock.Anything, mock.MatchedBy(func(m mailer.Message) bool {
		return m.To == "test@example.com" && m.Subject == "Test" && m.Body == "Body"
	})).Return(nil).Once().Run(func(mock.Arguments) { close(done) })

	err = asyncMailer.Send(context.Background(), msg)
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for mailer Send")
	}

	asyncMailer.Stop()
	err = asyncMailer.Send(context.Background(), msg)
	assert.ErrorIs(t, err, mailer.ErrMailerStopped)
}

func TestAsyncMailer_GracefulShutdown(t *testing.T) {
	t.Parallel()
	mockMailer := mailerMock.NewMockMailer(t)
	l, err := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)
	asyncMailer := mailer.NewAsyncMailer(mockMailer, 10, 1, l)
	asyncMailer.Start()
	defer asyncMailer.Stop()

	msg := mailer.Message{To: "drain@example.com", Subject: "Drain", Body: "Body"}

	sendCalled := make(chan struct{})
	mockMailer.On("Send", mock.Anything, mock.MatchedBy(func(m mailer.Message) bool {
		return m.To == "drain@example.com"
	})).Return(nil).Once().Run(func(mock.Arguments) { close(sendCalled) })

	err = asyncMailer.Send(context.Background(), msg)
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		asyncMailer.Stop()
		close(done)
	}()

	select {
	case <-sendCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for mailer to drain message")
	}
	<-done
}
