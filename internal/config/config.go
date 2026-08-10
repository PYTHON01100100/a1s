package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Region       string
	Profile      string
	Currency     string
	SARPerUSD    float64
	ReadOnly     bool
	Demo         bool
	AIBaseURL    string
	AIAPIKey     string
	AIModel      string
	AliyunConfig string
	ConfigSource string
}

type aliyunDiskConfig struct {
	Current  string `json:"current"`
	Profiles []struct {
		Name     string `json:"name"`
		RegionID string `json:"region_id"`
	} `json:"profiles"`
}

func FromEnv() Config {
	fx := 3.75
	if v := os.Getenv("A1S_SAR_PER_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			fx = f
		}
	}

	currency := strings.ToUpper(os.Getenv("A1S_CURRENCY"))
	if currency == "" {
		currency = "USD"
	}

	profile := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_PROFILE"))
	region := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_REGION_ID"))
	source := "environment"
	path := ""

	// If profile/region were not explicitly supplied, reuse the current
	// aliyun-cli profile from disk. We only read profile metadata here;
	// aliyun-cli remains responsible for credentials/authentication.
	if profile == "" || region == "" {
		if detected, detectedPath, ok := detectAliyunCLIConfig(); ok {
			if profile == "" {
				profile = detected.Current
			}
			if region == "" {
				region = regionForProfile(detected, profile)
			}
			path = detectedPath
			source = "aliyun-cli"
		}
	}

	return Config{
		Region:       region,
		Profile:      profile,
		Currency:     currency,
		SARPerUSD:    fx,
		AIBaseURL:    os.Getenv("A1S_AI_BASE_URL"),
		AIAPIKey:     os.Getenv("A1S_AI_API_KEY"),
		AIModel:      os.Getenv("A1S_AI_MODEL"),
		AliyunConfig: path,
		ConfigSource: source,
	}
}

func regionForProfile(c aliyunDiskConfig, profile string) string {
	if profile == "" {
		profile = c.Current
	}
	for _, p := range c.Profiles {
		if p.Name == profile {
			return strings.TrimSpace(p.RegionID)
		}
	}
	return ""
}

func detectAliyunCLIConfig() (aliyunDiskConfig, string, bool) {
	var zero aliyunDiskConfig
	seen := map[string]bool{}
	var candidates []string

	// Official credentials-go supports an explicit config path via this env var.
	if p := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_CREDENTIALS_FILE")); p != "" {
		candidates = append(candidates, p)
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".aliyun", "config.json"))
	}
	if up := strings.TrimSpace(os.Getenv("USERPROFILE")); up != "" {
		candidates = append(candidates, filepath.Join(up, ".aliyun", "config.json"))
	}

	// WSL convenience: if aliyun-cli was configured on Windows, reuse its
	// current profile metadata from the mounted Windows user home.
	if matches, _ := filepath.Glob("/mnt/c/Users/*/.aliyun/config.json"); len(matches) > 0 {
		candidates = append(candidates, matches...)
	}

	for _, p := range candidates {
		p = filepath.Clean(p)
		if seen[p] {
			continue
		}
		seen[p] = true

		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var c aliyunDiskConfig
		if err := json.Unmarshal(b, &c); err != nil {
			continue
		}
		if c.Current == "" && len(c.Profiles) == 0 {
			continue
		}
		return c, p, true
	}
	return zero, "", false
}
