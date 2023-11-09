/*
List of public DNS servers extracted from https://public-dns.info/
*/
package servers

import (
	"fmt"
	"text/template"
	"bytes"
	"golang.org/x/exp/slog"
)

type DNSServer struct {
	IP          string // IP of DNS server
	Name        string // Name of a provider company
	Location    string // Latitude and longitude of place location (comma separated) or just name of place
	CountryCode string // ISO alpha-2 country code
}

func (s DNSServer) String() string {
	var b bytes.Buffer
	if err := templ.Execute(&b, s); err != nil {
		slog.Error("Error while execting template", "erroe", err)
		return ""
	}
	return b.String()
}

func init() {
	const t = `
IP:          {{.IP}}
Name:        {{.Name}}
Location:    {{.Location}}
CountryCode: {{.CountryCode}}
`
	var err error
	templ, err = template.New("dnsServerTemplate").Parse(t)
	if err != nil {
		panic(fmt.Sprintf("Public DNS server template is not valid: %s", err))
	}
}

var templ *template.Template

var DNSServers = []DNSServer{
	{IP: "91.197.68.143:53", Name: "LLC Likonet", Location: "Kyiv", CountryCode: "UA"},
	{IP: "88.221.162.33:53", Name: "Akamai International B.V.", Location: "unknown", CountryCode: "NL"},
	{IP: "194.28.33.190:53", Name: "Info-Net Uslugi Teleinformatyczne S.C.", Location: "Ostrów Wielkopolski", CountryCode: "PL"},
	{IP: "91.26.45.59:53", Name: "Deutsche Telekom AG", Location: "Siegen", CountryCode: "DE"},
	{IP: "188.116.92.133:53", Name: "SELECT SYSTEM, s.r.o.", Location: "Sumperk", CountryCode: "CZ"},
	{IP: "187.102.222.46:53", Name: "MASTERNET TELECOMUNICACAO LTD", Location: "Sao Pedro do Suacui", CountryCode: "BR"},
	{IP: "96.45.46.46:53", Name: "FORTINET", Location: "Guelph", CountryCode: "CA"},
	{IP: "80.75.35.106:53", Name: "Telekom Austria", Location: "unknown", CountryCode: "AT"},
	{IP: "170.64.147.31:53", Name: "DIGITALOCEAN-ASN", Location: "Sydney", CountryCode: "AU"},
	{IP: "98.100.136.231:53", Name: "TWC-10796-MIDWEST", Location: "Milwaukee", CountryCode: "US"},
	{IP: "24.218.117.81:53", Name: "COMCAST-7922", Location: "Hamden", CountryCode: "US"},
	{IP: "8.8.4.4:53", Name: "Google", Location: "unknown", CountryCode: "US"},
}
