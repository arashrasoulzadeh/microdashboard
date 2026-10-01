package monitor

import (
	"io"
	"log"
	"net/http"
	"time"

	"microdashboard/internal/store"
)

type CheckResult struct {
	ID        string
	ElapsedMs int64
	Status    int
	BodyBytes int
	Error     error
}

type Monitor struct {
	store    *store.Store
	interval time.Duration
	stopChan chan struct{}
}

func Start(s *store.Store) {
	m := &Monitor{
		store:    s,
		interval: 30 * time.Second,
		stopChan: make(chan struct{}),
	}
	go m.run()
}

func (m *Monitor) run() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAll()
		case <-m.stopChan:
			return
		}
	}
}

func (m *Monitor) checkAll() {
	rows, err := m.store.GetAllMonitors()
	if err != nil {
		log.Printf("failed to query monitors: %v", err)
		return
	}

	for _, mon := range rows {
		result := m.checkOne(mon.ID, mon.URL, mon.Method, mon.Timeout)
		m.store.InsertMetric("", result.ID, int(result.ElapsedMs), result.Status, result.BodyBytes)

		if result.Error != nil {
			log.Printf("monitor %s error: %v", result.ID, result.Error)
		} else {
			log.Printf("monitor %s: elapsed=%dms status=%d", result.ID, result.ElapsedMs, result.Status)
		}
	}
}

func (m *Monitor) checkOne(id, url, method string, timeout int) *CheckResult {
	start := time.Now()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return &CheckResult{
			ID:        id,
			Error:     err,
			ElapsedMs: time.Since(start).Milliseconds(),
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return &CheckResult{
			ID:        id,
			Error:     err,
			ElapsedMs: elapsed,
			Status:    0,
			BodyBytes: 0,
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	return &CheckResult{
		ID:        id,
		ElapsedMs: elapsed,
		Status:    resp.StatusCode,
		BodyBytes: len(body),
	}
}