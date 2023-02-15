package partitions

type Device struct {
	Uevent
	Path       string
	Disk       *Device
	Partitions []*Device
}
