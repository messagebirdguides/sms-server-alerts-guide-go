package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	messagebird "github.com/messagebird/go-rest-api"
	"github.com/messagebird/go-rest-api/sms"
	log "github.com/sirupsen/logrus"
)

var (
	logLevelsInfo = []log.Level{
		log.InfoLevel,
		log.WarnLevel,
	}
	logLevelsSevere = []log.Level{
		log.ErrorLevel,
		log.PanicLevel,
		log.FatalLevel,
	}
	logLevelsAll = []log.Level{
		log.InfoLevel,
		log.WarnLevel,
		log.ErrorLevel,
		log.PanicLevel,
		log.FatalLevel,
	}
)

// defaultPath is a catchall HTTP handler
func defaultPath(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello. "+
		"Please enter a valid status code in the path to simulate a HTTP server status. "+
		"E.g. www.example.com/simulate/404")
}

// simulateHTTPStatus injects a http.StatusInternalServerError as the HTTP status
func simulateHTTPStatus(w http.ResponseWriter, r *http.Request) {
	// Get status code from path /simulate/xxx
	path := strings.Split(r.URL.Path, "/")
	simulateCode, err := strconv.Atoi(path[2])
	if err != nil {
		log.Error(err)
	} else if len(path[2]) != 3 {
		output := fmt.Sprintf("Unknown status code used in path: %s", path[2])
		log.Warningln(output)
		fmt.Fprintln(w, output)
		fmt.Fprintln(w, "Hello. "+
			"Please enter a valid status code in the path to simulate a HTTP server status. "+
			"E.g. www.example.com/simulate/404")
		return
	}

	// Once we've gotten the status code to simulate from the path /simulate/xxx
	// We write it into the page's header and decide if we need to log an error.
	w.WriteHeader(simulateCode)

	// Handle all possible Server Error class of HTTP status codes
	if simulateCode >= 500 && simulateCode < 600 {
		output := fmt.Sprintf("Server error. [%s %s] %d %s", r.Method, r.URL.Path, simulateCode, http.StatusText(simulateCode))
		log.Errorln(output)
		fmt.Fprintln(w, output)
	} else {
		output := fmt.Sprintf("Everything's ok on our end.[%s %s] %d %s", r.Method, r.URL.Path, simulateCode, http.StatusText(simulateCode))
		log.Infoln(output)
		fmt.Fprintln(w, output)
	}
	return
}

func main() {
	client := messagebird.New("<enter-your-api-key>")

	// Configure logger

	// Discards default output, so we rely entirely on the hooks for output
	log.SetOutput(ioutil.Discard)
	log.SetFormatter(&log.TextFormatter{})

	logfile, err := os.OpenFile("mbservermon.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error(err)
	}
	defer func() {
		logfile.Sync()
		logfile.Close()
	}()

	log.AddHook(&WriterHook{
		Writer:    os.Stdout,
		LogLevels: logLevelsAll,
	})
	log.AddHook(&WriterHook{
		Writer:    logfile,
		LogLevels: logLevelsAll,
	})
	log.AddHook(&WriterHook{
		Writer:    &MBContainer{client, "MBServerMon", []string{"<recipient_number_here>"}},
		LogLevels: logLevelsSevere,
	})

	// HTTP Routing and Server
	http.HandleFunc("/", defaultPath)
	http.HandleFunc("/simulate/", simulateHTTPStatus)

	port := ":8080"
	log.Println("Serving on" + port)
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Errorln(err)
	}
}

// WriterHook hooks into log events.
type WriterHook struct {
	Writer    io.Writer
	LogLevels []log.Level
}

// Fire tells WriterHook what to do when an event is logged.
func (hook *WriterHook) Fire(entry *log.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	_, err = hook.Writer.Write([]byte(line))
	if err != nil {
		return err
	}
	return nil
}

// Levels rewrites the Levels method to only include the
// log.Level specified in the WriterHook struct.
func (hook *WriterHook) Levels() []log.Level {
	return hook.LogLevels
}

// MBContainer giftwraps the messagebird client so that we
// can implement a io.Writer interface for it
type MBContainer struct {
	Client     *messagebird.Client
	Originator string
	Recipients []string
}

// Custom Write method so MBContainer can be used as an io.Writer
func (mb *MBContainer) Write(p []byte) (int, error) {
	msgBody := string(p[:])
	if len(msgBody) > 160 {
		msgBody = msgBody[:159]
	}
	msg, err := sms.Create(
		mb.Client,
		mb.Originator,
		mb.Recipients,
		msgBody,
		nil,
	)
	if err != nil {
		return 1, err
	}
	fmt.Printf("Message sent: %v", msg)
	return 0, nil
}
