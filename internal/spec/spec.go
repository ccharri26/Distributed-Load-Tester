package spec

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type TestSpec struct {
	ID          string            `json:"id"`
	TargetURL   string            `json:"target_url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	Duration    Duration          `json:"duration"`
	RPS         int               `json:"rps"`
	Concurrency int               `json:"concurrency"`
	Workers     int               `json:"workers"`
}

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	type Alias Duration
	return json.Marshal(&struct {
		Duration string `json:"duration"`
		*Alias
	}{
		Duration: d.String(),
		Alias:    (*Alias)(&d),
	})
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("duration must be a string like \"30s\": %w", err)
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

// Parse decodes, normalizes, and validates a test specification.
func Parse(data []byte) (*TestSpec, error) {
	var s TestSpec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	s.ApplyDefaults()

	if err := s.Validate(); err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *TestSpec) ApplyDefaults() {
	s.Method = strings.ToUpper(s.Method)

	if s.Method == "" {
		s.Method = "GET"
	}
	if s.Concurrency == 0 {
		s.Concurrency = 10
	}
}

func (s TestSpec) Validate() error {
	var errs []string

	if s.TargetURL == "" {
		errs = append(errs, "target_url is required")
	} else if u, err := url.ParseRequestURI(s.TargetURL); err != nil {
		errs = append(errs, "target_url is not a valid URL")
	} else if u.Scheme != "http" && u.Scheme != "https" {
		errs = append(errs, "target_url must use http or https")
	}

	if !isValidMethod(s.Method) {
		errs = append(errs, fmt.Sprintf("method %q is not a valid HTTP method", s.Method))
	}

	if s.Duration.Duration <= 0 {
		errs = append(errs, "duration must be positive")
	}

	if s.RPS <= 0 {
		errs = append(errs, "rps must be positive")
	}

	if s.Concurrency <= 0 {
		errs = append(errs, "concurrency must be positive")
	}

	if s.Workers <= 0 {
		errs = append(errs, "workers must be positive")
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid test spec: %s", strings.Join(errs, "; "))
	}
	return nil
}

func isValidMethod(m string) bool {
	switch m {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}
