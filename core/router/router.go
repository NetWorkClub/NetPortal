// Package router implements NetPortal's opt-in visual routing constructor IR,
// validation, deterministic two-tier rule evaluator, and sticky-fallback state machine.
package router

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/NetWorkClub/NetPortal/core/api"
)

// Flow describes a connection/packet metadata tuple for rule evaluation and UI testing.
type Flow struct {
	Process        string
	AndroidPackage string
	AppPath        string
	Domain         string
	DstIP          string
	DstPort        int
	Network        string // "tcp" or "udp"
	L7             string // "tls", "http", "quic", etc.
	Inbound        string // "tun", "proxy", "lan"
}

// NewEmptyProfile creates a RoutingProfile with strictly zero hidden default rules.
func NewEmptyProfile(id, nameRu, nameEn, defaultTarget string) api.RoutingProfile {
	if defaultTarget == "" {
		defaultTarget = "direct"
	}
	return api.RoutingProfile{
		ID:            id,
		NameRu:        nameRu,
		NameEn:        nameEn,
		DefaultTarget: defaultTarget,
		Rules:         []api.RoutingRule{},
	}
}

// ValidateProfile verifies regexes, CIDRs, IP pools, and required target fields before saving.
func ValidateProfile(prof api.RoutingProfile) error {
	if prof.ID == "" {
		return errors.New("router: profile ID must not be empty")
	}
	if strings.TrimSpace(prof.DefaultTarget) == "" {
		return errors.New("router: defaultTarget must not be empty")
	}
	for i, r := range prof.Rules {
		if r.Regex != "" {
			if _, err := regexp.Compile(r.Regex); err != nil {
				return fmt.Errorf("router: rule %d (%s) invalid regex: %w", i, r.Name, err)
			}
		}
		for _, cidr := range r.CIDRs {
			if _, err := netip.ParsePrefix(cidr); err != nil {
				if _, addrErr := netip.ParseAddr(cidr); addrErr != nil {
					return fmt.Errorf("router: rule %d (%s) invalid CIDR %q", i, r.Name, cidr)
				}
			}
		}
		for _, pool := range r.IPPools {
			if _, _, err := parseIPPool(pool); err != nil {
				return fmt.Errorf("router: rule %d (%s) invalid IP pool %q: %w", i, r.Name, pool, err)
			}
		}
		if strings.TrimSpace(r.Target) == "" {
			return fmt.Errorf("router: rule %d (%s) missing target", i, r.Name)
		}
	}
	return nil
}

// Evaluate runs two-tier deterministic first-match-wins evaluation across enabled rules.
func Evaluate(prof api.RoutingProfile, flow Flow) (target string, matched *api.RoutingRule, reason string) {
	// Tier 1: App-tier rules first, then Combined, then Network-tier.
	ordered := make([]api.RoutingRule, 0, len(prof.Rules))
	for _, tier := range []api.RuleTier{api.RuleTierApp, api.RuleTierCombined, api.RuleTierNetwork} {
		for _, r := range prof.Rules {
			if r.Enabled && r.Tier == tier {
				ordered = append(ordered, r)
			}
		}
	}

	for i := range ordered {
		r := ordered[i]
		ok, why := matchRule(r, flow)
		if r.Invert {
			ok = !ok
			if ok {
				why = "inverted match: " + r.Name
			}
		}
		if ok {
			cp := r
			return r.Target, &cp, why
		}
	}

	def := prof.DefaultTarget
	if def == "" {
		def = "direct"
	}
	return def, nil, "default target"
}

func matchRule(r api.RoutingRule, flow Flow) (bool, string) {
	if r.Inbound != "" && r.Inbound != api.InboundAny && !strings.EqualFold(string(r.Inbound), flow.Inbound) {
		return false, ""
	}

	// App matcher evaluation
	if r.AppMode != "" && r.AppMode != api.AppMatchOff {
		appMatched := matchAppList(r, flow)
		if r.AppMode == api.AppMatchInclude && !appMatched {
			return false, ""
		}
		if r.AppMode == api.AppMatchExclude && appMatched {
			return false, ""
		}
		if r.Tier == api.RuleTierApp && !r.ContinueToNetwork {
			return true, "matched app rule " + r.Name
		}
	} else if r.Tier == api.RuleTierApp {
		return false, ""
	}

	// Network matcher evaluation (all specified conditions must match; at least one condition needed for network/combined)
	hasNetCond := false

	if len(r.Domains) > 0 || len(r.Suffixes) > 0 || len(r.Keywords) > 0 || r.Regex != "" {
		hasNetCond = true
		if !matchDomainConditions(r, flow.Domain) {
			return false, ""
		}
	}

	if len(r.CIDRs) > 0 || len(r.IPPools) > 0 {
		hasNetCond = true
		if !matchIPConditions(r, flow.DstIP) {
			return false, ""
		}
	}

	if r.Network != "" && r.Network != api.TransportBoth {
		hasNetCond = true
		if !strings.EqualFold(string(r.Network), flow.Network) {
			return false, ""
		}
	}

	if len(r.Ports) > 0 {
		hasNetCond = true
		if !matchPortList(r.Ports, flow.DstPort) {
			return false, ""
		}
	}

	if len(r.L7) > 0 {
		hasNetCond = true
		if !containsFold(r.L7, flow.L7) {
			return false, ""
		}
	}

	if r.IpVer != "" && r.IpVer != api.IpVerAny && flow.DstIP != "" {
		hasNetCond = true
		addr, err := netip.ParseAddr(flow.DstIP)
		if err != nil {
			return false, ""
		}
		if r.IpVer == api.IpVerIPv4 && !addr.Is4() {
			return false, ""
		}
		if r.IpVer == api.IpVerIPv6 && !addr.Is6() {
			return false, ""
		}
	}

	if !hasNetCond && r.Tier != api.RuleTierCombined {
		return false, ""
	}
	return true, "matched rule " + r.Name
}

func matchAppList(r api.RoutingRule, flow Flow) bool {
	if flow.Process != "" && containsFold(r.Apps, flow.Process) {
		return true
	}
	if flow.AndroidPackage != "" && containsFold(r.AndroidApps, flow.AndroidPackage) {
		return true
	}
	if flow.AppPath != "" && containsFold(r.AppPaths, flow.AppPath) {
		return true
	}
	return false
}

func matchDomainConditions(r api.RoutingRule, domain string) bool {
	d := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
	if d == "" {
		return false
	}
	for _, exact := range r.Domains {
		if strings.ToLower(strings.TrimSpace(exact)) == d {
			return true
		}
	}
	for _, sfx := range r.Suffixes {
		s := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(sfx), "."))
		if s != "" && (d == s || strings.HasSuffix(d, "."+s)) {
			return true
		}
	}
	for _, kw := range r.Keywords {
		k := strings.ToLower(strings.TrimSpace(kw))
		if k != "" && strings.Contains(d, k) {
			return true
		}
	}
	if r.Regex != "" {
		if re, err := regexp.Compile(r.Regex); err == nil && re.MatchString(d) {
			return true
		}
	}
	return false
}

func matchIPConditions(r api.RoutingRule, dstIP string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(dstIP))
	if err != nil {
		return false
	}
	for _, raw := range r.CIDRs {
		if p, err := netip.ParsePrefix(strings.TrimSpace(raw)); err == nil && p.Contains(addr) {
			return true
		}
		if a, err := netip.ParseAddr(strings.TrimSpace(raw)); err == nil && a == addr {
			return true
		}
	}
	for _, pool := range r.IPPools {
		start, end, err := parseIPPool(pool)
		if err == nil && addr.Compare(start) >= 0 && addr.Compare(end) <= 0 {
			return true
		}
	}
	return false
}

func parseIPPool(raw string) (netip.Addr, netip.Addr, error) {
	left, right, ok := strings.Cut(strings.TrimSpace(raw), "-")
	if !ok {
		return netip.Addr{}, netip.Addr{}, errors.New("expected start-end range")
	}
	start, err := netip.ParseAddr(strings.TrimSpace(left))
	if err != nil {
		return netip.Addr{}, netip.Addr{}, err
	}
	end, err := netip.ParseAddr(strings.TrimSpace(right))
	if err != nil {
		return netip.Addr{}, netip.Addr{}, err
	}
	if start.Compare(end) > 0 {
		return netip.Addr{}, netip.Addr{}, errors.New("start IP exceeds end IP")
	}
	return start, end, nil
}

func matchPortList(ports []string, dstPort int) bool {
	if dstPort <= 0 {
		return false
	}
	for _, p := range ports {
		trimmed := strings.TrimSpace(p)
		if left, right, ok := strings.Cut(trimmed, "-"); ok {
			lo, err1 := strconv.Atoi(strings.TrimSpace(left))
			hi, err2 := strconv.Atoi(strings.TrimSpace(right))
			if err1 == nil && err2 == nil && dstPort >= lo && dstPort <= hi {
				return true
			}
			continue
		}
		if single, err := strconv.Atoi(trimmed); err == nil && dstPort == single {
			return true
		}
	}
	return false
}

func containsFold(slice []string, target string) bool {
	if target == "" {
		return false
	}
	for _, item := range slice {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

// StickyChain implements cross-protocol sticky-primary failover with hysteresis.
type StickyChain struct {
	mu                  sync.Mutex
	hops                []string
	activeIdx           int
	consecutiveFailures int
	failThreshold       int
	stickBackOnRecovery bool
}

// NewStickyChain creates a sticky-fallback chain ordered from primary (index 0) to backups.
func NewStickyChain(hops []string, failThreshold int, stickBackOnRecovery bool) (*StickyChain, error) {
	if len(hops) == 0 {
		return nil, errors.New("router: sticky chain requires at least one hop")
	}
	if failThreshold <= 0 {
		failThreshold = 2
	}
	return &StickyChain{
		hops:                append([]string(nil), hops...),
		failThreshold:       failThreshold,
		stickBackOnRecovery: stickBackOnRecovery,
	}, nil
}

// Active returns the currently pinned hop ID.
func (c *StickyChain) Active() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hops[c.activeIdx]
}

// ReportHealth updates healthcheck state for the current active hop (or primary recovery).
func (c *StickyChain) ReportHealth(activeHealthy bool, primaryHealthy bool) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeIdx > 0 && c.stickBackOnRecovery && primaryHealthy {
		c.activeIdx = 0
		c.consecutiveFailures = 0
		return c.hops[0]
	}

	if activeHealthy {
		c.consecutiveFailures = 0
		return c.hops[c.activeIdx]
	}

	c.consecutiveFailures++
	if c.consecutiveFailures >= c.failThreshold && c.activeIdx+1 < len(c.hops) {
		c.activeIdx++
		c.consecutiveFailures = 0
	}
	return c.hops[c.activeIdx]
}
