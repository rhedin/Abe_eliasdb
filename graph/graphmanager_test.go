/*
 * EliasDB
 *
 * Copyright 2016 Matthias Ladkau. All rights reserved.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package graph

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"devt.de/krotik/common/fileutil"
	"devt.de/krotik/eliasdb/graph/data"
	"devt.de/krotik/eliasdb/graph/graphstorage"
)

/*
Flag to enable / disable tests which use actual disk storage.
(Only used for test development - should never be false)
*/
const RunDiskStorageTests = true

const GraphManagerTestDBDir1 = "gmtest1"
const GraphManagerTestDBDir2 = "gmtest2"
const GraphManagerTestDBDir3 = "gmtest3"
const GraphManagerTestDBDir4 = "gmtest4"
const GraphManagerTestDBDir5 = "gmtest5"
const GraphManagerTestDBDir6 = "gmtest6"

var DBDIRS = []string{GraphManagerTestDBDir1, GraphManagerTestDBDir2,
	GraphManagerTestDBDir3, GraphManagerTestDBDir4, GraphManagerTestDBDir5,
	GraphManagerTestDBDir6}

const InvlaidFileName = "**" + "\x00"

// Main function for all tests in this package

func TestMain(m *testing.M) {
	flag.Parse()

	for _, dbdir := range DBDIRS {
		if res, _ := fileutil.PathExists(dbdir); res {
			if err := os.RemoveAll(dbdir); err != nil {
				fmt.Print("Could not remove test directory:", err.Error())
			}
		}
	}

	// Run the tests

	res := m.Run()

	// Teardown

	for _, dbdir := range DBDIRS {
		if res, _ := fileutil.PathExists(dbdir); res {
			if err := os.RemoveAll(dbdir); err != nil {
				fmt.Print("Could not remove test directory:", err.Error())
			}
		}
	}

	os.Exit(res)
}

/*
NewGraphManager returns a new GraphManager instance without loading rules.
*/
func newGraphManagerNoRules(gs graphstorage.Storage) *Manager {
	return createGraphManager(gs)
}

/*
Create a GraphManager which has some prefilled data.
*/
func songGraph() (*Manager, *graphstorage.MemoryGraphStorage) {

	mgs := graphstorage.NewMemoryGraphStorage("mystorage")
	gm := NewGraphManager(mgs)

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
		gm.StoreEdge("main", constructEdge(name+"-edge", node, node3, number))
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

	return gm, mgs.(*graphstorage.MemoryGraphStorage)
}
