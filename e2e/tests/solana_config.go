package e2e

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TODO: we can eventually move this to CTF framework. This can be the blockhain.Input
type SolConfig struct {
	Type                     string             `toml:"type" validate:"required,oneof=solana" envconfig:"net_type"`
	Image                    string             `toml:"image"`
	PullImage                bool               `toml:"pull_image"`
	Port                     string             `toml:"port"`
	WSPort                   string             `toml:"port_ws"`
	ChainID                  string             `toml:"chain_id"`
	DockerCmdParamsOverrides []string           `toml:"docker_cmd_params"`
	Out                      *blockchain.Output `toml:"out"`
}

func defaultSolana(in *SolConfig) {
	if in.Image == "" {
		in.Image = "solanalabs/solana:v1.18.26"
	}
	if in.ChainID == "" {
		in.ChainID = "localnet"
	}
	if in.Port == "" {
		in.Port = "8899"
	}
}

// newSolana initializes and starts a Solana container
// TODO: we can eventually move this to CTF framework here https://github.com/smartcontractkit/chainlink-testing-framework/tree/main/framework/components/blockchain
func (in *SolConfig) newSolana() (*blockchain.Output, error) {
	defaultSolana(in)
	ctx := context.Background()

	// Always use a temporary directory for the ledger
	tempDir, err := os.MkdirTemp("", "solana-ledger-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory for ledger: %w", err)
	}
	// Ensure the temp folder is cleaned up after the container stops
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("Failed to clean up temp directory %s: %v", tempDir, err)
		}
	}()

	containerName := framework.DefaultTCName("solana-node")
	bindPort := fmt.Sprintf("%s/tcp", in.Port)
	wsPortNumber, err := strconv.Atoi(in.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to convert port to integer: %w", err)
	}
	wsPortNumber += 1 // Increment by 1
	wsPortNumberStr := strconv.Itoa(wsPortNumber)
	fmt.Println("Creating container in port ", containerName, in.Image, in.Port)

	req := testcontainers.ContainerRequest{
		AlwaysPullImage: in.PullImage,
		Image:           in.Image,
		Labels:          framework.DefaultTCLabels(),
		Networks:        []string{framework.DefaultNetworkName},
		NetworkAliases: map[string][]string{
			framework.DefaultNetworkName: {containerName},
		},
		Entrypoint: []string{"solana-test-validator"}, // Override the default entrypoint
		Cmd: []string{
			"--rpc-port", in.Port,
			"--bind-address", "0.0.0.0",
			"--log",
		},
		ExposedPorts: []string{
			fmt.Sprintf("%s/tcp", in.Port),      // HTTP RPC port
			fmt.Sprintf("%d/tcp", wsPortNumber), // WebSocket port (RPC + 1)
		},
		HostConfigModifier: func(h *container.HostConfig) {
			h.PortBindings = framework.MapTheSamePort(bindPort)
			h.PortBindings[nat.Port(fmt.Sprintf("%d/tcp", wsPortNumber))] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: wsPortNumberStr,
				},
			}

		},
		WaitingFor: wait.ForListeningPort(nat.Port(in.Port)).
			WithStartupTimeout(30 * time.Second),
	}

	// Start the container
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Retrieve host and mapped port
	host, err := c.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}
	mp, err := c.MappedPort(ctx, nat.Port(bindPort))
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	return &blockchain.Output{
		UseCache:      true,
		Family:        "solana",
		ChainID:       in.ChainID,
		ContainerName: containerName,
		Nodes: []*blockchain.Node{
			{
				HostWSUrl:             fmt.Sprintf("ws://%s:%d", host, wsPortNumber),
				HostHTTPUrl:           fmt.Sprintf("http://%s:%s", host, mp.Port()),
				DockerInternalHTTPUrl: fmt.Sprintf("http://%s:%s", containerName, in.Port),
			},
		},
	}, nil
}
