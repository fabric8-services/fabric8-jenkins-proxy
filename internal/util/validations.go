package util

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// IsNotEmpty checks if value stored at given key is empty.
// if it is empty it returns an error.
func IsNotEmpty(value interface{}, key string) error {
	s, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprintf("Value for %s needs to be a string.", key))
	}

	if len(s) == 0 {
		return errors.New(fmt.Sprintf("Value for %s cannot be empty.", key))
	}
	return nil

}

// IsURL checks if value store at an given key is an URL.
func IsURL(value interface{}, key string) error {
	s, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprintf("Value for %s needs to be a string.", key))
	}

	if !strings.HasPrefix(s, "https://") && !strings.HasPrefix(s, "http://") {
		return errors.New(fmt.Sprintf("Value for %s needs to be a valid URL.", key))
	}

	_, err := url.ParseRequestURI(s)
	if err != nil {
		return errors.New(fmt.Sprintf("Value for %s needs to be a valid URL.", key))
	}
	return nil
}

// IsInt checks if values stored at a given key is an int.
func IsInt(value interface{}, key string) error {
	_, err := strconv.Atoi(value.(string))
	if err != nil {
		return errors.New(fmt.Sprintf("Value for %s needs to be an integer.", key))
	}
	return nil
}

// IsBool checks if value stored at a given key is a bool.
func IsBool(value interface{}, key string) error {
	_, err := strconv.ParseBool(value.(string))
	if err != nil {
		return errors.New(fmt.Sprintf("Value for %s needs to be an bool.", key))
	}
	return nil
}
