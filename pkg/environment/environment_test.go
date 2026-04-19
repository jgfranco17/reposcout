package environment

import (
	"os"
	"testing"
)

func TestIsRunningLocally(t *testing.T) {
	// Save original environment
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "JENKINS_URL"}
	originalValues := make(map[string]string)
	for _, v := range ciVars {
		originalValues[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range originalValues {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	// Test: no CI variables set → should return false
	result := IsRunningLocally()
	if result {
		t.Errorf("IsRunningLocally() = %v, want false when no CI vars set", result)
	}

	// Test: CI variable set → should return true
	os.Setenv("CI", "true")
	result = IsRunningLocally()
	if !result {
		t.Errorf("IsRunningLocally() = %v, want true when CI is set", result)
	}

	// Test: GITHUB_ACTIONS set → should return true
	os.Unsetenv("CI")
	os.Setenv("GITHUB_ACTIONS", "true")
	result = IsRunningLocally()
	if !result {
		t.Errorf("IsRunningLocally() = %v, want true when GITHUB_ACTIONS is set", result)
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	const testKey = "TEST_REPOSCOUT_ENV_KEY"
	const defaultVal = "default_value"
	const envVal = "env_value"

	// Ensure the test key is not set
	defer os.Unsetenv(testKey)

	// Test: env var not set → should return default
	result := GetEnvWithDefault(testKey, defaultVal)
	if result != defaultVal {
		t.Errorf("GetEnvWithDefault() = %q, want %q when env var not set", result, defaultVal)
	}

	// Test: env var set → should return env value
	os.Setenv(testKey, envVal)
	result = GetEnvWithDefault(testKey, defaultVal)
	if result != envVal {
		t.Errorf("GetEnvWithDefault() = %q, want %q when env var is set", result, envVal)
	}

	// Test: env var set to empty string → should return default
	os.Setenv(testKey, "")
	result = GetEnvWithDefault(testKey, defaultVal)
	if result != defaultVal {
		t.Errorf("GetEnvWithDefault() = %q, want %q when env var is empty", result, defaultVal)
	}
}
