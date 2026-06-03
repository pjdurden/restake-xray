package eigenlayer

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadConfigs reads the live LRT config set (configs/lrts.json).
func LoadConfigs(path string) ([]LRTConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfgs []LRTConfig
	if err := json.Unmarshal(b, &cfgs); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfgs, nil
}
