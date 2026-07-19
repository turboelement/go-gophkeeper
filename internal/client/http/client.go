package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client — HTTP клиент для взаимодействия с сервером GophKeeper.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// New создаёт новый HTTP клиент.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken устанавливает JWT токен для авторизации.
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetToken возвращает текущий JWT токен.
func (c *Client) GetToken() string {
	return c.token
}

// doRequest выполняет HTTP запрос и возвращает ответ.
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	return resp, nil
}

// APIResponse — стандартный ответ сервера.
type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

// Register отправляет запрос на регистрацию.
func (c *Client) Register(email, password string) (string, string, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}

	resp, err := c.doRequest("POST", "/api/v1/register", body)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	return c.parseAuthResponse(resp)
}

// Login отправляет запрос на аутентификацию.
func (c *Client) Login(email, password string) (string, string, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}

	resp, err := c.doRequest("POST", "/api/v1/login", body)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	return c.parseAuthResponse(resp)
}

// parseAuthResponse парсит ответ от /register и /login.
func (c *Client) parseAuthResponse(resp *http.Response) (string, string, error) {
	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	if !apiResp.Success {
		return "", "", fmt.Errorf("server error: %s", apiResp.Error)
	}

	var data struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return "", "", fmt.Errorf("parse auth data: %w", err)
	}

	return data.Token, data.UserID, nil
}

// CreateSecretRequest — запрос на создание секрета.
type CreateSecretRequest struct {
	Type             string `json:"type"`
	Title            string `json:"title"`
	Metadata         string `json:"metadata,omitempty"`
	EncryptedPayload []byte `json:"encrypted_payload"`
}

// CreateSecret отправляет запрос на создание секрета.
func (c *Client) CreateSecret(req CreateSecretRequest) error {
	resp, err := c.doRequest("POST", "/api/v1/secrets", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkSuccess(resp)
}

// GetSecret запрашивает секрет по ID.
func (c *Client) GetSecret(id string) ([]byte, error) {
	resp, err := c.doRequest("GET", "/api/v1/secrets/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.readResponseData(resp)
}

// ListSecrets запрашивает список секретов.
func (c *Client) ListSecrets() (json.RawMessage, error) {
	resp, err := c.doRequest("GET", "/api/v1/secrets", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.readResponseData(resp)
}

// UpdateSecret отправляет запрос на обновление секрета.
func (c *Client) UpdateSecret(id string, req CreateSecretRequest) error {
	resp, err := c.doRequest("PUT", "/api/v1/secrets/"+id, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkSuccess(resp)
}

// DeleteSecret отправляет запрос на удаление секрета.
func (c *Client) DeleteSecret(id string) error {
	resp, err := c.doRequest("DELETE", "/api/v1/secrets/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkSuccess(resp)
}

// checkSuccess проверяет, что ответ содержит success=true.
func (c *Client) checkSuccess(resp *http.Response) error {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("server error: %s", apiResp.Error)
	}

	return nil
}

// readResponseData читает и возвращает Data из APIResponse.
func (c *Client) readResponseData(resp *http.Response) ([]byte, error) {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("server error: %s", apiResp.Error)
	}

	return apiResp.Data, nil
}
