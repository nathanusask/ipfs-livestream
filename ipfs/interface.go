package ipfs

import shell "github.com/ipfs/go-ipfs-api"

type Interface interface {
	PublishName(name string) error
	PushFolder(path string) error
	PushFile(path string) (string, error)
	GetId() (*shell.IdOutput, error)
	ClearBootstrapList() error
	SetBootstrapList(list []string) error
}
