package daemon

import (
	"net"
	"net/http"
	"net/url"
)

// loopbackGuard is a local patch (not upstream): in loopback mode there is no
// bearer token, so a web page that rebinds its own domain to 127.0.0.1 (DNS
// rebinding) would become same-origin with the daemon and could drive /mcp.
// Rejecting any request whose Host — or Origin, when a browser sends one — is
// not a loopback name closes that hole without breaking local MCP clients or
// the /pair page opened at http://127.0.0.1:8765/pair.
func loopbackGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !isLoopbackHost(u.Host) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isLoopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
