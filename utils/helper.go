package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

func HashMD5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.Mkdir(dir, os.ModePerm)
	} else {
		return RemoveContents(dir)
	}
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func IpnsDownloadFile(peerId, newFilename string) (err error) {
	out, err := os.Create(newFilename)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get("http://127.0.0.1:8080/ipns/" + peerId)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to download the file, response code " + strconv.Itoa(resp.StatusCode))
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
