package docker_control

type NodeType struct {
	ImageName      string
	DockerfileName string
}

var (
	GoI2PNode = NodeType{
		ImageName:      "go-i2p-node",
		DockerfileName: "go-i2p-node.dockerfile",
	}
	I2PDNode = NodeType{
		ImageName:      "i2pd-node",
		DockerfileName: "i2pd-node.dockerfile",
	}
	I2PJavaNode = NodeType{
		ImageName:      "i2p-java-node",
		DockerfileName: "i2p-java-node.dockerfile",
	}
)
