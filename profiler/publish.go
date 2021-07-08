package profiler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"
)

var httpclient = &http.Client{
	Timeout:   10 * time.Second,
	Transport: http.DefaultTransport,
}

func (cfg *Config) detectTargetURL() string {
	if cfg.customTarget {
		return cfg.targetURL
	}
	// logic to detect default url
	// needed for deciding data sent to forwarder or agent
	return cfg.targetURL
}

func (cfg *Config) sendToAgent(ctx context.Context) {

	target := cfg.detectTargetURL()
	cfg.logf("sending profile data to url %s", target)

	for {
		select {
		case <-ctx.Done():
			return
		case d := <-cfg.out:
			// base64 encode pprof data
			d.Data = []byte(base64.StdEncoding.EncodeToString(d.Data))
			body, err := json.Marshal(d)
			if err != nil {
				cfg.logf("failed to marshal %s profile data %s", d.Type, err)
				break
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
			if err != nil {
				cfg.logf("failed to create %s profile request %s", d.Type, err)
				break
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := httpclient.Do(req)
			if err != nil {
				cfg.logf("failed to send %s profile data, %s", d.Type, err)
				break
			}
			if resp.StatusCode != http.StatusOK {
				cfg.logf("failed to send %s profile collected at %d response %s", d.Type, d.Timestamp, resp.Status)
			} else {
				cfg.logf("sent %s profile collected at %d response %s", d.Type, d.Timestamp, resp.Status)
			}
		}
	}
}

func (cfg *Config) writeToFile(ctx context.Context) {
	// start remove routine
	go cfg.removeOldFiles(ctx)

	// create directory
	_, err := os.Stat(DefaultProfilesDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(DefaultProfilesDir, 0644)
		if err != nil {
			cfg.logf("failed to create directory %s", err)
		}
	} else {
		cfg.logf("failed to list directory %s", err)
	}

	// write received profiles to file
	// file name service_timestamp_pid.profiletype
	for {
		select {
		case <-ctx.Done():
			return
		case d := <-cfg.out:
			file := path.Join(
				DefaultProfilesDir,
				fmt.Sprintf("%s_%d_%d.%s", cfg.service, d.Timestamp, d.PID, d.Type),
			)
			err := ioutil.WriteFile(file, d.Data, 0644)
			if err != nil {
				cfg.logf("failed to write profile %s, %s", d.Type, err)
			}
		}
	}
}

func (cfg *Config) removeOldFiles(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			files, err := ioutil.ReadDir(DefaultProfilesDir)
			if err != nil {
				cfg.logf("failed to read directory %s", err)
			}
			for _, f := range files {
				if time.Now().Sub(f.ModTime()) > DefaultProfilesAge {
					err := os.Remove(path.Join(DefaultProfilesDir, f.Name()))
					if err != nil {
						cfg.logf("failed to remove file %s", err)
					}
				}
			}
		}
	}
}
