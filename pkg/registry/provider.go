package registry

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"strings"
	"time"
)

type DockerRegistry interface {
	IsNonBackupImage(image string) bool
	Exists(ctx context.Context, image string) (bool, error)
	Backup(ctx context.Context, imageSource, imageDestination string) error
	BackupImageName(image string) (string, error)
}

type dockerRegistry struct {
	backupRegistry string
}

func NewDockerRegistry(b string) DockerRegistry {
	return &dockerRegistry{
		backupRegistry: b,
	}
}

func (d *dockerRegistry) IsNonBackupImage(image string) bool {
	return !strings.HasPrefix(image, d.backupRegistry)
}

func (d *dockerRegistry) Exists(ctx context.Context, image string) (bool, error) {
	return false, nil
}

func (d *dockerRegistry) Backup(ctx context.Context, imageSource, imageDestination string) error {
	time.Sleep(time.Second * 5)
	return nil
}

func (d *dockerRegistry) BackupImageName(image string) (string, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return "", fmt.Errorf("unexpected parse image reference error %w", err)
	}

	fmt.Println("Repo is " + ref.String())
	replacedName := strings.ReplaceAll(ref.Context().RepositoryStr(), "/", "_")

	return fmt.Sprintf("%s/%s:%s", d.backupRegistry, replacedName, ref.Identifier()), nil
}
