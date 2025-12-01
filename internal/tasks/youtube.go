package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/youtube"
)

// UploadToYouTubeArgs contains the arguments for uploading a video to YouTube
type UploadToYouTubeArgs struct {
	Input ArchiveVideoInput `json:"input"`
}

func (UploadToYouTubeArgs) Kind() string { return "upload_to_youtube" }

func (args UploadToYouTubeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 3,
		Queue:       "default",
		Tags:        []string{"youtube", "upload"},
	}
}

func (w UploadToYouTubeArgs) Timeout(job *river.Job[UploadToYouTubeArgs]) time.Duration {
	return 12 * time.Hour // YouTube uploads can take a long time
}

type UploadToYouTubeWorker struct {
	river.WorkerDefaults[UploadToYouTubeArgs]
}

func (w UploadToYouTubeWorker) Work(ctx context.Context, job *river.Job[UploadToYouTubeArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting YouTube upload task")

	// Get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// Start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

	// Get database items
	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// Create YouTube service
	youtubeService := youtube.NewService(store)

	// Upload video
	err = youtubeService.UploadVideo(ctx, dbItems.Video.ID.String())
	if err != nil {
		logger.Error().Err(err).Msg("YouTube upload failed")
		return err
	}

	logger.Info().Msg("YouTube upload completed successfully")
	return nil
}
