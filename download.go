package project02

import (
	"errors"
	"io"
	"net/http"
)

func Download(u string) ([]byte, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	return io.ReadAll(resp.Body)
}
