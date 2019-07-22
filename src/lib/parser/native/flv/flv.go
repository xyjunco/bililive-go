package flv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/xyjunco/bililive-go/src/lib/reader"
)

const (
	audioTag  uint8 = 8
	videoTag  uint8 = 9
	scriptTag uint8 = 18
)

var (
	flvSign      = []byte{0x46, 0x4c, 0x56, 0x01} // flv version01
	NotFlvStream = errors.New("not flv stream")
	UnknownTag   = errors.New("unknown tag")
)

type Metadata struct {
	HasVideo, HasAudio bool
}

type Parser struct {
	Metadata Metadata

	i              *reader.BufferedReader
	o              io.Writer
	avcHeaderCount uint8
	tagCount       uint32

	hc     *http.Client
	stopCh chan struct{}
}

func NewParser() *Parser {
	return &Parser{
		Metadata: Metadata{},
		hc:       &http.Client{},
		stopCh:   make(chan struct{}),
	}
}

func (p *Parser) ParseLiveStream(url *url.URL, file string) error {
	// init input
	req, err := http.NewRequest("GET", url.String(), nil)
	req.Header.Add("User-Agent", "Chrome/59.0.3071.115")
	if err != nil {
		return err
	}
	resp, err := p.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	p.i = reader.New(resp.Body)
	defer p.i.Free()

	// init output
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	p.o = f
	defer f.Close()

	// start parse
	return p.doParse()
}

func (p *Parser) Stop() error {
	close(p.stopCh)
	return nil
}

func (p *Parser) doParse() error {
	// header of flv
	b, err := p.i.ReadN(9)
	if err != nil {
		return err
	}
	// signature
	if !bytes.Equal(b[:4], flvSign) {
		return NotFlvStream
	}
	// flag
	p.Metadata.HasVideo = uint8(b[4])&(1<<2) != 0
	p.Metadata.HasAudio = uint8(b[4])&1 != 0

	// offset must be 9
	if binary.BigEndian.Uint32(b[5:]) != 9 {
		return NotFlvStream
	}

	// write flv header
	if err := p.doWrite(p.i.AllBytes()); err != nil {
		return err
	}
	p.i.Reset()

	for {
		select {
		case <-p.stopCh:
			return nil
		default:
			if err := p.parseTag(); err != nil {
				return err
			}
		}
	}
}

func (p *Parser) doCopy(n uint32) error {
	if n, err := io.CopyN(p.o, p.i, int64(n)); err != nil || n != int64(n) {
		if err == nil {
			err = io.EOF
		}
		return err
	}
	return nil
}

func (p *Parser) doWrite(b []byte) error {
	if n, err := p.o.Write(b); err != nil || n != len(b) {
		if err == nil {
			err = io.EOF
		}
		return err
	}
	return nil
}
