package dockerfiles

import (
	"embed"
	_ "embed"
)

//go:embed *.dockerfile
var DockerfilesFS embed.FS

func GetDockerfileContent(filename string) ([]byte, error) {
	return DockerfilesFS.ReadFile(filename)
}
