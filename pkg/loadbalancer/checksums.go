package loadbalancer

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"io/fs"
	"os"

	log "github.com/sirupsen/logrus"
)

func checksumsEqual(p1, p2 string) (bool, error) {
	p1CS, p1Err := checksumForFile(p1)
	p2CS, p2Err := checksumForFile(p2)

	if p1Err != nil {
		return false, p1Err
	}

	if p2Err != nil {
		return false, p2Err
	}

	if p1CS == nil || p2CS == nil {
		return false, nil
	}

	return bytes.Equal(p1CS, p2CS), nil
}

func checksumForFile(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return h.Sum(nil), nil
}
