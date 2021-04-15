package ffmpeg

import (
	"time"
)

type Interface interface {
	RecordScreen(filename string, length time.Duration) error
	GetAvailableDevices() (*Devices, error)
	ConvertVideo(filename, newExtension string) (string, error)
	SetDevices(videoDevice string, audioDevice string)
}
