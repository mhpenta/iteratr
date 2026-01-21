package nats

import (
	"errors"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StartEmbeddedNATS starts an embedded NATS server with JetStream enabled
// using the specified data directory for file-based storage.
// Returns the server instance or an error if startup fails.
func StartEmbeddedNATS(dataDir string) (*server.Server, error) {
	opts := &server.Options{
		JetStream:  true,
		StoreDir:   dataDir,
		DontListen: true, // No network ports - in-process only
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, err
	}

	// Start server in background goroutine
	go ns.Start()

	// Wait for server to be ready with timeout
	if !ns.ReadyForConnections(4 * time.Second) {
		return nil, errors.New("nats server failed to start within timeout")
	}

	return ns, nil
}

// ConnectInProcess creates an in-process connection to the embedded NATS server.
// This connection does not use network ports and communicates directly with the server.
func ConnectInProcess(ns *server.Server) (*nats.Conn, error) {
	return nats.Connect("", nats.InProcessServer(ns))
}

// CreateJetStream creates a JetStream context from a NATS connection.
// This context is used for all JetStream operations including creating streams,
// consumers, and publishing/subscribing to subjects.
func CreateJetStream(nc *nats.Conn) (jetstream.JetStream, error) {
	return jetstream.New(nc)
}

// Shutdown gracefully shuts down the NATS connection and server.
// It first drains and closes the connection, then shuts down the server
// with a timeout to allow in-flight operations to complete.
func Shutdown(nc *nats.Conn, ns *server.Server) error {
	// Close the connection first (drain buffered messages)
	if nc != nil {
		// Drain waits for published messages to be acknowledged
		// and subscriptions to complete before closing
		// Use a timeout for drain to prevent hanging
		drainDone := make(chan error, 1)
		go func() {
			drainDone <- nc.Drain()
		}()

		select {
		case err := <-drainDone:
			if err != nil {
				// Drain failed, force close
				nc.Close()
			}
		case <-time.After(2 * time.Second):
			// Drain timed out, force close
			nc.Close()
		}
	}

	// Shutdown the server with a grace period
	if ns != nil {
		ns.Shutdown()

		// WaitForShutdown with timeout to prevent hanging
		shutdownDone := make(chan struct{})
		go func() {
			ns.WaitForShutdown()
			close(shutdownDone)
		}()

		select {
		case <-shutdownDone:
			// Server shut down cleanly
		case <-time.After(5 * time.Second):
			// Shutdown timed out - force stop
			// Note: There's no force-stop API, but at least we don't hang forever
			return errors.New("NATS server shutdown timed out")
		}
	}

	return nil
}
