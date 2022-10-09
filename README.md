# AWS Login

AWS Access Keys and Secret Access Keys are required to interact with the AWS CLI.  Using an Access Key and Secret Access Key to interact with the AWS CLI as a user will grant the same privileges to the user as if they were using the AWS console.  It is possible to require that users have MFA to log in to the console, but it is not possible to require users to supply an MFA token to use the CLI.  This means that if a user's Access Key and Secret Access Key is stolen an attacker would be able to use it without having to enter a MFA token.

A solution to this is to grant the user no privileges and create a role that only the user can assume instead.  Grant all privileges to that role and require that MFA is used when assuming the role.  When someone uses the user's Access Key to interact with the AWS CLI they will have no permissions until they supply a MFA token to assume the role.

This utility automates assuming the role, requesting MFA tokens when required, and can additionally print out a federated URL to access the console using the role.

## Building

Building requires `go >= 1.15`.

```
go build -o aws-login .
```

## Installation

Copy `aws-login` to `/usr/local/bin` to install for all users or to `${HOME}/.local/bin` to install for the current user.

## Setup

1. Create an **IAM User** in the AWS console.  Make sure the user has **AWS Management Console access** _disabled_ and **Programmatic access** _enabled_ and do not attach any permissions policies to the user.
1. Download the access keys for the user and copy them into `~/.aws/credentials` under a profile with the AWS account number:
    ```
    [123456789012]
    aws_access_key_id = AKIA................
    aws_secret_access_key = .........................................
    ```
1. Edit the **Security Credentials** for the user and **Assign an MFA device**.
1. Create an **IAM Role** and attach the relevant permissions policies to the new role.
1. Edit the **Trust Relationship** for the role to allow the user to assume the role and require MFA to assume the role:
    ```
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "AWS": "arn:aws:iam::123456789012:user/test"
          },
          "Action": "sts:AssumeRole",
          "Condition": {
            "Bool": {
              "aws:MultiFactorAuthPresent": "true"
            }
          }
        }
      ]
    }
    ```
1. Optionally set the environment variable `AWS_CONFIG_DIR` to a directory to store the credentials and configuration in.  The default is `${HOME}/.aws`.
1. Create a configuration file `${AWS_CONFIG_DIR}/aws-login.yaml` containing the ARN of the role to assume and the ARN of the MFA device.
    ```
    accounts:
      123456789012:
        role-arn: arn:aws:iam::123456789012:role/test
        serial-number: arn:aws:iam::123456789012:mfa/test
    aliases:
      accountname: 123456789012
    ```

## Usage

Having followed the setup and installation instructions above, it should now be possible to run `aws-login`:
```
aws-login [options] <account-id-or-alias> -- [cmd [args...]]

  -h, --help                             print this help and exit

  --cache-directory <cache-directory>    use <cache-directory> to cache
                                         credentials (default ${AWS_CONFIG_DIR})

  --config-file <config-file>            read account IDs, role ARNs and MFA
                                         device serial numbers from
                                         <config-file> (default
                                         ${AWS_CONFIG_DIR}/aws-login.yaml)

  --console                              print a link to the AWS console on
                                         standard output
  
  --destination                          the AWS console URL to redirect to
                                         after authenticating (default
                                         https://console.aws.amazon.com/)

  --duration-seconds <duration-seconds>  request credentials valid for at least
                                         <duration-seconds> seconds (default
                                         3600 seconds or 1 hour)

  <account-id-or-alias>                  an account alias or 12-digit account ID

  [cmd [args...]]                        run cmd using the acquired credentials

If cmd is not present and --console is not specified, credentials will be
printed on standard output.
```

Example to run the AWS CLI:
```
user@localhost ~ $ aws-login accountname aws sts get-caller-identity
Enter MFA token: 123456
{
    "UserId": "AROAUSSL6QESRO52QDIDY:aws-login",
    "Account": "123456789012",
    "Arn": "arn:aws:sts::123456789012:assumed-role/test/aws-login"
}
```

Example to open a link to the console:
```
user@localhost ~ $ chromium-browser $(aws-login accountname --console)
Enter MFA token: 123456
```

Credentials are cached in `${AWS_CONFIG_DIR}` and will be reused if they will not have expired before the requested duration is up (specified on the command line as `--duration-seconds` or in `aws-login.yaml` as `duration-seconds` under an account).  This means that if `aws-login` is run to get credentials valid for one hour (the default) then immediately re-run, the cached credentials will not be valid for a full hour from when the command is run a second time so new credentials will be requested along with requiring a new MFA token to be entered.

This can be worked around by running the command to request long-lived credentials that can be cached, and have repeated commands re-use those credentials for a shorter duration:
```
user@localhost ~ $ aws-login --duration-seconds 43200 accountname aws sts get-caller-identity  # Request credentials valid for the next 12 hours
Enter MFA token: 123456
{
    "UserId": "AROAUSSL6QESRO52QDIDY:aws-login",
    "Account": "123456789012",
    "Arn": "arn:aws:sts::123456789012:assumed-role/test/aws-login"
}
user@localhost ~ $ aws-login accountname aws sts get-caller-identity # Credentials valid for at least an hour found in cache
{
    "UserId": "AROAUSSL6QESRO52QDIDY:aws-login",
    "Account": "123456789012",
    "Arn": "arn:aws:sts::123456789012:assumed-role/test/aws-login"
}
```

## Recovery

If a user no longer has access to their configured MFA device then the following steps should be followed to regain access to their account.

1. Have another user in the account remove the MFA device from the **IAM User** account and remove the `Condition` block from the **Trust Relationship** for the corresponding **IAM Role**.

1. Comment out the `serial_number` configured for the account in `${AWS_CONFIG_DIR}/aws-login.yaml`.

1. Run `aws-login` to log into the console.  The user should not be prompted for an MFA token.

1. Reconfigure the MFA device for the **IAM USER** as above and add the `Condition` block back to the **Trust Relationship** for the **IAM Role**.

1. Uncomment the `serial_number` configured for the account in `${AWS_CONFIG_DIR}/aws-login.yaml`.

## Contributing

Bug reports and pull requests are welcome on GitHub at https://github.com/garymacindoe/aws-login.


## License

This repository is available as open source under the terms of the [LGPL v3](https://opensource.org/licenses/LGPL-3.0).
