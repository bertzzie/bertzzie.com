package main

import (
	"bertzzie.com/routes"
	"context"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	initConfiguration()
	initLogging()
	initServer()
}

func initServer() {
	address := viper.GetString("http.address")
	log.Infof("Starting http server at %s", address)

	router := initRouters()
	server := &http.Server{
		Addr: address,
		WriteTimeout: time.Duration(viper.GetInt("http.timeouts.read")) * time.Second,
		ReadTimeout: time.Duration(viper.GetInt("http.timeouts.read")) * time.Second,
		IdleTimeout: time.Duration(viper.GetInt("http.timeouts.idle")) * time.Second,
		Handler: router,
	}

	go func() {
		log.Infof("Server ready to serve request at %s", address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Fail to serve http: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// we can safely blocks here because server is running in the background
	// this means we will blocks here until we got either SIGINT or SIGTERM
	<- quit

	grace := time.Duration(viper.GetInt("http.timeouts.grace")) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Server failed to shutdown gracefully: %s\n", err)
	}

	log.Infof("Shutting down server...")
	os.Exit(0)
}

func initRouters() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/status/health", routes.StatusHandler)

	return r
}

func initLogging() {
	log.SetFormatter(initLogFormat())
	log.SetLevel(initLogLevel())
	log.SetOutput(initLogOutput())
}

func initLogOutput() io.Writer {
	output := viper.GetString("logging.output")
	switch strings.ToLower(output) {
	case "stdout":
		return os.Stdout
	case "file":
		return initLogFile()
	}

	log.Warnf("Unknown log output %s. Setting up output as stdout", output)

	return os.Stdout
}

func initLogFile() io.Writer {
	filename := viper.GetString("logging.file")

	// check if directory exists
	path := filepath.Dir(filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0744)
		if err != nil {
			log.Errorf("Error creating log file directory %s: %s. Falling back to stdout", filename, err)
			return os.Stdout
		}
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Errorf("Error opening log file %s: %s. Falling back to stdout", filename, err)
		return os.Stdout
	}

	return file
}

func initLogLevel() log.Level {
	level, err := log.ParseLevel(viper.GetString("logging.level"))
	if err != nil {
		log.Fatalf("Error setting up logging level: %s\n", err)
	}

	return level
}

func initLogFormat() log.Formatter {
	format := viper.GetString("logging.format")
	switch strings.ToLower(format) {
	case "json":
		return &log.JSONFormatter{}
	case "text":
		return &log.TextFormatter{}
	}

	log.Warnf("Unknown log format %s. Setting up log as JSON", format)

	return &log.JSONFormatter{}
}

func initConfiguration() {
	viper.SetConfigName("configuration")
	viper.SetConfigType("yaml")

	viper.AddConfigPath("./config")
	viper.AddConfigPath("$HOME/.bertzzie.com")
	viper.AddConfigPath("/etc/bertzzie.com")

	initDefaultConfigurations()

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file: %s\n", err)
	}
}

func initDefaultConfigurations() {
	viper.SetDefault("logging.file", "./log/application.log")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.output", "stdout")

	viper.SetDefault("http.address", "0.0.0.0:8080")
	viper.SetDefault("http.timeouts.write", 15)
	viper.SetDefault("http.timeouts.read", 15)
	viper.SetDefault("http.timeouts.idle", 60)
}
