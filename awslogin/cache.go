package awslogin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type Cache struct {
	cacheDirectory string
}

func makeCache(cacheDirectory string) Cache {
	return Cache{cacheDirectory}
}

func (cache Cache) Get(accountID string, durationSeconds int32) (aws.Credentials, bool, error) {
	var credentials aws.Credentials
	data, err := os.ReadFile(filepath.Join(cache.cacheDirectory, "aws-login-"+accountID+".json"))
	if err != nil && !os.IsNotExist(err) {
		return credentials, false, err
	}

	if err := json.Unmarshal(data, &credentials); err != nil {
		return credentials, false, err
	}

	if time.Now().Add(time.Second * time.Duration(durationSeconds)).Before(credentials.Expires) {
		return credentials, true, nil
	}

	return credentials, false, nil
}

func (cache Cache) Put(accountID string, credentials aws.Credentials) error {
	data, err := json.Marshal(credentials)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(cache.cacheDirectory, "aws-login-"+accountID+".json"), data, 0600)
}
