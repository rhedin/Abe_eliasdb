/*
 * EliasDB
 *
 * Copyright 2016 Matthias Ladkau. All rights reserved.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"devt.de/krotik/common/stringutil"
	"devt.de/krotik/eliasdb/api"
	"devt.de/krotik/eliasdb/graphql"
)

/*
EndpointGraphQL is the GraphQL endpoint URL (rooted). Handles everything under graphql/...
*/
const EndpointGraphQL = api.APIRoot + APIv1 + "/graphql/"

/*
GraphQLEndpointInst creates a new endpoint handler.
*/
func GraphQLEndpointInst() api.RestEndpointHandler {
	return &graphQLEndpoint{}
}

/*
Handler object for GraphQL operations.
*/
type graphQLEndpoint struct {
	*api.DefaultEndpointHandler
}

/*
HandlePOST handles GraphQL queries.
*/
func (e *graphQLEndpoint) HandlePOST(w http.ResponseWriter, r *http.Request, resources []string) {

	dec := json.NewDecoder(r.Body)
	data := make(map[string]interface{})

	if err := dec.Decode(&data); err != nil {
		http.Error(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	partData, ok := data["partition"]
	if !ok && len(resources) > 0 {
		partData = resources[0]
		ok = true
	}
	if !ok || partData == "" {
		http.Error(w, "Need a partition", http.StatusBadRequest)
		return
	}

	part := fmt.Sprint(partData)

	if _, ok := data["variables"]; !ok {
		data["variables"] = nil
	}

	if _, ok := data["operationName"]; !ok {
		data["operationName"] = nil
	}

	res, err := graphql.RunQuery(stringutil.CreateDisplayString(part)+" query",
		part, data, api.GM, nil, false)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("content-type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(res)
}

/*
SwaggerDefs is used to describe the endpoint in swagger.
*/
func (e *graphQLEndpoint) SwaggerDefs(s map[string]interface{}) {

	s["paths"].(map[string]interface{})["/v1/graphql"] = map[string]interface{}{
		"post": map[string]interface{}{
			"summary":     "GraphQL interface.",
			"description": "The GraphQL interface can be used to query and modify data.",
			"consumes": []string{
				"application/json",
			},
			"produces": []string{
				"text/plain",
				"application/json",
			},
			"parameters": []map[string]interface{}{
				map[string]interface{}{
					"name":        "partition",
					"in":          "path",
					"description": "Partition to query.",
					"required":    false,
					"type":        "string",
				},
				map[string]interface{}{
					"name":        "partition",
					"in":          "body",
					"description": "Partition to query.",
					"required":    false,
					"type":        "string",
				},
				map[string]interface{}{
					"name":        "operationName",
					"in":          "body",
					"description": "GraphQL query operation name.",
					"required":    false,
				},
				map[string]interface{}{
					"name":        "query",
					"in":          "body",
					"description": "GraphQL query.",
					"required":    true,
				},
				map[string]interface{}{
					"name":        "variables",
					"in":          "body",
					"description": "GraphQL query variable values.",
					"required":    false,
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "The operation was successful.",
				},
				"default": map[string]interface{}{
					"description": "Error response",
					"schema": map[string]interface{}{
						"$ref": "#/definitions/Error",
					},
				},
			},
		},
	}
}
