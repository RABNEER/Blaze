package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/tools/go/analysis/passes/nilfunc"
)

type State struct {
	Pid        int       `json:"pid"`
	ServerID   string    `json:"server_id"`
	ServerName string    `json:"server_name"`
	Hostname   string    `json:"hostname"`
	Proto      string    `json:"proto"`
	ConfigPath string    `json:"config_path"`
	StartedAt  time.Time `json:"started_at"`	
}

type CachedCreds struct {
	Username	string		`json:"username"`
	Password	string		`json:"password"`
	LastUpdated	string		`json:last_updated"`
	FetchedAt	time.Time	`json:"fetched_at"`	
}

func BaseDir() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == ""{
		home, _ := os.UserHomeDir()
		local = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(local, "BlazeVPN")
}

func ConfigsDir() string { return filepath.Join(BaseDir(), "configs")}
func StatePath() string { return filepath.Join(BaseDir(), "state.json")}
func CredsPath() string {return filepath.Join(BaseDir(), "creds.json")}
func StopFlagPath() string {filepath.Join(BaseDir(), "stop.flag")}

func EnsureDirs() error {
	for _, d := range []string {BaseDir(), ConfigsDir()} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func ClearState() { _= os.Remove(StatePath()) }

func LoadCreds() (*CachedCreds, error) {
	data, err := os.ReadFile(CredsPath())
	if err != nil {
		return nil, err
	}
	var c CachedCreds
	if err := json.Unmarshal(data, &c); err !=nil {
		return nil, err
	}
	return &c, nil
}

func SaveCreds(c *CachedCreds) error {
	data err := json.MarshalIndent(c, "", "	")
	if err != nil {
		return err
	}
	return os.WriteFile(CredsPath(), data. 0o600)
}