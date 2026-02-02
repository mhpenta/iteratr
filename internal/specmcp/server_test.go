package specmcp

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	specDir := "/tmp/test-specs"
	srv := New(specDir)

	require.NotNil(t, srv)
	assert.Equal(t, specDir, srv.specDir)
	assert.NotNil(t, srv.questionCh, "questionCh should be initialized")
	assert.Nil(t, srv.mcpServer, "mcpServer should be nil before Start")
	assert.Nil(t, srv.httpServer, "httpServer should be nil before Start")
	assert.Equal(t, 0, srv.port, "port should be 0 before Start")
}

func TestStart_Success(t *testing.T) {
	ctx := context.Background()
	srv := New(t.TempDir())

	port, err := srv.Start(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, srv.Stop())
	}()

	// Verify port is valid
	assert.Greater(t, port, 0, "port should be positive")
	assert.LessOrEqual(t, port, 65535, "port should be in valid range")
	assert.Equal(t, port, srv.port, "port should match srv.port")

	// Verify servers are initialized
	require.NotNil(t, srv.mcpServer, "mcpServer should be initialized")
	require.NotNil(t, srv.httpServer, "httpServer should be initialized")

	// Verify URL is correct
	expectedURL := fmt.Sprintf("http://localhost:%d/mcp", port)
	assert.Equal(t, expectedURL, srv.URL())
}

func TestStart_AlreadyStarted(t *testing.T) {
	ctx := context.Background()
	srv := New(t.TempDir())

	// Start server
	port1, err := srv.Start(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, srv.Stop())
	}()

	// Try to start again
	port2, err := srv.Start(ctx)
	assert.Error(t, err, "starting already-started server should fail")
	assert.Contains(t, err.Error(), "already started")
	assert.Equal(t, 0, port2, "port should be 0 on error")
	assert.Equal(t, port1, srv.port, "original port should be unchanged")
}

func TestStart_MultipleServers(t *testing.T) {
	ctx := context.Background()

	// Start multiple servers to ensure they get different ports
	srv1 := New(t.TempDir())
	port1, err := srv1.Start(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, srv1.Stop())
	}()

	srv2 := New(t.TempDir())
	port2, err := srv2.Start(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, srv2.Stop())
	}()

	// Verify different ports
	assert.NotEqual(t, port1, port2, "servers should get different ports")

	// Verify both are valid
	assert.Greater(t, port1, 0)
	assert.Greater(t, port2, 0)
}

func TestStop_NotStarted(t *testing.T) {
	srv := New(t.TempDir())

	// Stop should be idempotent and safe even if never started
	err := srv.Stop()
	assert.NoError(t, err, "Stop on unstarted server should succeed")
}

func TestStop_Success(t *testing.T) {
	ctx := context.Background()
	srv := New(t.TempDir())

	port, err := srv.Start(ctx)
	require.NoError(t, err)
	require.Greater(t, port, 0)

	// Verify server is running by checking if port is listening
	url := fmt.Sprintf("http://localhost:%d/mcp", port)
	resp, err := http.Get(url)
	if err == nil {
		_ = resp.Body.Close()
	}
	// We don't care about the response, just that the server is reachable

	// Stop the server
	err = srv.Stop()
	assert.NoError(t, err)

	// Verify cleanup
	assert.Nil(t, srv.httpServer, "httpServer should be nil after Stop")
	assert.Nil(t, srv.mcpServer, "mcpServer should be nil after Stop")

	// Verify server is no longer listening
	time.Sleep(50 * time.Millisecond) // Give server time to shut down
	_, err = http.Get(url)
	assert.Error(t, err, "server should not be reachable after Stop")
}

func TestStop_Idempotent(t *testing.T) {
	ctx := context.Background()
	srv := New(t.TempDir())

	// Start server
	_, err := srv.Start(ctx)
	require.NoError(t, err)

	// Stop once
	err = srv.Stop()
	assert.NoError(t, err)

	// Stop again should be safe
	err = srv.Stop()
	assert.NoError(t, err, "Stop should be idempotent")
}

func TestStop_MultipleConcurrent(t *testing.T) {
	ctx := context.Background()
	srv := New(t.TempDir())

	_, err := srv.Start(ctx)
	require.NoError(t, err)

	// Call Stop concurrently from multiple goroutines
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			done <- srv.Stop()
		}()
	}

	// All should succeed (idempotent)
	for i := 0; i < 3; i++ {
		err := <-done
		assert.NoError(t, err, "concurrent Stop calls should succeed")
	}
}

func TestURL_BeforeStart(t *testing.T) {
	srv := New(t.TempDir())

	// URL before start should return port 0
	url := srv.URL()
	assert.Equal(t, "http://localhost:0/mcp", url)
}

func TestURL_AfterStart(t *testing.T) {
	ctx := context.Background()
	srv := New(t.TempDir())

	port, err := srv.Start(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, srv.Stop())
	}()

	url := srv.URL()
	expectedURL := fmt.Sprintf("http://localhost:%d/mcp", port)
	assert.Equal(t, expectedURL, url)
}

func TestURL_AfterStop(t *testing.T) {
	ctx := context.Background()
	srv := New(t.TempDir())

	port, err := srv.Start(ctx)
	require.NoError(t, err)
	expectedURL := fmt.Sprintf("http://localhost:%d/mcp", port)

	err = srv.Stop()
	require.NoError(t, err)

	// URL should still return the port that was used
	// (port field is not cleared on Stop)
	url := srv.URL()
	assert.Equal(t, expectedURL, url)
}

func TestQuestionChannel(t *testing.T) {
	srv := New(t.TempDir())

	ch := srv.QuestionChannel()
	require.NotNil(t, ch, "QuestionChannel should not be nil")

	// Verify it returns a read-only channel
	// (We can't directly compare types, but we can verify it works)
	select {
	case <-ch:
		t.Fatal("channel should be empty")
	default:
		// Expected - channel is empty
	}
}

func TestQuestionChannel_SendReceive(t *testing.T) {
	srv := New(t.TempDir())
	ch := srv.QuestionChannel()

	// Send a question in a goroutine
	testQuestion := &QuestionRequest{
		Questions: []*Question{
			{
				Question: "Test question?",
				Header:   "Test",
				Options:  []QuestionOption{{Label: "A", Description: "Option A"}},
				Multiple: false,
			},
		},
		AnswerCh: make(chan<- []any),
	}

	done := make(chan bool)
	go func() {
		srv.questionCh <- testQuestion
		done <- true
	}()

	// Receive from channel
	select {
	case received := <-ch:
		assert.Equal(t, testQuestion, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for question")
	}

	<-done
}

func TestStartStop_Lifecycle(t *testing.T) {
	ctx := context.Background()
	srv := New(t.TempDir())

	// Start -> Stop -> Start -> Stop cycle
	for i := 0; i < 2; i++ {
		t.Run(fmt.Sprintf("cycle_%d", i), func(t *testing.T) {
			// Start
			port, err := srv.Start(ctx)
			require.NoError(t, err)
			require.Greater(t, port, 0)

			// Verify running
			require.NotNil(t, srv.httpServer)
			require.NotNil(t, srv.mcpServer)

			// Stop
			err = srv.Stop()
			require.NoError(t, err)

			// Verify stopped
			assert.Nil(t, srv.httpServer)
			assert.Nil(t, srv.mcpServer)
		})
	}
}
