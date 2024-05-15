package classdata

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/influxdata/toml"
)

type Handler interface {
	GetDataForClass(name string) (string, bool)
	Update() error
}

type DirecroryHandler struct {
	data map[string]string
	path string
	mu   sync.RWMutex
}

func NewDirectoryHandler(path string) (*DirecroryHandler, error) {
	handler := &DirecroryHandler{
		path: path,
		data: make(map[string]string),
	}

	if err := handler.readClassData(); err != nil {
		return nil, fmt.Errorf("failed to read telegaf class data: %w", err)
	}

	if err := handler.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate telegraf class data: %w", err)
	}

	return handler, nil
}

func (h *DirecroryHandler) GetDataForClass(name string) (string, bool) {
	h.mu.RLock()
	data, ok := h.data[name]
	h.mu.RUnlock()

	return data, ok
}

func (h *DirecroryHandler) Update() error {
	// Make a copy of the current data in case the update fails
	cp := make(map[string]string)
	h.mu.RLock()
	for k, v := range h.data {
		cp[k] = v
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.readClassData(); err != nil {
		h.data = cp
		return fmt.Errorf("failed to update class data: %w", err)
	}

	if err := h.validate(); err != nil {
		h.data = cp
		return fmt.Errorf("validate to validate updated class data, reverting to previous: %w", err)
	}

	return nil
}

func (h *DirecroryHandler) validate() error {
	if len(h.data) == 0 {
		return fmt.Errorf("failed to validate class data, no data could be found")
	}

	for file, data := range h.data {
		if _, err := toml.Parse([]byte(data)); err != nil {
			return fmt.Errorf("failed to validate class data for file: %s, error: %w", file, err)
		}
	}

	return nil
}

func (h *DirecroryHandler) readClassData() error {
	files, err := os.ReadDir(h.path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %s, error: %w", h.path, err)
	}

	for _, file := range files {
		fpath := filepath.Join(h.path, file.Name())
		stat, err := os.Stat(fpath)
		if err != nil {
			return fmt.Errorf("failed to stat: %s, error: %w", file.Name(), err)
		}

		if stat.Mode().IsRegular() {
			data, err := os.ReadFile(fpath)
			if err != nil {
				return fmt.Errorf("failed to read data from file: %s, error: %w", file.Name(), err)
			}
			h.data[file.Name()] = string(data)
		}
	}

	return nil
}
