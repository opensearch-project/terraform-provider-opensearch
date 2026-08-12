package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/olivere/elastic/uritemplates"
)

const dashboardsXSRFHeader = "osd-xsrf"

func resourceOpensearchDashboard() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a bundle of OpenSearch Dashboards saved objects (dashboards, visualizations, index patterns, searches, and so on) via the Dashboards Saved Objects API. `source` is ndjson content to import, such as an export produced by Dashboards' Saved Objects Management UI (Stack Management > Saved Objects > Export) or `file(\"export.ndjson\")`. On update, objects are re-imported with `overwrite=true`, and any object present in a prior `source` but absent from the new one is deleted. Requires the Dashboards Saved Objects API to be reachable; see the provider's `dashboards_url` setting.",
		Create:      resourceOpensearchDashboardCreate,
		Read:        resourceOpensearchDashboardRead,
		Update:      resourceOpensearchDashboardUpdate,
		Delete:      resourceOpensearchDashboardDelete,
		Schema: map[string]*schema.Schema{
			"source": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ndjson content to import, typically loaded via `file(\"export.ndjson\")`.",
			},
			"objects": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The saved objects contained in `source`, as reported by the most recent import.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The saved object type, e.g. `dashboard`, `visualization`, `index-pattern`, or `search`.",
						},
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The saved object id.",
						},
					},
				},
			},
		},
	}
}

func resourceOpensearchDashboardCreate(d *schema.ResourceData, m interface{}) error {
	conf := m.(*ProviderConf)

	results, err := dashboardImportSavedObjects(conf, d.Get("source").(string))
	if err != nil {
		return err
	}

	id, err := uuid.GenerateUUID()
	if err != nil {
		return fmt.Errorf("could not generate resource id: %+v", err)
	}
	d.SetId(id)

	return setDashboardObjects(d, results)
}

func resourceOpensearchDashboardRead(d *schema.ResourceData, m interface{}) error {
	conf := m.(*ProviderConf)

	for _, ref := range getDashboardObjects(d) {
		err := dashboardGetSavedObject(conf, ref)
		if isDashboardsNotFound(err) {
			log.Printf("[WARN] Dashboard saved object (%s/%s) not found, removing opensearch_dashboard (%s) from state", ref.Type, ref.ID, d.Id())
			d.SetId("")
			return nil
		}
		if err != nil {
			return fmt.Errorf("could not read Dashboard saved object (%s/%s): %+v", ref.Type, ref.ID, err)
		}
	}

	return nil
}

func resourceOpensearchDashboardUpdate(d *schema.ResourceData, m interface{}) error {
	conf := m.(*ProviderConf)
	previous := getDashboardObjects(d)

	results, err := dashboardImportSavedObjects(conf, d.Get("source").(string))
	if err != nil {
		return err
	}

	current := make(map[dashboardSavedObjectRef]bool, len(results))
	for _, r := range results {
		current[dashboardSavedObjectRef(r)] = true
	}

	for _, ref := range previous {
		if current[ref] {
			continue
		}
		if err := dashboardDeleteSavedObject(conf, ref); err != nil {
			return fmt.Errorf("failed to delete saved object (%s/%s) that was removed from source: %+v", ref.Type, ref.ID, err)
		}
	}

	return setDashboardObjects(d, results)
}

func resourceOpensearchDashboardDelete(d *schema.ResourceData, m interface{}) error {
	conf := m.(*ProviderConf)

	for _, ref := range getDashboardObjects(d) {
		if err := dashboardDeleteSavedObject(conf, ref); err != nil {
			return fmt.Errorf("failed to delete saved object (%s/%s): %+v", ref.Type, ref.ID, err)
		}
	}

	return nil
}

type dashboardSavedObjectRef struct {
	Type string
	ID   string
}

type dashboardImportResult struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type dashboardImportError struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Title string `json:"title"`
	Error struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
}

type dashboardImportResponse struct {
	Success        bool                    `json:"success"`
	SuccessCount   int                     `json:"successCount"`
	SuccessResults []dashboardImportResult `json:"successResults"`
	Errors         []dashboardImportError  `json:"errors"`
}

// dashboardImportSavedObjects imports source (ndjson) via the Dashboards
// Saved Objects _import API with overwrite=true, so that re-applying updates
// existing objects in place. The API always returns HTTP 200, even on
// partial or complete failure, so success is determined from the response
// body.
func dashboardImportSavedObjects(conf *ProviderConf, source string) ([]dashboardImportResult, error) {
	client, err := getDashboardsClient(conf)
	if err != nil {
		return nil, fmt.Errorf("could not create Dashboards client: %+v", err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="import.ndjson"`},
		"Content-Type":        {"application/ndjson"},
	})
	if err != nil {
		return nil, fmt.Errorf("could not build saved objects import request: %+v", err)
	}
	if _, err := part.Write([]byte(source)); err != nil {
		return nil, fmt.Errorf("could not build saved objects import request: %+v", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("could not build saved objects import request: %+v", err)
	}

	respBody, err := client.do(context.TODO(), dashboardsRequest{
		method:      "POST",
		path:        dashboardsSavedObjectsPath(conf, "/_import"),
		query:       url.Values{"overwrite": []string{"true"}},
		body:        &body,
		contentType: mw.FormDataContentType(),
		headers:     map[string]string{dashboardsXSRFHeader: "true"},
	})
	if err != nil {
		return nil, fmt.Errorf("saved objects import request failed: %+v", err)
	}

	var result dashboardImportResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("could not parse saved objects import response: %+v: %s", err, respBody)
	}
	if !result.Success {
		return nil, fmt.Errorf("failed to import %d of %d saved object(s): %s", len(result.Errors), len(result.Errors)+result.SuccessCount, formatDashboardImportErrors(result.Errors))
	}

	return result.SuccessResults, nil
}

func formatDashboardImportErrors(errs []dashboardImportError) string {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		reason := e.Error.Type
		if e.Error.Reason != "" {
			reason = fmt.Sprintf("%s: %s", e.Error.Type, e.Error.Reason)
		}
		msgs = append(msgs, fmt.Sprintf("%s/%s (%s)", e.Type, e.ID, reason))
	}
	return strings.Join(msgs, "; ")
}

func dashboardGetSavedObject(conf *ProviderConf, ref dashboardSavedObjectRef) error {
	client, err := getDashboardsClient(conf)
	if err != nil {
		return fmt.Errorf("could not create Dashboards client: %+v", err)
	}

	path, err := uritemplates.Expand(dashboardsSavedObjectsPath(conf, "/{type}/{id}"), map[string]string{
		"type": ref.Type,
		"id":   ref.ID,
	})
	if err != nil {
		return fmt.Errorf("error building URL path for saved object: %+v", err)
	}

	_, err = client.do(context.TODO(), dashboardsRequest{method: "GET", path: path})
	return err
}

func dashboardDeleteSavedObject(conf *ProviderConf, ref dashboardSavedObjectRef) error {
	client, err := getDashboardsClient(conf)
	if err != nil {
		return fmt.Errorf("could not create Dashboards client: %+v", err)
	}

	path, err := uritemplates.Expand(dashboardsSavedObjectsPath(conf, "/{type}/{id}"), map[string]string{
		"type": ref.Type,
		"id":   ref.ID,
	})
	if err != nil {
		return fmt.Errorf("error building URL path for saved object: %+v", err)
	}

	_, err = client.do(context.TODO(), dashboardsRequest{
		method:  "DELETE",
		path:    path,
		headers: map[string]string{dashboardsXSRFHeader: "true"},
	})
	if err != nil && !isDashboardsNotFound(err) {
		return err
	}
	return nil
}

func getDashboardObjects(d *schema.ResourceData) []dashboardSavedObjectRef {
	raw := d.Get("objects").([]interface{})
	refs := make([]dashboardSavedObjectRef, 0, len(raw))
	for _, o := range raw {
		m := o.(map[string]interface{})
		refs = append(refs, dashboardSavedObjectRef{
			Type: m["type"].(string),
			ID:   m["id"].(string),
		})
	}
	return refs
}

func setDashboardObjects(d *schema.ResourceData, results []dashboardImportResult) error {
	objects := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		objects = append(objects, map[string]interface{}{
			"type": r.Type,
			"id":   r.ID,
		})
	}
	return d.Set("objects", objects)
}

// dashboardsAPIError is returned for non-2xx responses from the Dashboards
// Saved Objects API. Dashboards uses a Kibana/hapi-style error shape
// ({"statusCode":N,"error":"...","message":"..."}), which is entirely
// different from Elasticsearch/OpenSearch's own error shape
// ({"error":{"type":...,"reason":...}}). Requests to Dashboards must not be
// made through the elastic7 client: its response parsing expects the
// OpenSearch shape and silently discards the real message on a mismatch,
// surfacing only a bare "elastic: Error 400 (Bad Request)".
type dashboardsAPIError struct {
	StatusCode int
	Message    string
}

func (e *dashboardsAPIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("dashboards API error %d (%s): %s", e.StatusCode, http.StatusText(e.StatusCode), e.Message)
	}
	return fmt.Sprintf("dashboards API error %d (%s)", e.StatusCode, http.StatusText(e.StatusCode))
}

func isDashboardsNotFound(err error) bool {
	var apiErr *dashboardsAPIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// dashboardsClient issues plain HTTP requests to the Dashboards Saved
// Objects API, reusing the same underlying *http.Client (and thus the same
// TLS/proxy/AWS request signing) as the primary OpenSearch client, but none
// of elastic7's Elasticsearch-specific request/response handling.
type dashboardsClient struct {
	http     *http.Client
	baseURL  string
	username string
	password string
}

type dashboardsRequest struct {
	method      string
	path        string
	query       url.Values
	body        io.Reader
	contentType string
	headers     map[string]string
}

func getDashboardsClient(conf *ProviderConf) (*dashboardsClient, error) {
	rawUrl := conf.rawUrl
	parsedUrl := conf.parsedUrl
	if conf.dashboardsUrl != "" {
		rawUrl = conf.dashboardsUrl
		parsedUrl = conf.parsedDashboardsUrl
	}

	username, password := conf.username, conf.password
	if username == "" && parsedUrl.User.Username() != "" {
		username = parsedUrl.User.Username()
		password, _ = parsedUrl.User.Password()
	}

	var httpClient *http.Client
	var err error
	switch {
	case awsUrlRegexp.MatchString(parsedUrl.Hostname()) && conf.signAWSRequests:
		region := awsUrlRegexp.FindStringSubmatch(parsedUrl.Hostname())[1]
		log.Printf("[INFO] Using AWS for Dashboards: %+v", region)
		httpClient, err = awsHttpClient(region, conf, map[string]string{})
	case conf.awsRegion != "" && conf.signAWSRequests:
		log.Printf("[INFO] Using AWS for Dashboards: %+v", conf.awsRegion)
		httpClient, err = awsHttpClient(conf.awsRegion, conf, map[string]string{})
	case conf.token != "":
		httpClient = tokenHttpClient(conf, map[string]string{})
	case conf.insecure || conf.cacertFile != "":
		httpClient = tlsHttpClient(conf, map[string]string{})
	default:
		httpClient = defaultHttpClient(conf, map[string]string{})
	}
	if err != nil {
		return nil, err
	}

	return &dashboardsClient{
		http:     httpClient,
		baseURL:  strings.TrimRight(rawUrl, "/"),
		username: username,
		password: password,
	}, nil
}

func (c *dashboardsClient) do(ctx context.Context, r dashboardsRequest) ([]byte, error) {
	reqUrl := c.baseURL + r.path
	if len(r.query) > 0 {
		reqUrl += "?" + r.query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, r.method, reqUrl, r.body)
	if err != nil {
		return nil, fmt.Errorf("could not build Dashboards request: %+v", err)
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	req.Header.Set("Accept", "application/json")
	if r.contentType != "" {
		req.Header.Set("Content-Type", r.contentType)
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dashboards request failed: %+v", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read Dashboards response: %+v", err)
	}

	if res.StatusCode >= 300 {
		apiErr := &dashboardsAPIError{StatusCode: res.StatusCode}
		var parsed struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &parsed) == nil && parsed.Message != "" {
			apiErr.Message = parsed.Message
		} else {
			apiErr.Message = string(respBody)
		}
		return nil, apiErr
	}

	return respBody, nil
}
