package registry

import (
	"context"
	"testing"
)

func TestDockerRegistry_Backup(t *testing.T) {
	//t.Skip()
	registry := "docker.io/marcosquesada/"
	username := "marcosquesada"
	token := "ab96c1c9-3044-4d74-8755-28b0fe8dec1a"
	r := NewDockerRegistry(registry, username, token)

	origin := "nginx:1.14.2"
	dst := "docker.io/marcosquesada/library_nginx:1.14.2"
	if err := r.Backup(context.Background(), origin, dst); err != nil {
		t.Fatalf("unable to backup container, error %v", err)
	}

}
