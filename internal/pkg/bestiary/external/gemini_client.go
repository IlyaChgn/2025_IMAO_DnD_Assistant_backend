package external

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type GeminiClient struct {
	BaseURL string
	Client  *http.Client
}

func NewGeminiClient(baseURL string) *GeminiClient {
	return &GeminiClient{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

func (g *GeminiClient) GenerateFromDescription(desc string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/create_struct_from_desc/", g.BaseURL)

	payload := map[string]string{"desc": desc}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseJSONResponse(resp)
}

func (g *GeminiClient) GenerateFromImage(imagePath string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/parse_card_from_img/", g.BaseURL)

	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(imagePath))
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(part, file); err != nil {
		return nil, err
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseJSONResponse(resp)
}

func parseJSONResponse(resp *http.Response) (map[string]interface{}, error) {
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(b))
	}

	var result map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return nil, errors.New("failed to parse JSON response")
	}

	return result, nil
}
