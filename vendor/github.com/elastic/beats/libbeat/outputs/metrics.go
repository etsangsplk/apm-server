package outputs

import "github.com/elastic/beats/libbeat/monitoring"

// Stats implements the Observer interface, for collecting metrics on common
// outputs events.
type Stats struct {
	//
	// Output event stats
	//
	batches *monitoring.Uint // total number of batches processed by output
	events  *monitoring.Uint // total number of events processed by output

	acked  *monitoring.Uint // total number of events ACKed by output
	failed *monitoring.Uint // total number of events failed in output
	active *monitoring.Uint // events sent and waiting for ACK/fail from output

	//
	// Output network connection stats
	//
	writeBytes  *monitoring.Uint // total amount of bytes written by output
	writeErrors *monitoring.Uint // total number of errors on write

	readBytes  *monitoring.Uint // total amount of bytes read
	readErrors *monitoring.Uint // total number of errors while waiting for response on output
}

// NewStats creates a new Stats instance using a backing monitoring registry.
// This function will create and register a number of metrics with the registry passed.
// The registry must not be null.
func NewStats(reg *monitoring.Registry) *Stats {
	return &Stats{
		batches: monitoring.NewUint(reg, "events.batches"),
		events:  monitoring.NewUint(reg, "events.total"),
		acked:   monitoring.NewUint(reg, "events.acked"),
		failed:  monitoring.NewUint(reg, "events.failed"),
		active:  monitoring.NewUint(reg, "events.active"),

		writeBytes:  monitoring.NewUint(reg, "write.bytes"),
		writeErrors: monitoring.NewUint(reg, "write.errors"),

		readBytes:  monitoring.NewUint(reg, "read.bytes"),
		readErrors: monitoring.NewUint(reg, "read.errors"),
	}
}

// NewBatch updates active batch and event metrics.
func (s *Stats) NewBatch(n int) {
	if s != nil {
		s.batches.Inc()
		s.events.Add(uint64(n))
		s.active.Add(uint64(n))
	}
}

// Acked updates active and acked event metrics.
func (s *Stats) Acked(n int) {
	if s != nil {
		s.acked.Add(uint64(n))
		s.active.Sub(uint64(n))
	}
}

// Failed updates active and failed event metrics.
func (s *Stats) Failed(n int) {
	if s != nil {
		s.failed.Add(uint64(n))
		s.active.Sub(uint64(n))
	}
}

// Dropped updates total number of event drops as reported by the output.
// Outputs will only report dropped events on fatal errors which lead to the
// event not being publishabel. For example encoding errors or total event size
// being bigger then maximum supported event size.
func (s *Stats) Dropped(n int) {
	// number of dropped events (e.g. encoding failures)
	if s != nil {
		s.active.Sub(uint64(n))
	}
}

// Cancelled updates the active event metrics.
func (s *Stats) Cancelled(n int) {
	if s != nil {
		s.active.Sub(uint64(n))
	}
}

// WriteError increases the write I/O error metrics.
func (s *Stats) WriteError(err error) {
	if s != nil {
		s.writeErrors.Inc()
	}
}

// WriteBytes updates the total number of bytes written/send by an output.
func (s *Stats) WriteBytes(n int) {
	if s != nil {
		s.writeBytes.Add(uint64(n))
	}
}

// ReadError increases the read I/O error metrics.
func (s *Stats) ReadError(err error) {
	if s != nil {
		s.readErrors.Inc()
	}
}

// ReadBytes updates the total number of bytes read/received by an output.
func (s *Stats) ReadBytes(n int) {
	if s != nil {
		s.readBytes.Add(uint64(n))
	}
}
