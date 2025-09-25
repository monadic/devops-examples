package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ConfigHubSpace represents a space from ConfigHub
type ConfigHubSpace struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Labels map[string]string `json:"labels"`
}

// ConfigHubUnit represents a unit from ConfigHub
type ConfigHubUnit struct {
	Name  string `json:"name"`
	Space string `json:"space"`
	Type  string `json:"type"`
}

// GetConfigHubSpaces dynamically fetches spaces from ConfigHub
func GetConfigHubSpaces() ([]string, error) {
	// Use cub CLI to get spaces (without --json as it might not work)
	cmd := exec.Command("cub", "space", "list")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to environment variable if provided
		if envSpace := os.Getenv("CUB_SPACE"); envSpace != "" {
			return []string{envSpace}, nil
		}
		return []string{}, fmt.Errorf("failed to get spaces: %v", err)
	}

	// Parse the table output (skip header line)
	lines := strings.Split(string(output), "\n")
	var spaceNames []string
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header and empty lines
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && !strings.Contains(fields[0], "---") {
			spaceNames = append(spaceNames, fields[0])
		}
	}

	// If we couldn't get any spaces, use env variable
	if len(spaceNames) == 0 {
		if envSpace := os.Getenv("CUB_SPACE"); envSpace != "" {
			return []string{envSpace}, nil
		}
	}

	return spaceNames, nil
}

// GetConfigHubUnits dynamically fetches units from a space
func GetConfigHubUnits(space string) ([]string, error) {
	// Use cub CLI to get units (without --json)
	cmd := exec.Command("cub", "unit", "list", "--space", space)
	output, err := cmd.Output()
	if err != nil {
		return []string{}, fmt.Errorf("failed to get units: %v", err)
	}

	// Parse the table output (skip header line)
	lines := strings.Split(string(output), "\n")
	var unitNames []string
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header and empty lines
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && !strings.Contains(fields[0], "---") {
			unitNames = append(unitNames, fields[0])
		}
	}

	return unitNames, nil
}

// GetDynamicConfigHubInfo fetches ConfigHub info dynamically
func GetDynamicConfigHubInfo() ConfigHubInfo {
	info := ConfigHubInfo{
		Connected: false,
	}

	// Get spaces
	spaces, err := GetConfigHubSpaces()
	if err == nil && len(spaces) > 0 {
		info.Spaces = spaces
		info.Connected = true

		// Get units from the first space or env-specified space
		targetSpace := os.Getenv("CUB_SPACE")
		if targetSpace == "" && len(spaces) > 0 {
			targetSpace = spaces[0]
		}

		if targetSpace != "" {
			units, err := GetConfigHubUnits(targetSpace)
			if err == nil {
				info.Units = units
			}
		}
	}

	return info
}