package db

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/matrixorigin/matrixone/pkg/container/batch"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/dbi"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/logstore/sm"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/metadata/v1"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/mock"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/testutils"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/wal"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/aoe/storage/wal/shard"
	"github.com/stretchr/testify/assert"
)

type mockShard struct {
	sm.ClosedState
	sm.StateMachine
	gen      *shard.MockShardIndexGenerator
	database *metadata.Database
	queue    sm.Queue
	inst     *DB
}

type requestCtx struct {
	sync.WaitGroup
	err     error
	result  interface{}
	request interface{}
}

func newCtx() *requestCtx {
	ctx := new(requestCtx)
	ctx.Add(1)
	return ctx
}

func (ctx *requestCtx) setDone(err error, result interface{}) {
	ctx.err = err
	ctx.result = result
	ctx.Done()
}

func newMockShard(inst *DB, gen *shard.MockShardIndexGenerator) *mockShard {
	s := &mockShard{
		inst: inst,
		gen:  gen,
	}
	var err error
	dbName := strconv.FormatUint(s.gen.ShardId, 10)
	s.database, err = inst.CreateDatabase(dbName, s.gen.ShardId)
	if err != nil {
		panic(err)
	}
	wg := new(sync.WaitGroup)
	s.queue = sm.NewWaitableQueue(1000, 1, s, wg, nil, nil, s.onItems)
	s.queue.Start()
	return s
}

func (s *mockShard) Stop() {
	s.queue.Stop()
}

func (s *mockShard) sendRequest(ctx *requestCtx) {
	_, err := s.queue.Enqueue(ctx)
	if err != nil {
		ctx.Done()
	}
}

func (s *mockShard) onItems(items ...interface{}) {
	item := items[0]
	ctx := item.(*requestCtx)
	switch r := ctx.request.(type) {
	case *metadata.Schema:
		err := s.createTable(r)
		ctx.setDone(err, nil)
	case *dbi.DropTableCtx:
		err := s.dropTable(r)
		ctx.setDone(err, nil)
	case *dbi.AppendCtx:
		err := s.insert(r)
		ctx.setDone(err, nil)
	default:
		panic("")
	}
}

func (s *mockShard) createTable(schema *metadata.Schema) error {
	_, err := s.inst.CreateTable(s.database.Name, schema, s.gen.Next())
	return err
}

func (s *mockShard) dropTable(ctx *dbi.DropTableCtx) error {
	ctx.ShardId = s.gen.ShardId
	ctx.DBName = s.database.Name
	ctx.OpIndex = s.gen.Alloc()
	_, err := s.inst.DropTable(*ctx)
	return err
}

func (s *mockShard) insert(ctx *dbi.AppendCtx) error {
	ctx.ShardId = s.gen.ShardId
	ctx.DBName = s.database.Name
	ctx.OpIndex = s.gen.Alloc()
	ctx.OpSize = 1
	err := s.inst.Append(*ctx)
	return err
}

func (s *mockShard) getSafeId() uint64 {
	return s.inst.Wal.GetShardCheckpointId(s.gen.ShardId)
}

type mockClient struct {
	t       *testing.T
	schemas []*metadata.Schema
	router  map[string]int
	shards  []*mockShard
	bats    []*batch.Batch
}

func newClient(t *testing.T, shards []*mockShard, schemas []*metadata.Schema,
	bats []*batch.Batch) *mockClient {
	router := make(map[string]int)
	for i, info := range schemas {
		routed := i % len(shards)
		router[info.Name] = routed
	}
	return &mockClient{
		t:       t,
		schemas: schemas,
		shards:  shards,
		router:  router,
		bats:    bats,
	}
}

func (cli *mockClient) routing(name string) *mockShard {
	shardPos := cli.router[name]
	shard := cli.shards[shardPos]
	// cli.t.Logf("table-%s routed to shard-%d", info.Name, shard.id)
	return shard
}

func (cli *mockClient) insert(pos int) error {
	info := cli.schemas[pos]
	shard := cli.routing(info.Name)
	ctx := newCtx()
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(len(cli.bats))
	ctx.request = &dbi.AppendCtx{TableName: info.Name, Data: cli.bats[n]}
	shard.sendRequest(ctx)
	ctx.Wait()
	return ctx.err
}

func (cli *mockClient) dropTable(pos int) error {
	info := cli.schemas[pos]
	shard := cli.routing(info.Name)
	ctx := newCtx()
	ctx.request = &dbi.DropTableCtx{TableName: info.Name}
	shard.sendRequest(ctx)
	ctx.Wait()
	return ctx.err
}

func (cli *mockClient) createTable(pos int) error {
	info := cli.schemas[pos]
	shard := cli.routing(info.Name)
	ctx := newCtx()
	ctx.request = info
	shard.sendRequest(ctx)
	ctx.Wait()
	return ctx.err
}

func TestShard1(t *testing.T) {
	tableCnt := 20
	schemas := make([]*metadata.Schema, tableCnt)
	for i := 0; i < tableCnt; i++ {
		schemas[i] = metadata.MockSchema(20)
		schemas[i].Name = fmt.Sprintf("mock-%d", i)
	}

	initDBTest()
	inst, gen := initDB(wal.BrokerRole)

	shardCnt := 8
	shards := make([]*mockShard, shardCnt)
	for i := 0; i < shardCnt; i++ {
		idxGen := gen.Shard(uint64(i + 1))
		shards[i] = newMockShard(inst, idxGen)
	}

	var wg sync.WaitGroup
	clients := make([]*mockClient, 2)
	for i, _ := range clients {
		clients[i] = newClient(t, shards, schemas[i*tableCnt/2:(i+1)*tableCnt/2], nil)
		wg.Add(1)
		go func(cli *mockClient) {
			defer wg.Done()
			for pos := 0; pos < len(cli.schemas); pos++ {
				err := cli.createTable(pos)
				assert.Nil(t, err)
			}
			for pos := 0; pos < len(cli.schemas); pos++ {
				err := cli.dropTable(pos)
				assert.Nil(t, err)
			}
		}(clients[i])
	}

	wg.Wait()
	for _, shard := range shards {
		testutils.WaitExpect(200, func() bool {
			return shard.gen.Get() == shard.getSafeId()
		})
		assert.Equal(t, shard.gen.Get(), shard.getSafeId())
		t.Logf("shard-%d safeid %d, logid-%d", shard.gen.ShardId, shard.getSafeId(), shard.gen.Get())
		shard.Stop()
	}
	inst.Close()
}

func TestShard2(t *testing.T) {
	// Create 10 Table [0,1,2,3,4,5]
	// Insert To 10 Table [0,1,2,3,4]
	// Drop 10 Table
	tableCnt := 10
	schemas := make([]*metadata.Schema, tableCnt)
	for i := 0; i < tableCnt; i++ {
		schemas[i] = metadata.MockSchema(20)
		schemas[i].Name = fmt.Sprintf("mock-%d", i)
	}

	initDBTest()
	inst, _ := initDB(wal.BrokerRole)

	gen := shard.NewMockIndexAllocator()

	shardCnt := 4
	shards := make([]*mockShard, shardCnt)
	for i := 0; i < shardCnt; i++ {
		shards[i] = newMockShard(inst, gen.Shard(uint64(i)+1))
	}

	batches := make([]*batch.Batch, 10)
	for i, _ := range batches {
		step := inst.Store.Catalog.Cfg.BlockMaxRows / 10
		rows := (uint64(i) + 1) * 2 * step
		batches[i] = mock.MockBatch(schemas[0].Types(), rows)
	}

	var wg sync.WaitGroup
	clients := make([]*mockClient, 2)
	for i, _ := range clients {
		clients[i] = newClient(t, shards, schemas[i*tableCnt/2:(i+1)*tableCnt/2], batches)
		wg.Add(1)
		go func(cli *mockClient) {
			defer wg.Done()
			for pos := 0; pos < len(cli.schemas); pos++ {
				err := cli.createTable(pos)
				assert.Nil(t, err)
			}
			for n := 0; n < 2; n++ {
				for pos := 0; pos < len(cli.schemas); pos++ {
					err := cli.insert(pos)
					assert.Nil(t, err)
				}
			}
			for pos := 0; pos < len(cli.schemas); pos++ {
				err := cli.dropTable(pos)
				assert.Nil(t, err)
			}
		}(clients[i])
	}

	wg.Wait()
	for _, shard := range shards {
		testutils.WaitExpect(400, func() bool {
			return shard.gen.Get() == shard.getSafeId()
		})
		assert.Equal(t, shard.gen.Get(), shard.getSafeId())
		t.Logf("shard-%d safeid %d, logid-%d", shard.gen.ShardId, shard.getSafeId(), shard.gen.Get())
		shard.Stop()
	}

	total := 0
	for _, shard := range shards {
		view := shard.database.View(0)
		assert.Empty(t, view.Database.TableSet)
		view = shard.database.View(shard.getSafeId())
		total += len(view.Database.TableSet)
	}
	assert.Equal(t, 0, total)
	for _, shard := range shards {
		testutils.WaitExpect(400, func() bool {
			return shard.gen.Get() == shard.getSafeId()
		})
	}
	dbCompacts := 0
	tblCompacts := 0
	dbListener := new(metadata.BaseDatabaseListener)
	dbListener.DatabaseCompactedFn = func(database *metadata.Database) {
		dbCompacts++
	}
	tblListener := new(metadata.BaseTableListener)
	tblListener.TableCompactedFn = func(t *metadata.Table) {
		tblCompacts++
	}
	inst.Store.Catalog.Compact(dbListener, tblListener)
	assert.Equal(t, 0, dbCompacts)
	assert.Equal(t, tableCnt, tblCompacts)

	for _, shard := range shards {
		err := inst.DropDatabase(shard.database.Name, shard.gen.Alloc())
		assert.Nil(t, err)
	}
	for _, shard := range shards {
		testutils.WaitExpect(400, func() bool {
			return (shard.gen.Get() == shard.getSafeId()) && (shard.database.IsHardDeleted())
		})
		assert.Equal(t, shard.gen.Get(), shard.getSafeId())
	}
	dbCompacts = 0
	tblCompacts = 0
	inst.Store.Catalog.Compact(dbListener, tblListener)
	assert.Equal(t, len(shards), dbCompacts)
	assert.Equal(t, 0, tblCompacts)

	t.Log(inst.Store.Catalog.IndexWal.String())

	inst.Close()
}
