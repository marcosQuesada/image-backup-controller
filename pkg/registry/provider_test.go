package registry

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDockerRegistryExistenceOnTopOfFakedHttpServerWithSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	r := NewDockerRegistry(u.Host, "fakeUser", "fakePassword")
	res, err := r.Exists(context.Background(), u.Host+"/marcosquesada/nginx:1.14.2")
	if err != nil {
		t.Fatalf("unable to check container existence, error %v", err)
	}

	if !res {
		t.Fatal("expected existence")
	}
}

func TestDockerRegistryExistenceOnTopOfFakedHttpServerAndNotFoundImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.WriteHeader(http.StatusOK)

		case "/v2/marcosquesada/nginx/manifests/1.14.2":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusOK)
		}

	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	r := NewDockerRegistry(u.Host, "fakeUser", "fakePassword")
	res, err := r.Exists(context.Background(), u.Host+"/marcosquesada/nginx:1.14.2")
	if err != nil {
		t.Fatalf("unable to check container existence, error %v", err)
	}

	if res {
		t.Fatal("expected not existence")
	}
}

func TestDockerRegistryIsAbleToCreateBackupFromFakeImage(t *testing.T) {
	s := httptest.NewServer(registry.New())
	defer s.Close()

	u, _ := url.Parse(s.URL)
	src := fmt.Sprintf("%s/marcosquesada/nginx:1.14.2", u.Host)
	dst := fmt.Sprintf("%s/backupregistry/nginx:1.14.2", u.Host)

	img, err := random.Image(1024, 5)
	if err != nil {
		t.Fatalf("unable to backup container, error %v", err)
	}

	err = crane.Push(img, src)
	if err != nil {
		t.Fatalf("unable to backup container, error %v", err)
	}

	reg := u.Host + "/marcosquesada/"
	username := "marcosquesada"
	token := "fakeToken"
	r := NewDockerRegistry(reg, username, token)
	if err := r.Backup(context.Background(), src, dst); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestDockerRegistryIsAbleToCreateBackupFromNonPushedFakeImageReturnsError(t *testing.T) {
	s := httptest.NewServer(registry.New())
	defer s.Close()

	u, _ := url.Parse(s.URL)
	src := fmt.Sprintf("%s/marcosquesada/nginx:1.14.2", u.Host)
	dst := fmt.Sprintf("%s/backupregistry/nginx:1.14.2", u.Host)

	username := "marcosquesada"
	token := "fakeToken"
	r := NewDockerRegistry(u.Host+"/marcosquesada/", username, token)
	if err := r.Backup(context.Background(), src, dst); err == nil {
		t.Fatal("expected error")
	}
}

func TestBackupImageNameGeneration(t *testing.T) {
	var testSamples = []struct {
		registry string
		image    string
		expected string
	}{
		{
			registry: "docker.io/backupregistry/",
			image:    "nginx:1.14.2",
			expected: "docker.io/backupregistry/library_nginx:1.14.2",
		},
		{
			registry: "docker.io/backupregistry/",
			image:    "marcosquesada/foo:1.0.0",
			expected: "docker.io/backupregistry/marcosquesada_foo:1.0.0",
		},
		{
			registry: "docker.io/backupregistry/",
			image:    "docker.io/marcosquesada/foo:1.0.0",
			expected: "docker.io/backupregistry/marcosquesada_foo:1.0.0",
		},
	}

	for _, sample := range testSamples {
		r := NewDockerRegistry(sample.registry, "bar", "zoom")
		res, err := r.BackupImageName(sample.image)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if expected, got := sample.expected, res; expected != got {
			t.Fatalf("values do not match, expected %s got %s", expected, got)
		}
	}
}
