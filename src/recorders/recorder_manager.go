package recorders

import (
	"context"
	"sync"

	"github.com/xyjunco/bililive-go/src/api"
	"github.com/xyjunco/bililive-go/src/instance"
	"github.com/xyjunco/bililive-go/src/lib/events"
	"github.com/xyjunco/bililive-go/src/listeners"
)

func NewIRecorderManager(ctx context.Context) IRecorderManager {
	rm := &RecorderManager{
		savers: make(map[api.LiveId]*Recorder),
		lock:   new(sync.RWMutex),
	}
	instance.GetInstance(ctx).RecorderManager = rm
	return rm
}

type IRecorderManager interface {
	AddRecorder(ctx context.Context, live api.Live) error
	RemoveRecorder(ctx context.Context, liveId api.LiveId) error
	GetRecorder(ctx context.Context, liveId api.LiveId) (*Recorder, error)
	HasRecorder(ctx context.Context, liveId api.LiveId) bool
}

type RecorderManager struct {
	savers map[api.LiveId]*Recorder
	lock   *sync.RWMutex
}

func (r *RecorderManager) Start(ctx context.Context) error {
	inst := instance.GetInstance(ctx)
	if inst.Config.RPC.Enable || len(inst.Lives) > 0 {
		inst.WaitGroup.Add(1)
	}
	inst.Logger.Info("RecorderManager Start")
	ed := inst.EventDispatcher.(events.IEventDispatcher)

	// 开播事件
	ed.AddEventListener(listeners.LiveStart, events.NewEventListener(func(event *events.Event) {
		r.AddRecorder(ctx, event.Object.(api.Live))
	}))

	// 下播事件
	ed.AddEventListener(listeners.LiveEnd, events.NewEventListener(func(event *events.Event) {
		r.RemoveRecorder(ctx, event.Object.(api.Live).GetLiveId())
	}))

	// 监听关闭事件
	ed.AddEventListener(listeners.ListenStop, events.NewEventListener(func(event *events.Event) {
		r.RemoveRecorder(ctx, event.Object.(api.Live).GetLiveId())
	}))

	return nil
}

func (r *RecorderManager) Close(ctx context.Context) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for id, recorder := range r.savers {
		recorder.Close()
		delete(r.savers, id)
	}
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
	inst.Logger.Info("RecorderManager Closed")
}

func (r *RecorderManager) AddRecorder(ctx context.Context, live api.Live) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, ok := r.savers[live.GetLiveId()]; ok {
		return recorderExistError
	}
	recorder, err := NewRecorder(ctx, live)
	if err != nil {
		return err
	}
	r.savers[live.GetLiveId()] = recorder
	recorder.Start()
	return nil

}

func (r *RecorderManager) RemoveRecorder(ctx context.Context, liveId api.LiveId) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if recorder, ok := r.savers[liveId]; !ok {
		return recorderNotExistError
	} else {
		recorder.Close()
		delete(r.savers, liveId)
		return nil
	}
}

func (r *RecorderManager) GetRecorder(ctx context.Context, liveId api.LiveId) (*Recorder, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r, ok := r.savers[liveId]; !ok {
		return nil, recorderNotExistError
	} else {
		return r, nil
	}
}

func (r *RecorderManager) HasRecorder(ctx context.Context, liveId api.LiveId) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	_, ok := r.savers[liveId]
	return ok
}
