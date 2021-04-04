package main

/*
Read /etc/nagios/nrpe.d, add a prometheus metric for every check
loop through running checks at regular intervals, store result
promhttp to provide infor from that store
status url for result text
*/

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

const configPath = "./nrpe_prometheus_exporter.yaml"
const checkInterval = 300

/* config file yaml format:
port: 2112
instance: check-unit-1
host: test-host-1
site: test-site-1
checks:
  - name: check_disk_root
    command: /usr/lib/nagios/plugins/check_disk -u GB -w 25% -c 20% -K 5% -p /
  - name: check_load
    command: /usr/lib/nagios/plugins/check_load -w 256,128,64 -c 512,256,128
*/

// struct to define a single check, with results to be populated later
type Check struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

// global because this is keeping track of all status for all checks
var allResults = make(map[string]string)

// struct to define the config file fields
type Config struct {
	Checks   []Check `yaml:"checks"`
	Instance string  `yaml:"instance"`
	Host     string  `yaml:"host"`
	Site     string  `yaml:"site"`
	Port     string  `yaml:"port"`
}

// Generic result checker
func handleError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

// Run checks (goroutine)
func runCheck(name string, checkCommand string, nrpeCheck prometheus.Gauge) {
	for {
		// Init return code
		retCode := 0
		// split command into command and args, and run it
		command := strings.Fields(checkCommand)
		plugin := command[0]
		args := command[1:]
		fmt.Printf("%s Running %s with args: %s\n", name, plugin, args) // TODO remove this debug output
		cmd := exec.Command(plugin, args...)                            // TODO timeout, return unknown if too slow
		CmdOutput, err := cmd.Output()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				retCode = exitError.ExitCode()
			}
		}
		allResults[name] = string(CmdOutput)
		fmt.Printf("Output: %s\n", allResults[name]) // TODO remove this debug output
		nrpeCheck.Set(float64(retCode))
		fmt.Printf("Check: %s, result: %s", name, CmdOutput) // TODO remove this debug output
		time.Sleep(checkInterval * time.Second)              // TODO jitter
	}
}

// setup and run Prometheus metrics
func (config Config) recordMetrics() {
	var nrpeChecks []prometheus.Gauge
	for i, check := range config.Checks {
		// loop through config.Checks, each check (type Check):
		nrpeChecks = append(nrpeChecks, promauto.NewGauge(prometheus.GaugeOpts{
			Name:        check.Name,
			Help:        "Return code from Nagios plugin execution",
			ConstLabels: prometheus.Labels{"job": "NRPE", "instance": config.Instance, "hostname": config.Host},
		}))
		allResults[check.Name] = "initializing"
		fmt.Println("Initializing check: ", check.Name)
		go runCheck(check.Name, check.Command, nrpeChecks[i])
	}
}

// Show the text output from check command execution
func ShowStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", allResults[r.URL.Path[1:]])
	// TODO what to do with empty results and bad URLs
}

// main
func main() {

	f, err := os.Open(configPath)
	handleError(err)
	cfg := yaml.NewDecoder(f)
	var config Config
	err = cfg.Decode(&config)
	handleError(err)
	config.recordMetrics()
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", ShowStatus)
	http.ListenAndServe(":"+config.Port, nil) // TODO handle default port

}
