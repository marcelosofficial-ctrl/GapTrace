package logcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/marcelosofficial-ctrl/gaptrace/internal/engine"
	"github.com/marcelosofficial-ctrl/gaptrace/internal/probe"
)

var header = []string{
	"timestamp",
	"status",
	"dns_ok",
	"dns_ms",
	"dns_target",
	"dns_error",
	"tcp_ok",
	"tcp_ms",
	"tcp_target",
	"tcp_error",
	"http_ok",
	"http_ms",
	"http_target",
	"http_error",
}

type Writer struct {
	file *os.File
	csv  *csv.Writer
}

func Open(path string) (*Writer, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	writer := csv.NewWriter(file)

	if err := writer.Write(header); err != nil {
		_ = file.Close()
		return nil, err
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		_ = file.Close()
		return nil, err
	}

	return &Writer{file: file, csv: writer}, nil
}

func (w *Writer) Write(sample engine.Sample) error {
	record := []string{
		sample.Timestamp.Format(time.RFC3339Nano),
		string(sample.Status),
		strconv.FormatBool(sample.DNS.OK),
		durationMS(sample.DNS.Latency),
		sample.DNS.Target,
		sample.DNS.Error,
		strconv.FormatBool(sample.TCP.OK),
		durationMS(sample.TCP.Latency),
		sample.TCP.Target,
		sample.TCP.Error,
		strconv.FormatBool(sample.HTTP.OK),
		durationMS(sample.HTTP.Latency),
		sample.HTTP.Target,
		sample.HTTP.Error,
	}

	if err := w.csv.Write(record); err != nil {
		return err
	}

	w.csv.Flush()
	return w.csv.Error()
}

func (w *Writer) Close() error {
	w.csv.Flush()

	if err := w.csv.Error(); err != nil {
		_ = w.file.Close()
		return err
	}

	return w.file.Close()
}

func Read(path string) ([]engine.Sample, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	gotHeader, err := reader.Read()
	if err != nil {
		return nil, err
	}

	if len(gotHeader) != len(header) {
		return nil, fmt.Errorf("unexpected CSV header width")
	}

	var samples []engine.Sample

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		sample, err := parseRecord(record)
		if err != nil {
			return nil, err
		}

		samples = append(samples, sample)
	}

	return samples, nil
}

func parseRecord(record []string) (engine.Sample, error) {
	if len(record) != len(header) {
		return engine.Sample{}, fmt.Errorf("unexpected CSV record width")
	}

	timestamp, err := time.Parse(time.RFC3339Nano, record[0])
	if err != nil {
		return engine.Sample{}, err
	}

	dnsResult, err := parseProbe("dns", record[2], record[3], record[4], record[5])
	if err != nil {
		return engine.Sample{}, err
	}

	tcpResult, err := parseProbe("tcp", record[6], record[7], record[8], record[9])
	if err != nil {
		return engine.Sample{}, err
	}

	httpResult, err := parseProbe("http", record[10], record[11], record[12], record[13])
	if err != nil {
		return engine.Sample{}, err
	}

	return engine.Sample{
		Timestamp: timestamp,
		Status:    engine.Status(record[1]),
		DNS:       dnsResult,
		TCP:       tcpResult,
		HTTP:      httpResult,
	}, nil
}

func parseProbe(name, okText, msText, target, errorText string) (probe.Result, error) {
	ok, err := strconv.ParseBool(okText)
	if err != nil {
		return probe.Result{}, err
	}

	ms, err := strconv.ParseFloat(msText, 64)
	if err != nil {
		return probe.Result{}, err
	}

	return probe.Result{
		Name:    name,
		Target:  target,
		OK:      ok,
		Latency: time.Duration(ms * float64(time.Millisecond)),
		Error:   errorText,
	}, nil
}

func durationMS(d time.Duration) string {
	return fmt.Sprintf("%.3f", float64(d)/float64(time.Millisecond))
}
