package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"text/template"
	"time"

	"golang.org/x/exp/maps"

	"github.com/miekg/dns"

	"dnspropagation/internal/servers"
)

type ServerResponse struct {
	Status       string
	Server       servers.DNSServer
	RecordType   string
	RecordValues []string
}

func (r ServerResponse) String() string {
	var b bytes.Buffer
	if err := responseTemplate.Execute(&b, r); err != nil {
		return ""
	}
	return b.String()
}

var responseTemplate *template.Template

const StatusSuccess = "OK"
const StatusError = "ERROR"

const ColorRed = "\033[31m"
const ColorGreen = "\033[32m"
const ColorReset = "\033[0m"

const ServerResponseTimeout = 5 * time.Second

var flagListDNSServers = flag.Bool("dns-servers", false, "List of publicly accessible DNS servers used in propagation checks.")
var flagDomain = flag.String("domain", "", "Domain for accessing records.")
var flagRecordType = flag.String("record-type", "", fmt.Sprintf("Request for DNS record type. Supported types: %s",
	strings.Join(maps.Keys(SupportedDNSTypes), ",")))

var SupportedDNSTypes = map[string]uint16{
	"A":     dns.TypeA,
	"AAAA":  dns.TypeAAAA,
	"CNAME": dns.TypeCNAME,
	"MX":    dns.TypeMX,
	"NS":    dns.TypeNS,
	"PTR":   dns.TypePTR,
	"SRV":   dns.TypeSRV,
	"SOA":   dns.TypeSOA,
	"TXT":   dns.TypeTXT,
	"CAA":   dns.TypeCAA,
	"DS":    dns.TypeDS,
}

func init() {
	const templ = `
DNS Server:
 Name:           {{.Server.Name}}
 IP:             {{.Server.IP}}
 Location:       {{.Server.Location}}
 Country Code:   {{.Server.CountryCode}}
 Status:         {{clr .Status}}
 Type:           {{.RecordType}}
 Values:         {{join .RecordValues ","}}`
	clr := func(s string) string {
		switch s {
		case StatusSuccess:
			return ColorGreen + s + ColorReset
		case StatusError:
			return ColorRed + s + ColorReset
		default:
			return s
		}
	}
	var err error
	responseTemplate, err = template.New("responseTemplate").
		Funcs(template.FuncMap{"join": strings.Join, "clr": clr}).Parse(templ)
	if err != nil {
		panic(fmt.Sprintf("Response record template is not valid: %s", err))
	}
}

func getA(r dns.RR) []string {
	if v, ok := r.(*dns.A); ok && v != nil {
		return []string{v.A.String()}
	}
	return []string{}
}

func getAAAA(r dns.RR) []string {
	if v, ok := r.(*dns.AAAA); ok && v != nil {
		return []string{v.AAAA.String()}
	}
	return []string{}
}

func getCNAME(r dns.RR) []string {
	if v, ok := r.(*dns.CNAME); ok && v != nil {
		return []string{v.Target}
	}
	return []string{}
}

func getMX(r dns.RR) []string {
	if v, ok := r.(*dns.MX); ok && v != nil {
		return []string{v.Mx}
	}
	return []string{}
}

func getNS(r dns.RR) []string {
	if v, ok := r.(*dns.NS); ok && v != nil {
		return []string{v.Ns}
	}
	return []string{}
}

func getPTR(r dns.RR) []string {
	if v, ok := r.(*dns.PTR); ok && v != nil {
		return []string{v.Ptr}
	}
	return []string{}
}

func getSRV(r dns.RR) []string {
	if v, ok := r.(*dns.SRV); ok && v != nil {
		return []string{v.Target}
	}
	return []string{}
}

func getSOA(r dns.RR) []string {
	if v, ok := r.(*dns.SOA); ok && v != nil {
		return []string{fmt.Sprintf("Ns: %s, Mbox: %s", v.Ns, v.Mbox)}
	}
	return []string{}
}

func getTXT(r dns.RR) []string {
	if v, ok := r.(*dns.TXT); ok && v != nil {
		return v.Txt
	}
	return []string{}
}

func getCAA(r dns.RR) []string {
	if v, ok := r.(*dns.CAA); ok && v != nil {
		return []string{v.Value}
	}
	return []string{}
}

func getDS(r dns.RR) []string {
	if v, ok := r.(*dns.DS); ok && v != nil {
		return []string{v.Digest}
	}
	return []string{}
}

func extractFn(rtype uint16) func(dns.RR) []string {
	switch rtype {
	case dns.TypeA:
		return getA
	case dns.TypeAAAA:
		return getAAAA
	case dns.TypeCNAME:
		return getCNAME
	case dns.TypeMX:
		return getMX
	case dns.TypeNS:
		return getNS
	case dns.TypePTR:
		return getPTR
	case dns.TypeSRV:
		return getSRV
	case dns.TypeSOA:
		return getSOA
	case dns.TypeTXT:
		return getTXT
	case dns.TypeCAA:
		return getCAA
	case dns.TypeDS:
		return getDS
	default:
		return nil
	}
}

func queryDNSServers(domain, record string, result chan<- ServerResponse) error {
	rtype, ok := SupportedDNSTypes[record]
	if !ok {
		return fmt.Errorf("record type %s is not recognized as valid DNS record type or it's not supported", record)
	}
	extractAnswer := extractFn(rtype)
	if extractAnswer == nil {
		return fmt.Errorf("function for extracting answer for record type: %d is not defined", rtype)
	}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), rtype)
	c := &dns.Client{Dialer: &net.Dialer{Timeout: ServerResponseTimeout}}
	for _, server := range servers.DNSServers {
		go func(s servers.DNSServer) {
			if r, _, err := c.Exchange(m, s.IP); err == nil {
				answers := make([]string, 0)
				for _, record := range r.Answer {
					answers = append(answers, extractAnswer(record)...)
				}
				result <- ServerResponse{
					Status:       StatusSuccess,
					Server:       s,
					RecordType:   record,
					RecordValues: answers,
				}
			} else {
				result <- ServerResponse{
					Status:     StatusError,
					Server:     s,
					RecordType: record,
				}
			}
		}(server)
	}

	return nil
}

func printDNSServers() {
	fmt.Println("List of publicly accessible DNS servers used in application:")
	for _, s := range servers.DNSServers {
		fmt.Println(s)
	}
}

func printResults(domain, record string) {
	_, ok := SupportedDNSTypes[strings.ToUpper(record)]
	if !ok {
		fmt.Fprintf(os.Stderr, "'%s' DNS record type is not supported\n", record)
		return
	}
	res := make(chan ServerResponse)
	if err := queryDNSServers(domain, record, res); err != nil {
		fmt.Fprintf(os.Stderr, "Querying DNS serves failed, due to error: %s\name", err)
		return
	}
	for range servers.DNSServers {
		fmt.Println(<-res)
	}
}

func main() {
	flag.Parse()
	if flag.NFlag() == 0 {
		fmt.Print("DNSPropagation Checker is a tool that reports on the status of DNS propagation across publicly available DNS servers.\nThe list of public DNS servers is built into the application and can be accessed using the '-dns-servers' command line argument.\n\nList of command line arguments:\n\n")
		flag.PrintDefaults()
		return
	}
	if *flagListDNSServers {
		printDNSServers()
		return
	}
	if len(*flagDomain) == 0 || len(*flagRecordType) == 0 {
		fmt.Fprintln(os.Stderr, "Both -domain and -record-type should be specified.")
		return
	}
	printResults(*flagDomain, *flagRecordType)
}
