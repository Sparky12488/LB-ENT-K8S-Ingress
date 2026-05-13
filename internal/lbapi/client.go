package lbapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type LBRequest struct {
	LBCLI []map[string]string `json:"lbcli"`
}

type LBListResponse struct {
	LBAPI [][]struct {
		Iteration []map[string]interface{} `json:"itteration"`
	} `json:"lbapi"`
}

type Client struct {
	BaseURL    string
	APIKey     string
	LBUser     string
	LBPass     string
	HTTPClient *http.Client
}

// ExecuteAction performs the API call and returns the raw body for parsing
func (c *Client) ExecuteAction(action map[string]string) ([]byte, error) {
	payload := LBRequest{
		LBCLI: []map[string]string{action},
	}
	jsonData, _ := json.Marshal(payload)

	// Base64 encode the key (Matches echo -n | base64)
	encodedKey := base64.StdEncoding.EncodeToString([]byte(c.APIKey))

	req, err := http.NewRequest("POST", c.BaseURL+"/api/v2/", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.LBUser, c.LBPass)
	req.Header.Set("X-LB-APIKEY", encodedKey)
	req.Header.Set("X-API-KEY", encodedKey)

	fmt.Printf("DEBUG: API POST to %s/api/v2/\n", c.BaseURL)
	fmt.Printf("DEBUG: Payload: %s\n", string(jsonData))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		fmt.Printf("DEBUG: Connection Error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// This is the "Truth" line. If this prints, we are talking to the LB.
	fmt.Printf("DEBUG: Status: %d | Response: %s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("appliance error: %d - %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// SendAction is a helper for actions where we don't care about the response body
func (c *Client) SendAction(action map[string]string) error {
	_, err := c.ExecuteAction(action)
	return err
}

func (c *Client) ListVirtual(vipName string) ([]string, error) {
	action := map[string]string{
		"action":   "list",
		"function": "dumpconfig",
	}

	// 1. Get the raw bytes
	body, err := c.ExecuteAction(action)
	if err != nil {
		return nil, fmt.Errorf("failed to dump config: %w", err)
	}

	// 2. Unmarshal the bytes into our response struct
	var resp LBListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse dumpconfig JSON: %w", err)
	}

	var currentIPs []string

	// 3. Now we can navigate the struct (using resp.LBAPI)
	for _, apiLayer := range resp.LBAPI {
		for _, iterationWrap := range apiLayer {
			for _, item := range iterationWrap.Iteration {
				// Dig into haproxy -> virtual
				if haproxy, ok := item["haproxy"].(map[string]interface{}); ok {
					if virtual, ok := haproxy["virtual"].(map[string]interface{}); ok {
						// Check if this is the VIP we are looking for
						if label, ok := virtual["label"].(string); ok && label == vipName {
							// Find the 'real' servers array
							if realServers, ok := virtual["real"].([]interface{}); ok {
								for _, srv := range realServers {
									if srvMap, ok := srv.(map[string]interface{}); ok {
										if ip, ok := srvMap["server"].(string); ok {
											currentIPs = append(currentIPs, ip)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	fmt.Printf("DEBUG: Found %d existing RIPs on appliance for %s\n", len(currentIPs), vipName)
	return currentIPs, nil
}

func (c *Client) DeleteVIP(name string) error {
	return c.SendAction(map[string]string{"action": "delete-vip", "vip": name})
}

func (c *Client) DeleteRIP(vip, rip string) error {
	return c.SendAction(map[string]string{"action": "delete-rip", "vip": vip, "rip": rip})
}

func (c *Client) ReloadConfig() error {
	return c.SendAction(map[string]string{"action": "reload-haproxy"})
}
