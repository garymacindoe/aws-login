package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/pborman/getopt/v2"
)

func main() {
	var console = false
	var destination = "https://console.aws.amazon.com/"
	var sessionDuration int32 = 0
	var help = false

	getopt.FlagLong(&console, "console", 0, "print a link to the AWS console on standard output")
	getopt.FlagLong(&destination, "destination", 0, "the AWS Console URL to redirect to after authenticating")
	getopt.FlagLong(&sessionDuration, "session-duration", 0, "duration of the console session")
	getopt.FlagLong(&help, "help", 'h', "print this help and exit")

	getopt.Parse()
	args := getopt.Args()

	if help {
		getopt.Usage()
		os.Exit(1)
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "missing account ID or alias")
		os.Exit(1)
	}

	accountID := args[0]

	cfg, err := awsConfig.LoadDefaultConfig(
		context.TODO(),
		awsConfig.WithSharedConfigProfile(accountID),
		awsConfig.WithAssumeRoleCredentialOptions(func(o *stscreds.AssumeRoleOptions) {
			o.TokenProvider = stdinTokenProvider
		}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to get credentials: %v\n", err)
		os.Exit(1)
	}
	var credentials aws.Credentials
	if credentials, err = cfg.Credentials.Retrieve(context.TODO()); err != nil {
		fmt.Fprintf(os.Stderr, "unable to get credentials: %v\n", err)
		os.Exit(1)
	}

	if console {
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_enable-console-custom-url.html
		var request = "https://signin.aws.amazon.com/federation?Action=getSigninToken"

		session, err := json.Marshal(map[string]string{
			"sessionId":    credentials.AccessKeyID,
			"sessionKey":   credentials.SecretAccessKey,
			"sessionToken": credentials.SessionToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal session: %v\n", err)
			os.Exit(1)
		}
		request += "&Session=" + url.QueryEscape(string(session))

		if sessionDuration > 0 {
			request += fmt.Sprintf("&SessionDuration=%d", sessionDuration)
		}

		resp, err := http.Get(request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch URL: %v\n", err)
			os.Exit(1)
		}

		defer resp.Body.Close()

		var response map[string]interface{}
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to decode response: %v\n", err)
			os.Exit(1)
		}

		signinToken, ok := response["SigninToken"]
		if !ok {
			fmt.Fprintln(os.Stderr, "Response does not contain SigninToken")
			os.Exit(1)
		}

		fmt.Println("https://signin.aws.amazon.com/federation" +
			"?Action=login" +
			"&Issuer=aws-login" +
			"&SigninToken=" + signinToken.(string) +
			"&Destination=" + url.QueryEscape(destination))
		os.Exit(0)
	}

	env := map[string]string{
		"AWS_ACCESS_KEY_ID":     credentials.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY": credentials.SecretAccessKey,
		"AWS_SESSION_TOKEN":     credentials.SessionToken,
	}

	if len(args) == 1 {
		for k, v := range env {
			fmt.Printf("export %s=\"%s\"\n", k, v)
		}
	} else {
		cmd := exec.Command(args[1], args[2:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(),
			"AWS_ACCESS_KEY_ID="+credentials.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY="+credentials.SecretAccessKey,
			"AWS_SESSION_TOKEN="+credentials.SessionToken)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "unable to run command: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func stdinTokenProvider() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	// Enter MFA code for arn:aws:iam::032368440683:mfa/gary.macindoe@bbc.co.uk:
	if _, err := fmt.Fprintln(os.Stderr, "Enter MFA Token:"); err != nil {
		return "", err
	}
	tokenCode, err := reader.ReadString('\n')
	return strings.Trim(tokenCode, "\n"), err
}
