package models

const (
	Counter = "counter"
	Gauge   = "gauge"
)

type MetricSnapshot struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

// Delta и Value объявлены через указатели, чтобы отличать значение "0", от не заданного значения
type Metric struct {
	ID    string   `json:"id" validate:"required"`
	MType string   `json:"type" validate:"required,oneof=gauge counter"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type GetMetricRequest struct {
	ID    string `json:"id" validate:"required"`
	MType string `json:"type" validate:"required,oneof=gauge counter"`
}
