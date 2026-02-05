package sandbox

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/nanobot/log"
)

func PipeOut(ctx context.Context, outRead io.Reader, serverName string) {
	scanner := bufio.NewScanner(outRead)
	scanner.Buffer(make([]byte, 0, 1024), 10*1024*1024)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.TrimSpace(text) != "" {
			log.StderrMessages(ctx, serverName, text)
		}
	}
}
