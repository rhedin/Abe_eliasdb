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
	"github.com/rhedin/Abe_eliasdb/graph/util"
	"github.com/rhedin/Abe_eliasdb/hash"
)

/*
NodeKeyIterator can be used to iterate node keys of a certain node kind.
*/
type NodeKeyIterator struct {
	gm        *Manager            // GraphManager which created the iterator
	it        *hash.HTreeIterator // Internal HTree iterator
	LastError error               // Last encountered error
}

/*
Next returns the next node key. Sets the LastError attribute if an error occurs.
*/
func (it *NodeKeyIterator) Next() string {

	// Take reader lock
	it.gm.mutex.RLock()
	defer it.gm.mutex.RUnlock()

	for {

		/***/

		k, _ := it.it.Next()

		/***

		k, something := it.it.Next()
		fmt.Printf("k = %#v, something = %#v\n", k, something)

		***/

		if it.it.LastError != nil {
			it.LastError = &util.GraphError{
				Type:   util.ErrReading,
				Detail: it.it.LastError.Error(),
			}
			return ""
		}
		if len(k) == 0 {
			return ""
		}

		// Only accept entries that start with PrefixNSAttrs (0x01)
		if len(k) > 0 && k[0] == PrefixNSAttrs[0] {
			return string(k[len(PrefixNSAttrs):])
		}

		// Skip everything else (individual attribute values, etc.)
	}
}

/***
Grok and my belief is that this code, for iterating through the keys available for a given kind
for a node, is wrong.  It only happens to work because we didn't run across any PrefixNSAttr's
or PrefixNSSpecs, or PrefixNSEdge's.
If and when we are proven correct, and this code malfunctions, insert the fix that already
exists in (it *EdgeKeyIterator) Next() below.
***/

/***
Here's some information I discovered.  It was in globals.go.

# Nodes database

Each node kind database stores:

	PrefixNSAttrs + node key -> [ ATTRS ]
	(a list of attributes of a certain node)

	PrefixNSAttr + node key + attr num -> value
	(attribute value of a certain node)

	PrefixNSSpecs + node key -> map[spec]<empty string>
	(a lookup for available specs for a certain node)

	PrefixNSEdge + node key + spec -> map[edge key]edgeinfo{other node key, other node kind}]
	(connection from one node to another via a spec)

# Edges database

Each edge kind database stores:

	PrefixNSAttrs + edge key -> [ ATTRS ]
	(a list of attributes of a certain edge)

	PrefixNSAttr + edge key + attr num -> value
	(attribute value of a certain edge)

And farther down on the page.

// PREFIXES for Node storage
// =========================

// Prefixes are only one byte. They should be followed by the node key so
// similar entries are stored near each other physically.
//

//
// PrefixNSAttrs is the prefix for storing attributes of a node
//
const PrefixNSAttrs = "\x01"

//
// PrefixNSAttr is the prefix for storing the value of a node attribute
//
const PrefixNSAttr = "\x02"

//
// PrefixNSSpecs is the prefix for storing specs of edges related to a node
//
const PrefixNSSpecs = "\x03"

//
// PrefixNSEdge is the prefix for storing a link from a node (and a spec) to an edge
//
const PrefixNSEdge = "\x04"

With this information, it should be easier to interpret the output pasted in
under the (it *EdgeKeyIterator) Next() implementation.

(after looking at it) Although I thought PrefixNSAttrs was a list of multiple
attributes, and PrefixNSAttr was just one attribute, apparently not.  I should
look at the "attribute" rather than the "value", instead of the attrS rather
then the attr .  Attrs is the key, and Attr is the value.

I checked the output.  Only Attrs and Attr, 0x1 and 0x2, occur in the edge hash store.
For kind Connection in our sample database, at least.
rickhedin@Ricks-MacBook-Pro Abe_eliasdb % grep -v 'k = ..byte{0x2,' /tmp/3.txt >/tmp/4.txt
rickhedin@Ricks-MacBook-Pro Abe_eliasdb % sdiff -w 200 /tmp/4.txt /tmp/3.txt | less
rickhedin@Ricks-MacBook-Pro Abe_eliasdb % cat /tmp/4.txt
Entering for iter.HasNext() loop.  Going to pull out all the keys for kind Connection and not process them.
Came out of for iter.HasNext() loop.
rickhedin@Ricks-MacBook-Pro Abe_eliasdb %
***/

/***
I have chickened out.  I'm going to insert the fix into (it *NodeKeyIterator) Next().  I have gotten
the tests working, and succeeding across the board.  That makes me a little more confident that
I can make a change without breaking anything.  But honestly, I don't think the coverage is
good enough to catch a change like this.  We haven't even characterized how to make the
error happen.
***/

/***
I think I was wrong.  I don't think I needed to filter types in (it *NodeKeyIterator) Next() string.
In the nodes, but not in the edges, there's a separate data structure reserved for the keys.
Oh well.  Some extra filtering.

// getNodeStorageHTree gets two HTree instances which can be used to store nodes.
// This function ensures that depending entries in other datastructures do exist.
func (gm *Manager) getNodeStorageHTree(part string, kind string,
	create bool) (*hash.HTree, *hash.HTree, error) {

	. . .

	// Return the actual storage

	gs := gm.gs.StorageManager(caseSensitiveName(part, kind, StorageSuffixNodes), create)
	if gs == nil {
		return nil, nil, nil
	}

	attrTree, err := gm.getHTree(gs, RootIDNodeHTree)
	if err != nil {
		return nil, nil, err
	}

	valTree, err := gm.getHTree(gs, RootIDNodeHTreeSecond)
	if err != nil {
		return nil, nil, err
	}

	return attrTree, valTree, nil     <<<<<< returns two trees, one for attributes (keys) and one for the values of those keys.
}

// getEdgeStorageHTree gets a HTree which can be used to store edges. This function ensures that depending
// entries in other datastructures do exist.
func (gm *Manager) getEdgeStorageHTree(part string, kind string, create bool) (*hash.HTree, error) {

	. . .

	// Return the actual storage

	gs := gm.gs.StorageManager(caseSensitiveName(part, kind, StorageSuffixEdges), create)
	if gs == nil {
		return nil, nil
	}

	return gm.getHTree(gs, RootIDNodeHTree)     <<<<<< returns one key, combining keys and values
}
***/

/*
HasNext returns if there is a next node key.
*/
func (it *NodeKeyIterator) HasNext() bool {
	return it.it.HasNext()
}

/*
Error returns the last encountered error.
*/
func (it *NodeKeyIterator) Error() error {
	return it.LastError
}

//
// Grok and I discovered that EdgeKeyIterator didn't exist,
// but I have a need for it.  This is part of its implementation.
//

/*
EdgeKeyIterator can be used to iterate edge keys of a certain edge kind.
*/
type EdgeKeyIterator struct {
	gm        *Manager            // GraphManager which created the iterator
	it        *hash.HTreeIterator // Internal HTree iterator
	LastError error               // Last encountered error
}

/*
Next returns the next edge key. Sets the LastError attribute if an error occurs.
*/
func (it *EdgeKeyIterator) Next() string {

	// Take reader lock
	it.gm.mutex.RLock()
	defer it.gm.mutex.RUnlock()

	for {

		/***/

		k, _ := it.it.Next()

		/***

		k, something := it.it.Next()
		fmt.Printf("k = %#v, something = %#v\n", k, something)

		***/

		if it.it.LastError != nil {
			it.LastError = &util.GraphError{
				Type:   util.ErrReading,
				Detail: it.it.LastError.Error(),
			}
			return ""
		}
		if len(k) == 0 {
			return ""
		}

		// Only accept entries that start with PrefixNSAttrs (0x01)
		if len(k) > 0 && k[0] == PrefixNSAttrs[0] {
			return string(k[len(PrefixNSAttrs):])
		}

		// Skip everything else (individual attribute values, etc.)
	}
}

/***
Here's Grok's explanation of his code.  He changed it to have 3 possibilities,
rather than 2.
  * If there was an error → set LastError and return
  * If we got an empty key → return
  * If this entry has the right prefix → return the clean key
Otherwise → do nothing and fall through to the bottom of the loop (which continues to the next iteration)
***/

/***
Here's some of what the debugging Printf above revealed.
The edgeKey = "~~~" is from this application code.

		for iter.HasNext() {

			// ........Get the key
			// ........(key comes from the iterator)

			edgeKey := iter.Next()
			if iter.Error() != nil {
				fmt.Printf("Iterator through a edge's keys had an error.  iter.Error() = $v\n", iter.Error())
			}

			fmt.Printf("edgeKey = %#v\n", edgeKey)

			I removed the rest of the processing from the loop.

		}

k = []byte{0x1, 0x31, 0x36, 0x30, 0x3a, 0x32, 0x36, 0x36}, something = []string{"\t\x00\x00\x00", "\b\x00\x00\x00", "\n\x00\x00\x00", "\v\x00\x00\x00", "\x03\x00\x00\x00", "\x04\x00\x00\x00", "\a\x00\x00\x00", "\x05\x00\x00\x00"}
edgeKey = "160:266"
k = []byte{0x2, 0x32, 0x32, 0x3a, 0x31, 0x31, 0x31, 0x4, 0x0, 0x0, 0x0}, something = "22"
edgeKey = "22:111\x04\x00\x00\x00"
k = []byte{0x2, 0x31, 0x35, 0x30, 0x3a, 0x32, 0x32, 0x37, 0x3, 0x0, 0x0, 0x0}, something = "Endpoint"
edgeKey = "150:227\x03\x00\x00\x00"
k = []byte{0x2, 0x31, 0x33, 0x36, 0x3a, 0x31, 0x39, 0x31, 0x7, 0x0, 0x0, 0x0}, something = "Station"
edgeKey = "136:191\a\x00\x00\x00"
k = []byte{0x2, 0x31, 0x37, 0x36, 0x3a, 0x32, 0x33, 0x34, 0x4, 0x0, 0x0, 0x0}, something = "176"
edgeKey = "176:234\x04\x00\x00\x00"
k = []byte{0x2, 0x33, 0x30, 0x3a, 0x31, 0x37, 0x36, 0x4, 0x0, 0x0, 0x0}, something = "30"
edgeKey = "30:176\x04\x00\x00\x00"
k = []byte{0x2, 0x39, 0x3a, 0x32, 0x33, 0x32, 0x3, 0x0, 0x0, 0x0}, something = "Endpoint"
edgeKey = "9:232\x03\x00\x00\x00"
k = []byte{0x2, 0x35, 0x32, 0x3a, 0x31, 0x5, 0x0, 0x0, 0x0}, something = false
edgeKey = "52:1\x05\x00\x00\x00"
k = []byte{0x2, 0x31, 0x37, 0x36, 0x3a, 0x32, 0x33, 0x34, 0x8, 0x0, 0x0, 0x0}, something = "Endpoint"
edgeKey = "176:234\b\x00\x00\x00"
k = []byte{0x2, 0x37, 0x34, 0x3a, 0x32, 0x38, 0x37, 0x3, 0x0, 0x0, 0x0}, something = "Endpoint"
edgeKey = "74:287\x03\x00\x00\x00"
k = []byte{0x2, 0x32, 0x34, 0x34, 0x3a, 0x32, 0x39, 0x35, 0x9, 0x0, 0x0, 0x0}, something = false
edgeKey = "244:295\t\x00\x00\x00"
k = []byte{0x2, 0x31, 0x31, 0x36, 0x3a, 0x31, 0x31, 0x37, 0x7, 0x0, 0x0, 0x0}, something = "Station"
edgeKey = "116:117\a\x00\x00\x00"
k = []byte{0x1, 0x32, 0x34, 0x32, 0x3a, 0x32, 0x36, 0x35}, something = []string{"\a\x00\x00\x00", "\n\x00\x00\x00", "\x05\x00\x00\x00", "\x03\x00\x00\x00", "\b\x00\x00\x00", "\v\x00\x00\x00", "\x04\x00\x00\x00", "\t\x00\x00\x00"}
edgeKey = "242:265"
k = []byte{0x2, 0x31, 0x32, 0x3a, 0x32, 0x35, 0x37, 0x8, 0x0, 0x0, 0x0}, something = "Endpoint"
edgeKey = "12:257\b\x00\x00\x00"
k = []byte{0x2, 0x31, 0x31, 0x37, 0x3a, 0x31, 0x31, 0x38, 0x3, 0x0, 0x0, 0x0}, something = "Endpoint"
edgeKey = "117:118\x03\x00\x00\x00"
k = []byte{0x2, 0x31, 0x32, 0x3a, 0x35, 0x36, 0x4, 0x0, 0x0, 0x0}, something = "12"
edgeKey = "12:56\x04\x00\x00\x00"
k = []byte{0x2, 0x32, 0x34, 0x38, 0x3a, 0x32, 0x38, 0x35, 0x5, 0x0, 0x0, 0x0}, something = false
edgeKey = "248:285\x05\x00\x00\x00"
k = []byte{0x2, 0x34, 0x3a, 0x32, 0x30, 0x31, 0xb, 0x0, 0x0, 0x0}, something = "Station"
edgeKey = "4:201\v\x00\x00\x00"
k = []byte{0x2, 0x34, 0x3a, 0x32, 0x30, 0x31, 0x4, 0x0, 0x0, 0x0}, something = "4"
edgeKey = "4:201\x04\x00\x00\x00"
k = []byte{0x2, 0x31, 0x31, 0x30, 0x3a, 0x32, 0x36, 0x35, 0x8, 0x0, 0x0, 0x0}, something = "Endpoint"
edgeKey = "110:265\b\x00\x00\x00"
k = []byte{0x2, 0x37, 0x33, 0x3a, 0x31, 0x38, 0x32, 0xb, 0x0, 0x0, 0x0}, something = "Station"
edgeKey = "73:182\v\x00\x00\x00"
k = []byte{0x2, 0x31, 0x31, 0x37, 0x3a, 0x31, 0x31, 0x38, 0xb, 0x0, 0x0, 0x0}, something = "Station"
edgeKey = "117:118\v\x00\x00\x00"
k = []byte{0x1, 0x32, 0x33, 0x3a, 0x34, 0x31}, something = []string{"\t\x00\x00\x00", "\a\x00\x00\x00", "\n\x00\x00\x00", "\x04\x00\x00\x00", "\b\x00\x00\x00", "\x05\x00\x00\x00", "\v\x00\x00\x00", "\x03\x00\x00\x00"}
edgeKey = "23:41"
k = []byte{0x2, 0x32, 0x32, 0x31, 0x3a, 0x32, 0x39, 0x34, 0xa, 0x0, 0x0, 0x0}, something = "294"
edgeKey = "221:294\n\x00\x00\x00"
k = []byte{0x2, 0x37, 0x33, 0x3a, 0x31, 0x8, 0x0, 0x0, 0x0}, something = "Endpoint"
edgeKey = "73:1\b\x00\x00\x00"
k = []byte{0x2, 0x34, 0x32, 0x3a, 0x31, 0x32, 0x30, 0x7, 0x0, 0x0, 0x0}, something = "Station"
edgeKey = "42:120\a\x00\x00\x00"
k = []byte{0x2, 0x38, 0x30, 0x3a, 0x32, 0x33, 0x31, 0x7, 0x0, 0x0, 0x0}, something = "Station"
edgeKey = "80:231\a\x00\x00\x00"
k = []byte{0x2, 0x32, 0x30, 0x30, 0x3a, 0x32, 0x38, 0x39, 0xb, 0x0, 0x0, 0x0}, something = "Station"
edgeKey = "200:289\v\x00\x00\x00"
k = []byte{0x2, 0x33, 0x36, 0x3a, 0x32, 0x38, 0x39, 0x5, 0x0, 0x0, 0x0}, something = false
edgeKey = "36:289\x05\x00\x00\x00"
***/

/*
HasNext returns if there is a next edge key.
*/
func (it *EdgeKeyIterator) HasNext() bool {
	return it.it.HasNext()
}

/*
Error returns the last encountered error.
*/
func (it *EdgeKeyIterator) Error() error {
	return it.LastError
}
