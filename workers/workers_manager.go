package workers

import (
	"github.com/gynshu/workerpool/entities"
)

type Manager interface {
	PauseAll()
	ResumeAll()
	StopAll()
	Report() (running, idle int)
}

type manager struct {
	workers map[int]entities.Worker
}

func (m *manager) PauseAll() {
	for _, w := range m.workers {
		w.Toggle()
	}
}
func (m *manager) ResumeAll() {
	m.PauseAll()
}
func (m *manager) StopAll() {
	for _, w := range m.workers {
		w.Stop()
	}
}

func (m *manager) Report() (running, idle int) {
	for i := range m.workers {
		if m.workers[i].Busy() {
			running++
		} else {
			idle++
		}
	}
	return
}

func NewManager(workers map[int]entities.Worker) Manager {
	return &manager{
		workers: workers,
	}
}
