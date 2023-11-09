# atlantis-drift-detector
Runs a cronjob plan and draws some plots which can be useful to monitor infrastructure drift

## Settings
| Env name                               | Example                                                | Description                                  |
| -------------------------------------- | ------------------------------------------------------ | -------------------------------------------- |
| `DRIFT_DETECTOR_ALLOWLIST`             | "github.com/org/repo,github.com/org/repo2"             | List of your repositories separated by comma |
| `DRIFT_DETECTOR_GH_APP_SLUG`           | "drift-detector"                                       | Name of your Github App                      |
| `DRIFT_DETECTOR_GH_APP_ID`             | "123456"                                               | Github App ID                                |
| `DRIFT_DETECTOR_GH_APP_KEY_FILE`       | "key/key.pem"                                          | Path to the key file                         |
| `DRIFT_DETECTOR_GH_INSTALLATION_ID`    | "12345678"                                             | Github App Installation ID                   |
| `DRIFT_DETECTOR_CRON`                  | "* 17 * * *"                                           | Cron expression to run drift detection       |
| `DRIFT_DETECTOR_SLACK_CHANNEL`         | "drift-channel"                                        | Slack channel name                           |
| `DRIFT_DETECTOR_SLACK_TOKEN`           | "xoxb-xxx"                                             | Slack token                                  |

# Build
```bash
docker build -t repo/atlantis-drift-detector .
```

# Run
```bash
docker run -p 8080:8080 -d repo/atlantis-drift-detector:tag --env ...
```

The app is running on `localhost:8080/drift-detector/report`

<img width="1440" alt="image" src="https://github.com/ovceev/atlantis-drift-detector/assets/54960661/00ce428e-693a-4e01-87a9-eb49fa3d0cbf">
