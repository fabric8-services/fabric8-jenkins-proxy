package util

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_IsURL(t *testing.T) {
	var testURLs = []struct {
		url    string
		errors []string
	}{
		{"http://localhist:9999/api", []string{}},
		{"http://localhist:9999/api/", []string{}},
		{"foo", []string{"Value for FOO needs to be a valid URL."}},
		{"/foo", []string{"Value for FOO needs to be a valid URL."}},
		{"ftp://localhost", []string{"Value for FOO needs to be a valid URL."}},
		{"", []string{"Value for FOO needs to be a valid URL."}},
	}

	for _, testURL := range testURLs {
		err := IsURL(testURL.url, "FOO")
		var errors []string
		if err == nil {
			errors = []string{}
		} else {
			errors = strings.Split(err.Error(), "\n")
		}

		assert.Equal(t, testURL.errors, errors, fmt.Sprintf("Unexpected error for %s", testURL.url))
	}
}

func Test_IsBool(t *testing.T) {
	var testBools = []struct {
		value    string
		expected bool
		errors   []string
	}{
		{"true", true, []string{}},
		{"false", false, []string{}},
		{"0", false, []string{}},
		{"1", true, []string{}},
		{"snafu", false, []string{"Value for FOO needs to be an bool."}},
		{"", false, []string{"Value for FOO needs to be an bool."}},
	}

	for _, testBool := range testBools {
		err := IsBool(testBool.value, "FOO")
		var errors []string
		if err == nil {
			errors = []string{}
		} else {
			errors = strings.Split(err.Error(), "\n")
		}

		assert.Equal(t, testBool.errors, errors, fmt.Sprintf("Unexpected error for %s", testBool.value))
	}
}

func Test_IsDuration(t *testing.T) {
	var tt = []struct {
		name     string
		value    string
		expected time.Duration
		errors   []string
	}{
		{"second", "10s", 10 * time.Second, []string{}},
		{"minute", "2m", 2 * time.Minute, []string{}},
		{"hour", "3h", 3 * time.Hour, []string{}},
		{"FOO", "invalid", 0, []string{"Value for duration needs to be a time duration."}},
		{"empty", "", 0, []string{"Value for duration needs to be a time duration."}},
	}

	for _, testcase := range tt {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			err := IsDuration(testcase.value, "duration")
			errors := []string{}
			if err != nil {
				errors = strings.Split(err.Error(), "\n")
			}

			assert.Equal(t, testcase.errors, errors, fmt.Sprintf("Unexpected error for %s", testcase.value))
		})
	}
}
