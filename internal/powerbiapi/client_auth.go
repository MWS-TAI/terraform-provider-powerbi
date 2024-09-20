package powerbiapi

import (
	// "context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	"github.com/hashicorp/go-cleanhttp"
	// "github.com/hashicorp/terraform-plugin-log/tflog"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

type cliTokenResponse struct {
    AccessToken  string `json:"accessToken"`
    ExpiresOn    string `json:"expiresOn"`
    ExpiresOnTS  int64  `json:"expires_on"`
    Subscription string `json:"subscription"`
    Tenant       string `json:"tenant"`
    TokenType    string `json:"tokenType"`
}

type bearerTokenRoundTripper struct {
	innerRoundTripper http.RoundTripper
	getToken          func(*http.Client) (string, error)
	mux               sync.Mutex
	token             string
}

func newBearerTokenRoundTripper(getToken func(*http.Client) (string, error), next http.RoundTripper) http.RoundTripper {
	return &bearerTokenRoundTripper{
		innerRoundTripper: next,
		getToken:          getToken,
	}
}

func (rt *bearerTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	newRequest := *req

	if rt.token == "" {
		err := func() error {
			rt.mux.Lock()
			defer rt.mux.Unlock()

			if rt.token == "" {

				// create own http client so we dont try to add token to request to get tokens
				httpClient := cleanhttp.DefaultClient()
				httpClient.Transport = newErrorOnUnsuccessfulRoundTripper(httpClient.Transport)

				token, err := rt.getToken(httpClient)
				if err != nil {
					return err
				}
				rt.token = token
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	newRequest.Header.Set("Authorization", "Bearer "+rt.token)

	return rt.innerRoundTripper.RoundTrip(&newRequest)
}

func getAuthTokenWithPassword(
	httpClient *http.Client,
	tenant string,
	clientID string,
	clientSecret string,
	username string,
	password string,
) (string, error) {

	authURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", url.PathEscape(tenant))
	resp, err := httpClient.Post(authURL, "application/x-www-form-urlencoded", strings.NewReader(url.Values{
		"grant_type":    {"password"},
		"scope":         {"https://analysis.windows.net/powerbi/api/.default"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"username":      {username},
		"password":      {password},
	}.Encode()))

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status: %d, body: %s", resp.StatusCode, data)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var dataObj tokenResponse
	err = json.Unmarshal(data, &dataObj)
	return dataObj.AccessToken, err
}

func getAuthTokenWithClientCredentials(
	httpClient *http.Client,
	tenant string,
	clientID string,
	clientSecret string,
) (string, error) {

	authURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", url.PathEscape(tenant))
	resp, err := httpClient.Post(authURL, "application/x-www-form-urlencoded", strings.NewReader(url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {"https://analysis.windows.net/powerbi/api/.default"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}.Encode()))

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status: %d, body: %s", resp.StatusCode, data)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var dataObj tokenResponse
	err = json.Unmarshal(data, &dataObj)
	return dataObj.AccessToken, err
}

func getAuthTokenWithAzureCLI() (string, error) {
    // Execute the az account get-access-token command
    cmd := exec.Command("az", "account", "get-access-token", "--resource", "https://analysis.windows.net/powerbi/api")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to execute az command: %v", err)
    }

    // Parse the output
    var dataObj cliTokenResponse
    if err := json.Unmarshal(output, &dataObj); err != nil {
        return "", fmt.Errorf("failed to parse az command output: %v", err)
    }

    return dataObj.AccessToken, nil
}
