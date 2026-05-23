package web

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

func (s *Server) githubPreviewPathPrefix() string {
	return normalizePreviewPathPrefix(s.cfg.GitHub.PreviewPathPrefix)
}

func normalizePreviewPathPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "/previews"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimRight(prefix, "/")
}

func (s *Server) handleGitHubAutomationPreviewProxy(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil || !s.automation.Enabled() {
		WriteError(w, NewAPIError(ErrCodeNotFound, "GitHub automation is disabled"))
		return
	}

	slug, upstreamPath, ok := parsePreviewProxyPath(r.URL.Path, s.githubPreviewPathPrefix())
	if !ok {
		WriteError(w, NewAPIError(ErrCodeNotFound, "Preview route not found"))
		return
	}
	target, ok := s.automation.GetPreviewTarget(slug)
	if !ok {
		WriteError(w, NewAPIError(ErrCodeNotFound, "Preview target not found"))
		return
	}

	targetURL, err := url.Parse(target)
	if err != nil || !isAllowedPreviewTargetURL(targetURL) {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Preview target is invalid"))
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL) // #nosec G704 -- target is restricted to loopback preview URLs.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = joinURLPath(targetURL.Path, upstreamPath)
		req.URL.RawPath = ""
		req.Host = targetURL.Host
		req.Header.Set("X-Forwarded-Prefix", s.githubPreviewPathPrefix()+"/"+slug)
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		previewBase := s.githubPreviewPathPrefix() + "/" + slug
		rewritePreviewLocation(resp, previewBase)
		rewritePreviewCookiePath(resp, previewBase)
		if err := injectPreviewBaseScript(resp, previewBase); err != nil {
			return err
		}
		return nil
	}
	proxy.ServeHTTP(w, r) // #nosec G704 -- proxy target is loopback-validated above.
}

func parsePreviewProxyPath(path, prefix string) (slug, upstreamPath string, ok bool) {
	prefix = normalizePreviewPathPrefix(prefix)
	path = "/" + strings.TrimLeft(path, "/")
	if path != prefix && !strings.HasPrefix(path, prefix+"/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimLeft(rest, "/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.SplitN(rest, "/", 2)
	slug = strings.TrimSpace(parts[0])
	if slug == "" {
		return "", "", false
	}
	upstreamPath = "/"
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		upstreamPath = "/" + parts[1]
	}
	return slug, upstreamPath, true
}

func joinURLPath(basePath, childPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	childPath = "/" + strings.TrimLeft(childPath, "/")
	if basePath == "" {
		return childPath
	}
	return basePath + childPath
}

func isAllowedPreviewTargetURL(targetURL *url.URL) bool {
	if targetURL == nil || targetURL.Host == "" {
		return false
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return false
	}
	host := targetURL.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func rewritePreviewLocation(resp *http.Response, previewBase string) {
	location := resp.Header.Get("Location")
	if location == "" || !strings.HasPrefix(location, "/") || strings.HasPrefix(location, previewBase+"/") {
		return
	}
	resp.Header.Set("Location", strings.TrimRight(previewBase, "/")+location)
}

func rewritePreviewCookiePath(resp *http.Response, previewBase string) {
	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) == 0 {
		return
	}
	resp.Header.Del("Set-Cookie")
	for _, cookie := range cookies {
		if strings.Contains(strings.ToLower(cookie), "path=/") {
			cookie = strings.Replace(cookie, "Path=/", "Path="+previewBase+"/", 1)
			cookie = strings.Replace(cookie, "path=/", "Path="+previewBase+"/", 1)
		}
		resp.Header.Add("Set-Cookie", cookie)
	}
}

func injectPreviewBaseScript(resp *http.Response, previewBase string) error {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/html") {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	script := []byte(`<script>(function(){const basePath=` + strconv.Quote(previewBase) + `;window.K13D_BASE_PATH=basePath;window.k13dPath=function(path){if(!basePath||typeof path!=='string'||!path.startsWith('/'))return path;if(path.startsWith(basePath+'/'))return path;return basePath+path;};if(!window.fetch)return;const nativeFetch=window.fetch.bind(window);window.fetch=function(input,init){if(typeof input==='string'){input=window.k13dPath(input);}else if(input instanceof URL&&input.origin===window.location.origin){input=new URL(window.k13dPath(input.pathname)+input.search+input.hash,input.origin);}else if(input instanceof Request){const reqURL=new URL(input.url,window.location.origin);if(reqURL.origin===window.location.origin&&reqURL.pathname.startsWith('/api/')){const rewritten=new URL(window.k13dPath(reqURL.pathname)+reqURL.search+reqURL.hash,reqURL.origin);input=new Request(rewritten,input);}}return nativeFetch(input,init);};})();</script>`)
	if bytes.Contains(body, []byte("</head>")) {
		body = bytes.Replace(body, []byte("</head>"), append(script, []byte("</head>")...), 1)
	} else {
		body = append(script, body...)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}
