package proxy

import (
	"time"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	log "github.com/sirupsen/logrus"
)


type Proxy struct {
	RequestBuffer map[string]*BufferedReuqests
	VisitStats *map[string]time.Time
	bufferLock *sync.Mutex
	service string
	bufferCheckSleep time.Duration
	tenant clients.Tenant
	wit clients.WIT
	idler clients.Idler
}

type BufferedReuqests struct {
	LastRequest time.Time
	Requests []BufferedReuqest
}

type BufferedReuqest struct {
	Request *http.Request
	Body []byte
}

func NewProxy(t clients.Tenant, w clients.WIT, i clients.Idler) Proxy {
	rb := make(map[string]*BufferedReuqests)
	vs := make(map[string]time.Time)
	p := Proxy{
		RequestBuffer: rb,
		VisitStats: &vs,
		bufferLock: &sync.Mutex{},
		tenant: t,
		wit: w,
		idler: i,
		bufferCheckSleep: 5,
	}
	go func() {
		p.ProcessBuffer()
	}()
	return p
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	isGH := false
	if ua, exist := r.Header["User-Agent"]; exist {
		isGH = strings.HasPrefix(ua[0], "GitHub-Hookshot")
	}

	if isGH {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
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
		ns, err := p.GetUser(gh)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(""))
			return
		}
		route, err := p.idler.GetRoute(ns)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(""))
			return
		}
		r.Host = fmt.Sprintf(route)

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
			log.Info("Webhook request buffered")
			w.Write([]byte(""))
			return
		} else {
			r.Body = ioutil.NopCloser(bytes.NewReader(body))
		}
	} else {
	/*
		Here will be a proxy
	
		vs := p.VisitStats 
		(*vs)[ns]=time.Now().UTC()
		r.Host = fmt.Sprintf(p.newUrl, "vpavlin")
		//Switch or add OSO token
		r.Header["Authorization"] = []string{fmt.Sprintf("Bearer %s", p.GetUserToken(""))}
		*/
	}

	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req = r
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
	log.Info(pl.Repository.CloneURL)
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
					req, err := http.NewRequest("", "", nil)
					if err != nil {
						log.Error("Request error ", err)
						continue
					}
					req = p.prepareRequest(req, rb.Request, rb.Body)
					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						log.Error("Error: ", err)
						break
					}

					if resp.StatusCode != 200 {
						log.Error(fmt.Sprintf("Got status %s after retrying request", resp.Status))
						break
					}

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

func (p *Proxy) prepareRequest(dst *http.Request, src *http.Request, body []byte) *http.Request {
	dst.URL = src.URL
	dst.URL.Host = src.Host
	dst.URL.Scheme = "https" //FIXME
	dst.Host = src.Host
	dst.Method = src.Method

	for k, v := range src.Header {
		dst.Header[k] = v
	}
	dst.Header["Server"] = []string{"Webhook-Proxy"}

	if len(body) == 0 {
		dst.Body = src.Body
	} else {
		dst.Body = ioutil.NopCloser(bytes.NewReader(body))
	}
	
	return dst
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