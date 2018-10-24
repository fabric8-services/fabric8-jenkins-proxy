package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/auth"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	log "github.com/sirupsen/logrus"
)

func (p *Proxy) handleJenkinsUIRequest(w http.ResponseWriter, r *http.Request, logger *log.Entry) (cacheKey, ns string, okToForward bool) {
	logger.Infof("Incoming request: %s Cookies: %v ", r.URL.Path, cookieNames(r.Cookies()))

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

		pci, osioToken, err := p.processToken([]byte(tj[0]), tjLogger)
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error processing token_json to get osio-token: %q", err), tjLogger)
			return
		}

		ns = pci.NS
		clusterURL := pci.ClusterURL

		nsLogger := tjLogger.WithFields(log.Fields{"ns": ns, "cluster": clusterURL})
		nsLogger.Infof("found ns : %q, cluster: %q", ns, clusterURL)

		osoToken, err := auth.DefaultClient().OSOTokenForCluster(pci.ClusterURL, osioToken)
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error when fetching OSO token: %s", err), nsLogger)
			return
		}
		nsLogger.Info("Fetched OSO token from OSIO token")

		// we don't care about code here since only the state of jenkins pod -
		// running or not is what is relevant
		state, _, err := p.startJenkins(ns, clusterURL)
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error when starting Jenkins: %s", err), nsLogger)
			return
		}

		if state != clients.Running {
			// Break the process if Jenkins isn't running.

			nsLogger.Infof("setting idled cookie: %v ", pci)

			// jenkins is idled and there could be old jsession, so delete them as
			// it will be invalid at this point
			expireCookiesMatching(w, r, isSessionCookie)

			// Set "idled" cookie to indicate that jenkins is idled
			// also cache the ns & cluster for faster lookup next time
			uuid := setIdledCookie(w)
			p.ProxyCache.SetDefault(uuid, pci)

			// Redirect to set the idled cookied and to  get rid of token in URL
			nsLogger.Info("Redirecting to remove token from URL")
			http.Redirect(w, r, redirectURL.String(), http.StatusFound)
			return
		}

		// Jenkins is running at this point; login and set the jenkins cookies
		status, jenkinsCookies, err := p.loginJenkins(pci, osoToken, nsLogger)
		if err != nil {
			p.HandleError(w, fmt.Errorf("Error when logging into jenkins: %s", err), nsLogger)
			return
		}

		nsLogger.Infof("Jenkins Login returned: %v", cookieNames(jenkinsCookies))

		if status != http.StatusOK {
			nsLogger.Errorf("Jenkins login returned status %d", status)
			http.Redirect(w, r, redirectURL.String(), http.StatusFound)
			return
		}

		// there could be old session cookies, so lets clear it
		expireCookiesMatching(w, r, isSessionOrIdledCookie)

		// set all cookies that we got from jenkins
		jsessionCookie := setJenkinsCookies(w, jenkinsCookies)
		if jsessionCookie == nil {
			// for some reason, login didn't return a session cookie
			p.HandleError(w, fmt.Errorf("could not find cookie %q for %q", SessionCookie, ns), nsLogger)
			return
		}

		// Update proxy-cache to associate pci with the session cookie
		// the cache so that, the subsequent request that would contain the
		// the jession cookie can be used to lookup the cache
		p.ProxyCache.SetDefault(jsessionCookie.Value, pci)
		nsLogger.Infof("Cached Jenkins route %q in %q", pci.Route, jsessionCookie.Value)

		// If all good, redirect to self to remove token from url
		nsLogger.Infof("Redirecting to %q", redirectURL)
		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
		return
	}

	if len(r.Cookies()) > 0 {
		// Check cookies and proxy cache to find user info
		cookieLogger := logger.WithField("part", "cookie")

		for _, cookie := range r.Cookies() {

			if !isSessionOrIdledCookie(cookie) {
				continue // only the session and idled cookies are cached
			}

			cacheVal, ok := p.ProxyCache.Get(cookie.Value)
			if !ok {
				// if the cookie is not in cache, it could be an old idled or jsessionid
				// cookie so lets clear it
				cookieLogger.Infof("cookies: %q is session or idle is OLD; expiring", cookie.Name)
				expireCookie(w, cookie)
				continue
			}

			cacheKey = cookie.Value
			pci := cacheVal.(CacheItem)
			ns = pci.NS
			clusterURL := pci.ClusterURL

			nsLogger := log.WithFields(log.Fields{"ns": ns, "cluster": clusterURL, "cookie": cookie.Name})
			nsLogger.Infof("cookie: %q is in cache", cookie.Name)

			if isSessionCookie(cookie) {
				// We found a session cookie in cache
				scLogger := nsLogger.WithField("cookietype", "session")
				scLogger.Infof("Cache has Jenkins route %q in %q", pci.Route, cookie.Value)

				// ensure jenkins is running
				state, _, err := p.startJenkins(ns, pci.ClusterURL)
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
				expireCookiesMatching(w, r, isSessionOrIdledCookie)
				p.recordStatistics(pci.NS, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?

				break // and do a reauth
			}

			if isIdledCookie(cookie) {
				// Found a cookie saying Jenkins is idled
				icLogger := nsLogger.WithField("cookietype", "idle")
				icLogger.Infof("Cache has Jenkins route %q in %q", pci.Route, cookie.Value)

				needsAuth = false
				state, code, err := p.startJenkins(ns, clusterURL)
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
				statusCode, _, err = p.loginJenkins(pci, "", icLogger)

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

					expireCookiesMatching(w, r, isSessionOrIdledCookie)
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
		redirAuth := auth.DefaultClient().CreateRedirectURL(redirectURL.String())
		logger.Infof("Redirecting to auth: %q", redirAuth)

		// clear session and idle cookies as this is a fresh start
		expireCookiesMatching(w, r, isSessionOrIdledCookie)
		w.Header().Set("Cache-Control", "no-cache")
		http.Redirect(w, r, redirAuth, http.StatusTemporaryRedirect)
	}
	return
}

func (p *Proxy) loginJenkins(pci CacheItem, osoToken string, logger *log.Entry) (int, []*http.Cookie, error) {
	//Login to Jenkins with OSO token to get cookies
	jenkinsURL := fmt.Sprintf("%s://%s/securityRealm/commenceLogin?from=%%2F", pci.Scheme, pci.Route)

	req, _ := http.NewRequest("GET", jenkinsURL, nil)
	if len(osoToken) > 0 {
		logger.WithField("ns", pci.NS).Infof("Jenkins login for %s", jenkinsURL)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", osoToken))
	} else {
		logger.WithField("ns", pci.NS).Infof("Accessing Jenkins route %s", jenkinsURL)
	}
	c := http.DefaultClient
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Cookies(), err
}

func (p *Proxy) processToken(tokenData []byte, logger *log.Entry) (pci CacheItem, osioToken string, err error) {
	tokenJSON := &auth.TokenJSON{}
	err = json.Unmarshal(tokenData, tokenJSON)
	if err != nil {
		return
	}

	uid, err := auth.DefaultClient().UIDFromToken(tokenJSON.AccessToken)
	if err != nil {
		return
	}

	ti, err := p.tenant.GetTenantInfo(uid)
	if err != nil {
		return
	}
	osioToken = tokenJSON.AccessToken

	namespace, err := clients.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return
	}

	logger.WithField("ns", namespace.Name).Debug("Extracted information from token")
	route, scheme, err := p.constructRoute(namespace.ClusterURL, namespace.Name)
	if err != nil {
		return
	}

	//Prepare an item for proxyCache - Jenkins info and OSO token
	pci = NewCacheItem(namespace.Name, scheme, route, namespace.ClusterURL)

	return
}

// unidles Jenkins only if it is idled and returns the
// state of the pod, the http status of calling unidle, and error if any
func (p *Proxy) startJenkins(ns, clusterURL string) (state clients.PodState, code int, err error) {
	// Assume pods are starting and unidle only if it is in "idled" state
	code = http.StatusAccepted
	nsLogger := log.WithFields(log.Fields{"ns": ns, "cluster": clusterURL})

	state, err = p.idler.State(ns, clusterURL)
	if err != nil {
		return
	}
	nsLogger.Infof("state : %q", state)

	if state == clients.Idled {
		// Unidle only if needed
		nsLogger.Infof("Unidling jenkins")
		if code, err = p.idler.UnIdle(ns, clusterURL); err != nil {
			return
		}
	}
	if code == http.StatusOK {
		// XHR relies on 202 to retry and 200 to stop retrying and reload
		// since we just started jenkins pods, change the code to 202 so
		// that it retries
		// SEE: static/html/index.html
		code = http.StatusAccepted
	}
	return state, code, nil
}
