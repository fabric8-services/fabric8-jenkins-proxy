package testutils

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
		"route": "jenkins-vpavlin-jenkins.d800.free-int.openshiftapps.com",
		"tls": true,
		"is_idle": false
		}`)
}

func IdlerData2() []byte {
	return []byte(`{
		"service": "jenkins",
		"route": "localhost:8888",
		"tls": false,
		"is_idle": true
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

func TenantData2() []byte {
	return []byte(`{"errors":[{"code":"not_found","detail":"/","id":"2Q/BAc8b","status":"404","title":"Not Found"}]}`)
}

func GetGHData() []byte {
	return []byte(`{
		"ref": "refs/heads/master",
		"before": "2b2f45994f0b9643876c09f0c6169bbd3dff09fe",
		"after": "94ae982d70bf112fb553de1379b313936d07d18c",
		"created": false,
		"deleted": false,
		"forced": false,
		"base_ref": null,
		"compare": "https://github.com/vpavlin/vpavlin-prod-prev-test/compare/2b2f45994f0b...94ae982d70bf",
		"commits": [
			{
				"id": "94ae982d70bf112fb553de1379b313936d07d18c",
				"tree_id": "16220cee67bee2a62635caee0ca4e142f6964731",
				"distinct": true,
				"message": "Update README.adoc",
				"timestamp": "2017-10-19T16:04:44+02:00",
				"url": "https://github.com/vpavlin/vpavlin-prod-prev-test/commit/94ae982d70bf112fb553de1379b313936d07d18c",
				"author": {
					"name": "Vaclav Pavlin",
					"email": "vaclav.pavlin@gmail.com",
					"username": "vpavlin"
				},
				"committer": {
					"name": "GitHub",
					"email": "noreply@github.com",
					"username": "web-flow"
				},
				"added": [
	
				],
				"removed": [
	
				],
				"modified": [
					"README.adoc"
				]
			}
		],
		"head_commit": {
			"id": "94ae982d70bf112fb553de1379b313936d07d18c",
			"tree_id": "16220cee67bee2a62635caee0ca4e142f6964731",
			"distinct": true,
			"message": "Update README.adoc",
			"timestamp": "2017-10-19T16:04:44+02:00",
			"url": "https://github.com/vpavlin/vpavlin-prod-prev-test/commit/94ae982d70bf112fb553de1379b313936d07d18c",
			"author": {
				"name": "Vaclav Pavlin",
				"email": "vaclav.pavlin@gmail.com",
				"username": "vpavlin"
			},
			"committer": {
				"name": "GitHub",
				"email": "noreply@github.com",
				"username": "web-flow"
			},
			"added": [
	
			],
			"removed": [
	
			],
			"modified": [
				"README.adoc"
			]
		},
		"repository": {
			"id": 107372849,
			"name": "vpavlin-prod-prev-test",
			"full_name": "vpavlin/vpavlin-prod-prev-test",
			"owner": {
				"name": "vpavlin",
				"email": "vaclav.pavlin@gmail.com",
				"login": "vpavlin",
				"id": 4759808,
				"avatar_url": "https://avatars2.githubusercontent.com/u/4759808?v=4",
				"gravatar_id": "",
				"url": "https://api.github.com/users/vpavlin",
				"html_url": "https://github.com/vpavlin",
				"followers_url": "https://api.github.com/users/vpavlin/followers",
				"following_url": "https://api.github.com/users/vpavlin/following{/other_user}",
				"gists_url": "https://api.github.com/users/vpavlin/gists{/gist_id}",
				"starred_url": "https://api.github.com/users/vpavlin/starred{/owner}{/repo}",
				"subscriptions_url": "https://api.github.com/users/vpavlin/subscriptions",
				"organizations_url": "https://api.github.com/users/vpavlin/orgs",
				"repos_url": "https://api.github.com/users/vpavlin/repos",
				"events_url": "https://api.github.com/users/vpavlin/events{/privacy}",
				"received_events_url": "https://api.github.com/users/vpavlin/received_events",
				"type": "User",
				"site_admin": false
			},
			"private": false,
			"html_url": "https://github.com/vpavlin/vpavlin-prod-prev-test",
			"description": null,
			"fork": false,
			"url": "https://github.com/vpavlin/vpavlin-prod-prev-test",
			"forks_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/forks",
			"keys_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/keys{/key_id}",
			"collaborators_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/collaborators{/collaborator}",
			"teams_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/teams",
			"hooks_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/hooks",
			"issue_events_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/issues/events{/number}",
			"events_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/events",
			"assignees_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/assignees{/user}",
			"branches_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/branches{/branch}",
			"tags_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/tags",
			"blobs_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/git/blobs{/sha}",
			"git_tags_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/git/tags{/sha}",
			"git_refs_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/git/refs{/sha}",
			"trees_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/git/trees{/sha}",
			"statuses_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/statuses/{sha}",
			"languages_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/languages",
			"stargazers_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/stargazers",
			"contributors_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/contributors",
			"subscribers_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/subscribers",
			"subscription_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/subscription",
			"commits_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/commits{/sha}",
			"git_commits_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/git/commits{/sha}",
			"comments_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/comments{/number}",
			"issue_comment_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/issues/comments{/number}",
			"contents_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/contents/{+path}",
			"compare_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/compare/{base}...{head}",
			"merges_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/merges",
			"archive_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/{archive_format}{/ref}",
			"downloads_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/downloads",
			"issues_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/issues{/number}",
			"pulls_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/pulls{/number}",
			"milestones_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/milestones{/number}",
			"notifications_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/notifications{?since,all,participating}",
			"labels_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/labels{/name}",
			"releases_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/releases{/id}",
			"deployments_url": "https://api.github.com/repos/vpavlin/vpavlin-prod-prev-test/deployments",
			"created_at": 1508311383,
			"updated_at": "2017-10-18T07:23:07Z",
			"pushed_at": 1508421885,
			"git_url": "git://github.com/vpavlin/vpavlin-prod-prev-test.git",
			"ssh_url": "git@github.com:vpavlin/vpavlin-prod-prev-test.git",
			"clone_url": "https://github.com/vpavlin/vpavlin-prod-prev-test.git",
			"svn_url": "https://github.com/vpavlin/vpavlin-prod-prev-test",
			"homepage": "",
			"size": 37,
			"stargazers_count": 0,
			"watchers_count": 0,
			"language": "HTML",
			"has_issues": false,
			"has_projects": true,
			"has_downloads": false,
			"has_wiki": false,
			"has_pages": false,
			"forks_count": 0,
			"mirror_url": null,
			"open_issues_count": 0,
			"forks": 0,
			"open_issues": 0,
			"watchers": 0,
			"default_branch": "master",
			"stargazers": 0,
			"master_branch": "master"
		},
		"pusher": {
			"name": "vpavlin",
			"email": "vaclav.pavlin@gmail.com"
		},
		"sender": {
			"login": "vpavlin",
			"id": 4759808,
			"avatar_url": "https://avatars2.githubusercontent.com/u/4759808?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/vpavlin",
			"html_url": "https://github.com/vpavlin",
			"followers_url": "https://api.github.com/users/vpavlin/followers",
			"following_url": "https://api.github.com/users/vpavlin/following{/other_user}",
			"gists_url": "https://api.github.com/users/vpavlin/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/vpavlin/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/vpavlin/subscriptions",
			"organizations_url": "https://api.github.com/users/vpavlin/orgs",
			"repos_url": "https://api.github.com/users/vpavlin/repos",
			"events_url": "https://api.github.com/users/vpavlin/events{/privacy}",
			"received_events_url": "https://api.github.com/users/vpavlin/received_events",
			"type": "User",
			"site_admin": false
		}
	}`)
}