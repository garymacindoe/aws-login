package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/garymacindoe/aws-login/awslogin"
	"github.com/pborman/getopt/v2"
)

func main() {
	var awsConfigDir, set = os.LookupEnv("AWS_CONFIG_DIR")
	if !set {
		awsConfigDir = filepath.Join(os.Getenv("HOME"), ".aws")
	}
	var cacheDirectory = awsConfigDir
	var configFile = filepath.Join(awsConfigDir, "aws-login.yaml")
	var console = false
	var destination = "https://console.aws.amazon.com/"
	var help = false
	var durationSeconds int32 = 3600

	getopt.FlagLong(&cacheDirectory, "cache-directory", 0, "use <cache-directory> to cache credentials (can be overridden by setting ${AWS_CONFIG_DIR})")
	getopt.FlagLong(&configFile, "config-file", 0, "read account IDs, role ARNs and MFA device serial numbers from <config-file> (default ${AWS_CONFIG_DIR}/aws-login.yaml)")
	getopt.FlagLong(&console, "console", 0, "print a link to the AWS console on standard output")
	getopt.FlagLong(&destination, "destination", 0, "the AWS Console URL to redirect to after authenticating")
	getopt.FlagLong(&help, "help", 'h', "print this help and exit")
	getopt.FlagLong(&durationSeconds, "duration-seconds", 0, "request credentials valid for at least <duration-seconds> seconds")

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

	awsLogin, err := awslogin.MakeAWSLogin(cacheDirectory, configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create AWSLogin: %v\n", err)
		os.Exit(1)
	}

	if console {
		url, _, err := awsLogin.Console(accountID, destination, durationSeconds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to generate console link: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(url)
		os.Exit(0)
	}

	credentials, _, err := awsLogin.Credentials(accountID, durationSeconds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to get credentials: %v\n", err)
		os.Exit(1)
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
