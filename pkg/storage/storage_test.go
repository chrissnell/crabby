package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/chrissnell/crabby/pkg/job"
)

// mockBackend implements Backend only (no MetricSender or EventSender).
type mockBackend struct {
	name    string
	started bool
	closed  bool
	startFn func(context.Context) error
	closeFn func() error
}

func (m *mockBackend) Name() string { return m.name }
func (m *mockBackend) Start(ctx context.Context) error {
	m.started = true
	if m.startFn != nil {
		return m.startFn(ctx)
	}
	return nil
}
func (m *mockBackend) Close() error {
	m.closed = true
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

// mockMetricBackend implements Backend + MetricSender.
type mockMetricBackend struct {
	mockBackend
	metrics []job.Metric
}

func (m *mockMetricBackend) SendMetric(_ context.Context, met job.Metric) error {
	m.metrics = append(m.metrics, met)
	return nil
}

// mockEventBackend implements Backend + EventSender.
type mockEventBackend struct {
	mockBackend
	events []job.Event
}

func (m *mockEventBackend) SendEvent(_ context.Context, e job.Event) error {
	m.events = append(m.events, e)
	return nil
}

// mockFullBackend implements Backend + MetricSender + EventSender.
type mockFullBackend struct {
	mockBackend
	metrics []job.Metric
	events  []job.Event
}

func (m *mockFullBackend) SendMetric(_ context.Context, met job.Metric) error {
	m.metrics = append(m.metrics, met)
	return nil
}

func (m *mockFullBackend) SendEvent(_ context.Context, e job.Event) error {
	m.events = append(m.events, e)
	return nil
}

func TestDistributor_AddBackend(t *testing.T) {
	d := NewDistributor()
	if len(d.backends) != 0 {
		t.Fatalf("expected 0 backends, got %d", len(d.backends))
	}
	d.AddBackend(&mockBackend{name: "a"})
	d.AddBackend(&mockBackend{name: "b"})
	if len(d.backends) != 2 {
		t.Fatalf("expected 2 backends, got %d", len(d.backends))
	}
}

func TestDistributor_Start(t *testing.T) {
	tests := []struct {
		name    string
		backends []Backend
		wantErr bool
	}{
		{
			name:    "all succeed",
			backends: []Backend{&mockBackend{name: "a"}, &mockBackend{name: "b"}},
		},
		{
			name: "first fails",
			backends: []Backend{
				&mockBackend{name: "fail", startFn: func(_ context.Context) error {
					return errors.New("boom")
				}},
				&mockBackend{name: "ok"},
			},
			wantErr: true,
		},
		{
			name:     "empty backends",
			backends: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDistributor()
			for _, b := range tt.backends {
				d.AddBackend(b)
			}
			err := d.Start(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDistributor_Start_calls_all_backends(t *testing.T) {
	a := &mockBackend{name: "a"}
	b := &mockBackend{name: "b"}
	d := NewDistributor()
	d.AddBackend(a)
	d.AddBackend(b)

	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.started || !b.started {
		t.Errorf("expected both backends started, a=%v b=%v", a.started, b.started)
	}
}

func TestDistributor_Close(t *testing.T) {
	tests := []struct {
		name    string
		backends []Backend
		wantErr bool
	}{
		{
			name:    "all succeed",
			backends: []Backend{&mockBackend{name: "a"}, &mockBackend{name: "b"}},
		},
		{
			name: "first fails returns error",
			backends: []Backend{
				&mockBackend{name: "fail", closeFn: func() error { return errors.New("close error") }},
				&mockBackend{name: "ok"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDistributor()
			for _, b := range tt.backends {
				d.AddBackend(b)
			}
			err := d.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDistributor_Close_calls_all_backends(t *testing.T) {
	a := &mockBackend{name: "a"}
	b := &mockBackend{name: "b"}
	d := NewDistributor()
	d.AddBackend(a)
	d.AddBackend(b)

	d.Close()
	if !a.closed || !b.closed {
		t.Errorf("expected both backends closed, a=%v b=%v", a.closed, b.closed)
	}
}

func TestDistributor_SendMetrics(t *testing.T) {
	metricOnly := &mockMetricBackend{mockBackend: mockBackend{name: "metric"}}
	eventOnly := &mockEventBackend{mockBackend: mockBackend{name: "event"}}
	full := &mockFullBackend{mockBackend: mockBackend{name: "full"}}
	plain := &mockBackend{name: "plain"}

	d := NewDistributor()
	d.AddBackend(metricOnly)
	d.AddBackend(eventOnly)
	d.AddBackend(full)
	d.AddBackend(plain)

	metrics := []job.Metric{
		{Job: "j1", Timing: "dns", Value: 1.0},
		{Job: "j2", Timing: "connect", Value: 2.0},
	}
	d.SendMetrics(context.Background(), metrics)

	if len(metricOnly.metrics) != 2 {
		t.Errorf("metricOnly got %d metrics, want 2", len(metricOnly.metrics))
	}
	if len(eventOnly.events) != 0 {
		t.Errorf("eventOnly should have 0 events, got %d", len(eventOnly.events))
	}
	if len(full.metrics) != 2 {
		t.Errorf("full got %d metrics, want 2", len(full.metrics))
	}
}

func TestDistributor_SendEvents(t *testing.T) {
	metricOnly := &mockMetricBackend{mockBackend: mockBackend{name: "metric"}}
	eventOnly := &mockEventBackend{mockBackend: mockBackend{name: "event"}}
	full := &mockFullBackend{mockBackend: mockBackend{name: "full"}}
	plain := &mockBackend{name: "plain"}

	d := NewDistributor()
	d.AddBackend(metricOnly)
	d.AddBackend(eventOnly)
	d.AddBackend(full)
	d.AddBackend(plain)

	events := []job.Event{
		{Name: "e1", ServerStatus: 200},
		{Name: "e2", ServerStatus: 500},
	}
	d.SendEvents(context.Background(), events)

	if len(eventOnly.events) != 2 {
		t.Errorf("eventOnly got %d events, want 2", len(eventOnly.events))
	}
	if len(metricOnly.metrics) != 0 {
		t.Errorf("metricOnly should have 0 metrics, got %d", len(metricOnly.metrics))
	}
	if len(full.events) != 2 {
		t.Errorf("full got %d events, want 2", len(full.events))
	}
}
