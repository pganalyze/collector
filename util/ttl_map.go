package util

import (
    "sync"
    "time"
)

type item struct {
    value     string
    createdAt int64
}

type TTLMap struct {
    m map[string]*item
    l sync.Mutex
}

func NewTTLMap(maxTTL int64, checkFrequency int64) (m *TTLMap) {
    m = &TTLMap{ m: make(map[string]*item) }
    go func() {
        for now := range time.Tick(time.Second * time.Duration(checkFrequency)) {
            m.l.Lock()
            for k, v := range m.m {
                if now.Unix() - v.createdAt > maxTTL {
                    delete(m.m, k)
                }
            }
            m.l.Unlock()
        }
    }()
    return
}

func (m *TTLMap) Len() int {
    return len(m.m)
}

func (m *TTLMap) Put(k, v string) {
    m.l.Lock()
    it, ok := m.m[k]
    if !ok {
        it = &item{value: v}
        m.m[k] = it
    }
    it.createdAt = time.Now().Unix()
    m.l.Unlock()
}

func (m *TTLMap) Get(k string) (v string) {
    m.l.Lock()
    if it, ok := m.m[k]; ok {
        v = it.value
    }
    m.l.Unlock()
    return
}
