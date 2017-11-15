package proxy

import (
	"crypto/rsa"
	"time"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/patrickmn/go-cache"
	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	log "github.com/sirupsen/logrus"
)


type Proxy struct {
	RequestBuffer map[string]*BufferedReuqests
	VisitStats *map[string]time.Time
	TenantCache *cache.Cache
	ProxyCache *cache.Cache
	bufferLock *sync.Mutex
	service string
	bufferCheckSleep time.Duration
	tenant clients.Tenant
	wit clients.WIT
	idler clients.Idler
	redirect string
	publicKey *rsa.PublicKey
	authURL string
}

type BufferedReuqests struct {
	LastRequest time.Time
	Requests []BufferedReuqest
	Targets []string
}

type BufferedReuqest struct {
	Request *http.Request
	Body []byte
}

type TokenJSON struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType string `json:"token_type"`
	ExpiresIn int `json:"expires_in"`
	RefreshExpiresIn int `json:"refresh_expires_in"`
}

func NewProxy(t clients.Tenant, w clients.WIT, i clients.Idler, keycloakURL string, authURL string, redirect string) (Proxy, error) {
	rb := make(map[string]*BufferedReuqests)
	vs := make(map[string]time.Time)
	p := Proxy{
		RequestBuffer: rb,
		VisitStats: &vs,
		TenantCache: cache.New(30*time.Minute, 40*time.Minute),
		ProxyCache: cache.New(15*time.Minute, 10*time.Minute),
		bufferLock: &sync.Mutex{},
		tenant: t,
		wit: w,
		idler: i,
		bufferCheckSleep: 5,
		redirect: redirect,
		authURL: authURL,
	}
	
	pk, err := GetPublicKey(keycloakURL)
	if err != nil {
		return p, err
	}

	p.publicKey = pk

	go func() {
		p.ProcessBuffer()
	}()
	return p, nil
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	isGH := false
	if ua, exist := r.Header["User-Agent"]; exist {
		isGH = strings.HasPrefix(ua[0], "GitHub-Hookshot")
	}

	var body []byte
	var err error
	var ns string
	if isGH {
		defer r.Body.Close()
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error("Could not load request body: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Could not load request body: %s", err)))
			return
		}
		gh := GHHookStruct{}
		err = json.Unmarshal(body, &gh)
		if err != nil {
			log.Error("Could not parse GH payload: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Could not parse GH payload: %s", err)))
			return
		}
		ns, err = p.GetUser(gh)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(""))
			return
		}
		scheme, route, err := p.idler.GetRoute(ns)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(""))
			return
		}
		r.URL.Scheme = scheme
		r.URL.Host = route
		r.Host = route

		isIdle, err := p.idler.IsIdle(ns)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(""))
			return
		}

		if isIdle {
			w.Header().Set("Server", "Webhook-Proxy")
			p.bufferLock.Lock()
			if _, exist := p.RequestBuffer[ns]; !exist {
				brs := &BufferedReuqests{
					LastRequest: time.Now().UTC(),
					Requests: make([]BufferedReuqest, 0, 50),
				}
				p.RequestBuffer[ns] = brs
			}
			brs := p.RequestBuffer[ns]
			(*brs).Requests = append((*brs).Requests, BufferedReuqest{Request: r, Body: body})
			(*brs).LastRequest = time.Now().UTC()
			p.bufferLock.Unlock()
			log.Info("Webhook request buffered for ", ns)
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(""))
			return
		}
	} else {
		redirect := true

		redirectURL, err := url.ParseRequestURI(fmt.Sprintf("%s%s", strings.TrimRight(p.redirect, "/"), r.URL.Path))
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		//fmt.Printf("Query: %s", r.URL)

		if _, ok := r.Header["Authorization"]; ok { //FIXME Do we need this?
			redirect = false
		}

		//FIXME Refactor error handling!
		if tj, ok := r.URL.Query()["token_json"]; ok {
			log.Info("Found token info in query")
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			redirect = false
			tokenJSON := &TokenJSON{}
			err = json.Unmarshal([]byte(tj[0]), tokenJSON)
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			log.Info("Extracted JWT token")

			uid, err := GetTokenUID(tokenJSON.AccessToken, p.publicKey)
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			log.Info("Extracted UID from JWT token")

			ti, err := p.tenant.GetTenantInfo(uid)
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			namespace, err := p.tenant.GetNamespaceByType(ti, "jenkins")
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			ns = namespace.Name

			log.Info(fmt.Sprintf("Extracted Tenant Info: %s", ns))

			scheme, route, err := p.idler.GetRoute(ns)
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			osoToken, err := GetOSOToken(fmt.Sprintf("%s/api/token?for=%s", strings.TrimRight(p.authURL, "/"), namespace.ClusterURL), tokenJSON.AccessToken)
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			log.Info("Loaded OSO token")

			nr, _ := http.NewRequest("GET", fmt.Sprintf("%s://%s/", scheme, route), nil)
			nr.Header.Set("Authorization", fmt.Sprintf("Bearer %s", osoToken))
			c := http.DefaultClient
			nresp, _ := c.Do(nr)
			if nresp.StatusCode == http.StatusOK {
				cached := false
				for _, cookie := range nresp.Cookies() {
					http.SetCookie(w, cookie)
					if strings.HasPrefix(cookie.Name, "JSESSIONID") { //FIXME magic const
						url := url.URL{}
						url.Scheme = scheme
						url.Host = route
						p.ProxyCache.SetDefault(cookie.Value, url)
						log.Info(fmt.Sprintf("Cached Jenkins route %s", route))
						cached = true
					}
				}
				if cached {
					http.Redirect(w, r, redirectURL.String(), http.StatusFound)
				} else {
					err = fmt.Errorf("Could not find cookie JSESSIONID for %s", namespace.Name)
					log.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
				}
				return
			} //FIXME what about else?

		}
		if len(r.Cookies()) > 0 {
			session := ""
			for _, cookie := range r.Cookies() {
				if strings.HasPrefix(cookie.Name, "JSESSIONID") { //FIXME magic const
					session = cookie.Value
					break
				}
			}
			if cacheVal, ok := p.ProxyCache.Get(session); ok {
					url := cacheVal.(url.URL)
					r.Host = url.Host
					r.URL.Host = url.Host
					r.URL.Scheme = url.Scheme
					redirect = false
			}
		}

		if redirect {
			redir := fmt.Sprintf("%s/api/login?redirect=%s", strings.TrimRight(p.authURL, "/"), redirectURL.String())
			log.Info(fmt.Sprintf("Redirecting to %s", redir))
			http.Redirect(w, r, redir, 301)
		}
	}
	vs := p.VisitStats
	(*vs)[ns]=time.Now().UTC()

	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			if len(body) > 0 {
				req.Body = ioutil.NopCloser(bytes.NewReader(body))
			}
		},
	}).ServeHTTP(w, r)
}

type GHHookStruct struct {
	Repository struct {
		Name string `json:"name"`
		FullName string `json:"full_name"`
		GitURL string `json:"git_url"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

func (p *Proxy) GetUser(pl GHHookStruct) (res string, err error) {
	if n, found := p.TenantCache.Get(pl.Repository.CloneURL); found {
		log.Info("Cache hit")
		res = n.(string)
		return
	}
	log.Info("Cache miss")
	wi, err := p.wit.SearchCodebase(pl.Repository.CloneURL)
	if err != nil {
		return
	}

	ti, err := p.tenant.GetTenantInfo(wi.OwnedBy)
	if err != nil {
		return
	}

	n, err := p.tenant.GetNamespaceByType(ti, "jenkins")
	if err != nil {
		return
	}
	res = n.Name
	p.TenantCache.SetDefault(pl.Repository.CloneURL, res)
	return
}

func (p *Proxy) ProcessBuffer() {
	for {
		for namespace, rbs := range p.RequestBuffer {
			reqs := (*rbs).Requests 
			for i:=0;i<len(reqs);i=i {
				rb := reqs[i]
				log.Info("Retrying request for ", namespace)
				isIdle, err := p.idler.IsIdle(namespace)
				if err != nil {
					log.Error(err)
					break
				}
				if !isIdle {
					req, err := p.prepareRequest(rb.Request, rb.Body)
					if err != nil {
						log.Error("Error generating request: ", err)
						break
					}
					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						log.Error("Error: ", err)
						break
					}

					if resp.StatusCode != 200 && resp.StatusCode != 404 {
						log.Error(fmt.Sprintf("Got status %s after retrying request on %s", resp.Status, req.URL))
						break
					} else if resp.StatusCode == 404 {
						log.Warn(fmt.Sprintf("Got status %s after retrying request on %s, throwing away the request", resp.Status, req.URL))
					}

					log.Info(fmt.Sprintf("Request for %s to %s forwarded.", namespace, req.Host))

					p.bufferLock.Lock()
					reqs = append((*rbs).Requests[:i], (*rbs).Requests[i+1:]...)
					(*rbs).Requests = reqs
					p.bufferLock.Unlock()
				} else {
					//Do not try other requests for user if Jenkins is not running
					break
				}
			}
		}
		time.Sleep(p.bufferCheckSleep*time.Second)
	}
}

func (p *Proxy) prepareRequest(src *http.Request, body []byte) (dst *http.Request, err error) {
	b := ioutil.NopCloser(bytes.NewReader(body))
	dst, err = http.NewRequest(src.Method, src.URL.String(), b)
	for k, v := range src.Header {
		for _, vv := range v {
			dst.Header.Add(k, vv)
		}
	}
	dst.Header["Server"] = []string{"Webhook-Proxy"}
	return
}

func (p *Proxy) GetBufferInfo(namespace string) (int, string) {
	l := 0
	t := time.Time{}
	p.bufferLock.Lock()
	if rb, ok := p.RequestBuffer[namespace]; ok {
		l = len((*rb).Requests)
		t = (*rb).LastRequest
	}
	p.bufferLock.Unlock()

	return l, t.Format(time.RFC3339)
}

func (p *Proxy) GetLastVisitString(namespace string) string {
	vs := p.VisitStats
	return (*vs)[namespace].Format(time.RFC3339)
}