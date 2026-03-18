//go:build diskbased

/*
 * EliasDB
 *
 * Copyright 2016 Matthias Ladkau. All rights reserved.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

/***

If you have the server running somewhere else, you'll encounter this.

rickhedin@Ricks-MacBook-Pro graphql % go test -tags=diskbased
--- FAIL: TestSimpleQueries (0.50s)
panic: Could not take ownership of lockfile queryDiskbasedTestDBDir/mainAuthor.nodeidx: Could not write lockfile - read result after writing: 1773370882578508000(expected: 1773370884268406000)<nil> [recovered, repanicked]

goroutine 49 [running]:
testing.tRunner.func1.2({0x100c02160, 0x1400040c030})
	/Users/rickhedin/.goenv/versions/1.25.4/src/testing/testing.go:1872 +0x190
testing.tRunner.func1()
	/Users/rickhedin/.goenv/versions/1.25.4/src/testing/testing.go:1875 +0x31c
panic({0x100c02160?, 0x1400040c030?})
	/Users/rickhedin/.goenv/versions/1.25.4/src/runtime/panic.go:783 +0x120
github.com/rhedin/Abe_eliasdb/storage.initByteDiskStorageManager(0x14000418f80)
	/Users/rickhedin/work/260102/Abe_eliasdb/storage/diskstoragemanager.go:657 +0x8c8
	...
	github.com/rhedin/Abe_eliasdb/graphql.TestSimpleQueries(0x14000582380)
	/Users/rickhedin/work/260102/Abe_eliasdb/graphql/query_diskbased_test.go:64 +0x28
testing.tRunner(0x14000582380, 0x100c59128)
	/Users/rickhedin/.goenv/versions/1.25.4/src/testing/testing.go:1934 +0xc8
created by testing.(*T).Run in goroutine 1
	/Users/rickhedin/.goenv/versions/1.25.4/src/testing/testing.go:1997 +0x364
exit status 2
FAIL	github.com/rhedin/Abe_eliasdb/graphql	2.389s
rickhedin@Ricks-MacBook-Pro graphql %

Actually, it turned out the tests were interfering with each other.  The first
test still had its lockfile monitor running, which meant that the second test
couldn't get hold of the file.

Hey.  The most useful way to understand this file is to do a:

sdiff -w 200 graphql/query_diskbased_test.go graphql/query_test.go | less

***/

package graphql

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	// "time"

	"github.com/rhedin/Abe_common/fileutil"
	"github.com/rhedin/Abe_eliasdb/graph"
	"github.com/rhedin/Abe_eliasdb/graph/data"
	"github.com/rhedin/Abe_eliasdb/graph/graphstorage"
)

const queryDiskbasedTestDBDir = "querydiskbasedtestdbdir"

var dbdirs = []string{queryDiskbasedTestDBDir}

// Main function for all tests in this package

func TestMain(m *testing.M) {
	flag.Parse()

	// createDirectory()  Actually, these are called from each test.

	// Run the tests

	res := m.Run()

	// Teardown

	// destroyDirectory()

	os.Exit(res)
}

// Dude!  You completely put the remove-directory logic into create-directory also!
// No, that's correct.  I thought I was creating the directory beforehand, and
// removing the directory after, but I was removing it in both cases.  The directory
// is created by the NewDiskGraphStorage call.

/*** never existed
func createDirectory() {
	for _, dbdir := range dbdirs {
		if res, _ := fileutil.PathExists(dbdir); res {
			if err := os.RemoveAll(dbdir); err != nil {
				fmt.Print("Could not remove test directory:", err.Error())
			}
		}
	}
}
***/

func destroyDirectory(directoryName string) {
	if res, _ := fileutil.PathExists(directoryName); res {
		if err := os.RemoveAll(directoryName); err != nil {
			fmt.Print("Could not remove test directory:", err.Error(), "\n")
		}
	}
}

func printContentsOfDirectory(directoryName string) {

	fmt.Printf("--- Contents of directory: %q ---\n", directoryName)

	// Check if it even exists anymore
	if _, err := os.Stat(directoryName); os.IsNotExist(err) {
		fmt.Println("  Directory does not exist (successfully removed)")
		return
	} else if err != nil {
		fmt.Printf("  Cannot stat directory: %v\n", err)
		return
	}

	// Walk the directory tree
	err := filepath.Walk(directoryName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("  Walk error at %s: %v\n", path, err)
			return nil // continue anyway
		}

		// Skip the root directory itself in output (less noise)
		if path == directoryName {
			return nil
		}

		rel, _ := filepath.Rel(directoryName, path)
		if info.IsDir() {
			fmt.Printf("  directory  %s/\n", rel)
		} else {
			fmt.Printf("  file %s  (size %d bytes)\n", rel, info.Size())
		}
		return nil
	})

	if err != nil {
		fmt.Printf("  Walk failed: %v\n", err)
	}

	fmt.Println("--- End of directory listing ---")
}

func TestErrorCases(t *testing.T) {

	// const directoryName = "queryDiskBasedTestErrorCasesDir"
	const directoryName = "queryDiskBasedTestSharedDir"

	destroyDirectory(directoryName)
	fmt.Printf("After destroyDirectory at beginning of TestErrorCases:\n")
	printContentsOfDirectory(directoryName)

	gm, dgs := songGraphGroups(directoryName)

	query := map[string]interface{}{
		"operationName": "foo",
		"variables":     nil,
	}

	_, err := RunQuery("test", "main", query, gm, nil, false)
	if err == nil || err.Error() != "Mandatory field 'query' missing from query object" {
		t.Error("Unexpected result:", err)
		return
	}

	query = map[string]interface{}{
		"query":     nil,
		"variables": nil,
	}

	_, err = RunQuery("test", "main", query, gm, nil, false)
	if err == nil || err.Error() != "Mandatory field 'operationName' missing from query object" {
		t.Error("Unexpected result:", err)
		return
	}

	query = map[string]interface{}{
		"operationName": "foo",
		"query":         nil,
	}

	_, err = RunQuery("test", "main", query, gm, nil, false)
	if err == nil || err.Error() != "Mandatory field 'variables' missing from query object" {
		t.Error("Unexpected result:", err)
		return
	}

	// Added code to close disk graph storage.
	if err := dgs.Close(); err != nil {
		t.Error(err)
		return
	}

	destroyDirectory(directoryName)
	fmt.Printf("After destroyDirectory at end of TestErrorCases:\n")
	printContentsOfDirectory(directoryName)

	// time.Sleep(5 * time.Second)  Seeing whether it works without a time delay.
	// Even though there was a visible pause before the first test (this one) finished,
	// the directory on disk apparently still survived.  The second test failed,
	// even though when it used an entirely different directory on disk, it succeeded.
}

func TestSimpleQueries(t *testing.T) {

	// const directoryName = "queryDiskBasedTestSimpleQueriesDir"
	const directoryName = "queryDiskBasedTestSharedDir"

	destroyDirectory(directoryName)
	fmt.Printf("After destroyDirectory at beginning of TestSimpleQueries:\n")
	printContentsOfDirectory(directoryName)

	gm, dgs := songGraphGroups(directoryName)

	ast, err := ParseQuery("test", "{ name }")

	if err != nil || ast.String() != `
Document
  ExecutableDefinition
    OperationDefinition
      SelectionSet
        Field
          Name: name
`[1:] {
		t.Error("Unexpected result:", ast, err)
	}

	ast, err = ParseQuery("test", "{ name ")

	if err == nil || err.Error() != "Parse error in test: Unexpected end (Line:1 Pos:7)" {
		t.Error("Unexpected result:", ast, err)
	}

	query := map[string]interface{}{
		"operationName": "foo",
		"query": `
{
  Song {
	key
  }
}
`,
		"variables": nil,
	}

	_, err = RunQuery("test", "main", query, gm, nil, false)
	if err == nil || err.Error() != "Fatal GraphQL operation error in test: Missing operation (Operation foo not found) (Line:2 Pos:2)" {
		t.Error("Unexpected result:", err)
		return
	}

	query = map[string]interface{}{
		"operationName": nil,
		"query":         nil,
		"variables":     nil,
	}

	_, err = RunQuery("test", "main", query, gm, nil, false)
	if err == nil || err.Error() != "Fatal GraphQL operation error in test: Missing operation (No executable expression found) (Line:1 Pos:0)" {
		t.Error("Unexpected result:", err)
		return
	}

	query = map[string]interface{}{
		"operationName": nil,
		"query": `
fragment friendFields on User {
  id
  name
  profilePic(size: 50)
}
`,
		"variables": nil,
	}

	res, err := RunQuery("test", "main", query, gm, nil, false)
	if err == nil || err.Error() != "Fatal GraphQL operation error in test: Missing operation (No executable expression found) (Line:2 Pos:2)" {
		t.Error("Unexpected result:", res, err)
		return
	}

	query = map[string]interface{}{
		"operationName": nil,
		"query": `
{
  Song1 : Song {
    song_key : key
    song_key1 : key
    song_key1 : name # This is illegal and will be ignored
    song_key1 : key
	name
	name
  }
  group {
	key
  },
}
`,
		"variables": nil,
	}

	if rerr := checkResult(`
{
  "data": {
    "Song1": [
      {
        "name": "StrangeSong1",
        "song_key": "StrangeSong1",
        "song_key1": "StrangeSong1"
      },
      {
        "name": "FightSong4",
        "song_key": "FightSong4",
        "song_key1": "FightSong4"
      },
      {
        "name": "DeadSong2",
        "song_key": "DeadSong2",
        "song_key1": "DeadSong2"
      },
      {
        "name": "LoveSong3",
        "song_key": "LoveSong3",
        "song_key1": "LoveSong3"
      },
      {
        "name": "MyOnlySong3",
        "song_key": "MyOnlySong3",
        "song_key1": "MyOnlySong3"
      },
      {
        "name": "Aria1",
        "song_key": "Aria1",
        "song_key1": "Aria1"
      },
      {
        "name": "Aria2",
        "song_key": "Aria2",
        "song_key1": "Aria2"
      },
      {
        "name": "Aria3",
        "song_key": "Aria3",
        "song_key1": "Aria3"
      },
      {
        "name": "Aria4",
        "song_key": "Aria4",
        "song_key1": "Aria4"
      }
    ],
    "group": [
      {
        "key": "Best"
      }
    ]
  },
  "errors": [
    {
      "locations": [
        {
          "column": 17,
          "line": 3
        }
      ],
      "message": "Field identifier song_key1 used multiple times",
      "path": [
        "Song1"
      ]
    },
    {
      "locations": [
        {
          "column": 17,
          "line": 3
        }
      ],
      "message": "Field identifier name used multiple times",
      "path": [
        "Song1"
      ]
    }
  ]
}`[1:], query, gm); rerr != nil {
		t.Error(rerr)
		return
	}

	query = map[string]interface{}{
		"operationName": nil,
		"query": `
fragment Song on Song {
  key
  name1 : name
  ...SongKind
}
query b {  # This should be executed
  Song {
	key
	...Song
  }
}
query a {
  Song1 {
	key1
  }
}
fragment SongKind on Song {
  kind
  name2 : name
  key
}
`,
		"variables": nil,
	}

	if rerr := checkResult(`
{
  "data": {
    "Song": [
      {
        "key": "StrangeSong1",
        "kind": "Song",
        "name1": "StrangeSong1",
        "name2": "StrangeSong1"
      },
      {
        "key": "FightSong4",
        "kind": "Song",
        "name1": "FightSong4",
        "name2": "FightSong4"
      },
      {
        "key": "DeadSong2",
        "kind": "Song",
        "name1": "DeadSong2",
        "name2": "DeadSong2"
      },
      {
        "key": "LoveSong3",
        "kind": "Song",
        "name1": "LoveSong3",
        "name2": "LoveSong3"
      },
      {
        "key": "MyOnlySong3",
        "kind": "Song",
        "name1": "MyOnlySong3",
        "name2": "MyOnlySong3"
      },
      {
        "key": "Aria1",
        "kind": "Song",
        "name1": "Aria1",
        "name2": "Aria1"
      },
      {
        "key": "Aria2",
        "kind": "Song",
        "name1": "Aria2",
        "name2": "Aria2"
      },
      {
        "key": "Aria3",
        "kind": "Song",
        "name1": "Aria3",
        "name2": "Aria3"
      },
      {
        "key": "Aria4",
        "kind": "Song",
        "name1": "Aria4",
        "name2": "Aria4"
      }
    ]
  },
  "errors": [
    {
      "locations": [
        {
          "column": 9,
          "line": 8
        }
      ],
      "message": "Field identifier key used multiple times",
      "path": [
        "Song"
      ]
    }
  ]
}`[1:], query, gm); rerr != nil {
		t.Error(rerr)
		return
	}

	// I added code to close disk graph storage.  It was keeping ahold of the lock file.
	// Note I replaced a _ return from songGraphGroups with dgs.
	if err := dgs.Close(); err != nil {
		t.Error(err)
		return
	}

	destroyDirectory(directoryName)
	fmt.Printf("After destroyDirectory at end of TestSimpleQueries:\n")
	printContentsOfDirectory(directoryName)
}

func checkResult(expectedResult string, query map[string]interface{}, gm *graph.Manager) error {
	actualResultObject, err := RunQuery("test", "main", query, gm, nil, false)

	if err == nil {
		var actualResultBytes []byte
		actualResultBytes, err = json.MarshalIndent(actualResultObject, "", "  ")
		actualResult := string(actualResultBytes)

		if err == nil {
			if expectedResult != actualResult {
				err = fmt.Errorf("Unexpected result:\nExpected:\n%s\nActual:\n%s",
					expectedResult, actualResult)
			}
		}

	}

	if err != nil {
		fmt.Println(ParseQuery("test", fmt.Sprint(query["query"])))
	}

	return err
}

func songGraph(directoryName string) (*graph.Manager, *graphstorage.DiskGraphStorage) {

	mgs, err := graphstorage.NewDiskGraphStorage(directoryName, false) // false means not readonly
	// I don't have t in this function, so I can't say t.Error(err).
	// If it errors, I'll just say so, return, and hope for the best.
	if err != nil {
		fmt.Printf("graphstorage.NewDiskGraphStorage(~) returned an error.  err = %v\n", err)
		// return It got mad because I didn't return *gm, *gs.  I didn't want to import os
		// so I could use os.Exit(1).  I just go on.  I wrote out what happened.
	}
	gm := graph.NewGraphManager(mgs)

	constructEdge := func(key string, node1 data.Node, node2 data.Node, number int) data.Edge {
		edge := data.NewGraphEdge()

		edge.SetAttr("key", key)
		edge.SetAttr("kind", "Wrote")

		edge.SetAttr(data.EdgeEnd1Key, node1.Key())
		edge.SetAttr(data.EdgeEnd1Kind, node1.Kind())
		edge.SetAttr(data.EdgeEnd1Role, "Author")
		edge.SetAttr(data.EdgeEnd1Cascading, true)

		edge.SetAttr(data.EdgeEnd2Key, node2.Key())
		edge.SetAttr(data.EdgeEnd2Kind, node2.Kind())
		edge.SetAttr(data.EdgeEnd2Role, "Song")
		edge.SetAttr(data.EdgeEnd2Cascading, false)

		edge.SetAttr("number", number)

		return edge
	}

	storeSong := func(node data.Node, name string, ranking int, number int) {
		node3 := data.NewGraphNode()
		node3.SetAttr("key", name)
		node3.SetAttr("kind", "Song")
		node3.SetAttr("name", name)
		node3.SetAttr("ranking", ranking)
		gm.StoreNode("main", node3)
		gm.StoreEdge("main", constructEdge(name, node, node3, number))
	}

	node0 := data.NewGraphNode()
	node0.SetAttr("key", "000")
	node0.SetAttr("kind", "Author")
	node0.SetAttr("name", "John")
	gm.StoreNode("main", node0)

	storeSong(node0, "Aria1", 8, 1)
	storeSong(node0, "Aria2", 2, 2)
	storeSong(node0, "Aria3", 4, 3)
	storeSong(node0, "Aria4", 18, 4)

	node1 := data.NewGraphNode()
	node1.SetAttr("key", "123")
	node1.SetAttr("kind", "Author")
	node1.SetAttr("name", "Mike")
	gm.StoreNode("main", node1)

	storeSong(node1, "LoveSong3", 1, 3)
	storeSong(node1, "FightSong4", 3, 4)
	storeSong(node1, "DeadSong2", 6, 2)
	storeSong(node1, "StrangeSong1", 5, 1)

	node2 := data.NewGraphNode()
	node2.SetAttr("key", "456")
	node2.SetAttr("kind", "Author")
	node2.SetAttr("name", "Hans")
	gm.StoreNode("main", node2)

	storeSong(node2, "MyOnlySong3", 19, 3)

	return gm, mgs.(*graphstorage.DiskGraphStorage)
}

func songGraphGroups(directoryName string) (*graph.Manager, *graphstorage.DiskGraphStorage) {
	gm, mgs := songGraph(directoryName)

	node0 := data.NewGraphNode()
	node0.SetAttr("key", "Best")
	node0.SetAttr("kind", "group")
	gm.StoreNode("main", node0)

	constructEdge := func(songkey string) data.Edge {
		edge := data.NewGraphEdge()

		edge.SetAttr("key", songkey)
		edge.SetAttr("kind", "Contains")

		edge.SetAttr(data.EdgeEnd1Key, node0.Key())
		edge.SetAttr(data.EdgeEnd1Kind, node0.Kind())
		edge.SetAttr(data.EdgeEnd1Role, "group")
		edge.SetAttr(data.EdgeEnd1Cascading, false)

		edge.SetAttr(data.EdgeEnd2Key, songkey)
		edge.SetAttr(data.EdgeEnd2Kind, "Song")
		edge.SetAttr(data.EdgeEnd2Role, "Song")
		edge.SetAttr(data.EdgeEnd2Cascading, false)

		return edge
	}

	gm.StoreEdge("main", constructEdge("LoveSong3"))
	gm.StoreEdge("main", constructEdge("Aria3"))
	gm.StoreEdge("main", constructEdge("MyOnlySong3"))
	gm.StoreEdge("main", constructEdge("StrangeSong1"))

	return gm, mgs
}
