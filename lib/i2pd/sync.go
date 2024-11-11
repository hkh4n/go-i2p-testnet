package i2pd

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/common/base64"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func SyncNetDbToShared(cli *client.Client, ctx context.Context, containerID string) error {
	sourcePath := "/root/.i2pd/netDb/"
	destinationPath := "testnet/shared/netDb/"

	// Ensure destination directory exists
	err := os.MkdirAll(destinationPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating destination directory: %v", err)
	}

	// Copy netDb from container to host directory
	reader, _, err := cli.CopyFromContainer(ctx, containerID, sourcePath)
	if err != nil {
		return fmt.Errorf("error copying from container: %v", err)
	}
	defer reader.Close()

	// Untar the content to the destination directory
	tr := tar.NewReader(reader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar header: %v", err)
		}

		// Construct full host path
		hostPath := filepath.Join(destinationPath, hdr.Name)
		if hdr.FileInfo().IsDir() {
			os.MkdirAll(hostPath, hdr.FileInfo().Mode())
		} else {
			file, err := os.OpenFile(hostPath, os.O_CREATE|os.O_WRONLY, hdr.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("error creating file: %v", err)
			}
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return fmt.Errorf("error copying file content: %v", err)
			}
			file.Close()
		}
	}

	fmt.Println("Successfully synchronized netDb to shared directory")
	return nil
}

func SyncSharedToNetDb(cli *client.Client, ctx context.Context, containerID string) error {
	sourcePath := "testnet/shared/netDb/"
	destinationPath := "/root/.i2pd/netDb/"

	// Ensure source directory exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %v", sourcePath)
	}

	// Create a tar archive of the netDb directory
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	err := filepath.Walk(sourcePath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Create tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(strings.Replace(file, sourcePath, "", -1), string(filepath.Separator))
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// If not a dir, write file content
		if !fi.IsDir() {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error creating tar archive: %v", err)
	}
	tw.Close()

	// Copy the tar archive to the container
	err = cli.CopyToContainer(ctx, containerID, destinationPath, bytes.NewReader(buf.Bytes()), types.CopyToContainerOptions{AllowOverwriteDirWithFile: true})
	if err != nil {
		return fmt.Errorf("error copying to container: %v", err)
	}

	fmt.Println("Successfully synchronized netDb from shared directory to container")
	return nil
}

func SyncRouterInfoToNetDb(cli *client.Client, ctx context.Context, containerID string) error {
	// Get RouterInfo, routerInfoString, and the generated filename
	ri, routerInfoString, filename, err := GetRouterInfoWithFilename(cli, ctx, containerID)
	if err != nil {
		return fmt.Errorf("error getting router info: %v", err)
	}
	log.Debugf("got filename: %s\n", filename)

	// Extract the first two characters of the encoded hash
	identHash := ri.IdentHash()
	encodedHash := base64.EncodeToString(identHash[:])
	directory := "r" + encodedHash[:1]

	// Define the target directory in shared netDb
	targetDir := filepath.Join("testnet/shared/netDb", directory)

	// Ensure the target directory exists
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	// Write the RouterInfo to the target directory
	filePath := filepath.Join(targetDir, filename)
	err = ioutil.WriteFile(filePath, []byte(routerInfoString), 0644)
	if err != nil {
		return fmt.Errorf("error writing RouterInfo file: %v", err)
	}

	fmt.Printf("Successfully synced RouterInfo to %s\n", targetDir)
	return nil
}
