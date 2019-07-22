package listeners

import (
	"context"
	"sync"

	"github.com/xyjunco/bililive-go/src/api"
	"github.com/xyjunco/bililive-go/src/instance"
)

func NewIListenerManager(ctx context.Context) IListenerManager {
	lm := &ListenerManager{
		savers: make(map[api.LiveId]*Listener),
		lock:   new(sync.RWMutex),
	}
	instance.GetInstance(ctx).ListenerManager = lm
	return lm
}

// 监听管理器接口
type IListenerManager interface {
	AddListener(ctx context.Context, live api.Live) error
	RemoveListener(ctx context.Context, liveId api.LiveId) error
	GetListener(ctx context.Context, liveId api.LiveId) (*Listener, error)
	HasListener(ctx context.Context, liveId api.LiveId) bool
}

type ListenerManager struct {
	savers map[api.LiveId]*Listener
	lock   *sync.RWMutex
}

func (l *ListenerManager) Start(ctx context.Context) error {
	inst := instance.GetInstance(ctx)
	if inst.Config.RPC.Enable || len(inst.Lives) > 0 {
		inst.WaitGroup.Add(1)
	}
	instance.GetInstance(ctx).Logger.Info("ListenerManager Start")
	return nil
}

func (l *ListenerManager) Close(ctx context.Context) {
	l.lock.Lock()
	defer l.lock.Unlock()
	for id, listener := range l.savers {
		listener.Close()
		delete(l.savers, id)
	}
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
	instance.GetInstance(ctx).Logger.Info("ListenerManager Closed")
}

func (l *ListenerManager) AddListener(ctx context.Context, live api.Live) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if _, ok := l.savers[live.GetLiveId()]; ok {
		return listenerExistError
	}
	listener := NewListener(ctx, live)
	listener.Start()
	l.savers[live.GetLiveId()] = listener
	return nil
}

func (l *ListenerManager) RemoveListener(ctx context.Context, liveId api.LiveId) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if listener, ok := l.savers[liveId]; !ok {
		return listenerNotExistError
	} else {
		listener.Close()
		delete(l.savers, liveId)
		return nil
	}
}

func (l *ListenerManager) GetListener(ctx context.Context, liveId api.LiveId) (*Listener, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	if r, ok := l.savers[liveId]; !ok {
		return nil, listenerNotExistError
	} else {
		return r, nil
	}
}

func (l *ListenerManager) HasListener(ctx context.Context, liveId api.LiveId) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	_, ok := l.savers[liveId]
	return ok
}
