// Copyright 2021 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dataio

import (
	// "encoding/binary"
	// "fmt"
	// "os"
	// "path/filepath"
	"testing"

	"github.com/matrixorigin/matrixone/pkg/encoding"

	"github.com/matrixorigin/matrixone/pkg/compress"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	gvector "github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/logutil"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/common"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/container/batch"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/container/vector"
	// "github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/layout/base"
	// "github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/layout/index"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/metadata/v1"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/testutils"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/wal/shard"

	"github.com/stretchr/testify/assert"
)

var (
	moduleName = "Meta"
)

// Unused
// func getTestPath(t *testing.T) string {
// 	return testutils.GetDefaultTestPath(moduleName, t)
// }

func initTestEnv(t *testing.T) string {
	testutils.RemoveDefaultTestPath(moduleName, t)
	return testutils.MakeDefaultTestPath(moduleName, t)
}

// Unused
// func mockUnSortedSegmentFile(t *testing.T, dirname string, id common.ID, indices []index.Index, blkCnt int) base.ISegmentFile {
// 	baseid := id
// 	var dir string
// 	for i := 0; i < blkCnt; i++ {
// 		id.BlockID = uint64(i)
// 		name := common.MakeBlockFileName(dirname, id.ToBlockFileName(), id.TableID, false)
// 		dir = filepath.Dir(name)
// 		if _, err := os.Stat(dir); os.IsNotExist(err) {
// 			err = os.MkdirAll(dir, 0755)
// 			if err != nil {
// 				panic(fmt.Sprintf("err: %s", err))
// 			}
// 		}
// 		t.Log(name)
// 		w, err := os.Create(name)
// 		if err != nil {
// 			panic(err)
// 		}
// 		//buf, err := index.DefaultRWHelper.WriteIndices(indices)
// 		//if err != nil {
// 		//	panic(err)
// 		//}
// 		//_, err = w.Write(buf)
// 		//if err != nil {
// 		//	panic(err)
// 		//}
// 		algo := uint8(0)
// 		cols := uint16(0)
// 		count := uint64(0)
// 		logRange := new(metadata.LogRange)
// 		err = binary.Write(w, binary.BigEndian, &algo)
// 		assert.Nil(t, err)
// 		err = binary.Write(w, binary.BigEndian, &cols)
// 		assert.Nil(t, err)
// 		err = binary.Write(w, binary.BigEndian, &count)
// 		assert.Nil(t, err)
// 		rangeBuf, err := logRange.Marshal()
// 		assert.Nil(t, err)
// 		err = binary.Write(w, binary.BigEndian, &rangeBuf)
// 		prevIdx := shard.Index{
// 			Id: shard.SimpleIndexId(uint64(0)),
// 		}
// 		buf, err := prevIdx.Marshal()
// 		assert.Nil(t, err)
// 		var sz int32
// 		sz = int32(len(buf))
// 		err = binary.Write(w, binary.BigEndian, &sz)
// 		assert.Nil(t, err)
// 		err = binary.Write(w, binary.BigEndian, &buf)
// 		assert.Nil(t, err)
// 		idx := shard.Index{
// 			Id: shard.SimpleIndexId(uint64(0)),
// 		}
// 		buf, err = idx.Marshal()
// 		assert.Nil(t, err)
// 		var sz_ int32
// 		sz_ = int32(len(buf))
// 		err = binary.Write(w, binary.BigEndian, &sz_)
// 		assert.Nil(t, err)
// 		err = binary.Write(w, binary.BigEndian, &buf)
// 		assert.Nil(t, err)
// 		w.Close()
// 	}
// 	segFile := NewUnsortedSegmentFile(dirname, baseid)
// 	id = baseid
// 	for i := 0; i < blkCnt; i++ {
// 		id.BlockID = uint64(i)
// 		segFile.RefBlock(id)
// 	}

// 	return segFile
// }

// TODO(zzl)
//func TestAll(t *testing.T) {
//	dir := initTestEnv(t)
//	bufMgr := bmgr.MockBufMgr(26 * 4)
//	colCnt := 2
//	indices := index.MockInt32ZmIndices(colCnt)
//	id := common.ID{}
//	blkCnt := 4
//	droppedBlocks := []uint64{}
//	blkCB := func(v interface{}) {
//		droppedBlocks = append(droppedBlocks, v.(*index.BlockIndexHolder).ID.BlockID)
//	}
//	segFile := mockUnSortedSegmentFile(t, dir, id, indices, blkCnt)
//	tblHolder := index.NewTableHolder(bufMgr, id.TableID)
//	segHolder := tblHolder.RegisterSegment(id, base.UNSORTED_SEG, nil)
//	segHolder.Unref()
//	for i := 0; i < blkCnt; i++ {
//		blkId := id
//		blkId.BlockID = uint64(i)
//		blkHolder := segHolder.RegisterBlock(blkId, base.PERSISTENT_BLK, blkCB)
//		blkHolder.Unref()
//		blkHolder.Init(segFile)
//	}
//	t.Log(bufMgr.String())
//	t.Log(tblHolder.String())
//	//assert.Equal(t, colCnt*blkCnt, bufMgr.NodeCount())
//	for bidx := 0; bidx < blkCnt; bidx++ {
//		blk := segHolder.StrongRefBlock(uint64(bidx))
//		for i, _ := range blk.Indices {
//			node := blk.GetIndexNode(i)
//			mnode := node.GetManagedNode()
//			zm := mnode.DataNode.(*index.BlockZoneMapIndex)
//			assert.Equal(t, zm.MinV.(int32), int32(i*100+1))
//			assert.Equal(t, zm.MaxV.(int32), int32(i*100+99))
//			t.Logf("zm index: min=%v,max=%v", zm.MinV, zm.MaxV)
//			mnode.Close()
//			node.Unref()
//		}
//		blk.Unref()
//	}
//	t.Log(bufMgr.String())
//	t.Log(tblHolder.String())
//
//	dropblkid := uint64(1)
//	droppBlk := segHolder.DropBlock(dropblkid)
//	t.Log(droppBlk.RefCount())
//	droppBlk.Unref()
//	t.Log(bufMgr.String())
//	t.Log(tblHolder.String())
//	t.Log(droppBlk.RefCount())
//	t.Log(droppedBlocks)
//	assert.Equal(t, 1, len(droppedBlocks))
//	assert.Equal(t, droppedBlocks[0], dropblkid)
//
//	fsMgr := NewManager(dir, false)
//	segFile = fsMgr.GetUnsortedFile(id)
//	assert.Nil(t, segFile)
//	segFile, err := fsMgr.RegisterUnsortedFiles(id)
//	assert.Nil(t, err)
//	segFile = fsMgr.GetUnsortedFile(id)
//	assert.NotNil(t, segFile)
//	fsMgr.UnregisterUnsortedFile(id)
//	err = fsMgr.Close()
//	assert.Nil(t, err)
//	logutil.Infof(fsMgr.String())
//}

func TestIVectorNodeWriter(t *testing.T) {
	dir := initTestEnv(t)
	vecType := types.Type{
		Oid:       types.T_int32,
		Size:      4,
		Width:     4,
		Precision: 0}
	capacity := uint64(40)
	vec0 := vector.NewStdVector(vecType, 4)
	defer vec0.Close()
	vec1 := vector.NewStrVector(types.Type{
		Oid:       types.T(types.T_varchar),
		Size:      24,
		Width:     0,
		Precision: 0}, 4)
	defer vec1.Close()
	err := vec0.Append(4, []int32{int32(3), int32(1), int32(2), int32(0)})
	assert.Nil(t, err)
	str0 := "str1"
	str1 := "str0"
	str2 := "str2"
	str3 := "str3"
	strs := [][]byte{[]byte(str0), []byte(str1), []byte(str2), []byte(str3)}
	err = vec1.Append(len(strs), strs)

	catalog := metadata.MockCatalog(dir, capacity, uint64(10), nil, nil)
	schema := metadata.MockSchema(2)
	schema.PrimaryKey = 1
	gen := shard.NewMockIndexAllocator()
	shardId := uint64(100)
	tblMeta := metadata.MockDBTable(catalog, "db1", schema, nil, 1, gen.Shard(shardId))
	segMeta := tblMeta.SimpleGetSegment(uint64(1))
	assert.NotNil(t, segMeta)
	meta := segMeta.SimpleGetBlock(uint64(1))
	assert.NotNil(t, meta)

	bat, err := batch.NewBatch([]int{0, 1}, []vector.IVector{vec0, vec1})
	assert.Nil(t, err)
	w := NewIBatchWriter(bat, meta, dir)
	err = w.Execute()
	assert.Nil(t, err)

	id := *meta.AsCommonID()
	segFile := NewUnsortedSegmentFile(dir, *meta.Segment.AsCommonID())
	f := NewBlockFile(segFile, id, nil)
	bufs := make([][]byte, 2)
	for i, _ := range bufs {
		sz := f.PartSize(uint64(i), id, false)
		osz := f.PartSize(uint64(i), id, true)
		node := common.GPool.Alloc(uint64(sz))
		defer common.GPool.Free(node)
		buf := node.Buf[:sz]
		f.ReadPart(uint64(i), id, buf)
		obuf := make([]byte, osz)
		_, err = compress.Decompress(buf, obuf, compress.Lz4)
		assert.Nil(t, err)
		if i == 0 {
			vec := vector.NewEmptyStdVector()
			err = vec.Unmarshal(obuf)
			assert.Nil(t, err)
			vc, err := vec.GetLatestView().CopyToVector()
			assert.Nil(t, err)
			t.Log(vc.String())
			assert.Equal(t, 4, vec.Length())
		} else {
			vec := vector.NewEmptyStrVector()
			err = vec.Unmarshal(obuf)
			assert.Nil(t, err)
			vc, err := vec.GetLatestView().CopyToVector()
			assert.Nil(t, err)
			t.Log(vc.String())
			assert.Equal(t, 4, vec.Length())
		}
	}

	var vecs []*gvector.Vector
	v0c, err := vec0.CopyToVector()
	assert.Nil(t, err)
	v1c, err := vec1.CopyToVector()
	assert.Nil(t, err)
	logutil.Infof("v0c is %v, v1c is %v\n", v0c, v1c)
	vecs = append(vecs, v0c)
	vecs = append(vecs, v1c)
	segMeta.Table.Schema.ColDefs[1].Type.Oid = types.T_varchar
	bw := NewBlockWriter(vecs, meta, dir)
	err = bw.Execute()
	assert.Nil(t, err)
	logutil.Infof(" %s | Memtable | Flushing", bw.GetFileName())

	segFile1 := NewUnsortedSegmentFile(dir, *meta.Segment.AsCommonID())
	nb := NewBlockFile(segFile1, id, nil)
	bufs = make([][]byte, 2)
	for i, _ := range bufs {
		sz := nb.PartSize(uint64(i), id, false)
		osz := nb.PartSize(uint64(i), id, true)
		buf := make([]byte, sz)
		nb.ReadPart(uint64(i), id, buf)
		originSize := uint64(osz)
		node1 := common.GPool.Alloc(originSize)
		defer common.GPool.Free(node1)
		_, err = compress.Decompress(buf, node1.Buf[:originSize], compress.Lz4)
		data := node1.Buf[:originSize]
		t1 := encoding.DecodeType(data[:encoding.TypeSize])
		v := gvector.New(t1)
		err = v.Read(data)
		logutil.Infof("nb.v is %v.\n", v)
		switch i {
		case 0:
			assert.Equal(t, int32(1), v.Col.([]int32)[0])
			assert.Equal(t, int32(3), v.Col.([]int32)[1])
			assert.Equal(t, int32(2), v.Col.([]int32)[2])
			assert.Equal(t, int32(0), v.Col.([]int32)[3])
		case 1:
			assert.Equal(t, []byte("str0"), v.Col.(*types.Bytes).Data[0:4])
			assert.Equal(t, []byte("str1"), v.Col.(*types.Bytes).Data[4:8])
			assert.Equal(t, []byte("str2"), v.Col.(*types.Bytes).Data[8:12])
			assert.Equal(t, []byte("str3"), v.Col.(*types.Bytes).Data[12:16])
		}
	}

	col0Vf := segFile.MakeVirtualPartFile(&id)
	assert.NotNil(t, col0Vf)
	stat0 := col0Vf.Stat()
	t.Log(stat0.Name())
	buf := make([]byte, col0Vf.Stat().Size())
	len, err := col0Vf.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, int64(len), col0Vf.Stat().Size())
	col0Vf.Unref()
}

func TestTransientBlock(t *testing.T) {
	dir := initTestEnv(t)
	rowCount, blkCount := uint64(10), uint64(4)
	catalog := metadata.MockCatalog(dir, rowCount, blkCount, nil, nil)
	schema := metadata.MockSchema(2)
	gen := shard.NewMockIndexAllocator()
	shardId := uint64(100)
	tbl := metadata.MockDBTable(catalog, "db1", schema, nil, 1, gen.Shard(shardId))

	segMeta := tbl.SimpleGetSegment(uint64(1))
	assert.NotNil(t, segMeta)
	blkMeta := segMeta.SimpleGetBlock(uint64(1))
	assert.NotNil(t, blkMeta)

	segFile := NewUnsortedSegmentFile(dir, *blkMeta.Segment.AsCommonID())

	tblk := NewTBlockFile(segFile, *blkMeta.AsCommonID())
	defer tblk.Unref()

	// rows := uint64(2)
	// bat1 := mock.MockBatch(schema.Types(), rows)
	// bat2 := mock.MockBatch(schema.Types(), rowCount)

	// ok := tblk.PreSync(uint32(bat1.Vecs[0].Length()))
	// assert.True(t, ok)
	// err = tblk.Sync(bat1.Vecs, blkMeta, blkMeta.Segment.Table.Conf.Dir)
	// assert.Nil(t, err)
	// ok = tblk.PreSync(uint32(bat2.Vecs[0].Length()))
	// assert.True(t, ok)
	// err = tblk.Sync(bat2.Vecs, blkMeta, blkMeta.Segment.Table.Conf.Dir)
	// assert.Nil(t, err)
	// ok = tblk.PreSync(uint32(bat2.Vecs[0].Length()))
	// assert.False(t, ok)
}
