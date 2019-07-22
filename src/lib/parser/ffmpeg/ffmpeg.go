package ffmpeg

import (
	"io"
	"net/url"
	"os/exec"
)

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
)

type Parser struct {
	cmd      *exec.Cmd
	cmdStdIn io.WriteCloser
}

func New() *Parser {
	return new(Parser)
}

func (p *Parser) ParseLiveStream(url *url.URL, file string) error {
	p.cmd = exec.Command(
		"ffmpeg",
		"-loglevel", "warning",
		"-y", "-re",
		"-user_agent", userAgent,
		"-timeout", "60000000",
		"-i", url.String(),
		"-c", "copy",
		"-bsf:a", "aac_adtstoasc",
		"-f", "mp4",
		file,
	)
	stdIn, err := p.cmd.StdinPipe()
	if err != nil {
		return err
	}
	p.cmdStdIn = stdIn
	p.cmd.Start()
	return p.cmd.Wait()
}

func (p *Parser) Stop() error {
	if p.cmd.ProcessState == nil {
		p.cmdStdIn.Write([]byte("q"))
	}
	return nil
}
