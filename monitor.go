package desync

import (
	"encoding/json"
	"sync"

	"github.com/reiver/go-oi"
	"github.com/reiver/go-telnet"
	"github.com/reiver/go-telnet/telsh"
	log "github.com/sirupsen/logrus"

	"io"
)

// Monitor represents informations to display on telnet server
type Monitor struct {
	mu           sync.Mutex
	Progress     int    `json:"progress,omitempty"`
	LocalStore   string `json:"local-store"`
	CurrentStore string `json:"current-store"`
}

var (
	onceMon   sync.Once
	singleMon *Monitor
)

// NewMonitor create a single telnet server listening on the given addr (":5555", "35.24.12.23:8957", ...)
func NewMonitor(addr string) *Monitor {

	onceMon.Do(func() {
		singleMon = new(Monitor)
		go singleMon.init(addr)
	})

	return singleMon
}

func (monitor *Monitor) SetProgress(progress int) {
	monitor.Progress = progress
}

func (monitor *Monitor) SetCurrentStore(currentStore string) {
	monitor.CurrentStore = currentStore
}

func (monitor *Monitor) SetLocalStore(localStore string) {
	monitor.LocalStore = localStore
}

func (monitor *Monitor) init(addr string) {
	shellHandler := telsh.NewShellHandler()

	shellHandler.WelcomeMessage = ""
	shellHandler.ExitMessage = ""
	shellHandler.Prompt = ""

	// Register the "progress" preload file command.
	commandName := "progress"
	commandProducer := telsh.ProducerFunc(monitor.progressProducer)

	shellHandler.Register(commandName, commandProducer)

	if err := telnet.ListenAndServe(addr, shellHandler); nil != err {
		Log.WithFields(log.Fields{
			"err": err,
		}).Error("Monitor init Error")
	}
}

func (monitor *Monitor) progressProducer(ctx telnet.Context, name string, args ...string) telsh.Handler {
	return telsh.PromoteHandlerFunc(monitor.progressHandler)
}

func (monitor *Monitor) progressHandler(stdin io.ReadCloser, stdout io.WriteCloser, stderr io.WriteCloser, args ...string) error {
	// building json
	jsonStat, err := json.Marshal(monitor)
	if err != nil {
		Log.WithFields(log.Fields{
			"err": err,
		}).Error("Monitor json Marshal Error")
	}

	oi.LongWrite(stdout, jsonStat)
	return err
}
