package awslogin

import (
	"bufio"
	"context"
	"encoding/json"
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
	cache  cache
	config config
}

func MakeAWSLogin(cacheDir, configFile string) (AWSLogin, error) {
	config, err := makeConfig(configFile)
	if err != nil {
		return AWSLogin{}, err
	}
	cache := makeCache(cacheDir)
	return AWSLogin{cache, config}, nil
}

func (awsLogin AWSLogin) Credentials(accountID string, durationSeconds int32) (aws.Credentials, bool, error) {
	var err error
	if accountID, err = awsLogin.config.resolveAccountID(accountID); err != nil {
		return aws.Credentials{}, false, err
	}

	var roleArn string
	if roleArn, err = awsLogin.config.roleArn(accountID); err != nil {
		return aws.Credentials{}, false, err
	}

	var assumeRoleInput = sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: aws.String("aws-login"),
	}
	if serialNumber, ok, err := awsLogin.config.serialNumber(accountID); err != nil {
		return aws.Credentials{}, false, err
	} else if ok {
		assumeRoleInput.SerialNumber = &serialNumber
	}
	if durationSeconds == 0 {
		if durationSeconds, ok, err := awsLogin.config.durationSeconds(accountID); err != nil {
			return aws.Credentials{}, false, err
		} else if ok {
			assumeRoleInput.DurationSeconds = &durationSeconds
		}
	} else {
		assumeRoleInput.DurationSeconds = &durationSeconds
	}

	if credentials, ok, err := awsLogin.cache.get(accountID, durationSeconds); err != nil {
		return aws.Credentials{}, false, err
	} else if ok {
		return credentials, true, nil
	}

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithSharedConfigProfile(accountID))
	if err != nil {
		return aws.Credentials{}, false, err
	}

	svc := sts.NewFromConfig(cfg)

	if assumeRoleInput.SerialNumber != nil {
		reader := bufio.NewReader(os.Stdin)
		if _, err := fmt.Fprintln(os.Stderr, "Enter MFA Token:"); err != nil {
			return aws.Credentials{}, false, err
		}
		if tokenCode, err := reader.ReadString('\n'); err != nil {
			return aws.Credentials{}, false, err
		} else {
			assumeRoleInput.TokenCode = aws.String(strings.Trim(tokenCode, "\n"))
		}
	}

	resp, err := svc.AssumeRole(context.TODO(), &assumeRoleInput)

	if err != nil {
		return aws.Credentials{}, false, err
	}

	var credentials = aws.Credentials{
		AccessKeyID:     *resp.Credentials.AccessKeyId,
		SecretAccessKey: *resp.Credentials.SecretAccessKey,
		SessionToken:    *resp.Credentials.SessionToken,
		Source:          *aws.String("aws-login"),
		CanExpire:       true,
		Expires:         *resp.Credentials.Expiration,
	}

	return credentials, false, awsLogin.cache.put(accountID, credentials)
}

func (awsLogin AWSLogin) Console(accountID, destination string, durationSeconds int32) (string, bool, error) {
	var request = "https://signin.aws.amazon.com/federation?Action=getSigninToken"
	var reuse = false

	if durationSeconds == 0 {
		if accountID, err := awsLogin.config.resolveAccountID(accountID); err != nil {
			return "", false, err
		} else {
			if durationSeconds, ok, err := awsLogin.config.durationSeconds(accountID); err != nil {
				return "", false, err
			} else if ok {
				request += "&SessionDuration=" + strconv.FormatInt(int64(durationSeconds), 10)
			}
		}
	} else {
		request += "&SessionDuration=" + strconv.FormatInt(int64(durationSeconds), 10)
	}

	if assumedRole, ok, err := awsLogin.Credentials(accountID, durationSeconds); err != nil {
		return "", false, err
	} else {
		reuse = ok
		session, err := json.Marshal(map[string]string{
			"sessionId":    assumedRole.AccessKeyID,
			"sessionKey":   assumedRole.SecretAccessKey,
			"sessionToken": assumedRole.SessionToken,
		})
		if err != nil {
			return "", false, err
		}
		request += "&Session=" + url.QueryEscape(string(session))
	}

	resp, err := http.Get(request)
	if err != nil {
		return "", false, err
	}

	defer resp.Body.Close()

	var response map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", false, err
	}

	signinToken, ok := response["SigninToken"]
	if !ok {
		return "", false, &ResponseError{"SigninToken", ErrResponseError}
	}

	return "https://signin.aws.amazon.com/federation" +
		"?Action=login" +
		"&SigninToken=" + signinToken.(string) +
		"&Destination=" + url.QueryEscape(destination), reuse, nil
}
