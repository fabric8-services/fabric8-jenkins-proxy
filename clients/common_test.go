package clients_test

import (
	"net/http"
	"net/http/httptest"
)

func MockServer(b []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, req *http.Request) {
		w.Write(b)
		w.Header().Set("Content-Type", "application/json")
	}))
}

func WITData1() []byte {
  return []byte(`{
		"data": [
				{
						"attributes": {
								"createdAt": "2017-10-16T09:09:06.400763Z",
								"last_used_workspace": "",
								"stackId": "vert.x",
								"type": "git",
								"url": "https://github.com/vpavlin/vpavlin-prod-prev-test.git"
						},
						"id": "ee978aa4-54af-4292-bd64-7f4f536e5181",
						"links": {
								"edit": "https://api.prod-preview.openshift.io/api/codebases/ee978aa4-54af-4292-bd64-7f4f536e5181/edit",
								"related": "https://api.prod-preview.openshift.io/api/codebases/ee978aa4-54af-4292-bd64-7f4f536e5181",
								"self": "https://api.prod-preview.openshift.io/api/codebases/ee978aa4-54af-4292-bd64-7f4f536e5181"
						},
						"relationships": {
								"space": {
										"data": {
												"id": "a7f45d87-c95a-4bbf-ad4b-7027de5ce270",
												"type": "spaces"
										},
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/a7f45d87-c95a-4bbf-ad4b-7027de5ce270",
												"self": "https://api.prod-preview.openshift.io/api/spaces/a7f45d87-c95a-4bbf-ad4b-7027de5ce270"
										}
								}
						},
						"type": "codebases"
				},
				{
						"attributes": {
								"createdAt": "2017-10-18T07:23:24.341083Z",
								"last_used_workspace": "",
								"stackId": "vert.x",
								"type": "git",
								"url": "https://github.com/vpavlin/vpavlin-prod-prev-test.git"
						},
						"id": "6d50505e-7cfc-443b-bd7d-c6003cdbc22c",
						"links": {
								"edit": "https://api.prod-preview.openshift.io/api/codebases/6d50505e-7cfc-443b-bd7d-c6003cdbc22c/edit",
								"related": "https://api.prod-preview.openshift.io/api/codebases/6d50505e-7cfc-443b-bd7d-c6003cdbc22c",
								"self": "https://api.prod-preview.openshift.io/api/codebases/6d50505e-7cfc-443b-bd7d-c6003cdbc22c"
						},
						"relationships": {
								"space": {
										"data": {
												"id": "4a3aec5e-2a07-41f5-97ad-bf1f657f908e",
												"type": "spaces"
										},
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e",
												"self": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e"
										}
								}
						},
						"type": "codebases"
				}
		],
		"included": [
				{
						"attributes": {
								"created-at": "2017-10-18T07:22:39.543885Z",
								"description": "",
								"name": "vpavlin-prod-prev-test",
								"updated-at": "2017-10-18T07:22:39.543885Z",
								"version": 0
						},
						"id": "4a3aec5e-2a07-41f5-97ad-bf1f657f908e",
						"links": {
								"backlog": {
										"self": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/backlog"
								},
								"filters": "https://api.prod-preview.openshift.io/api/filters",
								"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e",
								"self": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e",
								"workitemlinktypes": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/workitemlinktypes",
								"workitemtypegroups": "https://api.prod-preview.openshift.io/api/spacetemplates/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/workitemtypegroups/",
								"workitemtypes": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/workitemtypes"
						},
						"relationships": {
								"areas": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/areas"
										}
								},
								"backlog": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/backlog"
										}
								},
								"codebases": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/codebases"
										}
								},
								"collaborators": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/collaborators"
										}
								},
								"filters": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/filters"
										}
								},
								"iterations": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/iterations"
										}
								},
								"labels": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/labels"
										}
								},
								"owned-by": {
										"data": {
												"id": "2e15e957-0366-4802-bf1e-0d6fe3f11bb6",
												"type": "identities"
										},
										"links": {
												"related": "/api/users/2e15e957-0366-4802-bf1e-0d6fe3f11bb6"
										}
								},
								"workitemlinktypes": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/workitemlinktypes"
										}
								},
								"workitems": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/workitems"
										}
								},
								"workitemtypegroups": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spacetemplates/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/workitemtypegroups/"
										}
								},
								"workitemtypes": {
										"links": {
												"related": "https://api.prod-preview.openshift.io/api/spaces/4a3aec5e-2a07-41f5-97ad-bf1f657f908e/workitemtypes"
										}
								}
						},
						"type": "spaces"
				}
		],
		"links": {
				"first": "https://api.prod-preview.openshift.io/api/search/codebases?page[offset]=0&page[limit]=20&url=https://github.com/vpavlin/vpavlin-prod-prev-test.git",
				"last": "https://api.prod-preview.openshift.io/api/search/codebases?page[offset]=0&page[limit]=20&url=https://github.com/vpavlin/vpavlin-prod-prev-test.git"
		},
		"meta": {
				"totalCount": 2
		}
	}`)
}

func IdlerData1() []byte {
	return []byte(`{
		"service": "jenkins",
		"route": "jenkins-vpavlin-jenkins.d800.free-int.openshiftapps.com"
		}`)
}

func TenantData1() []byte {
	return []byte(`{  
		"data":{  
			"attributes":{  
					"created-at":"2017-10-11T18:47:27.69333Z",
					"email":"vpavlin@redhat.com",
					"namespaces":[  
						{  
								"cluster-url":"https://api.free-int.openshift.com",
								"created-at":"2017-10-11T18:47:28.491233Z",
								"name":"vpavlin-jenkins",
								"state":"created",
								"type":"jenkins",
								"updated-at":"2017-10-11T18:47:28.491233Z",
								"version":"2.0.6"
						},
						{  
								"cluster-url":"https://api.free-int.openshift.com",
								"created-at":"2017-10-11T18:47:28.513893Z",
								"name":"vpavlin-che",
								"state":"created",
								"type":"che",
								"updated-at":"2017-10-11T18:47:28.513893Z",
								"version":"2.0.6"
						},
						{  
								"cluster-url":"https://api.free-int.openshift.com",
								"created-at":"2017-10-11T18:47:28.56173Z",
								"name":"vpavlin-run",
								"state":"created",
								"type":"run",
								"updated-at":"2017-10-11T18:47:28.56173Z",
								"version":"2.0.6"
						},
						{  
								"cluster-url":"https://api.free-int.openshift.com",
								"created-at":"2017-10-11T18:47:28.604475Z",
								"name":"vpavlin",
								"state":"created",
								"type":"user",
								"updated-at":"2017-10-11T18:47:28.604475Z",
								"version":"1.0.91"
						},
						{  
								"cluster-url":"https://api.free-int.openshift.com",
								"created-at":"2017-10-11T18:47:28.763171Z",
								"name":"vpavlin-stage",
								"state":"created",
								"type":"stage",
								"updated-at":"2017-10-11T18:47:28.763171Z",
								"version":"2.0.6"
						}
					]
			},
			"id":"2e15e957-0366-4802-bf1e-0d6fe3f11bb6",
			"type":"tenants"
		}
}`)
}