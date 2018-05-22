package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-proxy/internal/clients"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// CookieJenkinsIdled stores name of the cookie through which we can check if service is idled
	CookieJenkinsIdled = "JenkinsIdled"
	// ServiceName is name of service that we are trying to idle or unidle
	ServiceName = "jenkins"
	// SessionCookie stores name of the session cookie of the service in question
	SessionCookie = "JSESSIONID"
)

func (p *Proxy) handleJenkinsUIRequest(w http.ResponseWriter, r *http.Request, requestLogEntry *log.Entry) (cacheKey string, ns string, noProxy bool) {
	//redirect determines if we need to redirect to auth service
	needsAuth := true
	noProxy = true

	//redirectURL is used for auth service as a target of successful auth redirect
	redirectURL, err := url.ParseRequestURI(fmt.Sprintf("%s%s", strings.TrimRight(p.redirect, "/"), r.URL.Path))
	if err != nil {
		p.HandleError(w, err, requestLogEntry)
		return
	}

	//If the user provides OSO token, we can directly proxy
	if _, ok := r.Header["Authorization"]; ok { //FIXME Do we need this?
		needsAuth = false
	}

	if tj, ok := r.URL.Query()["token_json"]; ok { //If there is token_json in query, process it, find user info and login to Jenkins
		if len(tj) < 1 {
			p.HandleError(w, errors.New("could not read JWT token from URL"), requestLogEntry)
			return
		}

		pci, osioToken, err := p.processToken([]byte(tj[0]), requestLogEntry)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
			return
		}
		ns = pci.NS
		clusterURL := pci.ClusterURL
		nsLogger := requestLogEntry.WithFields(log.Fields{"ns": ns, "cluster": clusterURL})
		nsLogger.Debug("Found token info in query")

		state, err := p.idler.State(ns, clusterURL)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
			return
		}

		// Break the process if the Jenkins is idled, set a cookie and redirect to self
		if state != clients.Running {
			if _, err := p.idler.UnIdle(ns, clusterURL); err != nil {
				p.HandleError(w, err, requestLogEntry)
				return
			}

			// jenkins is idled and there could be old cookies, so delete them as it
			// will be invalid at this point
			clearCookiesMatching(w, r, anyCookie)
			p.setIdledCookie(w, pci)

			requestLogEntry.WithField("ns", ns).Info("Redirecting to remove token from URL")
			http.Redirect(w, r, redirectURL.String(), http.StatusFound) //Redirect to get rid of token in URL
			return
		}

		osoToken, err := GetOSOToken(p.authURL, pci.ClusterURL, osioToken)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
			return
		}
		requestLogEntry.WithField("ns", ns).Debug("Loaded OSO token")

		statusCode, jenkinsCookies, err := p.loginJenkins(pci, osoToken, requestLogEntry)
		if err != nil {
			p.HandleError(w, err, requestLogEntry)
			return
		}
		if statusCode == http.StatusOK {
			clearCookiesMatching(w, r, anyCookie)
			for _, cookie := range jenkinsCookies {
				if isIdledCookie(cookie) {
					continue
				}
				http.SetCookie(w, cookie)
				//Find session cookie and use it's value as a key for cache
				if isSessionCookie(cookie) {
					p.ProxyCache.SetDefault(cookie.Value, pci)
					requestLogEntry.WithField("ns", ns).Infof("Cached Jenkins route %s in %s", pci.Route, cookie.Value)
					requestLogEntry.WithField("ns", ns).Infof("Redirecting to %s", redirectURL.String())
					//If all good, redirect to self to remove token from url
					http.Redirect(w, r, redirectURL.String(), http.StatusFound)
					return
				}

				//If we got here, the cookie was not found - report error
				p.HandleError(w, fmt.Errorf("could not find cookie %s for %s", SessionCookie, pci.NS), requestLogEntry)
			}
		} else {
			p.HandleError(w, fmt.Errorf("could not login to Jenkins in %s namespace", ns), requestLogEntry)
		}
	}

	if len(r.Cookies()) > 0 { //Check cookies and proxy cache to find user info
		for _, cookie := range r.Cookies() {
			cacheVal, ok := p.ProxyCache.Get(cookie.Value)
			if !ok {
				continue
			}

			if isSessionCookie(cookie) {
				//We found a session cookie in cache
				// If jenkins is not "running", return loading page and status 202
				pci := cacheVal.(CacheItem)
				ns = pci.NS
				cacheKey = cookie.Value

				requestLogEntry.WithField("ns", ns).
					Infof("Cached Jenkins route %s in %s", pci.Route, cookie.Value)

				state, err := p.idler.State(ns, pci.ClusterURL)
				if err != nil {
					p.HandleError(w, err, requestLogEntry)
					return
				}
				if state != clients.Running {
					// we find a session cookie but the pod isn't running
					// so lets unidle and clear the cookie and show loading page
					code, err := p.idler.UnIdle(ns, pci.ClusterURL)
					if err != nil {
						p.HandleError(w, err, requestLogEntry)
						return
					}

					if code == http.StatusOK {
						code = http.StatusAccepted
					} else if code != http.StatusServiceUnavailable {
						p.HandleError(w, fmt.Errorf("Failed to send unidle request using fabric8-jenkins-idler"), requestLogEntry)
					}

					w.WriteHeader(code)
					clearCookiesMatching(w, r, isSessionCookie)
					p.setIdledCookie(w, pci)

					err = p.processTemplate(w, ns, requestLogEntry)
					if err != nil {
						p.HandleError(w, err, requestLogEntry)
						return
					}
					p.recordStatistics(pci.NS, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?

				} else {
					r.Host = pci.Route //Configure proxy upstream
					r.URL.Host = pci.Route
					r.URL.Scheme = pci.Scheme
					needsAuth = false //user is probably logged in, do not redirect
					noProxy = false
				}
				break

			} else if isIdledCookie(cookie) {
				// Found a cookie saying Jenkins is idled, verify and act accordingly
				cacheKey = cookie.Value
				needsAuth = false
				pci := cacheVal.(CacheItem)
				ns = pci.NS
				clusterURL := pci.ClusterURL
				state, err := p.idler.State(ns, clusterURL)
				if err != nil {
					p.HandleError(w, err, requestLogEntry)
					return
				}
				// If jenkins is not "running", return loading page and status 202
				if state != clients.Running {
					code, err := p.idler.UnIdle(ns, clusterURL)
					if err != nil {
						p.HandleError(w, err, requestLogEntry)
						return
					}

					if code == http.StatusOK {
						code = http.StatusAccepted
					} else if code != http.StatusServiceUnavailable {
						p.HandleError(w, fmt.Errorf("Failed to send unidle request using fabric8-jenkins-idler"), requestLogEntry)
					}

					w.WriteHeader(code)
					err = p.processTemplate(w, ns, requestLogEntry)
					if err != nil {
						p.HandleError(w, err, requestLogEntry)
						return
					}
					p.recordStatistics(pci.NS, time.Now().Unix(), 0) //FIXME - maybe do this at the beginning?
				} else { //If Jenkins is running, remove the cookie
					//OpenShift can take up to couple tens of second to update HAProxy configuration for new route
					//so even if the pod is up, route might still return 500 - i.e. we need to check the route
					//before claiming Jenkins is up
					var statusCode int
					statusCode, _, err = p.loginJenkins(pci, "", requestLogEntry)
					if err != nil {
						p.HandleError(w, err, requestLogEntry)
						return
					}
					if statusCode == 200 || statusCode == 403 {
						cookie.Expires = time.Unix(0, 0)
						http.SetCookie(w, cookie)
					} else {
						w.WriteHeader(http.StatusAccepted)
						err = p.processTemplate(w, ns, requestLogEntry)
						if err != nil {
							p.HandleError(w, err, requestLogEntry)
							return
						}
					}
				}

				if err != nil {
					p.HandleError(w, err, requestLogEntry)
				}
				break
			}
		}
		if len(cacheKey) == 0 { //If we do not have user's info cached, run through login process to get it
			requestLogEntry.WithField("ns", ns).Info("Could not find cache, redirecting to re-login")
		} else {
			requestLogEntry.WithField("ns", ns).Infof("Found cookie %s", cacheKey)
		}
	}

	//Check if we need to redirect to auth service
	if needsAuth {
		redirAuth := GetAuthURI(p.authURL, redirectURL.String())
		requestLogEntry.Infof("Redirecting to auth: %s", redirAuth)

		w.Header().Set("Cache-Control", "no-cache")
		http.Redirect(w, r, redirAuth, http.StatusTemporaryRedirect)
	}
	return
}

func (p *Proxy) loginJenkins(pci CacheItem, osoToken string, requestLogEntry *log.Entry) (int, []*http.Cookie, error) {
	//Login to Jenkins with OSO token to get cookies
	jenkinsURL := fmt.Sprintf("%s://%s/securityRealm/commenceLogin?from=%%2F", pci.Scheme, pci.Route)
	req, _ := http.NewRequest("GET", jenkinsURL, nil)
	if len(osoToken) > 0 {
		requestLogEntry.WithField("ns", pci.NS).Infof("Jenkins login for %s", jenkinsURL)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", osoToken))
	} else {
		requestLogEntry.WithField("ns", pci.NS).Infof("Accessing Jenkins route %s", jenkinsURL)
	}
	c := http.DefaultClient
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Cookies(), err
}

func clearCookiesMatching(w http.ResponseWriter, r *http.Request, expire func(cookie *http.Cookie) bool) {
	for _, cookie := range r.Cookies() {
		if expire(cookie) {
			cookie.Expires = time.Unix(0, 0)
		}
		http.SetCookie(w, cookie)
	}
}

func isSessionCookie(c *http.Cookie) bool {
	return strings.HasPrefix(c.Name, SessionCookie)
}

func isIdledCookie(c *http.Cookie) bool {
	return c.Name == CookieJenkinsIdled
}

func anyCookie(_ *http.Cookie) bool {
	return true
}

func (p *Proxy) setIdledCookie(w http.ResponseWriter, pci CacheItem) {
	c := &http.Cookie{}
	u1 := uuid.NewV4().String()
	c.Name = CookieJenkinsIdled
	c.Value = u1
	p.ProxyCache.SetDefault(u1, pci)
	http.SetCookie(w, c)
	return
}

func (p *Proxy) processToken(tokenData []byte, requestLogEntry *log.Entry) (pci CacheItem, osioToken string, err error) {
	tokenJSON := &TokenJSON{}
	err = json.Unmarshal(tokenData, tokenJSON)
	if err != nil {
		return
	}

	uid, err := GetTokenUID(tokenJSON.AccessToken, p.publicKey)
	if err != nil {
		return
	}

	ti, err := p.tenant.GetTenantInfo(uid)
	if err != nil {
		return
	}
	osioToken = tokenJSON.AccessToken

	namespace, err := p.tenant.GetNamespaceByType(ti, ServiceName)
	if err != nil {
		return
	}

	requestLogEntry.WithField("ns", namespace.Name).Debug("Extracted information from token")
	route, scheme, err := p.constructRoute(namespace.ClusterURL, namespace.Name)
	if err != nil {
		return
	}

	//Prepare an item for proxyCache - Jenkins info and OSO token
	pci = NewCacheItem(namespace.Name, scheme, route, namespace.ClusterURL)

	return
}
