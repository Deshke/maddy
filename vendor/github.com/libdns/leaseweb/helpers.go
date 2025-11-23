package leaseweb

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// @see https://developer.leaseweb.com/api-docs/domains_v2.html#tag/DNS/operation/domains-resourcerecordsets-post
var supportedTTLs = []int{60, 300, 1800, 3600, 14400, 28800, 43200, 86400}

func fromLibdns(zone string, records []libdns.Record) ([]leasewebRecordSet, error) {
	type recordInfo struct {
		name     string
		typ      string
		ttl      int
		value    string
		consumed bool
	}

	recordInfos := make([]recordInfo, 0, len(records))
	for _, record := range records {
		rr := record.RR()
		recordInfos = append(recordInfos, recordInfo{
			name:  absoluteRecordName(rr.Name, zone),
			typ:   rr.Type,
			ttl:   normalizeTTL(rr.TTL),
			value: rr.Data,
		})
	}

	var errors []string
	var recordSets []leasewebRecordSet

	for currentIdx := range recordInfos {
		currentRecordInfo := &recordInfos[currentIdx]

		if currentRecordInfo.consumed {
			continue
		}
		currentRecordInfo.consumed = true

		var newRecordSet = leasewebRecordSet{
			Name:    currentRecordInfo.name,
			Type:    currentRecordInfo.typ,
			TTL:     currentRecordInfo.ttl,
			Content: []string{currentRecordInfo.value},
		}

		for otherIdx := range recordInfos {
			otherRecordInfo := &recordInfos[otherIdx]
			if otherIdx == currentIdx || otherRecordInfo.consumed {
				continue
			}

			if otherRecordInfo.name == currentRecordInfo.name && otherRecordInfo.typ == currentRecordInfo.typ {
				otherRecordInfo.consumed = true

				if otherRecordInfo.ttl != newRecordSet.TTL {
					errors = append(errors, fmt.Sprintf("Found different TTL values for %s: %d and %d.", newRecordSet.Name, newRecordSet.TTL, otherRecordInfo.ttl))
				}

				newRecordSet.Content = append(newRecordSet.Content, otherRecordInfo.value)
			}
		}

		recordSets = append(recordSets, newRecordSet)
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("%v", errors)
	}

	return recordSets, nil
}

func fromLeaseweb(zone string, recordSets leasewebRecordSets) []libdns.Record {
	var records []libdns.Record
	for _, resourceRecordSet := range recordSets.ResourceRecordSets {
		name := libdns.RelativeName(resourceRecordSet.Name, zone)
		for _, content := range resourceRecordSet.Content {
			rr := libdns.RR{
				Name: name,
				Type: resourceRecordSet.Type,
				TTL:  time.Duration(resourceRecordSet.TTL) * time.Second,
				Data: content,
			}

			// Parse into the type-specific record struct so callers can rely
			// on type assertions. Unknown types are returned as libdns.RR.
			record, err := rr.Parse()
			if err != nil {
				record = rr
			}
			records = append(records, record)
		}
	}
	return records
}

// absoluteRecordName converts a libdns record name into a fully-qualified
// name with a trailing dot, as expected by the Leaseweb API.
//
// libdns uses "@" for the zone root and otherwise relative names, but some
// callers (e.g. dns_challenge_override_domain) may provide fully-qualified
// names, so both forms are accepted.
func absoluteRecordName(name, zone string) string {
	domainName := strings.TrimSuffix(zone, ".")

	if name == "" || name == "@" {
		return domainName + "."
	}

	// Cleanup record name and ensure it ends with domain.ext[dot] even if dns_challenge_override_domain is set
	// trimming both zone & domainName is probably overzealous, but better be safe then sorry
	// Example:
	//   zone: example.com.
	//   domainName: example.com
	//   dnsRecord.Name 1: _acme-challenge.example.com
	//   dnsRecord.Name 2: _acme-challenge.example.com.
	//   dnsRecord.Name 3: _acme-challenge.
	//   all after cleanup -> _acme-challenge.example.com.
	relativeName := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(name, "."), domainName), ".")
	if relativeName == "" {
		return domainName + "."
	}

	return relativeName + "." + domainName + "."
}

// normalizeTTL returns the TTL in seconds, falling back to the lowest
// supported value when the requested TTL is not supported by Leaseweb.
func normalizeTTL(ttl time.Duration) int {
	seconds := int(ttl.Seconds())
	if !slices.Contains(supportedTTLs, seconds) {
		// Use the first listed TTL if the user did not provide a TTL or provided a unsupported value
		// It would probably be nice to log a warning about this, but that doesn't seem to be supported in libdns
		// Note: Leaseweb uses a default value of 3600. But as this lib will probably mostly be used
		// with caddy-dns/leaseweb to solve ACME DNS-01 challenges, let's use the lowest possible value
		return supportedTTLs[0]
	}
	return seconds
}
