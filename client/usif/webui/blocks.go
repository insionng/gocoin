package webui

import (
	"time"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"github.com/piotrnar/gocoin/lib/btc"
	"github.com/piotrnar/gocoin/client/common"
	"github.com/piotrnar/gocoin/client/network"
)

func p_blocks(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	write_html_head(w, r)
	w.Write([]byte(load_template("blocks.html")))
	write_html_tail(w)
}


func json_blocks(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	type one_block struct {
		Height uint32
		Timestamp uint32
		Hash string
		TxCnt int
		Size int
		Version uint32
		Reward uint64
		Miner string
		FeeSPB float64

		Received uint32
		TimeDl, TimeVer, TimeQue int
		WasteCnt uint
		Sigops int
	}

	var blks []*one_block

	common.Last.Mutex.Lock()
	end := common.Last.Block
	common.Last.Mutex.Unlock()

	for cnt:=uint32(0); end!=nil && cnt<atomic.LoadUint32(&common.CFG.WebUI.ShowBlocks); cnt++ {
		bl, _, e := common.BlockChain.Blocks.BlockGet(end.BlockHash)
		if e != nil {
			return
		}
		block, e := btc.NewBlock(bl)
		if e!=nil {
			return
		}

		b := new(one_block)
		b.Height = end.Height
		b.Timestamp = block.BlockTime()
		b.Hash = end.BlockHash.String()
		b.TxCnt = block.TxCount
		b.Size = len(bl)
		b.Version = block.Version()

		cbasetx, cbaselen := btc.NewTx(bl[block.TxOffset:])
		for o := range cbasetx.TxOut {
			b.Reward += cbasetx.TxOut[o].Value
		}

		b.Miner, _ = common.TxMiner(cbasetx)
		if len(bl)-block.TxOffset-cbaselen != 0 {
			b.FeeSPB = float64(b.Reward-btc.GetBlockReward(end.Height)) / float64(len(bl)-block.TxOffset-cbaselen)
		}

		common.BlockChain.BlockIndexAccess.Lock()
		node := common.BlockChain.BlockIndex[end.BlockHash.BIdx()]
		common.BlockChain.BlockIndexAccess.Unlock()

		network.MutexRcv.Lock()
		rb := network.ReceivedBlocks[end.BlockHash.BIdx()]
		network.MutexRcv.Unlock()

		b.Received = uint32(rb.Time.Unix())
		b.Sigops = int(node.Sigops)

		if rb.TmDownload!=0 {
			b.TimeDl = int(rb.TmDownload/time.Millisecond)
		} else {
			b.TimeDl = -1
		}

		if rb.TmQueuing!=0 {
			b.TimeQue = int(rb.TmQueuing/time.Millisecond)
		} else {
			b.TimeQue = -1
		}

		if rb.TmAccept!=0 {
			b.TimeVer = int(rb.TmAccept/time.Millisecond)
		} else {
			b.TimeVer = -1
		}

		b.WasteCnt = rb.Cnt

		blks = append(blks, b)
		end = end.Parent
	}

	bx, er := json.Marshal(blks)
	if er == nil {
		w.Header()["Content-Type"] = []string{"application/json"}
		w.Write(bx)
	} else {
		println(er.Error())
	}

}
