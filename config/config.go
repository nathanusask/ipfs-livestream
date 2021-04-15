package config

import "time"

type Config struct {
	FFmpeg         string        `json:"ffmpeg"`
	SamplesPath    string        `json:"samples_path"`
	SampleDuration time.Duration `json:"sample_duration"`
}
