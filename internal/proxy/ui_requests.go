package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/proxy/cookies"
	log "github.com/sirupsen/logrus"
)

func (p *Proxy) handleJenkinsUIRequest(w http.ResponseWriter, r *http.Request, logger *log.Entry) (cacheKey, ns string, okToForward bool) {
	logger.Infof("Incoming request: %s Cookies: %v ", r.URL.Path, cookiesutil.CookieNames(r.Cookies()))

	needsAuth := true   // indicates if we need to redirect to auth service
	okToForward = false // indicates if its ready to forward requests to Jenkins

	// redirectURL is used for auth service as a target of successful auth redirect
	redirectURL, err := url.ParseRequestURI(strings.TrimRight(p.redirect, "/") + r.URL.Path)
	if err != nil {
		logger.Errorf("Failed to create redirection url from %q and %q - error %s", p.redirect, r.URL.Path, err)
		p.HandleError(w, err, logger)
		return
	}

	if tj, ok := r.URL.Query()["token_json"]; ok {
		// If there is token_json in query, process it, find user info and login to Jenkins
		tjLogger := logger.WithField("part", "token_json")

		if len(tj) < 1 {
			p.HandleError(w, fmt.Errorf("could not read JWT token from URL"), tjLogger)
			return
		}

		jenkins, osioToken, err := GetJenkins(p.clusters, p.idler, p.tenant, tj[0], logger)
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error processing token_json to get osio-token: %q", err), tjLogger)
			return
		}

		ns = jenkins.info.NS
		clusterURL := jenkins.info.ClusterURL

		nsLogger := tjLogger.WithFields(log.Fields{"ns": ns, "cluster": clusterURL})
		nsLogger.Infof("found ns : %q, cluster: %q", ns, clusterURL)

		authClient, err := auth.DefaultClient()
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error while getting default auth client: %q", err), tjLogger)
			return
		}
		osoToken, err := authClient.OSOTokenForCluster(jenkins.info.ClusterURL, osioToken)
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error when fetching OSO token: %s", err), nsLogger)
			return
		}
		nsLogger.Info("Fetched OSO token from OSIO token")

		// we don't care about code here since only the state of jenkins pod -
		// running or not is what is relevant
		state, _, err := jenkins.Start()
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error when starting Jenkins: %s", err), nsLogger)
			return
		}

		if state != clients.Running {
			// Break the process if Jenkins isn't running.

			nsLogger.Infof("setting idled cookie: %v ", jenkins.info)

			// jenkins is idled and there could be old jsession, so delete them as
			// it will be invalid at this point
			cookiesutil.ExpireCookiesMatching(w, r, cookiesutil.IsSessionCookie)

			// Set "idled" cookie to indicate that jenkins is idled
			// also cache the ns & cluster for faster lookup next time
			uuid := cookiesutil.SetIdledCookie(w)
			p.ProxyCache.SetDefault(uuid, jenkins.info)

			// Redirect to set the idled cookied and to  get rid of token in URL
			nsLogger.Info("Redirecting to remove token from URL")
			http.Redirect(w, r, redirectURL.String(), http.StatusFound)
			return
		}

		// Jenkins is running at this point; login and set the jenkins cookies
		status, jenkinsCookies, err := jenkins.Login(osoToken)
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error when logging into jenkins: %s", err), nsLogger)
			return
		}

		nsLogger.Infof("Jenkins Login returned: %v", cookiesutil.CookieNames(jenkinsCookies))

		if status != http.StatusOK {
			nsLogger.Errorf("Jenkins login returned status %d", status)
			http.Redirect(w, r, redirectURL.String(), http.StatusFound)
			return
		}

		// there could be old session cookies, so lets clear it
		cookiesutil.ExpireCookiesMatching(w, r, cookiesutil.IsSessionOrIdledCookie)

		// set all cookies that we got from jenkins
		jsessionCookie := cookiesutil.SetJenkinsCookies(w, jenkinsCookies)
		if jsessionCookie == nil {
			// for some reason, login didn't return a session cookie
			p.HandleError(w, fmt.Errorf("could not find cookie %q for %q", cookiesutil.SessionCookie, ns), nsLogger)
			return
		}

		// Update proxy-cache to associate pci with the session cookie
		// the cache so that, the subsequent request that would contain the
		// the jession cookie can be used to lookup the cache
		p.ProxyCache.SetDefault(jsessionCookie.Value, jenkins.info)
		nsLogger.Infof("Cached Jenkins route %q in %q", jenkins.info.Route, jsessionCookie.Value)

		// If all good, redirect to self to remove token from url
		nsLogger.Infof("Redirecting to %q", redirectURL)
		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
		return
	}

	if len(r.Cookies()) > 0 {
		// Check cookies and proxy cache to find user info
		cookieLogger := logger.WithField("part", "cookie")

		for _, cookie := range r.Cookies() {

			if !cookiesutil.IsSessionOrIdledCookie(cookie) {
				continue // only the session and idled cookies are cached
			}

			cacheVal, ok := p.ProxyCache.Get(cookie.Value)
			if !ok {
				// if the cookie is not in cache, it could be an old idled or jsessionid
				// cookie so lets clear it
				cookieLogger.Infof("cookies: %q is session or idle is OLD; expiring", cookie.Name)
				cookiesutil.ExpireCookie(w, cookie)
				continue
			}

			cacheKey = cookie.Value
			pci := cacheVal.(CacheItem)
			ns = pci.NS
			clusterURL := pci.ClusterURL
			jenkins, _, err := GetJenkins(nil, p.idler, p.tenant, "", cookieLogger)
			if err != nil {
				p.HandleError(w, err, cookieLogger)
				return
			}
			jenkins.info = pci

			nsLogger := log.WithFields(log.Fields{"ns": ns, "cluster": clusterURL, "cookie": cookie.Name})
			nsLogger.Infof("cookie: %q is in cache", cookie.Name)

			if cookiesutil.IsSessionCookie(cookie) {
				// We found a session cookie in cache
				scLogger := nsLogger.WithField("cookietype", "session")
				scLogger.Infof("Cache has Jenkins route %q in %q", pci.Route, cookie.Value)

				// ensure jenkins is running
				state, _, err := jenkins.Start()
				if err != nil {
					p.HandleError(w, err, scLogger)
					return
				}

				if state == clients.Running {
					scLogger.Infof("jenkins is running | so rev proxy: %+v", pci)

					needsAuth = false  // user is logged in; do not redirect
					okToForward = true // good to be reverse-proxied
					r.Host = pci.Route
					r.URL.Host = pci.Route
					r.URL.Scheme = pci.Scheme
					return
				}

				// we find a session cookie in cache but the pod is not running
				// so lets clear the cookie and the cache entry
				p.ProxyCache.Delete(cacheKey)
				cacheKey = "" // cacheKey isn't valid any more
				cookiesutil.ExpireCookiesMatching(w, r, cookiesutil.IsSessionOrIdledCookie)
				p.recordStatistics(pci.NS, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?

				break // and do a reauth
			}

			if cookiesutil.IsIdledCookie(cookie) {
				// Found a cookie saying Jenkins is idled
				icLogger := nsLogger.WithField("cookietype", "idle")
				icLogger.Infof("Cache has Jenkins route %q in %q", pci.Route, cookie.Value)

				needsAuth = false
				state, code, err := jenkins.Start()
				if err != nil {
					p.HandleError(w, err, icLogger)
					return
				}

				// error if unexpected code is returned
				if code != http.StatusServiceUnavailable && code != http.StatusAccepted {
					err = fmt.Errorf("Failed to send unidle request using fabric8-jenkins-idler: code: %d", code)
					p.HandleError(w, err, icLogger)
					return
				}

				if state != clients.Running {
					w.WriteHeader(code)
					err = p.processTemplate(w, ns, icLogger)
					if err != nil {
						p.HandleError(w, err, icLogger)
					}
					p.recordStatistics(ns, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?
					return
				}

				// Jenkins seems to be running but OpenShift can take up to couple of
				// tens of second to update HAProxy configuration for new route
				// so even if the pod is up, route might still return 500 - i.e.
				// we need to check the route before claiming Jenkins is up

				icLogger.Infof("check if login works: %v ", pci)

				// login is performed using "" token to only verify if jenkins is actually running
				var statusCode int
				statusCode, _, err = jenkins.Login("")

				if err != nil {
					p.HandleError(w, err, icLogger)
					return
				}

				if statusCode == http.StatusOK ||
					statusCode == http.StatusForbidden {
					icLogger.Infof("jenkins is running fine and returned %d", statusCode)

					// jenkins is up and running; so expire both session and idled cookies
					// so that the next request will end up in re-auth and thus try
					// acutal login to Jenkins with the token_json which will then setup
					// the jsession cookies

					cookiesutil.ExpireCookiesMatching(w, r, cookiesutil.IsSessionOrIdledCookie)
					p.ProxyCache.Delete(cacheKey)
					cacheKey = ""

				} else {
					icLogger.Infof("Jenkins isn't running yet so process template")
					w.WriteHeader(http.StatusAccepted)
					err = p.processTemplate(w, ns, icLogger)
					if err != nil {
						p.HandleError(w, err, icLogger)
					}
				}
			}
		}

		// If we do not have user's info cached, run through login process to get it
		if len(cacheKey) == 0 {
			logger.WithField("ns", ns).Info("Could not find cache, redirecting to re-login")
		} else {
			logger.WithField("ns", ns).Infof("Found cookie %s", cacheKey)
		}
	}

	//Check if we need to redirect to auth service
	if needsAuth {

		authClient, err := auth.DefaultClient()
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error while getting default auth client: %q", err), logger)
			return
		}

		redirAuth := authClient.CreateRedirectURL(redirectURL.String())
		logger.Infof("Redirecting to auth: %q", redirAuth)

		// clear session and idle cookies as this is a fresh start
		cookiesutil.ExpireCookiesMatching(w, r, cookiesutil.IsSessionOrIdledCookie)
		w.Header().Set("Cache-Control", "no-cache")
		http.Redirect(w, r, redirAuth, http.StatusTemporaryRedirect)
	}
	return
}
