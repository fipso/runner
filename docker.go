package main

import (
	"context"
	"log"
	"net"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

var docker *client.Client

func connectDocker() {
	var err error
	docker, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
}

func dockerRun(
	image, cmd string, env, port, hostPort *string,
	mountPath string,
) (string, error) {
	var cmdParts []string
	if cmd != "" {
		cmdParts = strings.Split(cmd, " ")
	}

	reader, err := docker.ImagePull(
		context.Background(),
		image,
		types.ImagePullOptions{},
	)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	// Read docker pull logs
	buf := new(strings.Builder)
	_, err = stdcopy.StdCopy(buf, buf, reader)
	pullLogContent := buf.String()
	log.Println(pullLogContent)

	containerConfig := container.Config{
		Image: image,
		Cmd:   cmdParts,
		Tty:   true,
	}

	if env != nil {
		containerConfig.Env = strings.Split(*env, "\n")
	}

	mountPathAbs, err := filepath.Abs(mountPath)

	hostConfig := container.HostConfig{
		// Resources: container.Resources{
		// 	Memory:         1e9, // 1GB RAM
		// 	NanoCPUs:       1e9, // 1 CPU Core
		// 	OomKillDisable: ptr(true),
		// },
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: mountPathAbs,
				Target: "/runner",
			},
		},
	}

	if port != nil && hostPort != nil {
		containerConfig.ExposedPorts = nat.PortSet{
			nat.Port(*port): struct{}{},
		}
		hostConfig.PortBindings = nat.PortMap{
			nat.Port(*port): []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: *hostPort,
				},
			},
		}
	}

	// Create container
	resp, err := docker.ContainerCreate(
		context.Background(),
		&containerConfig,
		&hostConfig,
		&network.NetworkingConfig{},
		nil,
		"",
	)
	if err != nil {
		return "", err
	}

	log.Println("New container", resp.ID)

	// Start container
	err = docker.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return resp.ID, err
	}

	return resp.ID, nil
}

func dockerLogs(id string) (string, error) {
	reader, err := docker.ContainerLogs(
		context.Background(),
		id,
		types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	buf := new(strings.Builder)
	_, err = stdcopy.StdCopy(buf, buf, reader)

	return buf.String(), err
}

func dockerShell(id string) (*net.Conn, error) {
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Tty:          true,
		Env:          []string{"TERM=vt100"},
		Cmd:          []string{"/bin/bash"},
	}

	createRes, err := docker.ContainerExecCreate(
		context.Background(),
		id,
		config,
	)
	if err != nil {
		return nil, err
	}

	attachRes, err := docker.ContainerExecAttach(
		context.Background(),
		createRes.ID,
		types.ExecStartCheck{},
	)
	if err != nil {
		return nil, err
	}

	return &attachRes.Conn, nil
}

func dockerWatch() error {
	// Watch container
	eChan, errChan := docker.Events(context.Background(), types.EventsOptions{})

	for {
		select {
		case err := <-errChan:
			return err
		case msg := <-eChan:
			if msg.Type == "container" {
				if msg.Action == "die" {
					log.Println("Container died:", msg.Actor.ID)
				}
			}
		}
	}
}

func dockerRemove(id string) error {
	return docker.ContainerRemove(
		context.Background(),
		id,
		types.ContainerRemoveOptions{},
	)
}

func dockerStart(id string) error {
	return docker.ContainerStart(
		context.Background(),
		id,
		types.ContainerStartOptions{},
	)
}

func dockerStop(id string) error {
	return docker.ContainerStop(
		context.Background(),
		id,
		container.StopOptions{},
	)
}
