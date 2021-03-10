package desync

import (
	"encoding/json"
	"fmt"
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
	Snapstore      *SnapStore
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

func (monitor *Monitor) SetSnapStore(snapstore *SnapStore) {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()
	monitor.Snapstore = snapstore
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
	statusCommand := "status"
	snapCommand := "snap"
	takeSnapCommand := "takeSnap"
	statusProducer := telsh.ProducerFunc(monitor.statusProducer)
	snapProducer := telsh.ProducerFunc(monitor.snapProducer)
	takeSnapProducer := telsh.ProducerFunc(monitor.takeSnapProducer)

	shellHandler.Register(statusCommand, statusProducer)
	shellHandler.Register(snapCommand, snapProducer)
	shellHandler.Register(takeSnapCommand, takeSnapProducer)

	if err := telnet.ListenAndServe(addr, shellHandler); nil != err {
		Log.WithFields(log.Fields{
			"err": err,
		}).Error("Monitor init Error")
	}
}

func (monitor *Monitor) statusProducer(ctx telnet.Context, name string, args ...string) telsh.Handler {
	return telsh.PromoteHandlerFunc(monitor.statusHandler)
}

func (monitor *Monitor) snapProducer(ctx telnet.Context, name string, args ...string) telsh.Handler {
	return telsh.PromoteHandlerFunc(monitor.snapHandler)
}

func (monitor *Monitor) takeSnapProducer(ctx telnet.Context, name string, args ...string) telsh.Handler {
	return telsh.PromoteHandlerFunc(monitor.takeSnapHandler, args...)
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

func (monitor *Monitor) snapHandler(stdin io.ReadCloser, stdout io.WriteCloser, stderr io.WriteCloser, args ...string) error {
	// building json
	msg := fmt.Sprintf("snap dir : %s\tcurrent snap : %s\t\n", monitor.Snapstore.dir, monitor.Snapstore.name)

	_, err := oi.LongWrite(stdout, []byte(msg))
	return err
}

func (monitor *Monitor) takeSnapHandler(stdin io.ReadCloser, stdout io.WriteCloser, stderr io.WriteCloser, args ...string) error {
	// building json
	var msg string
	if len(args) == 0 || len(args) > 2 {
		msg = fmt.Sprintf("takeSnap take 1 or 2 arg [snapshot_name] [true/false]\n")
	} else if args[1] != "true" && args[1] != "false" {
		msg = fmt.Sprintf("takeSnap[%s] invalid 2nd arg [true/false]: %s\n", args[0], args[1])
	} else {
		msg = fmt.Sprintf("takeSnap[%s] record: %s\n", args[0], args[1])

		if err := monitor.Snapstore.takeSnapshot(args[0], args[1] == "true"); err != nil {
			msg = fmt.Sprintf("takeSnap[%s] record: %s ERROR : %s\n", args[0], args[1], err)
		}
	}

	_, err := oi.LongWrite(stdout, []byte(msg))
	return err
}
