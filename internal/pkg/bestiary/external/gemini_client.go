package external

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
)

type geminiClient struct {
	baseURL string
	client  *http.Client
}

func NewGeminiClient(baseURL string) bestiaryinterfaces.GeminiAPI {
	return &geminiClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (g *geminiClient) GenerateFromDescription(desc string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/create_struct_from_desc/", g.baseURL)

	payload := map[string]string{"desc": desc}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseJSONResponse(resp)
}

func (g *geminiClient) GenerateFromImage(image []byte) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/parse_card_from_img/", g.baseURL)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "creature.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(image); err != nil {
		return nil, err
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := g.client.Do(req)
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
