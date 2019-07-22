package recorders

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xyjunco/bililive-go/src/api"
	"github.com/xyjunco/bililive-go/src/configs"
	"github.com/xyjunco/bililive-go/src/instance"
	"github.com/xyjunco/bililive-go/src/interfaces"
	"github.com/xyjunco/bililive-go/src/lib/events"
	"github.com/xyjunco/bililive-go/src/lib/parser"
	"github.com/xyjunco/bililive-go/src/lib/parser/ffmpeg"
	"github.com/xyjunco/bililive-go/src/lib/parser/native/flv"
	"github.com/xyjunco/bililive-go/src/lib/utils"
)

type Recorder struct {
	Live       api.Live
	OutPutPath string

	config               *configs.Config
	ed                   events.IEventDispatcher
	logger               *interfaces.Logger
	startOnce, closeOnce *sync.Once
	stop                 chan struct{}

	parser parser.Parser
}

func NewRecorder(ctx context.Context, live api.Live) (*Recorder, error) {
	inst := instance.GetInstance(ctx)
	return &Recorder{
		Live:       live,
		OutPutPath: instance.GetInstance(ctx).Config.OutPutPath,
		config:     inst.Config,
		ed:         inst.EventDispatcher.(events.IEventDispatcher),
		logger:     inst.Logger,
		startOnce:  new(sync.Once),
		closeOnce:  new(sync.Once),
		stop:       make(chan struct{}),
	}, nil
}

func (r *Recorder) run() {
	for {
		select {
		case <-r.stop:
			return
		default:
			urls, err := r.Live.GetStreamUrls()
			if err != nil {
				time.Sleep(5 * time.Second)
				continue
			}
			var (
				platformName = utils.ReplaceIllegalChar(r.Live.GetPlatformCNName())
				hostName     = utils.ReplaceIllegalChar(r.Live.GetCachedInfo().HostName)
				fileName     = fmt.Sprintf("%s_%s_%s.mp4", platformName, hostName, time.Now().Format("20060102_1504"))
				outputPath   = filepath.Join(r.OutPutPath, platformName, hostName)
				file         = filepath.Join(outputPath, fileName)
				url          = urls[0]
			)
			os.MkdirAll(outputPath, os.ModePerm)
			if strings.Contains(url.Path, ".flv") && r.config.Feature.UseNativeFlvParser {
				r.parser = flv.NewParser()
			} else {
				r.parser = ffmpeg.New()
			}
			r.logger.Debugln(r.parser.ParseLiveStream(url, file))
			if stat, err := os.Stat(file); err == nil && stat.Size() == 0 {
				os.Remove(file)
			}
		}
	}
}

func (r *Recorder) Start() error {
	r.startOnce.Do(func() {
		go r.run()
		r.logger.WithFields(r.Live.GetInfoMap()).Info("Recorde Start")
		r.ed.DispatchEvent(events.NewEvent(RecorderStart, r.Live))
	})
	return nil
}

func (r *Recorder) Close() {
	r.closeOnce.Do(func() {
		close(r.stop)
		r.parser.Stop()
		r.logger.WithFields(r.Live.GetInfoMap()).Info("Recorde End")
		r.ed.DispatchEvent(events.NewEvent(RecorderStop, r.Live))
	})
}
