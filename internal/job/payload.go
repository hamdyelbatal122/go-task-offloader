// Package job defines the contract (payload structures) between the Laravel
// application and the Go sidecar worker. Both sides must agree on this schema.
package job

import "encoding/json"

// LaravelPayload is the top-level JSON envelope that Laravel pushes onto the
// Redis queue list. It is compatible with Laravel's default queue payload format.
//
// Sample Laravel dispatch (PHP side):
//
//	ProcessImageJob::dispatch([
//	    'source_url'  => 's3://my-bucket/uploads/photo.jpg',
//	    'output_url'  => 's3://my-bucket/thumbnails/photo_thumb.jpg',
//	    'action'      => 'resize',
//	    'width'       => 800,
//	    'height'      => 600,
//	])->onQueue('image-processing');
type LaravelPayload struct {
	// UUID is a unique identifier for this job instance.
	UUID string `json:"uuid"`

	// DisplayName is the fully-qualified Laravel job class name.
	// The Go worker uses this to route the job to the correct handler.
	// Example: "App\\Jobs\\ProcessImageJob"
	DisplayName string `json:"displayName"`

	// MaxTries mirrors Laravel's $tries property on the job class.
	MaxTries int `json:"maxTries"`

	// Timeout mirrors Laravel's $timeout property (seconds).
	Timeout int `json:"timeout"`

	// Attempts tracks how many times this job has been attempted.
	// Incremented by the Go worker on each retry.
	Attempts int `json:"attempts"`

	// Data contains the job-specific payload. Its structure depends on
	// DisplayName and is decoded by each individual handler.
	Data json.RawMessage `json:"data"`

	// ID is an opaque identifier assigned by Laravel.
	ID string `json:"id"`
}

// ─── Typed Payload Structs ────────────────────────────────────────────────────
// Each handler defines its own typed struct so the "data" field can be
// safely decoded with json.Unmarshal.

// ImageJobData is the typed payload for image processing tasks.
// This maps to "App\\Jobs\\ProcessImageJob" in Laravel.
type ImageJobData struct {
	SourceURL    string `json:"source_url"`    // Input file path or S3 URI
	OutputURL    string `json:"output_url"`    // Destination path or S3 URI
	Action       string `json:"action"`        // "resize" | "watermark" | "convert"
	Width        int    `json:"width"`         // Target width in pixels
	Height       int    `json:"height"`        // Target height in pixels
	WatermarkURL string `json:"watermark_url"` // Overlay image path (watermark action)
}

// DataJobData is the typed payload for heavy data crunching tasks.
// This maps to "App\\Jobs\\CrunchDataJob" in Laravel.
type DataJobData struct {
	InputPath  string            `json:"input_path"`  // Absolute path to source CSV/JSON
	OutputPath string            `json:"output_path"` // Destination file path
	Operation  string            `json:"operation"`   // "aggregate" | "filter"
	Params     map[string]string `json:"params"`      // Operation-specific key/value config
}
