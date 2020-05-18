package util

import (
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Match the error to be already exist
type AlreadyExistMatcher struct {
}

func (matcher AlreadyExistMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}
	actualError := actual.(error)
	return apierrors.IsAlreadyExists(actualError), nil
}

func (matcher AlreadyExistMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to be already exist")
}

func (matcher AlreadyExistMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to be already exist")
}

// Match the error to be not found
type NotFoundMatcher struct {
}

func (matcher NotFoundMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, nil
	}
	actualError := actual.(error)
	return apierrors.IsNotFound(actualError), nil
}

func (matcher NotFoundMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to be already exist")
}

func (matcher NotFoundMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to be already exist")
}

// Match the error to error to take care of nil
func BeEquivalentToError(expected error) types.GomegaMatcher {
	return &ErrorMatcher{
		ExpectedError: expected,
	}
}

type ErrorMatcher struct {
	ExpectedError error
}

func (matcher ErrorMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return matcher.ExpectedError == nil, nil
	}
	actualError := actual.(error)
	return actualError.Error() == matcher.ExpectedError.Error(), nil
}

func (matcher ErrorMatcher) FailureMessage(actual interface{}) (message string) {
	actualError, actualOK := actual.(error)
	expectedError, expectedOK := matcher.ExpectedError.(error)
	if actualOK && expectedOK {
		return format.MessageWithDiff(actualError.Error(), "to equal", expectedError.Error())
	}

	return format.Message(actual, "to equal", matcher.ExpectedError)
}

func (matcher ErrorMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to equal", matcher.ExpectedError)
}
