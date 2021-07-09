package profiler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
		case p := <-cfg.outProfile:
			// base64 encode pprof data
			p.Data = []byte(base64.StdEncoding.EncodeToString(p.Data))

			err := pushToAgent(ctx, target, p)
			if err != nil {
				cfg.logf("failed to send %s profile, error: %s", p.Type, err)
			} else {
				cfg.logf("sent %s profile collected at %d", p.Type, p.Timestamp)
			}

		case m := <-cfg.outMetrics:
			err := pushToAgent(ctx, target, m)
			if err != nil {
				cfg.logf("failed to send metrics collected at %s, error: %s", m.Timestamp, err)
			} else {
				cfg.logf("sent metrics collected at %d", m.Timestamp)
			}

		}
	}
}

func pushToAgent(ctx context.Context, target string, data interface{}) error {

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to send data " + resp.Status)
	}

	return nil
}

func (cfg *Config) writeToFile(ctx context.Context) {
	// start remove routine
	go cfg.removeOldFiles(ctx)

	// create directory
	_, err := os.Stat(DefaultProfilesDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(DefaultProfilesDir, 0755)
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
		case p := <-cfg.outProfile:
			file := path.Join(
				DefaultProfilesDir,
				fmt.Sprintf("%s_%d_%d.%s", cfg.service, p.Timestamp, p.PID, p.Type),
			)
			err := ioutil.WriteFile(file, p.Data, 0644)
			if err != nil {
				cfg.logf("failed to write profile %s, %s", p.Type, err)
			}

		case m := <-cfg.outMetrics:
			file := path.Join(
				DefaultProfilesDir,
				fmt.Sprintf("%s_%d_%d.json", cfg.service, m.Timestamp, m.PID),
			)
			data, err := json.MarshalIndent(m, "", "  ")
			if err != nil {
				cfg.logf("failed to marshal metrics data %s", err)
				break
			}
			err = ioutil.WriteFile(file, data, 0644)
			if err != nil {
				cfg.logf("failed to write metrics %s", err)
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
