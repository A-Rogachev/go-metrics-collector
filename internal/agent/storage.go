package agent

import (
	"maps"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
	"github.com/shirou/gopsutil/v4/mem"
)

func pause(interval time.Duration) {
	time.Sleep(interval)
}

type AgentStorage struct {
	mutex    sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewAgentStorage() *AgentStorage {

	return &AgentStorage{
		gauges:   make(map[string]float64),
		counters: map[string]int64{"PollCount": int64(0)},
	}
}

func (s *AgentStorage) SetGauge(name string, value float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.gauges[name] = value
}

func (s *AgentStorage) AddCounter(name string, value int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.counters[name] += value
}

func (s *AgentStorage) GetGauge(key string) (float64, bool) {
	value, exist := s.gauges[key]
	if !exist {
		return float64(0), false
	}
	return value, true
}

func (s *AgentStorage) GetCounter(key string) (int64, bool) {
	value, exist := s.counters[key]
	if !exist {
		return int64(0), false
	}
	return value, true
}

func (s *AgentStorage) GetMetricsCount() (int, int) {
	return len(s.gauges), len(s.counters)
}

func (s *AgentStorage) Snapshot() models.MetricSnapshot {
	s.mutex.Lock()
	gauges, counters := maps.Clone(s.gauges), maps.Clone(s.counters)
	s.mutex.Unlock()
	return models.MetricSnapshot{Gauges: gauges, Counters: counters}
}

func (s *AgentStorage) SubtractCounters(ch <-chan map[string]int64) {
	for {
		sentCounters := <-ch
		s.mutex.Lock()
		for key, value := range sentCounters {
			s.counters[key] -= value
			if s.counters[key] < 0 {
				s.counters[key] = 0
			}
		}
		s.mutex.Unlock()
	}
}

func (s *AgentStorage) Collect(polling bool, interval time.Duration) {
	var wg sync.WaitGroup

	wg.Go(func() {
		for {
			m := runtime.MemStats{}
			runtime.ReadMemStats(&m)

			s.mutex.Lock()
			s.gauges["Alloc"] = float64(m.Alloc)
			s.gauges["BuckHashSys"] = float64(m.BuckHashSys)
			s.gauges["Frees"] = float64(m.Frees)
			s.gauges["GCCPUFraction"] = m.GCCPUFraction
			s.gauges["GCSys"] = float64(m.GCSys)
			s.gauges["HeapAlloc"] = float64(m.HeapAlloc)
			s.gauges["HeapIdle"] = float64(m.HeapIdle)
			s.gauges["HeapInuse"] = float64(m.HeapInuse)
			s.gauges["HeapObjects"] = float64(m.HeapObjects)
			s.gauges["HeapReleased"] = float64(m.HeapReleased)
			s.gauges["HeapSys"] = float64(m.HeapSys)
			s.gauges["LastGC"] = float64(m.LastGC)
			s.gauges["Lookups"] = float64(m.Lookups)
			s.gauges["MCacheInuse"] = float64(m.MCacheInuse)
			s.gauges["MCacheSys"] = float64(m.MCacheSys)
			s.gauges["MSpanInuse"] = float64(m.MSpanInuse)
			s.gauges["MSpanSys"] = float64(m.MSpanSys)
			s.gauges["Mallocs"] = float64(m.Mallocs)
			s.gauges["NextGC"] = float64(m.NextGC)
			s.gauges["NumForcedGC"] = float64(m.NumForcedGC)
			s.gauges["NumGC"] = float64(m.NumGC)
			s.gauges["OtherSys"] = float64(m.OtherSys)
			s.gauges["PauseTotalNs"] = float64(m.PauseTotalNs)
			s.gauges["StackInuse"] = float64(m.StackInuse)
			s.gauges["StackSys"] = float64(m.StackSys)
			s.gauges["Sys"] = float64(m.Sys)
			s.gauges["TotalAlloc"] = float64(m.TotalAlloc)
			s.gauges["RandomValue"] = rand.Float64()
			s.counters["PollCount"]++

			s.mutex.Unlock()
			if !polling { // однократное срабатывание при polling == false
				return
			}
			pause(interval)
		}
	})

	wg.Go(func() {
		for {
			v, _ := mem.VirtualMemory()

			s.mutex.Lock()
			s.gauges["TotalMemory"] = float64(v.Total)
			s.gauges["FreeMemory"] = float64(v.Free)
			s.gauges["CPUutilization1"] = float64(runtime.NumCPU())
			s.mutex.Unlock()

			if !polling { // однократное срабатывание при polling == false
				return
			}
			pause(interval)
		}
	})

	if !polling {
		wg.Wait()
	}
}

func (s *AgentStorage) PeriodicalSnapshot(
	ch chan models.MetricSnapshot,
	period time.Duration,
) {
	for {
		s.mutex.Lock()
		gauges, counters := maps.Clone(s.gauges), maps.Clone(s.counters)
		s.mutex.Unlock()

		ch <- models.MetricSnapshot{Gauges: gauges, Counters: counters}

		pause(period)
	}
}
