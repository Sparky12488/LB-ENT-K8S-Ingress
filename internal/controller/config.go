package controller

import (
	"os"
	"strings"
)

type Config struct {
	ApplianceIP string
	APIKey      string
	VIPPool     []string
	LBUser      string
	LBPass      string
}

func LoadConfig() *Config {
	return &Config{
		ApplianceIP: os.Getenv("LB_APPLIANCE_IP"),
		APIKey:      os.Getenv("API_KEY"),
		VIPPool:     strings.Split(os.Getenv("VIP_POOL"), ","),
		LBUser:      os.Getenv("LB_USER"),
		LBPass:      os.Getenv("LB_PASS"),
	}
}
