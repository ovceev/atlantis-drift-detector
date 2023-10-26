package drift

import (
	"atlantis-drift-detector/notifier"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	httpauth "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

type AuthTokenClaim struct {
	*jwt.StandardClaims
}

type InstallationAuthResponse struct {
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expires_at"`
	Permissions struct {
		Checks       string `json:"checks"`
		Contents     string `json:"contents"`
		Deployments  string `json:"deployments"`
		Metadata     string `json:"metadata"`
		PullRequests string `json:"pull_requests"`
		Statuses     string `json:"statuses"`
	} `json:"permissions"`
	RepositorySelection string `json:"repository_selection"`
}

func DetectDrift(repoList []string, ghAppSlug, ghAppId, ghAppKeyFile, ghInstallationId string) {

	const maxConcurrentGoroutines = 12
	semaphore := make(chan struct{}, maxConcurrentGoroutines)

	for _, repo := range repoList {

		repoFolder := strings.Split(repo, "/")[2]
		err := cloneRepo(repo, repoFolder, ghAppId, ghAppKeyFile, ghInstallationId)
		if err != nil {
			log.Errorf("error cloning repo: %v", err)
		}

		log.Info("looking for some drifts in " + repoFolder)

		driftFolders, err := findTerragruntDirs(repoFolder)
		if err != nil {
			log.Warnf("error finding Terragrunt directories: %v", err)
			continue
		}

		// Channels to collect results from goroutines.
		errorCh := make(chan string, len(driftFolders))
		driftedCh := make(chan string, len(driftFolders))
		freshCh := make(chan string, len(driftFolders))

		// Wait group to wait for all goroutines to finish.
		var wg sync.WaitGroup

		for _, driftFolder := range driftFolders {
			if strings.Split(driftFolder, "/")[1] != "prod" && strings.Split(driftFolder, "/")[1] != "dev" {
				continue
			}

			wg.Add(1)
			semaphore <- struct{}{} // Acquire
			go func(df string) {    // Start a new goroutine.
				defer func() {
					<-semaphore // Release
					wg.Done()
				}()

				drifted, err := planRun(repoFolder, df)
				if err != nil {
					errorCh <- df
					return
				}
				if drifted {
					driftedCh <- df
				} else {
					freshCh <- df
				}
			}(driftFolder)

		}

		// Wait for all the goroutines to finish.
		wg.Wait()
		close(errorCh)
		close(driftedCh)
		close(freshCh)

		// Convert channels to slices.
		errorProjects := make([]string, 0, len(driftFolders))
		for err := range errorCh {
			errorProjects = append(errorProjects, err)
		}
		driftedProjects := make([]string, 0, len(driftFolders))
		for dp := range driftedCh {
			driftedProjects = append(driftedProjects, dp)
		}
		freshProjects := make([]string, 0, len(driftFolders))
		for fp := range freshCh {
			freshProjects = append(freshProjects, fp)
		}

		err = os.RemoveAll(repoFolder)
		if err != nil {
			log.Warnf("error removing directory: %v", err)
		}
		notifier.Notify(repoFolder, driftedProjects, errorProjects, freshProjects)
	}
}

// findTerragruntDirs walks through the file tree starting from rootDir and
// returns a slice of directories that contain terragrunt.hcl.
func findTerragruntDirs(rootDir string) ([]string, error) {
	var dirs []string

	// Walk through each file and directory in the tree.
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// If the item is a directory and it contains terragrunt.hcl, add it to the slice.
		if info.IsDir() {
			if _, err := os.Stat(filepath.Join(path, "terragrunt.hcl")); err == nil {
				dirs = append(dirs, path)
			}
		}
		return nil
	})

	// Return the slice and any error encountered during the walk.
	return dirs, err
}

func cloneRepo(repo, repoFolder, ghAppId, ghAppKeyFile, ghInstallationId string) error {
	keyBytes, err := os.ReadFile(ghAppKeyFile)
	if err != nil {
		log.Warnf("error reading key: %v", err)
	}

	rsaPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		log.Warnf("error parsing RSA private key from pem: %v", err)
	}

	jwtToken := jwt.New(jwt.SigningMethodRS256)

	jwtToken.Claims = &AuthTokenClaim{
		&jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute * 1).Unix(),
			Issuer:    ghAppId,
		},
	}

	tokenString, err := jwtToken.SignedString(rsaPrivateKey)
	if err != nil {
		log.Warnf("error getting jwt token string: %v", err)
	}

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://api.github.com/app/installations/"+ghInstallationId+"/access_tokens", nil)
	req.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
	req.Header.Set("Authorization", "Bearer "+tokenString)
	res, err := client.Do(req)
	if err != nil {
		log.Warnf("error making github api request: %v", err)
		return err
	}

	decoder := json.NewDecoder(res.Body)
	var installationAuthResponse InstallationAuthResponse
	err = decoder.Decode(&installationAuthResponse)
	if err != nil {
		log.Warnf("error decoding auth response: %v", err)
	}

	//fmt.Printf("Token: %s\n", installationAuthResponse.Token)
	r, err := git.PlainClone(repoFolder, false, &git.CloneOptions{
		URL: "https://" + repo + ".git",
		Auth: &httpauth.BasicAuth{
			Username: "x-access-token", // Yes, this can be anything except an empty string.
			Password: installationAuthResponse.Token,
		},
	})

	if err != nil {
		log.Warnf("error cloning repository: %v", err)
		return err
	}

	_, err = r.Head()
	if err != nil {
		log.Warnf("error verifying repository was cloned correctly: %v", err)
		return err
	}

	return nil
}

func planRun(repoFolder, driftFolder string) (drifted bool, err error) {

	// Run plan
	log.Debug("running plan in " + driftFolder)
	cmdPlan := exec.Command("terragrunt", "plan", "-lock=false", "-out=tfplan.out")
	cmdPlan.Dir = driftFolder
	cmdPlan.Env = os.Environ()
	if strings.Contains(driftFolder, "prod") {
		cmdPlan.Env = append(cmdPlan.Env, "AWS_PROFILE=prod")
		cmdPlan.Env = append(cmdPlan.Env, "TF_VAR_aws_profile=prod")
	} else if strings.Contains(driftFolder, "dev") {
		cmdPlan.Env = append(cmdPlan.Env, "AWS_PROFILE=dev")
		cmdPlan.Env = append(cmdPlan.Env, "TF_VAR_aws_profile=dev")
	}

	out, err := cmdPlan.Output()
	if err != nil {
		log.Infof("error project %s: %s", driftFolder, err)
		return false, err
	}

	// Clean up
	cmdCleanUp := exec.Command("rm", "-rf", ".terragrunt-cache")
	cmdCleanUp.Dir = driftFolder
	err = cmdCleanUp.Run()
	if err != nil {
		log.Warnf("error cleaning cache: %s", err)
	}

	return parsePlanOutput(out, driftFolder)
}

func parsePlanOutput(out []byte, project string) (bool, error) {
	var drifted bool
	var err error

	if strings.Contains(string(out), "Error:") {
		log.Infof("error project %s\n", project)
		err = fmt.Errorf("error running plan for project %s", project)
	}
	if strings.Contains(string(out), "Terraform will perform the following actions:") {
		log.Infof("drifted project %s", project)
		drifted = true
	}
	if strings.Contains(string(out), "and found no differences, so no changes are needed.") {
		log.Infof("fresh project %s", project)
		drifted = false
		err = nil
	}

	return drifted, err
}
