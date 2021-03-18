package livestream

type Interface interface {
	UseDefaultDevices() error
	SetDevices(videoDevice, audioDevice string)
	Watch(ipnsAddress string) error
	Broadcast(samples int) error
}
