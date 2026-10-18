package otlp

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// Settings is everything the exporter needs, already resolved: the secret is a
// value here, read from the environment variable the config names.
type Settings struct {
	Endpoint    string
	HeaderName  string
	HeaderValue string
	ServiceName string
	Version     string
}

var (
	ErrEndpointInvalid     = errors.New("otlp endpoint must be an http or https URL with a host")
	ErrEndpointCredentials = errors.New("otlp endpoint must not carry credentials, a query or a fragment")
	ErrHeaderInvalid       = errors.New("otlp header name is not a plain HTTP header name")
	ErrSecretOverPlainHTTP = errors.New("otlp header value would travel over plain http to a non-loopback host")
)

var headerName = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

var reservedHeaders = map[string]bool{"host": true, "content-length": true, "content-type": true, "transfer-encoding": true}

// target validates the settings and returns the URL spans are posted to.
func (s Settings) target() (string, error) {
	u, err := url.Parse(strings.TrimSpace(s.Endpoint))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", ErrEndpointInvalid
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return "", ErrEndpointCredentials
	}
	if s.HeaderName != "" && (!headerName.MatchString(s.HeaderName) || reservedHeaders[strings.ToLower(s.HeaderName)]) {
		return "", ErrHeaderInvalid
	}
	if s.HeaderValue != "" && u.Scheme == "http" && !loopback(u.Hostname()) {
		return "", ErrSecretOverPlainHTTP
	}
	path := strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(path, "/v1/traces") {
		path += "/v1/traces"
	}
	u.Path = path
	return u.String(), nil
}

func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
