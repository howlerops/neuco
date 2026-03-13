package jobs

import (
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/neuco-ai/neuco/internal/config"
	"github.com/neuco-ai/neuco/internal/store"
)

// riverClient is set after the River client is created in main.
// Workers use this to chain jobs (enqueue the next step in a pipeline).
var (
	riverClientMu sync.RWMutex
	riverClient   *river.Client[pgx.Tx]
)

func SetRiverClient(c *river.Client[pgx.Tx]) {
	riverClientMu.Lock()
	defer riverClientMu.Unlock()
	riverClient = c
}

func getRiverClient() *river.Client[pgx.Tx] {
	riverClientMu.RLock()
	defer riverClientMu.RUnlock()
	return riverClient
}

// RegisterAllWorkers registers all worker types with the River workers registry.
func RegisterAllWorkers(workers *river.Workers, s *store.Store, cfg *config.Config) {
	river.AddWorker(workers, NewIngestWorker(s, cfg))
	river.AddWorker(workers, NewEmbedWorker(s, cfg))
	river.AddWorker(workers, NewFetchSignalsWorker(s))
	river.AddWorker(workers, NewClusterThemesWorker(s))
	river.AddWorker(workers, NewNameThemesWorker(s, cfg))
	river.AddWorker(workers, NewScoreCandidatesWorker(s))
	river.AddWorker(workers, NewWriteCandidatesWorker(s))
	river.AddWorker(workers, NewUpdateContextWorker(s, cfg))
	river.AddWorker(workers, NewSpecGenWorker(s, cfg))
	river.AddWorker(workers, NewFetchSpecWorker(s))
	river.AddWorker(workers, NewIndexRepoWorker(s, cfg))
	river.AddWorker(workers, NewBuildContextWorker(s))
	river.AddWorker(workers, NewGenerateCodeWorker(s, cfg))
	river.AddWorker(workers, NewCreatePRWorker(s, cfg))
	river.AddWorker(workers, NewNotifyWorker(s, cfg))
	river.AddWorker(workers, NewDigestAllProjectsWorker(s))
	river.AddWorker(workers, NewCopilotReviewWorker(s, cfg))
	river.AddWorker(workers, NewNangoSyncWorker(s, cfg))
	river.AddWorker(workers, NewSyncAllIntegrationsWorker(s, cfg))
	river.AddWorker(workers, NewIntercomSyncWorker(s, cfg))
	river.AddWorker(workers, NewSlackSyncWorker(s, cfg))
	river.AddWorker(workers, NewLinearSyncWorker(s, cfg))
	river.AddWorker(workers, NewJiraSyncWorker(s, cfg))
	river.AddWorker(workers, NewSendEmailWorker(s, cfg))
	river.AddWorker(workers, NewDigestEmailsWorker(s, cfg))
}
