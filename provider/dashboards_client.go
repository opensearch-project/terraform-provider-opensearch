package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const dashboardsXsrfHeader = "osd-xsrf"

type dashboardsAPIResponse struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
}

func dashboardsRequest(conf *ProviderConf, method, path string, requestBody interface{}, responseBody interface{}) error {
	if conf.dashboardsUrl == "" || conf.parsedDashboardsUrl == nil {
		return fmt.Errorf("dashboards_url must be configured to use OpenSearch Dashboards API resources")
	}

	body, err := encodeDashboardsBody(requestBody)
	if err != nil {
		return err
	}

	requestURL, err := dashboardsRequestURL(conf.parsedDashboardsUrl, path)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, requestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(dashboardsXsrfHeader, "true")

	if conf.parsedDashboardsUrl.User.Username() != "" {
		password, _ := conf.parsedDashboardsUrl.User.Password()
		req.SetBasicAuth(conf.parsedDashboardsUrl.User.Username(), password)
	}
	if conf.username != "" && conf.password != "" {
		req.SetBasicAuth(conf.username, conf.password)
	}

	client := dashboardsHTTPClient(conf)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return errDashboardsNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("OpenSearch Dashboards API request failed: %s: %s", resp.Status, string(response))
	}
	if responseBody == nil || len(response) == 0 {
		return nil
	}
	if err := json.Unmarshal(response, responseBody); err != nil {
		return fmt.Errorf("error unmarshalling OpenSearch Dashboards API response: %+v: %s", err, string(response))
	}
	return nil
}

func encodeDashboardsBody(requestBody interface{}) ([]byte, error) {
	if requestBody == nil {
		return nil, nil
	}
	switch body := requestBody.(type) {
	case string:
		return []byte(body), nil
	case []byte:
		return body, nil
	default:
		return json.Marshal(body)
	}
}

func dashboardsRequestURL(base *url.URL, path string) (string, error) {
	requestURL := *base
	requestURL.User = nil
	requestURL.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(path, "/")
	return requestURL.String(), nil
}

func dashboardsHTTPClient(conf *ProviderConf) *http.Client {
	headers := map[string]string{}
	if conf.token != "" {
		return tokenHttpClient(conf, headers)
	}
	if conf.insecure || conf.cacertFile != "" || conf.certPemPath != "" || conf.keyPemPath != "" {
		return tlsHttpClient(conf, headers)
	}
	return defaultHttpClient(conf, headers)
}

var errDashboardsNotFound = errors.New("OpenSearch Dashboards object not found")
