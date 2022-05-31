package registry

import "strings"

type DockerRegistry interface {
	IsNonBackupImage(image string) bool
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
