package awslogin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type cache struct {
	cacheDirectory string
}

func makeCache(cacheDirectory string) cache {
	return cache{cacheDirectory}
}

func (c cache) get(accountID string, durationSeconds int32) (aws.Credentials, bool, error) {
	var credentials aws.Credentials
	data, err := osReadFile(filepath.Join(c.cacheDirectory, "aws-login-"+accountID+".json"))
	if err != nil {
		if !os.IsNotExist(err) {
			return credentials, false, err
		}
		return credentials, false, nil
	}

	if err := json.Unmarshal(data, &credentials); err != nil {
		return credentials, false, err
	}

	if timeNow().Add(time.Second * time.Duration(durationSeconds)).Before(credentials.Expires) {
		return credentials, true, nil
	}

	return credentials, false, nil
}

func (c cache) put(accountID string, credentials aws.Credentials) error {
	data, err := json.Marshal(credentials)
	if err != nil {
		return err
	}

	return osWriteFile(filepath.Join(c.cacheDirectory, "aws-login-"+accountID+".json"), data, 0600)
}
