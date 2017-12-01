package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type WIT struct {
	witURL string
	authToken string
}

func NewWIT(url string, token string) WIT {
	return WIT{
		witURL: url,
		authToken: token,
	}
}

//WITinfo holds information about owner of a git repository
type WITInfo struct {
	OwnedBy string
}

//UnmarshalJSON parses byte slice representing quite complicated API response into a simple struct
func (wi *WITInfo) UnmarshalJSON(b []byte) (err error) {
	type data struct {
		Relationships struct {
			Space struct {
				Data struct {
					Id string
				}
			}
		}
	}

	type relationships struct {
		OwnedBy struct {
			Data struct {
				Id string
			}
		} `json:"owned-by"`
	}

	type included struct {
		Id string
		Type string
		Relationships relationships
	}


	type Info struct {
		Included []included
		Data []data
	}

	i := Info{}
	err = json.Unmarshal(b, &i)
	if err != nil {
		return err
	}

	for _, d := range i.Data {
		for _, i := range i.Included {
			if d.Relationships.Space.Data.Id == i.Id { //Find correct Relationship
				wi.OwnedBy = i.Relationships.OwnedBy.Data.Id
				return nil
			}
		}
	}

	return nil
}

//SearchCodebase finds and returns owner of a given repository based on URL
func (w WIT) SearchCodebase(repo string) (*WITInfo, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/search/codebases", w.witURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.authToken))

	q := req.URL.Query()
	q.Add("url", repo)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
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