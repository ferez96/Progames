package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"

	"github.com/moby/moby/client"
)

// BuildRunnerImage packages a compiled binary into a minimal scratch Docker image
// so the bot runs without any OS userland — reduces the attack surface for untrusted code.
func BuildRunnerImage(ctx context.Context, cli *client.Client, binaryPath, imageTag string) error {
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	dockerfile := "FROM scratch\nCOPY bot /bot\nENTRYPOINT [\"/bot\"]\n"
	_ = tw.WriteHeader(&tar.Header{Name: "Dockerfile", Size: int64(len(dockerfile)), Mode: 0o644})
	_, _ = io.WriteString(tw, dockerfile)
	_ = tw.WriteHeader(&tar.Header{Name: "bot", Size: int64(len(binary)), Mode: 0o755})
	_, _ = tw.Write(binary)
	_ = tw.Close()
	resp, err := cli.ImageBuild(ctx, &buf, client.ImageBuildOptions{
		Tags:   []string{imageTag},
		Remove: true,
	})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}
