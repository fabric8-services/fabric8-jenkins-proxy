package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// WIT describes work item tracker service of OSIO.
type WIT interface {
	SearchCodebase(repo string) (*WITInfo, error)
}

type wit struct {
	witURL    string
	authToken string
	client    *http.Client
}

// NewWIT creates an instance of WIT client.
func NewWIT(url string, token string) WIT {
	return &wit{
		witURL:    url,
		authToken: token,
		client:    &http.Client{},
	}
}

// WITInfo holds information about owner of a git repository.
type WITInfo struct {
	OwnedBy string
}

// UnmarshalJSON parses byte slice representing quite complicated API response into a simple struct.
func (wi *WITInfo) UnmarshalJSON(b []byte) (err error) {
	type data struct {
		Relationships struct {
			Space struct {
				Data struct {
					ID string
				}
			}
		}
	}

	type relationships struct {
		OwnedBy struct {
			Data struct {
				ID string
			}
		} `json:"owned-by"`
	}

	type included struct {
		ID            string
		Type          string
		Relationships relationships
	}

	type Info struct {
		Included []included
		Data     []data
	}

	i := Info{}
	err = json.Unmarshal(b, &i)
	if err != nil {
		return err
	}

	for _, d := range i.Data {
		for _, i := range i.Included {
			if d.Relationships.Space.Data.ID == i.ID { // Find correct Relationship
				wi.OwnedBy = i.Relationships.OwnedBy.Data.ID
				return nil
			}
		}
	}

	return nil
}

// SearchCodebase finds and returns owner of a given repository based on URL.
func (w *wit) SearchCodebase(repo string) (*WITInfo, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/search/codebases", w.witURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.authToken))

	q := req.URL.Query()
	q.Add("url", repo)
	req.URL.RawQuery = q.Encode()

	log.Infof("WIT Client: %s", req.URL)
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}

	wi := &WITInfo{}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, wi)
	if err != nil {
		return nil, err
	}

	return wi, nil
}
