package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

var Options struct {
	DataDir       string `envconfig:"DATA_DIR"`
	HTTPSKeyFile  string `envconfig:"HTTPS_KEY_FILE"`
	HTTPSCertFile string `envconfig:"HTTPS_CERT_FILE"`
	HTTPPort      string `envconfig:"HTTP_PORT" default:"8080"`
	HTTPSPort     string `envconfig:"HTTPS_PORT" default:"8443"`
}

func main() {
	log.SetReportCaller(true)
	err := envconfig.Process("fileserver", &Options)
	if err != nil {
		log.Fatalf("Failed to process config: %v\n", err)
	}

	http.HandleFunc("/rhcos/", func(w http.ResponseWriter, r *http.Request) {
		image, err := parseImageName(r.URL.Path)
		if err != nil {
			log.Errorf("failed to parse image name: %v\n", err)
			http.NotFound(w, r)
			return
		}

		isoReader, err := os.Open(filepath.Join(Options.DataDir, image))
		if err != nil {
			log.Errorf("failed to open image file: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer isoReader.Close()

		http.ServeContent(w, r, image, time.Now(), isoReader)
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	info := initServers(Options.HTTPPort, Options.HTTPSPort, Options.HTTPSKeyFile, Options.HTTPSCertFile)
	<-stop
	info.Shutdown()
}

var pathRegexp = regexp.MustCompile(`^/rhcos/(.+)`)

func parseImageName(path string) (string, error) {
	match := pathRegexp.FindStringSubmatch(path)
	if match == nil {
		return "", fmt.Errorf("malformed download path: %s", path)
	}
	return match[1], nil
}

type ServerInfo struct {
	HTTP          http.Server
	HTTPS         http.Server
	HTTPSKeyFile  string
	HTTPSCertFile string
}

func initServers(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile string) *ServerInfo {
	servers := ServerInfo{}
	if httpsPort != "" && HTTPSKeyFile != "" && HTTPSCertFile != "" {
		servers.HTTPS = http.Server{
			Addr: fmt.Sprintf(":%s", httpsPort),
		}
		servers.HTTPSCertFile = HTTPSCertFile
		servers.HTTPSKeyFile = HTTPSKeyFile
		go servers.httpsListen()
	}
	if httpPort != "" {
		servers.HTTP = http.Server{
			Addr: fmt.Sprintf(":%s", httpPort),
		}
		go servers.httpListen()
	}
	return &servers
}

func shutdown(name string, server *http.Server) {
	if err := server.Shutdown(context.TODO()); err != nil {
		log.Infof("%s shutdown failed: %v", name, err)
		if err := server.Close(); err != nil {
			log.Fatalf("%s emergency shutdown failed: %v", name, err)
		}
	} else {
		log.Infof("%s server terminated gracefully", name)
	}
}

func (s *ServerInfo) Shutdown() bool {
	if s.HTTPSKeyFile != "" && s.HTTPSCertFile != "" {
		shutdown("HTTPS", &s.HTTPS)
	}
	shutdown("HTTP", &s.HTTP)
	return true
}

func (s *ServerInfo) httpListen() {
	log.Infof("Starting http handler on %s...", s.HTTP.Addr)
	if err := s.HTTP.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP listener closed: %v", err)
	}
}

func (s *ServerInfo) httpsListen() {
	log.Infof("Starting https handler on %s...", s.HTTPS.Addr)
	if err := s.HTTPS.ListenAndServeTLS(s.HTTPSCertFile, s.HTTPSKeyFile); err != http.ErrServerClosed {
		log.Fatalf("HTTPS listener closed: %v", err)
	}
}
