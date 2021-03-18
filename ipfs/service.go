package ipfs

import (
	"log"
	"os"

	shell "github.com/ipfs/go-ipfs-api"
)

type service struct {
	shell *shell.Shell
}

func New() Interface {
	return &service{
		shell: shell.NewLocalShell(),
	}
}

func (s *service) PublishName(name string) error {
	return s.shell.Publish("", name)
}

func (s *service) PushFolder(path string) error {
	cid, err := s.shell.AddDir(path)
	if err != nil {
		log.Println("Error adding directory ", path, " with error ", err)
		return err
	}
	return s.shell.Publish("", cid)
}

func (s *service) PushFile(path string) (string, error) {
	reader, err := os.Open(path)
	defer func() {
		err := reader.Close()
		if err != nil {
			log.Println("Failed to close file ", path, " with error: ", err)
		}
	}()
	if err != nil {
		log.Println("Failed to open file ", path, " with error: ", err)
		return "", err
	}
	cid, err := s.shell.Add(reader)
	if err != nil {
		log.Println("Failed to add file ", path, " to IPFS with error: ", err)
		return "", err
	}
	return cid, nil
}

func (s *service) GetId() (*shell.IdOutput, error) {
	return s.shell.ID()
}

func (s *service) ClearBootstrapList() error {
	_, err := s.shell.BootstrapRmAll()
	return err
}

func (s *service) SetBootstrapList(list []string) error {
	err := s.ClearBootstrapList()
	if err != nil {
		log.Println("Failed to clear bootstrap list with error: ", err)
		return err
	}
	_, err = s.shell.BootstrapAdd(list)
	return err
}
