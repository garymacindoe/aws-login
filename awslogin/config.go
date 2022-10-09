package awslogin

import (
	"os"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

type config struct {
	Accounts map[string]map[string]string
	Aliases  map[string]string
}

func makeConfig(filename string) (config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return config{}, err
	}
	var c config
	err = yaml.Unmarshal(data, &c)
	return c, err
}

func (c config) resolveAccountID(accountName string) (string, error) {
	matched, err := regexp.MatchString("^\\d{12}$", accountName)
	if err != nil {
		return "", err
	}

	if matched {
		return accountName, nil
	}

	accountID, ok := c.Aliases[accountName]
	if !ok {
		return "", &AccountError{accountName, ErrUnknownAccountAlias}
	}
	return accountID, nil
}

func (c config) roleArn(accountID string) (string, error) {
	account, ok := c.Accounts[accountID]
	if ok {
		roleArn, ok := account["role-arn"]
		if ok {
			return roleArn, nil
		}
		return "", &AccountError{accountID, ErrNoRoleArn}
	}
	return "", &AccountError{accountID, ErrUnknownAccountID}
}

func (c config) serialNumber(accountID string) (string, bool, error) {
	account, ok := c.Accounts[accountID]
	if ok {
		serialNumber, ok := account["serial-number"]
		return serialNumber, ok, nil
	}
	return "", false, &AccountError{accountID, ErrUnknownAccountID}
}

func (c config) durationSeconds(accountID string) (int32, bool, error) {
	account, ok := c.Accounts[accountID]
	if ok {
		if durationSeconds, ok := account["duration-seconds"]; ok {
			res, err := strconv.ParseInt(durationSeconds, 10, 32)
			return int32(res), ok, err
		}
		return 0, false, nil
	}
	return 0, false, &AccountError{accountID, ErrUnknownAccountID}
}
