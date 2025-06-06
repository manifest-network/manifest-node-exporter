package collectors

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"resty.dev/v3"

	"github.com/liftedinit/manifest-node-exporter/pkg"
	"github.com/liftedinit/manifest-node-exporter/pkg/utils"
)

type GeoIPCollector struct {
	latitude  *prometheus.Desc
	longitude *prometheus.Desc
	metadata  *prometheus.Desc
	client    *resty.Client
	key       string
	stateFile string
}

type IPResponse struct {
	IP string `json:"ip"`
}

// GeoIPResponse represents the top‐level JSON returned by ipbase.
type GeoIPResponse struct {
	Data GeoIPData `json:"data"`
}

// GeoIPData holds the “ip” string and the nested “location” object.
type GeoIPData struct {
	IP       string        `json:"ip"`
	Location GeoIPLocation `json:"location"`
}

// GeoIPLocation holds country, region, city and zip.
// We only include the fields you asked for; all other JSON keys are ignored.
type GeoIPLocation struct {
	Latitude  float64      `json:"latitude"`
	Longitude float64      `json:"longitude"`
	Country   GeoIPCountry `json:"country"`
	Region    GeoIPRegion  `json:"region"`
	City      GeoIPCity    `json:"city"`
	Zip       string       `json:"zip"`
}

// GeoIPCountry contains country code and name.
type GeoIPCountry struct {
	Alpha2 string `json:"alpha2"`
	Name   string `json:"name"`
}

// GeoIPRegion contains region code and name.
type GeoIPRegion struct {
	Alpha2 string `json:"alpha2"`
	Name   string `json:"name"`
}

// GeoIPCity contains just the city name.
type GeoIPCity struct {
	Name string `json:"name"`
}

type cacheState struct {
	IP        string        `json:"ip"`
	Geo       GeoIPResponse `json:"geo"`
	NextFetch time.Time     `json:"next_fetch"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	ipifyURL  = "https://api.ipify.org?format=json"
	ipBaseUrl = "https://api.ipbase.com/v2/info?ip=%s&apikey=%s"
)

func NewGeoIPCollector(key, stateFile string) *GeoIPCollector {
	return &GeoIPCollector{
		client:    resty.New().SetHeader("Accept", "application/json").SetTimeout(pkg.ClientTimeout).SetRetryCount(pkg.ClientRetry),
		key:       key,
		stateFile: stateFile,
		latitude: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "geo", "latitude"),
			"Node's geographical latitude",
			[]string{"ip"},
			prometheus.Labels{"source": "geoip"},
		),
		longitude: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "geo", "longitude"),
			"Node's geographical longitude",
			[]string{"ip"},
			prometheus.Labels{"source": "geoip"},
		),
		metadata: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "geo", "metadata"),
			"Node's geographical information",
			[]string{"ip", "country_code", "country_name", "region_code", "region_name", "city", "zip_code"},
			prometheus.Labels{"source": "geoip"},
		),
	}
}

func (c *GeoIPCollector) loadState() (*cacheState, error) {
	data, err := os.ReadFile(c.stateFile)
	if err != nil {
		return nil, err
	}
	var st cacheState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (c *GeoIPCollector) saveState(st *cacheState) error {
	dir := filepath.Dir(c.stateFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.stateFile, data, 0o644)
}

func (c *GeoIPCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.latitude
	ch <- c.longitude
	ch <- c.metadata
}

func (c *GeoIPCollector) Collect(ch chan<- prometheus.Metric) {
	ip, err := getPublicIP(c.client)
	if err != nil {
		ReportInvalidMetric(ch, c.metadata, fmt.Errorf("failed to get public ip address: %w", err))
		return
	}

	maybeIp := net.ParseIP(ip)
	if maybeIp == nil {
		ReportInvalidMetric(ch, c.metadata, fmt.Errorf("invalid IP address: %s", ip))
		return
	}

	// Update the GeoIP info if the IP has changed or if the cache is expired.
	// The cache is valid for one month, with a random jitter to avoid all nodes updating at the same time.
	now := time.Now()
	var geoIP *GeoIPResponse
	st, err := c.loadState()
	useCache := err == nil && st.IP == ip && now.Before(st.NextFetch)

	if useCache {
		geoIP = &st.Geo
	} else {
		geoIP, err = getGeoIP(c.client, ip, c.key)
		if err != nil {
			ReportInvalidMetric(ch, c.metadata, err)
			return
		}
		nextMonth := now.AddDate(0, 1, 0)
		dur := nextMonth.Sub(now)
		jitter := time.Duration(rand.Int63n(int64(dur)))
		newState := &cacheState{IP: ip, Geo: *geoIP, NextFetch: now.Add(jitter)}
		if err := c.saveState(newState); err != nil {
			slog.Error("failed to save geoip state", "error", err)
		}
	}

	geoMetric, err := prometheus.NewConstMetric(
		c.metadata,
		prometheus.GaugeValue,
		1,
		ip,
		geoIP.Data.Location.Country.Alpha2,
		geoIP.Data.Location.Country.Name,
		geoIP.Data.Location.Region.Alpha2,
		geoIP.Data.Location.Region.Name,
		geoIP.Data.Location.City.Name,
		geoIP.Data.Location.Zip,
	)
	if err != nil {
		slog.Error("Failed to create geo metric", "error", err)
		return
	}

	latMetric, err := prometheus.NewConstMetric(
		c.latitude,
		prometheus.GaugeValue,
		geoIP.Data.Location.Latitude,
		ip,
	)
	if err != nil {
		slog.Error("Failed to create latitude metric", "error", err)
		return
	}

	lonMetric, err := prometheus.NewConstMetric(
		c.longitude,
		prometheus.GaugeValue,
		geoIP.Data.Location.Longitude,
		ip,
	)
	if err != nil {
		slog.Error("Failed to create longitude metric", "error", err)
		return
	}

	ch <- geoMetric
	ch <- latMetric
	ch <- lonMetric
}

func getPublicIP(client *resty.Client) (string, error) {
	ipResp := new(IPResponse)
	if err := utils.DoJSONRequest(client, ipifyURL, ipResp); err != nil {
		return "", fmt.Errorf("error getting public ip: %w", err)
	}
	return ipResp.IP, nil
}

func getGeoIP(client *resty.Client, ip, key string) (*GeoIPResponse, error) {
	geoIP := new(GeoIPResponse)
	url := fmt.Sprintf(ipBaseUrl, ip, key)
	if err := utils.DoJSONRequest(client, url, geoIP); err != nil {
		return nil, fmt.Errorf("error getting geoip: %w", err)
	}
	return geoIP, nil
}
