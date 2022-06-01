package registry

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"net/http"
	"strings"
)

// DockerRegistry defines docker registry provider
type DockerRegistry interface {
	IsNonImageBackup(image string) bool
	Exists(ctx context.Context, image string) (bool, error)
	Backup(ctx context.Context, imageSource, imageDestination string) error
	BackupImageName(image string) (string, error)
}

type dockerRegistry struct {
	backupRegistry string
	credentials    authn.Authenticator
}

// NewDockerRegistry instantiates docker registry provider
func NewDockerRegistry(backupRepository, username, token string) DockerRegistry {
	auth := authn.AuthConfig{
		Username: username,
		Password: token,
	}

	return &dockerRegistry{
		backupRegistry: backupRepository,
		credentials:    authn.FromConfig(auth),
	}
}

// IsNonImageBackup checks if provided image is non image backup
func (d *dockerRegistry) IsNonImageBackup(image string) bool {
	return !strings.HasPrefix(image, d.backupRegistry)
}

// Exists checks in docker register the image existence
func (d *dockerRegistry) Exists(ctx context.Context, image string) (bool, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return false, fmt.Errorf("unexpected parse image reference error %w", err)
	}

	_, err = remote.Index(ref, remote.WithContext(ctx), remote.WithAuth(d.credentials))
	if err != nil {
		e, ok := err.(*transport.Error)
		if !ok {
			return false, fmt.Errorf("unexpected get image %q error %w", ref, err)
		}

		if e.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("unexpected get image %q transport error %w", ref, err)
	}

	return true, nil
}

// Backup clones source image to backupRegistry destination
func (d *dockerRegistry) Backup(ctx context.Context, imageSource, imageDestination string) error {
	if err := crane.Copy(imageSource, imageDestination, crane.WithContext(ctx), crane.WithAuth(d.credentials)); err != nil {
		return fmt.Errorf("unexpected error copying image src %s dst %s, error %w", imageSource, imageDestination, err)
	}

	return nil
}

// BackupImageName formats backup image name properly
func (d *dockerRegistry) BackupImageName(image string) (string, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return "", fmt.Errorf("unexpected parse image reference error %w", err)
	}

	replacedName := strings.ReplaceAll(ref.Context().RepositoryStr(), "/", "_")

	return fmt.Sprintf("%s%s:%s", d.backupRegistry, replacedName, ref.Identifier()), nil
}
