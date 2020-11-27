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
	Active    bool   `json:"active"`
	Index     int    `json:"index"`
	Store     string `json:"store"`
	Timestamp string `json:"timestamp"`
	Reason    string `json:"reason"`
}

// Monitor represents informations to display on telnet server
type Monitor struct {
	mu             sync.Mutex
	Progress       int             `json:"progress,omitempty"`
	LocalStore     string          `json:"local-store"`
	FailoverStores []FailoverStore `json:"failover-stores"`
	lastIndex      int
	folen          int
}

var (
	onceMon   sync.Once
	singleMon *Monitor
)

// NewMonitor create a single telnet server listening on the given addr (":5555", "35.24.12.23:8957", ...)
func NewMonitor(addr string, folen int) *Monitor {

	onceMon.Do(func() {
		singleMon = &Monitor{FailoverStores: make([]FailoverStore, 0, folen), folen: folen}
		go singleMon.init(addr)
	})

	return singleMon
}

func (monitor *Monitor) SetProgress(progress int) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()
	monitor.Progress = progress
}

// SetCurrentStore set the current store. mutex is suppose to be used by the caller, but anyway...
func (monitor *Monitor) SetCurrentStore(currentStore string, previousReason string) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()
	if monitor.lastIndex >= monitor.folen {
		monitor.FailoverStores[monitor.folen-1].Active = false
		monitor.FailoverStores[monitor.folen-1].Reason = previousReason
	} else if monitor.lastIndex > 0 {
		monitor.FailoverStores[monitor.lastIndex-1].Active = false
		monitor.FailoverStores[monitor.lastIndex-1].Reason = previousReason
	}
	fo := FailoverStore{
		Active:    true,
		Index:     monitor.lastIndex,
		Store:     currentStore,
		Timestamp: time.Now().Format("2006-01-02T15:04:05.00-07:00"),
	}
	monitor.lastIndex++
	if len(monitor.FailoverStores) == monitor.folen {
		monitor.FailoverStores = append(monitor.FailoverStores[1:], fo)
	} else {
		monitor.FailoverStores = append(monitor.FailoverStores, fo)
	}
}

// SetLocalStore real name is cache store. even if a reference store is local, cache is called after all,
// but it's more clear like that.
func (monitor *Monitor) SetLocalStore(localStore string) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()
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
