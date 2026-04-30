package apitest

import (
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/internal/steptest"
)

func flowListNamespaces(t *testing.T, h *harness) {
	t.Helper()

	const (
		nsA = "46c14e5d-c048-4159-a26a-f37bc0110c85"
		nsB = "31d0952d-8d73-4119-8e95-b5f19d6f9df7"
		nsC = "c00420c4-1461-4824-a2cc-34e3d04524d4"
		nsD = "5565d865-a6dc-45e7-a086-28e49669e8a6"
		nsE = "aaecb6eb-f0c7-4cf4-976d-f8e7aefcf7ef"
	)
	Flow{
		NamespaceIDs: []string{nsA, nsB, nsC, nsD, nsE},
		Steps: []steptest.Step{
			{
				Name: "empty",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces",
				},
				Response: steptest.Response{
					Status: 200,
					Body:   steptest.Body{JSON: map[string]any{"total": 0, "offset": 0, "items": []any{}}},
				},
			},

			{
				Name: "create_a",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"a","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           nsA,
						"tag":          "a",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name: "create_b",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"b","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           nsB,
						"tag":          "b",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name: "create_c",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"c","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           nsC,
						"tag":          "c",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name: "create_d",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"d","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           nsD,
						"tag":          "d",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name: "create_e",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"e","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           nsE,
						"tag":          "e",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},

			{
				Name: "pagination_defaultLimit",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces",
				},
				Response: steptest.Response{
					Status: 200,
					Body: steptest.Body{JSON: map[string]any{
						"total":  5,
						"offset": 0,
						"items": []any{
							map[string]any{"id": nsB, "tag": "b", "pid_format": map[string]any{"pattern": "***-***", "characters": "full"}, "date_created": h.now},
							map[string]any{"id": nsA, "tag": "a", "pid_format": map[string]any{"pattern": "***-***", "characters": "full"}, "date_created": h.now},
						},
					}},
				},
			},
			{
				Name: "pagination_clampMaxLimit",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces?limit=999",
				},
				Response: steptest.Response{
					Status: 200,
					Body: steptest.Body{JSON: map[string]any{
						"total":  5,
						"offset": 0,
						"items": []any{
							map[string]any{"id": nsB, "tag": "b", "pid_format": map[string]any{"pattern": "***-***", "characters": "full"}, "date_created": h.now},
							map[string]any{"id": nsA, "tag": "a", "pid_format": map[string]any{"pattern": "***-***", "characters": "full"}, "date_created": h.now},
							map[string]any{"id": nsD, "tag": "d", "pid_format": map[string]any{"pattern": "***-***", "characters": "full"}, "date_created": h.now},
						},
					}},
				},
			},
			{
				Name: "pagination_offsetPastEnd",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces?offset=5",
				},
				Response: steptest.Response{
					Status: 200,
					Body:   steptest.Body{JSON: map[string]any{"total": 5, "offset": 5, "items": []any{}}},
				},
			},

			{
				Name:     "invalidQuery_limit_0",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces?limit=0"},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": `invalid query parameter "limit"`}}},
			},
			{
				Name:     "invalidQuery_limit_-1",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces?limit=-1"},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": `invalid query parameter "limit"`}}},
			},
			{
				Name:     "invalidQuery_limit_abc",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces?limit=abc"},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": `invalid query parameter "limit"`}}},
			},

			{
				Name:     "invalidQuery_offset_-1",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces?offset=-1"},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": `invalid query parameter "offset"`}}},
			},
			{
				Name:     "invalidQuery_offset_abc",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces?offset=abc"},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": `invalid query parameter "offset"`}}},
			},

			{
				Name: "filter_tag",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces?tag=c",
				},
				Response: steptest.Response{
					Status: 200,
					Body: steptest.Body{JSON: map[string]any{
						"total":  1,
						"offset": 0,
						"items": []any{
							map[string]any{"id": nsC, "tag": "c", "pid_format": map[string]any{"pattern": "***-***", "characters": "full"}, "date_created": h.now},
						},
					}},
				},
			},
		},
	}.Run(t, h)
}

func flowCreateNamespace(t *testing.T, h *harness) {
	t.Helper()

	const ns = "46c14e5d-c048-4159-a26a-f37bc0110c85"
	Flow{
		NamespaceIDs: []string{ns},
		Steps: []steptest.Step{
			{
				Name: "success",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"flow-tag","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           ns,
						"tag":          "flow-tag",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name:     "emptyBody",
				Request:  steptest.Request{Method: "POST", Path: "/resolver/namespaces"},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": "empty request body"}}},
			},
			{
				Name: "trailingJSON",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"x","pid_format":{"pattern":"***","characters":"full"}} {"tag":"y"}`,
				},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": "trailing JSON"}}},
			},
			{
				Name: "tooLargeBody",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"` + strings.Repeat("a", 512) + `","pid_format":{"pattern":"***","characters":"full"}}`,
				},
				Response: steptest.Response{Status: 413, Body: steptest.Body{JSON: map[string]any{"error": "request payload too large"}}},
			},
		},
	}.Run(t, h)
}

func flowListResources(t *testing.T, h *harness) {
	t.Helper()
	const ns = "46c14e5d-c048-4159-a26a-f37bc0110c85"
	Flow{
		NamespaceIDs: []string{ns},
		PIDs: []string{
			"xjc-cjy", // create_a
			"8zk-pwt", // create_b
			"6ez-s5t", // create_emptyTag
			"7yy-3dz", // create_c
			"nqs-vxz", // create_d (not asserted directly, but consumed)
		},
		Steps: []steptest.Step{
			{
				Name:     "namespaceMissing",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces/missing-ns/resources"},
				Response: steptest.Response{Status: 404, Body: steptest.Body{JSON: map[string]any{"error": "namespace not found"}}},
			},
			{
				Name: "create_namespace",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"list-res","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           ns,
						"tag":          "list-res",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name:     "empty",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces/" + ns + "/resources"},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{"total": 0, "offset": 0, "items": []any{}}}},
			},

			{
				Name: "create_a",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/a","metadata":"ext-1@sys-a","tag":"alpha"}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "create_b",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/b","metadata":"ext-2@sys-a","tag":"beta"}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "create_emptyTag",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/empty-tag","metadata":null,"tag":""}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "create_c",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/c","metadata":"ext-4@sys-a","tag":"alpha"}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "create_d",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/d","metadata":"ext-5@sys-a","tag":"alpha"}`,
				},
				Response: steptest.Response{Status: 201},
			},

			{
				Name:    "pagination_defaultLimit",
				Request: steptest.Request{Method: "GET", Path: "/resolver/namespaces/" + ns + "/resources"},
				Response: steptest.Response{
					Status: 200,
					Body: steptest.Body{JSON: map[string]any{
						"total":  5,
						"offset": 0,
						"items": []any{
							map[string]any{
								"pid":          "6ez-s5t",
								"url":          "https://example.com/empty-tag",
								"metadata":     nil,
								"date_created": h.now,
								"date_updated": h.now,
								"tag":          "",
								"deleted":      false,
							},
							map[string]any{
								"pid":          "7yy-3dz",
								"url":          "https://example.com/c",
								"metadata":     "ext-4@sys-a",
								"date_created": h.now,
								"date_updated": h.now,
								"tag":          "alpha",
								"deleted":      false,
							},
						},
					}},
				},
			},
			{
				Name: "pagination_clampMaxLimit",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces/" + ns + "/resources?limit=999",
				},
				Response: steptest.Response{Status: 200},
			},
			{
				Name: "pagination_offsetPastEnd",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces/" + ns + "/resources?offset=5",
				},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{"total": 5, "offset": 5, "items": []any{}}}},
			},
			{
				Name: "tagOmitted",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces/" + ns + "/resources",
				},
				Response: steptest.Response{Status: 200},
			},
			{
				Name: "tagFilter",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces/" + ns + "/resources?tag=alpha&limit=999",
				},
				Response: steptest.Response{Status: 200},
			},
			{
				Name: "tagNoMatch",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces/" + ns + "/resources?tag=other",
				},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{"total": 0, "offset": 0, "items": []any{}}}},
			},
			{
				Name: "tagEmptyMeansEmptyString",
				Request: steptest.Request{
					Method: "GET",
					Path:   "/resolver/namespaces/" + ns + "/resources?tag=&limit=999",
				},
				Response: steptest.Response{Status: 200},
			},
		},
	}.Run(t, h)
}

func flowCreateResource(t *testing.T, h *harness) {
	t.Helper()
	const ns = "46c14e5d-c048-4159-a26a-f37bc0110c85"
	Flow{
		NamespaceIDs: []string{
			ns,                                     // create-res
			"31d0952d-8d73-4119-8e95-b5f19d6f9df7", // hex-res
			"c00420c4-1461-4824-a2cc-34e3d04524d4", // readable6-res
			"5565d865-a6dc-45e7-a086-28e49669e8a6", // readable9-res
			"aaecb6eb-f0c7-4cf4-976d-f8e7aefcf7ef", // random64-res
		},
		PIDs: []string{
			"xjc-cjy", // success
			"8zk-pwt", // metadataNull_roundTrips
		},
		Steps: []steptest.Step{
			{
				Name: "create_namespace",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"create-res","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           ns,
						"tag":          "create-res",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name: "success",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/a","metadata":"ext-1@sys-a","tag":"alpha"}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"pid":          "xjc-cjy",
						"url":          "https://example.com/a",
						"metadata":     "ext-1@sys-a",
						"date_created": h.now,
						"date_updated": h.now,
						"tag":          "alpha",
						"deleted":      false,
					}},
				},
			},
			{
				Name: "metadataNull_roundTrips",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/metadata-null","metadata":null,"tag":"alpha"}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"pid":          "8zk-pwt",
						"url":          "https://example.com/metadata-null",
						"metadata":     nil,
						"date_created": h.now,
						"date_updated": h.now,
						"tag":          "alpha",
						"deleted":      false,
					}},
				},
			},
			{
				Name: "hexPattern_isLowercaseHex",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"hex-res","pid_format":{"pattern":"********-****-****-****-************","characters":"hex"}}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "readable6_isDeterministic",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"readable6-res","pid_format":{"pattern":"***-***","characters":"readable"}}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "readable9_isDeterministic",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"readable9-res","pid_format":{"pattern":"***-***-***","characters":"readable"}}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "random64_isDeterministic",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"random64-res","pid_format":{"pattern":"****************************************************************","characters":"full"}}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "namespaceMissing",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/absent-ns/resources",
					Body:   `{"url":"https://x","metadata":null,"tag":""}`,
				},
				Response: steptest.Response{Status: 404, Body: steptest.Body{JSON: map[string]any{"error": "namespace not found"}}},
			},
			{
				Name: "invalidNamespace",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/bad.ns/resources",
					Body:   `{"url":"https://x","metadata":null,"tag":""}`,
				},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": "invalid namespace"}}},
			},
			{
				Name: "tooLargeBody",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://` + strings.Repeat("a", 512) + `","metadata":"x","tag":"z"}`,
				},
				Response: steptest.Response{Status: 413, Body: steptest.Body{JSON: map[string]any{"error": "request payload too large"}}},
			},
		},
	}.Run(t, h)
}

func flowBatchCreateResources(t *testing.T, h *harness) {
	t.Helper()
	const ns = "46c14e5d-c048-4159-a26a-f37bc0110c85"
	Flow{
		NamespaceIDs: []string{ns},
		PIDs: []string{
			"xjc-cjy", // batch item 1
			"8zk-pwt", // batch item 2
		},
		Steps: []steptest.Step{
			{
				Name: "create_namespace",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"batch-res","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           ns,
						"tag":          "batch-res",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name: "success",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources:batch",
					Body:   `[{"url":"https://b1","metadata":"b1@batch","tag":"batch"},{"url":"https://b2","metadata":null,"tag":""}]`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: []any{
						map[string]any{
							"pid":          "xjc-cjy",
							"url":          "https://b1",
							"metadata":     "b1@batch",
							"date_created": h.now,
							"date_updated": h.now,
							"tag":          "batch",
							"deleted":      false,
						},
						map[string]any{
							"pid":          "8zk-pwt",
							"url":          "https://b2",
							"metadata":     nil,
							"date_created": h.now,
							"date_updated": h.now,
							"tag":          "",
							"deleted":      false,
						},
					}},
				},
			},
			{
				Name: "tooManyItems",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources:batch",
					Body:   `[{"url":"https://b1","metadata":"b1@batch","tag":""},{"url":"https://b2","metadata":"b2@batch","tag":""},{"url":"https://b3","metadata":"b3@batch","tag":""}]`,
				},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": "too many items: 3 > 2"}}},
			},
			{
				Name: "namespaceMissing",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/missing/resources:batch",
					Body:   `[{"url":"https://x","metadata":null,"tag":""}]`,
				},
				Response: steptest.Response{Status: 404, Body: steptest.Body{JSON: map[string]any{"error": "namespace not found"}}},
			},
			{
				Name: "tooLargeBody",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources:batch",
					Body:   `[{"url":"https://` + strings.Repeat("a", 512) + `","metadata":"x","tag":""}]`,
				},
				Response: steptest.Response{Status: 413, Body: steptest.Body{JSON: map[string]any{"error": "request payload too large"}}},
			},
		},
	}.Run(t, h)
}

func flowGetResource(t *testing.T, h *harness) {
	t.Helper()
	const ns = "46c14e5d-c048-4159-a26a-f37bc0110c85"
	const pid = "xjc-cjy"
	Flow{
		NamespaceIDs: []string{ns},
		PIDs:         []string{pid},
		Steps: []steptest.Step{
			{
				Name: "create_namespace",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"get-res","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           ns,
						"tag":          "get-res",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name: "create_resource",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/a","metadata":"ext-1@sys-a","tag":"alpha"}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name:    "success",
				Request: steptest.Request{Method: "GET", Path: "/resolver/namespaces/" + ns + "/resources/" + pid},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{
					"pid":          pid,
					"url":          "https://example.com/a",
					"metadata":     "ext-1@sys-a",
					"date_created": h.now,
					"date_updated": h.now,
					"tag":          "alpha",
					"deleted":      false,
				}}},
			},
			{
				Name:     "resourceNotFound",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces/" + ns + "/resources/99999"},
				Response: steptest.Response{Status: 404, Body: steptest.Body{JSON: map[string]any{"error": "resource not found"}}},
			},
			{
				Name:     "invalidNamespace",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces/bad.ns/resources/" + pid},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": "invalid namespace"}}},
			},
			{
				Name:     "invalidPID",
				Request:  steptest.Request{Method: "GET", Path: "/resolver/namespaces/" + ns + "/resources/bad.pid"},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": "invalid pid"}}},
			},
		},
	}.Run(t, h)
}

func flowUpdateResource(t *testing.T, h *harness) {
	t.Helper()
	const ns = "46c14e5d-c048-4159-a26a-f37bc0110c85"
	const pid = "xjc-cjy"
	Flow{
		NamespaceIDs: []string{ns},
		PIDs:         []string{pid},
		Steps: []steptest.Step{
			{
				Name: "create_namespace",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces",
					Body:   `{"tag":"update-res","pid_format":{"pattern":"***-***","characters":"full"}}`,
				},
				Response: steptest.Response{
					Status: 201,
					Body: steptest.Body{JSON: map[string]any{
						"id":           ns,
						"tag":          "update-res",
						"pid_format":   map[string]any{"pattern": "***-***", "characters": "full"},
						"date_created": h.now,
					}},
				},
			},
			{
				Name: "create_resource",
				Request: steptest.Request{
					Method: "POST",
					Path:   "/resolver/namespaces/" + ns + "/resources",
					Body:   `{"url":"https://example.com/a","metadata":"ext-1@sys-a","tag":"alpha"}`,
				},
				Response: steptest.Response{Status: 201},
			},
			{
				Name: "success_allFields",
				Request: steptest.Request{
					Method: "PATCH",
					Path:   "/resolver/namespaces/" + ns + "/resources/" + pid,
					Body:   `{"url":"https://example.com/updated","metadata":"ext-1b@sys-b","tag":"beta","deleted":false}`,
				},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{
					"pid":          pid,
					"url":          "https://example.com/updated",
					"metadata":     "ext-1b@sys-b",
					"date_created": h.now,
					"date_updated": h.now,
					"tag":          "beta",
					"deleted":      false,
				}}},
			},
			{
				Name: "partial_urlOnly",
				Request: steptest.Request{
					Method: "PATCH",
					Path:   "/resolver/namespaces/" + ns + "/resources/" + pid,
					Body:   `{"url":"https://example.com/url-only"}`,
				},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{
					"pid":          pid,
					"url":          "https://example.com/url-only",
					"metadata":     "ext-1b@sys-b",
					"date_created": h.now,
					"date_updated": h.now,
					"tag":          "beta",
					"deleted":      false,
				}}},
			},
			{
				Name: "partial_setMetadataNull",
				Request: steptest.Request{
					Method: "PATCH",
					Path:   "/resolver/namespaces/" + ns + "/resources/" + pid,
					Body:   `{"metadata":null}`,
				},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{
					"pid":          pid,
					"url":          "https://example.com/url-only",
					"metadata":     nil,
					"date_created": h.now,
					"date_updated": h.now,
					"tag":          "beta",
					"deleted":      false,
				}}},
			},
			{
				Name: "partial_setMetadataString",
				Request: steptest.Request{
					Method: "PATCH",
					Path:   "/resolver/namespaces/" + ns + "/resources/" + pid,
					Body:   `{"metadata":"ext-new@sys-c"}`,
				},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{
					"pid":          pid,
					"url":          "https://example.com/url-only",
					"metadata":     "ext-new@sys-c",
					"date_created": h.now,
					"date_updated": h.now,
					"tag":          "beta",
					"deleted":      false,
				}}},
			},
			{
				Name: "partial_deletedOnly",
				Request: steptest.Request{
					Method: "PATCH",
					Path:   "/resolver/namespaces/" + ns + "/resources/" + pid,
					Body:   `{"deleted":true}`,
				},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{
					"pid":          pid,
					"url":          "https://example.com/url-only",
					"metadata":     "ext-new@sys-c",
					"date_created": h.now,
					"date_updated": h.now,
					"tag":          "beta",
					"deleted":      true,
				}}},
			},
			{
				Name:    "emptyUpdate_noChanges",
				Request: steptest.Request{Method: "PATCH", Path: "/resolver/namespaces/" + ns + "/resources/" + pid, Body: `{}`},
				Response: steptest.Response{Status: 200, Body: steptest.Body{JSON: map[string]any{
					"pid":          pid,
					"url":          "https://example.com/url-only",
					"metadata":     "ext-new@sys-c",
					"date_created": h.now,
					"date_updated": h.now,
					"tag":          "beta",
					"deleted":      true,
				}}},
			},
			{
				Name: "resourceNotFound",
				Request: steptest.Request{
					Method: "PATCH",
					Path:   "/resolver/namespaces/" + ns + "/resources/99999",
					Body:   `{"url":"https://example.com/updated","metadata":"ext-1b@sys-b","tag":"beta","deleted":false}`,
				},
				Response: steptest.Response{Status: 404, Body: steptest.Body{JSON: map[string]any{"error": "resource not found"}}},
			},
			{
				Name: "invalidPID",
				Request: steptest.Request{
					Method: "PATCH",
					Path:   "/resolver/namespaces/" + ns + "/resources/bad.pid",
					Body:   `{"url":"https://example.com/updated","metadata":"ext-1b@sys-b","tag":"beta","deleted":false}`,
				},
				Response: steptest.Response{Status: 400, Body: steptest.Body{JSON: map[string]any{"error": "invalid pid"}}},
			},
			{
				Name: "tooLargeBody",
				Request: steptest.Request{
					Method: "PATCH",
					Path:   "/resolver/namespaces/" + ns + "/resources/" + pid,
					Body:   `{"url":"https://` + strings.Repeat("a", 512) + `","metadata":"x","tag":"z","deleted":false}`,
				},
				Response: steptest.Response{Status: 413, Body: steptest.Body{JSON: map[string]any{"error": "request payload too large"}}},
			},
		},
	}.Run(t, h)
}
