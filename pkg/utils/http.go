package utils

import (
	"fmt"

	"resty.dev/v3"
)

// DoJSONRequest performs a GET and unmarshals the JSON response into result.
func DoJSONRequest(client *resty.Client, url string, result interface{}) error {
	resp, err := client.R().
		SetHeader("Accept", "application/json").
		SetResult(result).
		Get(url)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("request to %s failed: %s", url, resp.Status())
	}
	return nil
}
