package desync

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"github.com/reiver/go-telnet/telsh"
	log "github.com/sirupsen/logrus"

	"io"
)

//
type FailoverStore struct {
	Store          string `json:"store"`
	Timestamp      string `json:"timestamp"`
	PreviousReason string `json:"previous-reason"`
}

// Monitor represents informations to display on telnet server
type Monitor struct {
	mu             sync.Mutex
	Progress       int             `json:"progress,omitempty"`
	LocalStore     string          `json:"local-store"`
	FailoverStores []FailoverStore `json:"failover-stores"`
}

var (
	onceMon   sync.Once
	singleMon *Monitor
)

// NewMonitor create a single telnet server listening on the given addr (":5555", "35.24.12.23:8957", ...)
func NewMonitor(addr string) *Monitor {

	onceMon.Do(func() {
		singleMon = &Monitor{FailoverStores: make([]FailoverStore, 0)}
		go singleMon.init(addr)
	})

	return singleMon
}

func (monitor *Monitor) SetProgress(progress int) {
	monitor.Progress = progress
}

func (monitor *Monitor) SetCurrentStore(currentStore string, previousReason string) {
	fo := FailoverStore{
		Store:          currentStore,
		PreviousReason: previousReason,
		Timestamp:      time.Now().Format("2006-01-02T15:04:05.00-07:00"),
	}
	monitor.FailoverStores = append(monitor.FailoverStores, fo)
}

func (monitor *Monitor) SetLocalStore(localStore string) {
	monitor.LocalStore = localStore
}

func (monitor *Monitor) init(addr string) {
	shellHandler := telsh.NewShellHandler()

	shellHandler.WelcomeMessage = ""
	shellHandler.ExitMessage = ""
	shellHandler.Prompt = ""

	// Register the "status" preload file command.
	commandName := "status"
	commandProducer := telsh.ProducerFunc(monitor.statusProducer)

	shellHandler.Register(commandName, commandProducer)

	if err := telnet.ListenAndServe(addr, shellHandler); nil != err {
		Log.WithFields(log.Fields{
			"err": err,
		}).Error("Monitor init Error")
	}
}

func (monitor *Monitor) statusProducer(ctx telnet.Context, name string, args ...string) telsh.Handler {
	return telsh.PromoteHandlerFunc(monitor.statusHandler)
}

func (monitor *Monitor) statusHandler(stdin io.ReadCloser, stdout io.WriteCloser, stderr io.WriteCloser, args ...string) error {
	// building json
	jsonStat, err := json.MarshalIndent(monitor, "", "  ")
	if err != nil {
		Log.WithFields(log.Fields{
			"err": err,
		}).Error("Monitor statusHandler json Marshal Error")
	}

	oi.LongWrite(stdout, jsonStat)
	return err
}
