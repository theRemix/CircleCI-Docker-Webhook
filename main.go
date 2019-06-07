package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/hcl"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"time"
)

type IncomingWebhook struct {
	Repository      string   `json:"repository"`
	Namespace       string   `json:"namespace"`
	Name            string   `json:"name"`
	DockerURL       string   `json:"docker_url"`
	Homepage        string   `json:"homepage"`
	Visibility      string   `json:"visibility"`
	BuildID         string   `json:"build_id"`
	DockerTags      []string `json:"docker_tags"`
	TriggerKind     string   `json:"trigger_kind"`
	TriggerID       string   `json:"trigger_id"`
	TriggerMetadata struct {
		DefaultBranch string `json:"default_branch"`
		Ref           string `json:"ref"`
		Commit        string `json:"commit"`
		CommitInfo    struct {
			URL     string `json:"url"`
			Message string `json:"message"`
			Date    string `json:"date"`
			Author  struct {
				Username  string `json:"username"`
				URL       string `json:"url"`
				AvatarURL string `json:"avatar_url"`
			} `json:"author"`
			Committer struct {
				Username  string `json:"username"`
				URL       string `json:"url"`
				AvatarURL string `json:"avatar_url"`
			} `json:"committer"`
		} `json:"commit_info"`
	} `json:"trigger_metadata"`
}

type SlackPayload struct {
	Text      string `json:"text"`
	Channel   string `json:"channel,omitempty"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
}

type Slack struct {
	WebhookUrl string `hcl:"webhookUrl"`
	Username   string `hcl:"username,omitempty"`
	Channel    string `hcl:"channel,omitempty"`
	IconEmoji  string `hcl:"iconEmoji,omitempty"`
}

type Service struct {
	Name          string `hcl:",key"`
	Repository    string `hcl:"repository"`
	Cmd           string `hcl:"cmd"`
	Conditions    string `hcl:"conditions"`
	DeployMessage string `hcl:"deployMessage"`
}

type Config struct {
	WebhookPath string    `hcl:"webhookPath"`
	Slack       *Slack    `hcl:"slack"`
	Services    []Service `hcl:"service"`
}

func validateConfig(config *Config) error {
	if config.WebhookPath == "" {
		return errors.New("( required ) webhookPath is missing from config.")
	}
	if config.WebhookPath == "CHANGE_ME__DO_NOT_ACTUALLY_USE_THIS_VALUE__SEE_README" {
		return errors.New("( required ) webhookPath has not been updated from the example. see Readme.md")
	}
	return nil
}

func timestamp() string {
	t := time.Now()
	return t.Format("2006-01-02 15:04:05")
}

func notifier(config *Config) func(string) {
	if config.Slack == nil {
		return func(_ string) {}
	} else {
		return func(message string) {
			payload := &SlackPayload{
				Text:      message,
				Channel:   config.Slack.Channel,
				Username:  config.Slack.Username,
				IconEmoji: config.Slack.IconEmoji,
			}
			payloadJson, jsonErr := json.Marshal(payload)
			if jsonErr != nil {
				fmt.Printf("[%s ERROR Slack Notification Json Marshal] %s\n", timestamp(), jsonErr)
				return
			}

			params := url.Values{}
			params.Set("payload", string(payloadJson))

			resp, err := http.Post(config.Slack.WebhookUrl, "application/x-www-form-urlencoded", bytes.NewBuffer([]byte(params.Encode())))
			if err != nil {
				fmt.Printf("[%s ERROR Slack Notification POST request] %s\n", timestamp(), jsonErr)
				return
			}

			if os.Getenv("DEBUG") != "" {
				fmt.Printf("[%s DEBUG PAYLOAD Slack Response] %+v\n", timestamp(), resp.Body)
			}

		}
	}
}

func deploy(svc Service, ref string) string {
	fmt.Printf("[%s Deploying] %s from %s\n", timestamp(), svc.Name, ref)

	content := []byte(ref)

	pattern := regexp.MustCompile(svc.Conditions)

	template := []byte(svc.Cmd)

	cmd := []byte{}

	for _, submatches := range pattern.FindAllSubmatchIndex(content, -1) {
		cmd = pattern.Expand(cmd, template, content, submatches)
	}

	fmt.Printf("[%s Executing Shell] %s\n", timestamp(), cmd)
	out, err := exec.Command("/bin/sh", "-c", string(cmd)).CombinedOutput()
	if err != nil {
		message := fmt.Sprintf("[%s ERROR] [exec shell] %s\n[output] %s\n", timestamp(), err, out)
		fmt.Print(message)
		return message
	}

	fmt.Printf("[%s Shell Output (begin)]\n", timestamp())
	fmt.Printf("%s\n", out)
	fmt.Printf("[%s Shell Output (end)]\n", timestamp())

	if svc.DeployMessage != "" {
		template = []byte(svc.DeployMessage)
		message := []byte{}
		for _, submatches := range pattern.FindAllSubmatchIndex(content, -1) {
			message = pattern.Expand(message, template, content, submatches)
		}
		return fmt.Sprintf(string(message))
	}

	return fmt.Sprintf("Successfully deployed %s from %s", svc.Name, ref)
}

func main() {
	var PORT string

	if os.Getenv("PORT") != "" {
		PORT = os.Getenv("PORT")
	} else {
		PORT = "2000"
	}

	if len(os.Args) < 2 {
		log.Fatal("Pass one argument to this program with the path to config.hcl")
	}

	configPath := os.Args[1]

	configContents, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	config := &Config{}
	decodeErr := hcl.Unmarshal(configContents, config)
	if decodeErr != nil {
		log.Fatal(decodeErr)
	}

	configValidationErr := validateConfig(config)
	if configValidationErr != nil {
		log.Fatal(configValidationErr)
	}

	fmt.Printf("[%s Config loaded] %s\n", timestamp(), configPath)

	notify := notifier(config)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _r *http.Request) {
		fmt.Fprintf(w, "ok")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fmt.Printf("[%s ERROR] [incorrect method] received: %s\n", timestamp(), r.Method)
			fmt.Fprintf(w, "ok")
			return
		}

		if r.RequestURI != "/"+config.WebhookPath {
			fmt.Printf("[%s ERROR] [incorrect webhookPath] received: %s\n", timestamp(), r.RequestURI)
			fmt.Fprintf(w, "ok")
			return
		}

		decoder := json.NewDecoder(r.Body)
		var payload IncomingWebhook
		err := decoder.Decode(&payload)

		if os.Getenv("DEBUG") != "" {
			fmt.Printf("[%s DEBUG PAYLOAD] %+v\n", timestamp(), payload)
		}

		if err != nil {
			fmt.Printf("[%s ERROR] [decode payload] %+v\n", timestamp(), err)
			fmt.Fprintf(w, "error")
			return
		}

		go func() {
			for _, svc := range config.Services {
				re := regexp.MustCompile(svc.Conditions)
				if svc.Repository == payload.Repository && len(re.Find([]byte(payload.TriggerMetadata.Ref))) != 0 {
					notify(fmt.Sprintf("Deploying %s", svc.Name))
					notify(deploy(svc, payload.TriggerMetadata.Ref))
				}
			}
		}()

		fmt.Fprintf(w, "ok")
	})

	http.ListenAndServe(":"+PORT, nil)
}
