package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"io"
	"mime/multipart"
	"net/http"

	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
)

type geminiClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewGeminiClient(baseURL, apiKey string, client *http.Client) bestiaryinterfaces.GeminiAPI {
	return &geminiClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  client,
	}
}

func (g *geminiClient) GenerateFromDescription(ctx context.Context, desc string) (map[string]interface{}, error) {
	l := logger.FromContext(ctx)

	url := fmt.Sprintf("%s/create_struct_from_desc/", g.baseURL)
	payload := map[string]string{"desc": desc}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	g.addHeaders(req)

	ctx = utils.SaveExternalRequestData(ctx, req)

	resp, err := g.client.Do(req)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, err
	}
	defer resp.Body.Close()

	return parseJSONResponse(ctx, resp)
}

func (g *geminiClient) GenerateFromImage(ctx context.Context, image []byte) (map[string]interface{}, error) {
	l := logger.FromContext(ctx)

	url := fmt.Sprintf("%s/parse_card_from_img/", g.baseURL)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "creature.jpg")
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, err
	}
	if _, err := part.Write(image); err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, err
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	g.addHeaders(req)

	ctx = utils.SaveExternalRequestData(ctx, req)

	resp, err := g.client.Do(req)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, err
	}
	defer resp.Body.Close()

	return parseJSONResponse(ctx, resp)
}

func parseJSONResponse(ctx context.Context, resp *http.Response) (map[string]interface{}, error) {
	l := logger.FromContext(ctx)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		l.ExternalError(ctx, apperrors.ApiErr, map[string]any{"body": string(b), "status": resp.StatusCode})
		return nil, apperrors.ApiErr
	}

	var result map[string]interface{}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.InvalidJSONError
	}

	return result, nil
}

func (g *geminiClient) addHeaders(req *http.Request) {
	req.Header.Set("X-API-Key", g.apiKey)
}
