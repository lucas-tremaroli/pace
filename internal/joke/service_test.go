package joke

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchJoke_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept: application/json header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "123", "joke": "Why did the scarecrow win an award? He was outstanding in his field.", "status": 200}`))
	}))
	defer server.Close()

	svc := NewServiceWithURL(server.URL)
	joke, err := svc.FetchJoke(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "Why did the scarecrow win an award? He was outstanding in his field."
	if joke != expected {
		t.Errorf("expected joke %q, got %q", expected, joke)
	}
}

func TestFetchJoke_BadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := NewServiceWithURL(server.URL)
	_, err := svc.FetchJoke(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain '500', got %v", err)
	}
}

func TestFetchJoke_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	svc := NewServiceWithURL(server.URL)
	_, err := svc.FetchJoke(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestFetchJoke_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "123", "joke": "test", "status": 200}`))
	}))
	defer server.Close()

	svc := NewServiceWithURL(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := svc.FetchJoke(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}
