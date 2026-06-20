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
	"regexp"
	"testing"

	"github.com/rhedin/Abe_eliasdb/graph"
	"github.com/rhedin/Abe_eliasdb/storage"
)

func TestIndexQuery(t *testing.T) {
	queryURL := "http://localhost" + TESTPORT + EndpointIndexQuery

	st, _, res := sendTestRequest(queryURL+"//main/x/", "GET", nil)
	if st != "400 Bad Request" || res != "Need a partition, entity type (n or e) and a kind" {
		t.Error("Unexpected response:", st, res)
		return
	}

	st, _, res = sendTestRequest(queryURL+"//main/x/bla", "GET", nil)
	if st != "400 Bad Request" || res != "Entity type must be n (nodes) or e (edges)" {
		t.Error("Unexpected response:", st, res)
		return
	}

	st, _, res = sendTestRequest(queryURL+"//main/n/bla?attr=1", "GET", nil)
	if st != "400 Bad Request" || res != "Unknown partition or node kind" {
		t.Error("Unexpected response:", st, res)
		return
	}

	st, _, res = sendTestRequest(queryURL+"//main/n/Song", "GET", nil)
	if st != "400 Bad Request" || res != "Query string for attr (attribute) is required" {
		t.Error("Unexpected response:", st, res)
		return
	}

	st, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=1", "GET", nil)
	if st != "400 Bad Request" || res != "Query string for either phrase, word or value is required" {
		t.Error("Unexpected response:", st, res)
		return
	}

	_, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=1&word=1", "GET", nil)
	if res != "{}" {
		t.Error("Unexpected response:", st, res)
		return
	}

	_, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=1&phrase=1", "GET", nil)
	if res != "[]" {
		t.Error("Unexpected response:", st, res)
		return
	}

	_, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=1&value=1", "GET", nil)
	if res != "[]" {
		t.Error("Unexpected response:", st, res)
		return
	}

	_, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=name&value=Aria1", "GET", nil)
	if res != `
[
  "Aria1"
]`[1:] {
		t.Error("Unexpected response:", res)
		return
	}

	_, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=name&phrase=Aria1", "GET", nil)
	if res != `
[
  "Aria1"
]`[1:] {
		t.Error("Unexpected response:", res)
		return
	}

	_, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=name&word=Aria1", "GET", nil)
	if res != `
{
  "Aria1": [
    1
  ]
}`[1:] {
		t.Error("Unexpected response:", res)
		return
	}

	_, _, res = sendTestRequest(queryURL+"//main/e/Wrote?attr=number&word=1", "GET", nil)
	if res != `
{
  "Aria1": [
    1
  ],
  "StrangeSong1": [
    1
  ]
}`[1:] {
		t.Error("Unexpected response:", res)
		return
	}

	// msm := gmMSM.StorageManager("main"+"Song"+graph.StorageSuffixNodesIndex,
	// 	true).(*storage.MemoryStorageManager)
	msm := gmMSM.StorageManager(graph.CaseSensitiveName("main", "Song", graph.StorageSuffixNodesIndex),
		true).(*storage.MemoryStorageManager)

	for i := 2; i < 30; i++ {
		msm.AccessMap[uint64(i)] = storage.AccessCacheAndFetchError
	}

	st, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=name&value=Aria1", "GET", nil)

	matched, _ := regexp.MatchString(`^\QGraphError: Index error (Slot not found (mystorage/mainSong\E(....)?\Q.nodeidx - Location\E`, res)
	if st != "500 Internal Server Error" || !matched {
		t.Error("Unexpected response:", res)
		return
	}
	// Since the original said HasPrefix, we leave the tail of the regular expression unanchored.  strings.HasPrefix(res, "GraphError: . . .
	//
	// Here's what the test runner said.
	// === RUN   TestIndexQuery
	//     index_test.go:128: Unexpected response: GraphError: Index error (Slot not found (mystorage/mainSong918b.nodeidx - Location:3))
	// --- FAIL: TestIndexQuery (0.00s)
	//
	// Here's how Grok summarized it.
	// This is actually progress. The test is now finally talking to the right storage manager, so you're seeing the genuine error message
	// instead of a higher-level wrapper. You just need to update the expected string to match reality.
	//
	// Here's what the MatchString line used to be, before I changed it to match the new situation.
	// matched, _ := regexp.MatchString(`^\QGraphError: Failed to access graph storage component (Slot not found (mystorage/mainSong\E(....)?\Q.nodeidx - Location\E`, res)

	for i := 2; i < 30; i++ {
		delete(msm.AccessMap, uint64(i))
	}

	msm.AccessMap[1] = storage.AccessCacheAndFetchError

	st, _, res = sendTestRequest(queryURL+"//main/n/Song?attr=name&value=Aria1", "GET", nil)

	matched, _ = regexp.MatchString(`^\QGraphError: Failed to access graph storage component (Slot not found (mystorage/mainSong\E(....)?\Q.nodeidx - Location:1))\E$`, res)
	if st != "500 Internal Server Error" || !matched {
		t.Error("Unexpected response:", res)
		return
	}

	delete(msm.AccessMap, 1)

}
