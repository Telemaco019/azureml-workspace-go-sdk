package workspace

import (
	"context"
	"fmt"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"go.uber.org/zap"
	"net/http"
)

const (
	amlApiVersion          = "2021-03-01-preview"
	amlWorkspaceApiBaseUrl = "https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.MachineLearningServices/workspaces/%s"
)

type WorkspaceHttpClientAPI interface {
}

type WorkspaceHttpClient struct {
	logger            *zap.SugaredLogger
	msalClient        confidential.Client
	subscriptionId    string
	resourceGroupName string
	workspaceName     string
	httpClient        *http.Client
}

func newWorkspaceHttpClient(
	logger *zap.SugaredLogger,
	msalClient confidential.Client,
	subscriptionId,
	resourceGroupName,
	workspaceName string) *WorkspaceHttpClient {
	return &WorkspaceHttpClient{
		logger:            logger,
		msalClient:        msalClient,
		subscriptionId:    subscriptionId,
		resourceGroupName: resourceGroupName,
		workspaceName:     workspaceName,
		httpClient:        &http.Client{},
	}
}

func (c WorkspaceHttpClient) getJwt() (string, error) {
	scopes := []string{DefaultAmlOauthScope}
	c.logger.Debug("Using cached JWT silently...")
	authResult, err := c.msalClient.AcquireTokenSilent(context.Background(), scopes)
	if err != nil {
		c.logger.Debug("Could not acquire JWT silently, now acquiring it with Client Credential flow...")
		authResult, err = c.msalClient.AcquireTokenByCredential(context.Background(), scopes)
		c.logger.Debug("JWT acquired")
	}
	return authResult.AccessToken, err
}

func (c *WorkspaceHttpClient) getWorkspaceApiBaseUrl() string {
	return fmt.Sprintf(amlWorkspaceApiBaseUrl, c.subscriptionId, c.resourceGroupName, c.workspaceName)
}

func (c *WorkspaceHttpClient) newGetRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return req, err
	}

	jwt, err := c.getJwt()
	if err != nil {
		return req, err
	}

	// Add required headers
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))

	// Add required query params
	q := req.URL.Query()
	q.Add("api-version", amlApiVersion)
	req.URL.RawQuery = q.Encode()

	return req, err
}

func (c *WorkspaceHttpClient) doGet(path string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", c.getWorkspaceApiBaseUrl(), path)
	request, err := c.newGetRequest(url)
	if err != nil {
		return nil, err
	}
	c.logger.Infof("GET > %s", url)
	return c.httpClient.Do(request)
}