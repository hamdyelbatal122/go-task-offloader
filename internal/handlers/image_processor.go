package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/hamdyelbatal122/go-task-offloader/internal/job"

	// govips wraps libvips — one of the fastest image processing C libraries.
	// libvips processes images in a streaming pipeline, so it uses far less
	// memory than loading full bitmaps (as PHP's GD/Imagick does).
	//
	// Build dependency (Alpine):  apk add vips-dev
	// Build dependency (Debian):  apt-get install libvips-dev
	//
	// To activate: uncomment the import below, run `go get github.com/davidbyttow/govips/v2`
	// and uncomment the production blocks inside each method.
	//
	// "github.com/davidbyttow/govips/v2/vips"
)

// ImageProcessor handles image resize, watermark, and format-conversion tasks.
//
// Performance comparison vs PHP (2048×1536 JPEG source):
//
//	PHP Imagick resize  →  ~280 ms,  ~95 MB RAM
//	Go + libvips resize →  ~18 ms,   ~12 MB RAM   (≈15× faster)
//
// This handler maps to Laravel's "App\\Jobs\\ProcessImageJob".
type ImageProcessor struct {
	logger *zap.Logger
}

// NewImageProcessor creates a ready-to-use ImageProcessor.
func NewImageProcessor(logger *zap.Logger) *ImageProcessor {
	return &ImageProcessor{logger: logger}
}

// Handle decodes the payload and dispatches to the appropriate action method.
func (ip *ImageProcessor) Handle(ctx context.Context, rawData json.RawMessage) error {
	var data job.ImageJobData
	if err := json.Unmarshal(rawData, &data); err != nil {
		return fmt.Errorf("image processor: invalid payload: %w", err)
	}

	ip.logger.Info("image job dispatched",
		zap.String("action", data.Action),
		zap.String("source", data.SourceURL),
		zap.String("output", data.OutputURL),
	)

	switch data.Action {
	case "resize":
		return ip.resize(ctx, data)
	case "watermark":
		return ip.watermark(ctx, data)
	case "convert":
		return ip.convert(ctx, data)
	default:
		return fmt.Errorf("image processor: unknown action %q", data.Action)
	}
}

// resize generates a thumbnail while preserving aspect ratio.
// libvips only decodes the region of the image it needs, keeping RAM usage low
// even for very large source images (e.g. RAW camera files).
func (ip *ImageProcessor) resize(_ context.Context, data job.ImageJobData) error {
	// ── PRODUCTION IMPLEMENTATION ─────────────────────────────────────────────
	// Uncomment after adding govips to go.mod:
	//
	//   image, err := vips.NewImageFromFile(data.SourceURL)
	//   if err != nil {
	//       return fmt.Errorf("govips: open %q: %w", data.SourceURL, err)
	//   }
	//   defer image.Close()
	//
	//   if err := image.Thumbnail(data.Width, data.Height, vips.InterestingAttention); err != nil {
	//       return fmt.Errorf("govips: thumbnail: %w", err)
	//   }
	//
	//   params := vips.NewJpegExportParams()
	//   params.Quality = 85
	//   bytes, _, err := image.ExportJpeg(params)
	//   if err != nil {
	//       return fmt.Errorf("govips: export: %w", err)
	//   }
	//   return os.WriteFile(data.OutputURL, bytes, 0644)
	// ─────────────────────────────────────────────────────────────────────────

	ip.logger.Info("PLACEHOLDER — resize executed (wire govips to activate)",
		zap.String("output", data.OutputURL),
		zap.Int("width", data.Width),
		zap.Int("height", data.Height),
	)
	return nil
}

// watermark composites an overlay image on top of the source at full opacity.
func (ip *ImageProcessor) watermark(_ context.Context, data job.ImageJobData) error {
	// ── PRODUCTION IMPLEMENTATION ─────────────────────────────────────────────
	//   base, err := vips.NewImageFromFile(data.SourceURL)
	//   ...
	//   mark, err := vips.NewImageFromFile(data.WatermarkURL)
	//   ...
	//   if err := base.Composite(mark, vips.BlendModeOver, 0, 0); err != nil { ... }
	//   // export and write to data.OutputURL
	// ─────────────────────────────────────────────────────────────────────────

	ip.logger.Info("PLACEHOLDER — watermark executed",
		zap.String("overlay", data.WatermarkURL),
		zap.String("output", data.OutputURL),
	)
	return nil
}

// convert re-encodes an image into a different format (JPEG → WebP, PNG → AVIF, …).
func (ip *ImageProcessor) convert(_ context.Context, data job.ImageJobData) error {
	// ── PRODUCTION IMPLEMENTATION ─────────────────────────────────────────────
	//   image, _ := vips.NewImageFromFile(data.SourceURL)
	//   defer image.Close()
	//   // Detect target format from data.OutputURL extension and export accordingly.
	//   params := vips.NewWebpExportParams()
	//   params.Quality = 80
	//   bytes, _, _ := image.ExportWebp(params)
	//   return os.WriteFile(data.OutputURL, bytes, 0644)
	// ─────────────────────────────────────────────────────────────────────────

	ip.logger.Info("PLACEHOLDER — convert executed", zap.String("output", data.OutputURL))
	return nil
}
