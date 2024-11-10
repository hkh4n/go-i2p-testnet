package i2pd

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/go-i2p/go-i2p/lib/common/base64"
	"github.com/go-i2p/go-i2p/lib/common/router_info"
	"go-i2p-testnet/lib/docker_control"
)

// GetRouterInfoWithFilename extracts RouterInfo and returns it with the routerInfoString and filename
func GetRouterInfoWithFilename(cli *client.Client, ctx context.Context, containerID string) (*router_info.RouterInfo, string, string, error) {
	routerInfoString, err := docker_control.ReadFileFromContainer(cli, ctx, containerID, "/root/.i2pd/router.info")
	if err != nil {
		return nil, "", "", err
	}
	ri, _, err := router_info.ReadRouterInfo([]byte(routerInfoString))
	if err != nil {
		return nil, "", "", err
	}
	identHash := ri.IdentHash()
	encodedHash := base64.EncodeToString(identHash[:])
	filename := "routerInfo-" + encodedHash + ".dat"
	return &ri, routerInfoString, filename, nil
}
