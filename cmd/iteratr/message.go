package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/session"
	natsserver "github.com/nats-io/nats-server/v2/server"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/spf13/cobra"
)

var messageFlags struct {
	name    string
	dataDir string
}

var messageCmd = &cobra.Command{
	Use:   "message <message>",
	Short: "Send a message to a running session",
	Long: `Send a message to a running session's inbox.

The message will appear in the session's inbox and be included
in the next iteration prompt. The agent can read and acknowledge
messages using the inbox_list and inbox_mark_read tools.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runMessage,
}

func init() {
	messageCmd.Flags().StringVarP(&messageFlags.name, "name", "n", "", "Session name (required)")
	messageCmd.MarkFlagRequired("name")
	messageCmd.Flags().StringVar(&messageFlags.dataDir, "data-dir", ".iteratr", "Data directory for NATS storage")
}

func runMessage(cmd *cobra.Command, args []string) error {
	// Join all args as the message content
	content := strings.Join(args, " ")

	// Get data directory (env override)
	dataDir := os.Getenv("ITERATR_DATA_DIR")
	if dataDir == "" {
		dataDir = messageFlags.dataDir
	}

	// Start embedded NATS server (temporary connection)
	ctx := context.Background()
	ns, nc, store, err := connectToNATS(ctx, dataDir)
	if err != nil {
		return err
	}
	defer func() {
		nc.Close()
		ns.Shutdown()
		ns.WaitForShutdown()
	}()

	// Send message
	_, err = store.InboxAdd(ctx, messageFlags.name, session.InboxAddParams{
		Content: content,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Printf("Message sent to session '%s'\n", messageFlags.name)
	return nil
}

// connectToNATS is a helper to start an embedded NATS server and create a store.
// This is used by commands that need to interact with session data without
// running a full orchestrator loop (e.g., message, gen-template).
func connectToNATS(ctx context.Context, dataDir string) (*natsserver.Server, *natsgo.Conn, *session.Store, error) {
	// Start embedded NATS server
	ns, err := nats.StartEmbeddedNATS(dataDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start NATS: %w", err)
	}

	// Wait for server to be ready
	if !ns.ReadyForConnections(4 * time.Second) {
		ns.Shutdown()
		return nil, nil, nil, fmt.Errorf("NATS server failed to start")
	}

	// Connect to NATS in-process
	nc, err := natsgo.Connect("", natsgo.InProcessServer(ns))
	if err != nil {
		ns.Shutdown()
		return nil, nil, nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		ns.Shutdown()
		return nil, nil, nil, fmt.Errorf("failed to create JetStream: %w", err)
	}

	// Setup stream
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		nc.Close()
		ns.Shutdown()
		return nil, nil, nil, fmt.Errorf("failed to setup stream: %w", err)
	}

	// Create store
	store := session.NewStore(js, stream)

	return ns, nc, store, nil
}
