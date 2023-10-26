package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// GetEnvWithDefault fetches the value of the environment variable named by the key.
// If the environment variable is not set, it returns the provided default value.
func GetEnvWithDefault(env, defaultValue string) string {
	if value, exists := os.LookupEnv(env); exists {
		return value
	}
	return defaultValue
}

func GetEnvStrict(env string) string {
	if value, exists := os.LookupEnv(env); exists {
		return value
	}
	log.Fatal("Environment variable " + env + " is not set")
	return ""
}

func InitEnvs() (string, string, string, string, string, string, string) {

	return GetEnvStrict("DRIFT_DETECTOR_ALLOWLIST"),
		GetEnvStrict("DRIFT_DETECTOR_GH_APP_SLUG"),
		GetEnvStrict("DRIFT_DETECTOR_GH_APP_ID"),
		GetEnvWithDefault("DRIFT_DETECTOR_GH_APP_KEY_FILE", "key.pem"),
		GetEnvStrict("DRIFT_DETECTOR_GH_INSTALLATION_ID"),
		GetEnvWithDefault("DRIFT_DETECTOR_CRON", "30 20 * * *"),
		GetEnvWithDefault("DRIFT_DETECTOR_MERGE_KUBECONFIGS", "false")

}

func InitSlackEnvs() (string, string) {

	return GetEnvWithDefault("DRIFT_DETECTOR_SLACK_CHANNEL", ""),
		GetEnvWithDefault("DRIFT_DETECTOR_SLACK_TOKEN", "")
}

func mergeKubeconfigs(files []string) (*clientcmdapi.Config, error) {
	mergedConfig := clientcmdapi.NewConfig()

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("failed to read kubeconfig from %s: %v", file, err)
			return nil, err
		}

		config, err := clientcmd.Load(content)
		if err != nil {
			log.Errorf("failed to load kubeconfig from %s: %v", file, err)
			return nil, err
		}

		for name, cluster := range config.Clusters {
			mergedConfig.Clusters[name] = cluster
		}

		for name, context := range config.Contexts {
			mergedConfig.Contexts[name] = context
		}

		for name, authInfo := range config.AuthInfos {
			mergedConfig.AuthInfos[name] = authInfo
		}
	}

	return mergedConfig, nil
}

func CreateKubeconfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Errorf("error determining home directory: %v\n", err)
	}
	kubeDir := filepath.Join(homeDir, ".kube")

	files, err := ioutil.ReadDir(kubeDir)
	if err != nil {
		log.Errorf("error reading ~/.kube directory: %v\n", err)
	}

	var kubeconfigs []string
	for _, file := range files {
		if strings.Contains(file.Name(), "kubeconfig") {
			kubeconfigs = append(kubeconfigs, filepath.Join(kubeDir, file.Name()))
		}
	}

	mergedConfig, err := mergeKubeconfigs(kubeconfigs)
	if err != nil {
		log.Errorf("error merging kubeconfig files: %v\n", err)
	}

	output, err := clientcmd.Write(*mergedConfig)
	if err != nil {
		log.Errorf("error writing merged kubeconfig: %v\n", err)
	}

	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	err = ioutil.WriteFile(kubeconfigPath, output, 0644)
	if err != nil {
		log.Errorf("error saving merged kubeconfig to %s: %v\n", kubeconfigPath, err)
	}

	log.Infof("merged kubeconfig saved to %s\n", kubeconfigPath)
}
