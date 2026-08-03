package server

import (
	"html/template"
	"net/http"
	"runtime"
	"time"
)

// serviceName is shown at the top of the status page.
const serviceName = "unifi-ha-presence-mqtt"

// statusView is the data rendered into the / page. It is built under lock from a
// snapshot of the Server so the template never touches shared state directly.
type statusView struct {
	Service   string
	Ready     bool
	Uptime    string
	GoVersion string

	DevicesReady bool
	Devices      []DeviceView

	PollInterval string
}

// statusTmpl is parsed once at package load. The markup is deliberately minimal and
// isolated here so it is the single place to iterate on styling later.
var statusTmpl = template.Must(template.New("status").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Service}} status</title>
</head>
<body>
<h1>{{.Service}}</h1>
<p>Status: {{if .Ready}}ready{{else}}not ready{{end}}</p>
<p>Uptime: {{.Uptime}}</p>
<h2>Devices</h2>
{{if .DevicesReady}}
<ul>
{{range .Devices}}<li>{{.Hostname}}: {{if .Present}}present{{else}}absent{{end}}</li>
{{end}}</ul>
{{else}}
<p>initialising</p>
{{end}}
<h2>Schedule</h2>
<ul>
<li>Poll interval: {{.PollInterval}}</li>
</ul>
<p>Go: {{.GoVersion}}</p>
</body>
</html>
`))

// handleRoot renders the HTML status page. It always returns 200 and never exposes
// credentials or per-device IP/MAC (omitted as mildly sensitive).
func (s *Server) handleRoot(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	view := statusView{
		Service:      serviceName,
		Ready:        s.ready,
		Uptime:       time.Since(s.startedAt).Round(time.Second).String(),
		GoVersion:    runtime.Version(),
		DevicesReady: s.devicesSet,
		Devices:      append([]DeviceView(nil), s.devices...),
		PollInterval: s.cfg.PollInterval.String(),
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = statusTmpl.Execute(w, view)
}
