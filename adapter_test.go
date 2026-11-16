package main

import (
	"testing"
	"time"
)

func TestPlatformAdapterIsolationAndLiveDelta(t *testing.T) {
	// Query live Windows interface table for each protocol family
	inStealth, outStealth, foundStealth := getPlatformWindscribeTrafficForProtocol("stealth")
	inUDP, outUDP, foundUDP := getPlatformWindscribeTrafficForProtocol("udp")
	inWG, outWG, foundWG := getPlatformWindscribeTrafficForProtocol("wireguard")
	inIKE, outIKE, foundIKE := getPlatformWindscribeTrafficForProtocol("ikev2")

	t.Logf("stealth:   found=%v in=%d out=%d", foundStealth, inStealth, outStealth)
	t.Logf("udp:       found=%v in=%d out=%d", foundUDP, inUDP, outUDP)
	t.Logf("wireguard: found=%v in=%d out=%d", foundWG, inWG, outWG)
	t.Logf("ikev2:     found=%v in=%d out=%d", foundIKE, inIKE, outIKE)

	if foundStealth {
		if !foundUDP || inStealth != inUDP || outStealth != outUDP {
			t.Fatalf("expected OpenVPN-family protocols (stealth/udp) to share the active OpenVPN adapter")
		}
		if foundWG || foundIKE {
			t.Fatalf("expected wireguard/ikev2 adapters to stay isolated while OpenVPN/Stealth is active (wg=%v, ikev2=%v)", foundWG, foundIKE)
		}

		// Measure 1s live delta while active
		time.Sleep(1 * time.Second)
		in2, out2, found2 := getPlatformWindscribeTrafficForProtocol("stealth")
		t.Logf("after 1s:  found=%v in=%d (delta=%d) out=%d (delta=%d)",
			found2, in2, in2-inStealth, out2, out2-outStealth)
	}
}
