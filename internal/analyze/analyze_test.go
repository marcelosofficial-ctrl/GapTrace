package analyze

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const header = "timestamp,status,dns_ok,dns_ms,dns_target,dns_error,tcp_ok,tcp_ms,tcp_target,tcp_error,http_ok,http_ms,http_target,http_error\n"

func line(ts, status, tcpOK, tcpMS string) string {
	return ts + "," + status + ",true,10,example.com,," + tcpOK + "," + tcpMS + ",1.1.1.1:443,,true,30,https://www.gstatic.com/generate_204,\n"
}
func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "x.csv")
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAnalysis(t *testing.T) {
	p := write(t, header+
		line("2026-09-11T00:00:00Z", "healthy", "true", "20")+
		line("2026-09-11T00:00:01Z", "dns_failure", "true", "20")+
		line("2026-09-11T00:00:02Z", "offline", "false", "400")+
		line("2026-09-11T00:00:03Z", "healthy", "true", "30")+
		line("2026-09-11T00:00:04Z", "latency_spike", "true", "700"))
	r, err := File(p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Samples != 5 || r.Outages != 1 || r.RecoveredOutages != 1 {
		t.Fatalf("unexpected counts: %+v", r)
	}
	if r.AvailabilityPercent != 60 {
		t.Fatalf("availability=%v", r.AvailabilityPercent)
	}
	if len(r.Events) != 1 || r.Events[0].Status != "mixed_failure" || r.Events[0].DurationMS != 2000 || r.Events[0].Samples != 2 || !r.Events[0].Recovered {
		t.Fatalf("event=%+v", r.Events)
	}
	if r.P95TCPMS != 30 {
		t.Fatalf("p95=%v", r.P95TCPMS)
	}
}
func TestOngoingOutage(t *testing.T) {
	p := write(t, header+line("2026-09-11T00:00:00Z", "offline", "false", "400")+line("2026-09-11T00:00:01Z", "offline", "false", "400"))
	r, err := File(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Events) != 1 || r.Events[0].Recovered || r.Events[0].DurationMS != 1000 {
		t.Fatalf("event=%+v", r.Events)
	}
}
func TestRejectHeader(t *testing.T) {
	p := write(t, strings.Replace(header, "timestamp", "time", 1))
	if _, err := File(p); err == nil {
		t.Fatal("expected error")
	}
}
func TestRejectUnknownStatus(t *testing.T) {
	p := write(t, header+line("2026-09-11T00:00:00Z", "mystery", "true", "20"))
	if _, err := File(p); err == nil {
		t.Fatal("expected error")
	}
}
func TestRejectBackwardsTime(t *testing.T) {
	p := write(t, header+line("2026-09-11T00:00:02Z", "healthy", "true", "20")+line("2026-09-11T00:00:01Z", "healthy", "true", "20"))
	if _, err := File(p); err == nil {
		t.Fatal("expected error")
	}
}
func TestRejectNegativeLatency(t *testing.T) {
	p := write(t, header+line("2026-09-11T00:00:00Z", "healthy", "true", "-1"))
	if _, err := File(p); err == nil {
		t.Fatal("expected error")
	}
}
func TestEmpty(t *testing.T) {
	p := write(t, header)
	r, err := File(p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Samples != 0 || r.Outages != 0 {
		t.Fatalf("report=%+v", r)
	}
}
