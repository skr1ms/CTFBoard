package load_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
)

const (
	loadTestServerIdleTimeout      = 5 * time.Second
	loadTestServerReadinessTimeout = 250 * time.Millisecond
)

func startLoadTestServer(pool *pgxpool.Pool, redisClient *redis.Client) (baseURL string, shutdown func(), err error) {
	deps, err := initLoadTestDeps(redisClient)
	if err != nil {
		return "", nil, err
	}

	repos := initLoadTestRepos(pool)

	storageDir, err := os.MkdirTemp("", "lt-storage-")
	if err != nil {
		return "", nil, fmt.Errorf("create storage dir: %w", err)
	}

	fileStorage, err := storage.NewFilesystemProvider(storageDir)
	if err != nil {
		_ = os.RemoveAll(storageDir)

		return "", nil, fmt.Errorf("create storage: %w", err)
	}

	cleanupStorage := func() {
		_ = fileStorage.Close()
		_ = os.RemoveAll(storageDir)
	}

	ctx, cancelServer := context.WithCancel(context.Background())

	hub := wskit.NewHub(
		wskit.WithRedis(redisClient, "lt:events"),
		wskit.WithOnConnect(func(sub wskit.Subscriber) {
			c, ok := sub.(*wskit.Client)
			if !ok {
				return
			}

			data, err := json.Marshal(wskit.NewEvent(websocket.EventTypeConnected, nil))
			if err == nil {
				c.Send(data)
			}
		}),
	)
	go hub.Run(ctx)
	go hub.SubscribeToRedis(ctx)

	uc := buildLoadTestUseCases(deps, repos, fileStorage, hub, redisClient)
	r := buildLoadTestRouter(ctx, deps.log, uc, deps.val, deps.jwt, storageDir, redisClient)

	ls := net.ListenConfig{}

	listener, err := ls.Listen(ctx, "tcp", ":0")
	if err != nil {
		cancelServer()
		cleanupStorage()

		return "", nil, fmt.Errorf("listen: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	srv := &http.Server{
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 200 * time.Second,
		IdleTimeout:  loadTestServerIdleTimeout,
	}

	go func() {
		err := srv.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("[load-test] server error: %v\n", err)
		}
	}()

	baseURL = fmt.Sprintf("http://localhost:%d", port)

	shutdownServer := func() {
		cancelServer()

		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		serr := srv.Shutdown(shutCtx)
		if serr != nil {
			fmt.Printf("[load-test] shutdown: %v\n", serr)
		}

		cleanupStorage()
	}

	ready := false

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		reqCtx, cancelReq := context.WithTimeout(context.Background(), loadTestServerReadinessTimeout)

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/api/v1/competition/status", http.NoBody)
		if err != nil {
			cancelReq()
			time.Sleep(50 * time.Millisecond)

			continue
		}

		resp, err := http.DefaultClient.Do(req)

		cancelReq()

		if err == nil && resp != nil {
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				ready = true

				break
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	if !ready {
		shutdownServer()

		return "", nil, fmt.Errorf("load-test server readiness: %s did not return OK before deadline", baseURL)
	}

	return baseURL, shutdownServer, nil
}
