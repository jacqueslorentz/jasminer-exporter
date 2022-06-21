package main

import (
    "crypto/md5"
    "crypto/rand"
    "encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
    "strings"
	"time"
	"os"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenAddress = flag.String("listen-address", ":5896", "Address to listen on for web interface and telemetry")
	metricsPath   = flag.String("telemetry-path", "/metrics", "Path to expose metrics of the exporter")
	jasminerUri = flag.String("jasminer-uri", "", "Uri to reach the jasminer dashboard")
	authUsername = flag.String("auth-username", "root", "Jasminer authentication username")
	authPassword = flag.String("auth-password", "root", "Jasminer authentication password")
	version string
	build   string

	jasminer_miner = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "miner"), "Type description", []string {"type"}, nil)
	jasminer_version = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "version"), "Version", []string {"datetime"}, nil)
	jasminer_mem_total = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "mem_total"), "Total memory", nil, nil)
	jasminer_mem_used = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "mem_used"), "Used memory", nil, nil)
	jasminer_mem_free = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "mem_free"), "Free memory", nil, nil)
	jasminer_network = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "network"), "Network configuration",
			[]string {"type", "mac", "ip", "mask", "gateway", "dns1", "dns2"}, nil)

	jasminer_uptime = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "uptime"), "Uptime in seconds", nil, nil)
	jasminer_rate_realtime = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "rate_realtime"), "Realtime hashrate (in MH/s)", nil, nil)
	jasminer_rate_average = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "rate_average"), "Average hashrate (in MH/s)", nil, nil)
	jasminer_reject_rate = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "reject_rate"), "Reject rate (in %)", nil, nil)
	jasminer_fan_speed = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "fan_speed"), "Fan speed", []string {"device"}, nil)
	jasminer_board_rate = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "board_rate"), "Board hashrate (in MH/s)",
			[]string {"device", "asics","freq"}, nil)
	jasminer_board_temp = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "board_temp"), "Board temperature (in °C)", []string {"device"}, nil)
	jasminer_pool_config = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "pool_config"), "Pool configuration",
			[]string {"pool", "status", "user", "url"}, nil)
	jasminer_pool_works = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "pool_works"), "Pool works", []string {"pool"}, nil)
	jasminer_pool_accepted = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "pool_accepted"), "Pool accepted", []string {"pool"}, nil)
	jasminer_pool_rejected = prometheus.NewDesc(prometheus.BuildFQName("jasminer", "", "pool_rejected"), "Pool rejected", []string {"pool"}, nil)
)


type JasminerExporter struct {
	client *http.Client
	uri string
	username string
	password string
}

func NewJasminerExporter(uri string, username string, password string) (*JasminerExporter, error) {
	h := &http.Client{Timeout: 10 * time.Second}

	return &JasminerExporter{ client: h, uri: uri, username: username, password: password }, nil
}

func (e *JasminerExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- jasminer_miner
	ch <- jasminer_version
	ch <- jasminer_mem_total
	ch <- jasminer_mem_used
	ch <- jasminer_mem_free
	ch <- jasminer_network
	ch <- jasminer_uptime
	ch <- jasminer_rate_realtime
	ch <- jasminer_rate_average
	ch <- jasminer_reject_rate
	ch <- jasminer_fan_speed
	ch <- jasminer_board_rate
	ch <- jasminer_board_temp
	ch <- jasminer_pool_config
	ch <- jasminer_pool_works
	ch <- jasminer_pool_accepted
	ch <- jasminer_pool_rejected
}

func (e *JasminerExporter) Collect(ch chan<- prometheus.Metric) {
	var infos map[string]string
	infosBody := HttpGetCallWithDigest(e.client, e.uri + "/cgi-bin/index.cgi", e.username, e.password)
	err := json.Unmarshal([]byte(infosBody), &infos)
	if err != nil {
        log.Fatal(err)
    }
	var infos2 map[string]interface{}
	infosBody2 := HttpGetCallWithDigest(e.client, e.uri + "/cgi-bin/minerStatus.cgi", e.username, e.password)
	err2 := json.Unmarshal([]byte(infosBody2), &infos2)
	if err2 != nil {
        log.Fatal(err2)
    }

	ch <- prometheus.MustNewConstMetric(jasminer_miner, prometheus.GaugeValue, 1, infos["minertype"])
	ch <- prometheus.MustNewConstMetric(jasminer_version, prometheus.GaugeValue, 1, infos["fs_version"])
	mem_total, _ := strconv.ParseFloat(infos["mem_total"], 64)
	ch <- prometheus.MustNewConstMetric(jasminer_mem_total, prometheus.GaugeValue, mem_total)
	mem_used, _ := strconv.ParseFloat(infos["mem_used"], 64)
	ch <- prometheus.MustNewConstMetric(jasminer_mem_used, prometheus.GaugeValue, mem_used)
	mem_free, _ := strconv.ParseFloat(infos["mem_free"], 64)
	ch <- prometheus.MustNewConstMetric(jasminer_mem_free, prometheus.GaugeValue, mem_free)
	ch <- prometheus.MustNewConstMetric(jasminer_network, prometheus.GaugeValue, 1, infos["nettype"], infos["macaddr"],
			infos["ipaddress"], infos["netmask"], infos["gateway"], infos["dns1"], infos["dns2"])

	summary := infos2["summary"].(map[string]interface{})
	ch <- prometheus.MustNewConstMetric(jasminer_uptime, prometheus.GaugeValue, summary["uptime"].(float64))
	rt, _ := strconv.ParseFloat(strings.Split(summary["rt"].(string), " ")[0], 64)
	ch <- prometheus.MustNewConstMetric(jasminer_rate_realtime, prometheus.GaugeValue, rt)
	avg, _ := strconv.ParseFloat(strings.Split(summary["avg"].(string), " ")[0], 64)
	ch <- prometheus.MustNewConstMetric(jasminer_rate_average, prometheus.GaugeValue, avg)
	rejectRate, _ := strconv.ParseFloat(strings.Split(summary["rejectRate"].(string), " ")[0], 64)
	ch <- prometheus.MustNewConstMetric(jasminer_reject_rate, prometheus.GaugeValue, rejectRate)
	boards := infos2["boards"].(map[string]interface{})
	ch <- prometheus.MustNewConstMetric(jasminer_fan_speed, prometheus.GaugeValue, boards["fan1"].(float64), "fan1")
	ch <- prometheus.MustNewConstMetric(jasminer_fan_speed, prometheus.GaugeValue, boards["fan2"].(float64), "fan2")
	for i, b := range boards["board"].([]interface{}) {
		board := b.(map[string]interface{})
		label := fmt.Sprintf("board%d", i)
		rate, _ := strconv.ParseFloat(strings.Split(board["rate"].(string), " ")[0], 64)
		ch <- prometheus.MustNewConstMetric(jasminer_board_rate, prometheus.GaugeValue, rate, label,
			strconv.FormatFloat(board["asics"].(float64), 'f', 0, 64),
			strconv.FormatFloat(board["freq"].(float64), 'f', 0, 64))
		ch <- prometheus.MustNewConstMetric(jasminer_board_temp, prometheus.GaugeValue, board["temp"].(float64), label)
	}
	pools := infos2["pools"].(map[string]interface{})
	for i, p := range pools["pool"].([]interface{}) {
		pool := p.(map[string]interface{})
		label := fmt.Sprintf("pool%d", i)
		user := pool["user"]
		if user == nil {
			user = ""
		}
		url := pool["url"]
		if url == nil {
			url = ""
		}
		ch <- prometheus.MustNewConstMetric(jasminer_pool_config, prometheus.GaugeValue, 1, label,
				pool["status"].(string), user.(string), url.(string))
		ch <- prometheus.MustNewConstMetric(jasminer_pool_works, prometheus.GaugeValue, pool["works"].(float64), label)
		ch <- prometheus.MustNewConstMetric(jasminer_pool_accepted, prometheus.GaugeValue, pool["accept"].(float64), label)
		ch <- prometheus.MustNewConstMetric(jasminer_pool_rejected, prometheus.GaugeValue, pool["reject"].(float64), label)
	}
}

func main() {
	flag.Parse()

	if *jasminerUri == "" {
		log.Fatal("Jasminer Uri required")
		os.Exit(1)
	}

	fmt.Println("Version:", version)
	fmt.Println("Build Time:", build)
	fmt.Println("Jasminer Uri:", *jasminerUri)
	fmt.Println("Metrics Path:", *metricsPath)

	exporter, err := NewJasminerExporter(*jasminerUri, *authUsername, *authPassword)
	if err != nil {
		log.Fatal("Error initializing exporter")
		os.Exit(1)
	}

	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})
	
	fmt.Println("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}



// HTTP call with digest auth utility methods

func HttpGetCallWithDigest(client *http.Client, uri string, username string, password string) (string) {
	req, err := http.NewRequest("GET", uri, nil)
	resp, err := client.Do(req)

    digestParts := digestParts(resp)
    digestParts["uri"] = uri
    digestParts["method"] = "GET"
    digestParts["username"] = username
    digestParts["password"] = password
	req2, err := http.NewRequest("GET", uri, nil)
	req2.Header.Add("Authorization", getDigestAuthrization(digestParts)) 
	resp2, err := client.Do(req2)

	if err != nil {
        log.Fatal(err)
    }

    defer resp2.Body.Close()

    body, err := ioutil.ReadAll(resp2.Body)

    if err != nil {
        log.Fatal(err)
    }

    return string(body)
}

func digestParts(resp *http.Response) map[string]string {
    result := map[string]string{}
    if len(resp.Header["Www-Authenticate"]) > 0 {
        wantedHeaders := []string{"nonce", "realm", "qop"}
        responseHeaders := strings.Split(resp.Header["Www-Authenticate"][0], ",")
        for _, r := range responseHeaders {
            for _, w := range wantedHeaders {
                if strings.Contains(r, w) {
                    result[w] = strings.Split(r, `"`)[1]
                }
            }
        }
    }
    return result
}

func getMD5(text string) string {
    hasher := md5.New()
    hasher.Write([]byte(text))
    return hex.EncodeToString(hasher.Sum(nil))
}

func getCnonce() string {
    b := make([]byte, 8)
    io.ReadFull(rand.Reader, b)
    return fmt.Sprintf("%x", b)[:16]
}

func getDigestAuthrization(digestParts map[string]string) string {
    d := digestParts
    ha1 := getMD5(d["username"] + ":" + d["realm"] + ":" + d["password"])
    ha2 := getMD5(d["method"] + ":" + d["uri"])
    nonceCount := 00000001
    cnonce := getCnonce()
    response := getMD5(fmt.Sprintf("%s:%s:%v:%s:%s:%s", ha1, d["nonce"], nonceCount, cnonce, d["qop"], ha2))
    authorization := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", cnonce="%s", nc="%v", qop="%s", response="%s"`,
        d["username"], d["realm"], d["nonce"], d["uri"], cnonce, nonceCount, d["qop"], response)
    return authorization
}
