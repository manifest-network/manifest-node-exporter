package collectors

import (
	"fmt"
	"log/slog"
	"net"

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
}

type IPResponse struct {
	IP string `json:"ip"`
}

type GeoIPResponse struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	RegionCode  string  `json:"region_code"`
	RegionName  string  `json:"region_name"`
	City        string  `json:"city"`
	ZipCode     string  `json:"zip_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

const (
	ipifyURL           = "https://api.ipify.org?format=json"
	freeGeoIPURLFormat = "https://freegeoip.live/json/%s"
)

func NewGeoIPCollector() *GeoIPCollector {
	return &GeoIPCollector{
		client: resty.New().SetHeader("Accept", "application/json").SetTimeout(pkg.ClientTimeout).SetRetryCount(pkg.ClientRetry),
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

	geoIP, err := getGeoIP(c.client, ip)
	if err != nil {
		ReportInvalidMetric(ch, c.metadata, err)
		return
	}

	if geoIP == nil {
		ReportInvalidMetric(ch, c.metadata, fmt.Errorf("geoIP response is nil"))
		return
	}

	geoMetric, err := prometheus.NewConstMetric(
		c.metadata,
		prometheus.GaugeValue,
		1,
		ip,
		geoIP.CountryCode,
		geoIP.CountryName,
		geoIP.RegionCode,
		geoIP.RegionName,
		geoIP.City,
		geoIP.ZipCode,
	)
	if err != nil {
		slog.Error("Failed to create geo metric", "error", err)
		return
	}

	latMetric, err := prometheus.NewConstMetric(
		c.latitude,
		prometheus.GaugeValue,
		geoIP.Latitude,
		ip,
	)
	if err != nil {
		slog.Error("Failed to create latitude metric", "error", err)
		return
	}

	lonMetric, err := prometheus.NewConstMetric(
		c.longitude,
		prometheus.GaugeValue,
		geoIP.Longitude,
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

func getGeoIP(client *resty.Client, ip string) (*GeoIPResponse, error) {
	geoIP := new(GeoIPResponse)
	url := fmt.Sprintf(freeGeoIPURLFormat, ip)
	if err := utils.DoJSONRequest(client, url, geoIP); err != nil {
		return nil, fmt.Errorf("error getting geoip: %w", err)
	}
	return geoIP, nil
}
