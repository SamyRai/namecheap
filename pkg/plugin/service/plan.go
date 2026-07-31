package service

import (
	"zonekit/pkg/dnsrecord"
)

// PlanOptions controls how a desired record set is reconciled with a zone.
type PlanOptions struct {
	// Replace drops records the desired set supersedes. Without it, a
	// same-kind collision is reported as a conflict and nothing is written.
	Replace bool
	// ForcePolicy overwrites existing SPF/DMARC policies instead of extending
	// them.
	ForcePolicy bool
}

// Plan is the reconciliation of a desired record set against an existing zone.
//
// Providers such as Namecheap apply DNS through a whole-zone write, so the
// published set must be the complete desired state. Plan makes the three
// categories explicit -- what survives untouched, what is replaced, and what is
// added -- so a caller can show them before committing and so nothing is
// dropped by omission.
type Plan struct {
	// Preserved are existing records the desired set does not own. They must
	// be republished verbatim or they are deleted.
	Preserved []dnsrecord.Record
	// Superseded are existing records the desired set replaces.
	Superseded []dnsrecord.Record
	// Desired is the service's own record set, after policy merging.
	Desired []dnsrecord.Record
}

// Conflicts returns the superseded records as "host TYPE" strings, for
// reporting when Replace was not requested.
func (p Plan) Conflicts() []string {
	out := make([]string, 0, len(p.Superseded))
	for _, r := range p.Superseded {
		out = append(out, r.HostName+" "+r.RecordType)
	}
	return out
}

// Records returns the complete zone to publish. Under Replace the superseded
// records are dropped; otherwise they are retained alongside the desired set.
func (p Plan) Records(replace bool) []dnsrecord.Record {
	out := append([]dnsrecord.Record{}, p.Preserved...)
	if !replace {
		out = append(out, p.Superseded...)
	}
	return append(out, p.Desired...)
}

// PlanZone reconciles desired against existing.
//
// Extracted so that every path which writes a zone -- the service setup
// command and provider-specific onboarding flows alike -- shares one
// implementation of the preserve/supersede rules, rather than each
// reimplementing them and getting the destructive edge cases wrong.
func PlanZone(existing, desired []dnsrecord.Record, opts PlanOptions) Plan {
	merged := mergePolicyRecords(desired, existing, opts.ForcePolicy)

	var plan Plan
	plan.Desired = merged
	for _, record := range existing {
		if supersededBy(record, merged) {
			plan.Superseded = append(plan.Superseded, record)
		} else {
			plan.Preserved = append(plan.Preserved, record)
		}
	}
	return plan
}
