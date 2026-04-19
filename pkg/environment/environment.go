package environment

import (
	"os"
)

const (
	ENV_KEY_ENVIRONMENT = "ENVIRONMENT"
	ENV_KEY_VERSION     = "APP_VERSION"
	ENV_KEY_LOG_LEVEL   = "LOG_LEVEL"
	ENV_LOG_FORMAT      = "LOG_FORMAT"
)

func IsRunningLocally() bool {
	envsFoundInCI := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "JENKINS_URL"}
	for _, env := range envsFoundInCI {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

func GetEnvWithDefault(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}
