// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

func getIniPath() string {
	iniPath, err := os.UserHomeDir()
	if err != nil {
		iniPath = "."
	}
	return iniPath + string(os.PathSeparator) + IniName
}

func LoadIni(createOnMissing bool) *ini.File {
	cfg, err := ini.Load(getIniPath())
	if err != nil {
		if !createOnMissing {
			log.Printf("Failed to read ini file: %v\n", err)
			os.Exit(1)
		}
		return ini.Empty()
	}
	return cfg
}

func SaveIni(cfg *ini.File) {
	if err := cfg.SaveTo(getIniPath()); err != nil {
		log.Printf("Failed to update ini file: %v\n", err)
		os.Exit(1)
	}
}

func ReflectValue(v interface{}) string {
	f := reflect.ValueOf(v)
	switch f.Kind() {
	case reflect.String:
		return f.String()
	case reflect.Int, reflect.Int64:
		return fmt.Sprint(f.Int())
	case reflect.Uint, reflect.Uint64:
		return fmt.Sprint(f.Uint())
	case reflect.Float64:
		return fmt.Sprint(f.Float())
	case reflect.Bool:
		return fmt.Sprint(f.Bool())
	case reflect.Slice:
		var s []string
		for _, el := range f.Interface().([]interface{}) {
			if reflect.ValueOf(el).Kind() == reflect.String {
				s = append(s, el.(string))
			}
		}
		return strings.Join(s, ",")
	default:
		// time.Time and others handled as string/JSON upstream
		return ""
	}
}

func BuildCoreUrl(project, resource, id string, params map[string]string) string {
	base := viper.GetString(DhCoreEndpoint) + "/api/" + viper.GetString(DhCoreApiVersion)

	var endpoint string
	if resource != "projects" && project != "" {
		endpoint += "/-/" + project
	}
	endpoint += "/" + resource
	if id != "" {
		endpoint += "/" + id
	}

	var qs string
	if len(params) > 0 {
		var sb strings.Builder
		sb.WriteString("?")
		for k, v := range params {
			if v != "" {
				sb.WriteString(k)
				sb.WriteString("=")
				sb.WriteString(v)
				sb.WriteString("&")
			}
		}
		qs = strings.TrimSuffix(sb.String(), "&")
	}

	return base + endpoint + qs
}

func PrepareRequest(method, url string, data []byte, accessToken string) *http.Request {
	var body io.Reader
	if data != nil {
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Printf("Failed to initialize request: %v\n", err)
		os.Exit(1)
	}
	if data != nil {
		req.Header.Add("Content-type", "application/json")
	}
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}
	return req
}

func DoRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error performing request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		msg := ""
		var bodyMap map[string]interface{}
		if json.Unmarshal(body, &bodyMap) == nil {
			if m, ok := bodyMap["message"].(string); ok {
				msg = " - " + m
			}
		}
		log.Printf("Core responded with: %v%v\n", resp.Status, msg)
		os.Exit(1)
	}
	return body, err
}

func TranslateFormat(format string) string {
	switch strings.ToLower(format) {
	case "json":
		return "json"
	case "yaml", "yml":
		return "yaml"
	default:
		return "short"
	}
}

func TranslateEndpoint(resource string) string {
	for key, val := range Resources {
		if key == resource || slices.Contains(val, resource) {
			return key
		}
	}
	log.Printf("Resource '%v' is not supported.\n", resource)
	os.Exit(1)
	return ""
}

func GetFirstIfList(m map[string]interface{}) (map[string]interface{}, error) {
	if content, ok := m["content"]; ok && reflect.ValueOf(content).Kind() == reflect.Slice {
		contentSlice := content.([]interface{})
		if len(contentSlice) >= 1 {
			return contentSlice[0].(map[string]interface{}), nil
		}
		return nil, errors.New("Resource not found")
	}
	return m, nil
}

func WaitForConfirmation(msg string) {
	for {
		buf := bufio.NewReader(os.Stdin)
		log.Printf(msg)
		userInput, err := buf.ReadBytes('\n')
		if err != nil {
			log.Printf("Error in reading user input: %v\n", err)
			os.Exit(1)
		}
		yn := strings.TrimSpace(string(userInput))
		switch strings.ToLower(yn) {
		case "y", "":
			return
		case "n":
			log.Println("Cancelling.")
			os.Exit(0)
		default:
			log.Println("Invalid input, must be y or n")
		}
	}
}

func PrintCommentForYaml(args ...string) {
	// fmt.Printf("# Generated on: %v\n", time.Now().Round(0))
	// fmt.Printf("#   from environment: %v (core version %v)\n", viper.GetString(DhCoreName), viper.GetString("dhcore_version"))
	// fmt.Printf("#   found at: %v\n", viper.GetString(DhCoreEndpoint))
	var parts []string
	for _, s := range args {
		if s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) > 0 {
		fmt.Printf("#   with parameters: %v\n", strings.Join(parts, " "))
	}
}

func CheckApiLevel(apiLevelKey string, min, max int) {
	fmt.Printf("Checking API level for %v command...\n", viper.GetString(apiLevelKey))

	apiLevelStr := viper.GetString(apiLevelKey)
	if apiLevelStr == "" {
		log.Println("ERROR: Unable to check compatibility, environment does not specify API level.")
		os.Exit(1)
	}

	apiLevel, err := strconv.Atoi(apiLevelStr)
	if err != nil {
		log.Printf("ERROR: API level %v is not an integer.\n", apiLevelStr)
		os.Exit(1)
	}

	inRange := true
	if min != 0 && apiLevel < min {
		inRange = false
	}
	if max != 0 && apiLevel > max {
		inRange = false
	}
	if !inRange {
		interval := "level"
		if min != 0 {
			interval = fmt.Sprintf("%v <= %s", min, interval)
		}
		if max != 0 {
			interval = fmt.Sprintf("%s <= %v", interval, max)
		}
		log.Printf("ERROR: API level %v is not within the supported interval: %v\n", apiLevel, interval)
		os.Exit(1)
	}
}

func GetStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func FetchConfig(configURL string) (map[string]interface{}, error) {
	resp, err := http.Get(configURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Core returned a non-200 status code: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func PrintResponseState(resp []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(resp, &m); err != nil {
		return err
	}
	if status, ok := m["status"].(map[string]interface{}); ok {
		if state, ok := status["state"].(string); ok {
			log.Printf("Core response successful, new state: %v\n", state)
			return nil
		}
	}
	log.Println("WARNING: core response successful, but unable to confirm new state.")
	return nil
}

func PrettyJSON(b []byte) string {
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		return string(b) // fallback non indentato
	}
	return out.String()
}
