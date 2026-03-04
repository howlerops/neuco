package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/neuco-ai/neuco/internal/config"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/nango"
	"github.com/neuco-ai/neuco/internal/store"
)

// NangoSyncWorker is a River worker that fetches data from a Nango-connected
// integration and inserts the extracted signals into the database.
//
// The provider field in NangoSyncJobArgs determines which sync method is
// invoked. Providers without a dedicated method fall back to SyncGeneric,
// which stores the raw API response as a single "event" signal.
type NangoSyncWorker struct {
	river.WorkerDefaults[NangoSyncJobArgs]
	store *store.Store
	cfg   *config.Config
}

// NewNangoSyncWorker constructs a NangoSyncWorker.
func NewNangoSyncWorker(s *store.Store, cfg *config.Config) *NangoSyncWorker {
	return &NangoSyncWorker{store: s, cfg: cfg}
}

// Work implements river.Worker[NangoSyncJobArgs].
func (w *NangoSyncWorker) Work(ctx context.Context, job *river.Job[NangoSyncJobArgs]) error {
	start := time.Now()
	args := job.Args

	StartTask(ctx, w.store, args.TaskID)

	slog.Info("nango_sync: starting",
		"project_id", args.ProjectID,
		"provider", args.Provider,
		"connection_id", args.ConnectionID,
		"integration_id", args.IntegrationID,
	)

	nc := nango.NewClient(w.cfg.NangoServerURL, w.cfg.NangoSecretKey)
	svc := nango.NewSyncService(nc, w.store)

	signals, err := w.fetchSignals(ctx, svc, args)
	if err != nil {
		slog.Error("nango_sync: fetch failed",
			"provider", args.Provider,
			"connection_id", args.ConnectionID,
			"error", err,
		)
		FailTask(ctx, w.store, args.TaskID, err)
		CheckPipelineCompletion(ctx, w.store, args.RunID)
		return fmt.Errorf("nango_sync: fetch signals: %w", err)
	}

	// Insert each signal into the DB.
	insertedCount := 0
	for i := range signals {
		if _, insertErr := w.store.InsertSignal(ctx, signals[i]); insertErr != nil {
			slog.Error("nango_sync: insert signal failed",
				"provider", args.Provider,
				"source_ref", signals[i].SourceRef,
				"error", insertErr,
			)
			// Continue — partial success is better than aborting.
			continue
		}
		insertedCount++
	}

	slog.Info("nango_sync: signals inserted",
		"provider", args.Provider,
		"connection_id", args.ConnectionID,
		"total_fetched", len(signals),
		"total_inserted", insertedCount,
	)

	// Stamp last_sync_at on the integration record.
	if args.IntegrationID.String() != "00000000-0000-0000-0000-000000000000" {
		if err := w.store.UpdateIntegrationLastSync(ctx, args.ProjectID, args.IntegrationID, time.Now().UTC()); err != nil {
			slog.Warn("nango_sync: failed to update last_sync_at",
				"integration_id", args.IntegrationID,
				"error", err,
			)
		}
	}

	// Chain an embedding job so the newly inserted signals get vector
	// embeddings without blocking the sync worker.
	if insertedCount > 0 {
		w.enqueueEmbed(ctx, args)
	}

	CompleteTask(ctx, w.store, args.TaskID, start)
	CheckPipelineCompletion(ctx, w.store, args.RunID)

	return nil
}

// fetchSignals dispatches to the correct provider-specific sync method.
func (w *NangoSyncWorker) fetchSignals(
	ctx context.Context,
	svc *nango.SyncService,
	args NangoSyncJobArgs,
) ([]domain.Signal, error) {
	switch args.Provider {
	case "gong":
		return svc.SyncGong(ctx, args.ConnectionID, args.ProjectID)
	case "intercom":
		return svc.SyncIntercom(ctx, args.ConnectionID, args.ProjectID)
	case "slack":
		return svc.SyncSlack(ctx, args.ConnectionID, args.ProjectID)
	default:
		slog.Info("nango_sync: no dedicated sync for provider, using generic",
			"provider", args.Provider,
		)
		return svc.SyncGeneric(ctx, args.Provider, args.ConnectionID, args.ProjectID)
	}
}

// enqueueEmbed inserts an EmbedJob into River so the freshly ingested signals
// receive vector embeddings asynchronously. It is best-effort: failures are
// logged but do not prevent the sync job from completing successfully.
func (w *NangoSyncWorker) enqueueEmbed(ctx context.Context, args NangoSyncJobArgs) {
	client := getRiverClient()
	if client == nil {
		slog.Warn("nango_sync: river client not available, skipping embed enqueue")
		return
	}

	embedArgs := EmbedJobArgs{
		ProjectID: args.ProjectID,
		RunID:     args.RunID,
		// SignalIDs is empty — EmbedWorker will pick up all unembedded signals
		// for the project, which includes the ones we just inserted.
	}

	// Resolve the embed task ID from the pipeline run when one exists.
	if args.RunID.String() != "00000000-0000-0000-0000-000000000000" {
		run, err := w.store.GetPipelineRun(ctx, args.RunID)
		if err == nil {
			for _, t := range run.Tasks {
				if t.Name == "embed" {
					embedArgs.TaskID = t.ID
					break
				}
			}
		}
	}

	if _, err := client.Insert(ctx, embedArgs, nil); err != nil {
		slog.Warn("nango_sync: failed to enqueue embed job", "error", err)
	}
}
