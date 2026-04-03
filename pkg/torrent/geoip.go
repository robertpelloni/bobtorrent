package torrent

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

// GeoIPService provides location information for IP addresses.
type GeoIPService struct {
	db *geoip2.Reader
}

// NewGeoIPService creates a new GeoIPService using the provided MaxMind database file.
func NewGeoIPService(dbPath string) (*GeoIPService, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoIP database: %w", err)
	}
	return &GeoIPService{db: db}, nil
}

// Close closes the underlying GeoIP database.
func (s *GeoIPService) Close() error {
	return s.db.Close()
}

// Lookup returns the country code and approximate coordinates for the given IP address.
func (s *GeoIPService) Lookup(ipStr string) (country string, lat, lon float64, err error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", 0, 0, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	record, err := s.db.City(ip)
	if err != nil {
		return "", 0, 0, fmt.Errorf("lookup failed: %w", err)
	}

	country = record.Country.IsoCode
	lat = record.Location.Latitude
	lon = record.Location.Longitude
	return country, lat, lon, nil
}
