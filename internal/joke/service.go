package joke

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultBaseURL = "https://icanhazdadjoke.com/"

type JokeResponse struct {
	ID     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

type Service struct {
	client  *http.Client
	baseURL string
}

func NewService() *Service {
	return &Service{
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: defaultBaseURL,
	}
}

// NewServiceWithURL creates a service with a custom base URL (for testing)
func NewServiceWithURL(baseURL string) *Service {
	return &Service{
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: baseURL,
	}
}

func (s *Service) FetchJoke(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Pace CLI (https://github.com/lucas-tremaroli/pace)")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var res JokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.Joke, nil
}
