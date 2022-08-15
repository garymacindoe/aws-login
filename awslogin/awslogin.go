package awslogin

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AWSLogin struct {
	cache  Cache
	config Config
}

func MakeAWSLogin(cacheDir, configFile string) (AWSLogin, error) {
	config, err := makeConfig(configFile)
	if err != nil {
		return AWSLogin{}, err
	}
	cache := makeCache(cacheDir)
	return AWSLogin{cache, config}, nil
}

func (awsLogin AWSLogin) Credentials(accountID string, durationSeconds int32) (aws.Credentials, error) {
	accountID, err := awsLogin.config.ResolveAccountID(accountID)
	if err != nil {
		return aws.Credentials{}, err
	}

	if durationSeconds == -1 {
		if durationSeconds, err = awsLogin.config.DurationSeconds(accountID); err != nil {
			return aws.Credentials{}, err
		}
		if durationSeconds == -1 {
			durationSeconds = 3600
		}
	}

	credentials, ok, err := awsLogin.cache.Get(accountID, durationSeconds)
	if err != nil {
		return credentials, err
	}
	if ok {
		return credentials, nil
	}

	var roleArn, serialNumber string
	if roleArn, err = awsLogin.config.RoleArn(accountID); err != nil {
		return credentials, nil
	}
	if serialNumber, err = awsLogin.config.SerialNumber(accountID); err != nil {
		return credentials, nil
	}

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithSharedConfigProfile(accountID))
	if err != nil {
		return credentials, err
	}

	svc := sts.NewFromConfig(cfg)

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintln(os.Stderr, "Enter MFA Token:")
	tokenCode, err := reader.ReadString('\n')
	if err != nil {
		return credentials, fmt.Errorf("failed to read MFA token, %v", err)
	}

	resp, err := svc.AssumeRole(context.TODO(), &sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: aws.String("aws-login"),
		DurationSeconds: &durationSeconds,
		SerialNumber:    &serialNumber,
		TokenCode:       aws.String(strings.Trim(tokenCode, "\n")),
	})

	if err != nil {
		return credentials, fmt.Errorf("failed to assume role, %v", err)
	}

	credentials = aws.Credentials{
		AccessKeyID:     *resp.Credentials.AccessKeyId,
		SecretAccessKey: *resp.Credentials.SecretAccessKey,
		SessionToken:    *resp.Credentials.SessionToken,
		Source:          *aws.String("aws-login"),
		CanExpire:       true,
		Expires:         *resp.Credentials.Expiration,
	}

	return credentials, awsLogin.cache.Put(accountID, credentials)
}

func (awsLogin AWSLogin) Console(accountID, destination string, durationSeconds int32) (string, error) {
	assumedRole, err := awsLogin.Credentials(accountID, durationSeconds)
	if err != nil {
		return "", fmt.Errorf("failed to generate credentials, %v", err)
	}

	accountID, err = awsLogin.config.ResolveAccountID(accountID)
	if err != nil {
		return "", err
	}

	if durationSeconds == -1 {
		if durationSeconds, err = awsLogin.config.DurationSeconds(accountID); err != nil {
			return "", err
		}
		if durationSeconds == -1 {
			durationSeconds = 3600
		}
	}

	session, err := json.Marshal(map[string]string{
		"sessionId":    assumedRole.AccessKeyID,
		"sessionKey":   assumedRole.SecretAccessKey,
		"sessionToken": assumedRole.SessionToken,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshall JSON request payload, %v", err)
	}

	resp, err := http.Get("https://signin.aws.amazon.com/federation" +
		"?Action=getSigninToken" +
		"&SessionDuration=" + strconv.FormatInt(int64(durationSeconds), 10) +
		"&Session=" + url.QueryEscape(string(session)))
	if err != nil {
		return "", fmt.Errorf("failed to generate signin token, %v", err)
	}

	defer resp.Body.Close()

	var response map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse signin token, %v", err)
	}

	signinToken, ok := response["SigninToken"]
	if !ok {
		return "", errors.New("SigninToken not included in response")
	}

	return "https://signin.aws.amazon.com/federation" +
		"?Action=login" +
		"&SigninToken=" + signinToken.(string) +
		"&Destination=" + url.QueryEscape(destination), nil
}
