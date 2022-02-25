package awslogin

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	accounts map[string]map[string]string
	aliases  map[string]string
}

type config struct {
	Accounts map[string]map[string]string
	Aliases  map[string]string
}

func makeConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	var config config
	err = yaml.Unmarshal(data, &config)
	return Config{config.Accounts, config.Aliases}, err
}

func (config Config) ResolveAccountID(accountName string) (string, error) {
	matched, err := regexp.MatchString("^\\d{12}$", accountName)
	if err != nil {
		return "", err
	}

	if matched {
		return accountName, nil
	}

	accountID, ok := config.aliases[accountName]
	if !ok {
		return "", fmt.Errorf("unknown account: %s", accountName)
	}
	return accountID, nil
}

func (config Config) RoleArn(accountID string) (string, error) {
	account, ok := config.accounts[accountID]
	if ok {
		roleArn, ok := account["role-arn"]
		if ok {
			return roleArn, nil
		}
		return "", fmt.Errorf("no role-arn configured for %s", accountID)
	}
	return "", fmt.Errorf("unknown account: %s", accountID)
}

func (config Config) SerialNumber(accountID string) (string, error) {
	account, ok := config.accounts[accountID]
	if ok {
		serialNumber := account["serial-number"]
		return serialNumber, nil
	}
	return "", fmt.Errorf("unknown account: %s", accountID)
}

func (config Config) DurationSeconds(accountID string) (int32, error) {
	account, ok := config.accounts[accountID]
	if ok {
		durationSeconds, err := strconv.ParseInt(account["duration-seconds"], 10, 32)
		return int32(durationSeconds), err
	}
	return -1, fmt.Errorf("unknown account: %s", accountID)
}
