package agent

import (
	"maps"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

const BatchSize = 50

type AgentStorage struct {
	mutex    sync.Mutex
	Gauges   map[string]float64
	Counters map[string]int64
}

func NewAgentStorage() *AgentStorage {
	return &AgentStorage{
		Gauges:   make(map[string]float64),
		Counters: map[string]int64{"PollCount": 0},
	}
}

func (s *AgentStorage) Snapshot() models.MetricSnapshot {
	s.mutex.Lock()
	gauges, counters := maps.Clone(s.Gauges), maps.Clone(s.Counters)
	s.mutex.Unlock()
	return models.MetricSnapshot{Gauges: gauges, Counters: counters}
}

func (s *AgentStorage) SubtractCounters(sentCounters map[string]int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for key, value := range sentCounters {
		s.Counters[key] -= value
		if s.Counters[key] < 0 {
			s.Counters[key] = 0
		}
	}
}

func (s *AgentStorage) Collect(polling bool, interval int) {
	for {
		m := runtime.MemStats{}
		runtime.ReadMemStats(&m)
		s.mutex.Lock()
		s.Gauges = map[string]float64{
			"Alloc":         float64(m.Alloc),
			"BuckHashSys":   float64(m.BuckHashSys),
			"Frees":         float64(m.Frees),
			"GCCPUFraction": m.GCCPUFraction,
			"GCSys":         float64(m.GCSys),
			"HeapAlloc":     float64(m.HeapAlloc),
			"HeapIdle":      float64(m.HeapIdle),
			"HeapInuse":     float64(m.HeapInuse),
			"HeapObjects":   float64(m.HeapObjects),
			"HeapReleased":  float64(m.HeapReleased),
			"HeapSys":       float64(m.HeapSys),
			"LastGC":        float64(m.LastGC),
			"Lookups":       float64(m.Lookups),
			"MCacheInuse":   float64(m.MCacheInuse),
			"MCacheSys":     float64(m.MCacheSys),
			"MSpanInuse":    float64(m.MSpanInuse),
			"MSpanSys":      float64(m.MSpanSys),
			"Mallocs":       float64(m.Mallocs),
			"NextGC":        float64(m.NextGC),
			"NumForcedGC":   float64(m.NumForcedGC),
			"NumGC":         float64(m.NumGC),
			"OtherSys":      float64(m.OtherSys),
			"PauseTotalNs":  float64(m.PauseTotalNs),
			"StackInuse":    float64(m.StackInuse),
			"StackSys":      float64(m.StackSys),
			"Sys":           float64(m.Sys),
			"TotalAlloc":    float64(m.TotalAlloc),
			"RandomValue":   rand.Float64(),
		}
		s.Counters["PollCount"]++
		s.mutex.Unlock()
		if !polling {
			return
		}
		time.Sleep(time.Second * time.Duration(interval))
	}
}
