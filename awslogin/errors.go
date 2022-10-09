package awslogin

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownAccountAlias = errors.New("unknown account alias")
	ErrUnknownAccountID    = errors.New("unknown account ID")
	ErrNoRoleArn           = errors.New("no role-arn configured for account ID")
	ErrResponseError       = errors.New("not included in response")
)

type AccountError struct {
	accountName string
	err         error
}

func (e *AccountError) Error() string {
	return fmt.Sprintf("%s: %s", e.err, e.accountName)
}

func (e *AccountError) Unwrap() error {
	return e.err
}

type ResponseError struct {
	field string
	err   error
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("%s %s", e.field, e.err)
}

func (e *ResponseError) Unwrap() error {
	return e.err
}
