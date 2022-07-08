package clipool

import "sync"

type Solana_ClientCreatorFunc func() interface{}
type Solana_WSClientCreatorFunc func() interface{}

type Solana_ClientPool struct {
	max  int
	init int
	cnt  int

	freeList   []interface{}
	freeListWs []interface{}

	lock        sync.Mutex
	allocFunc   Solana_ClientCreatorFunc
	allocFuncWs Solana_WSClientCreatorFunc
}

func (p *Solana_ClientPool) Init(init int, max int, allocFunc Solana_ClientCreatorFunc, allocFuncWs Solana_WSClientCreatorFunc) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.init = init
	p.max = max
	p.allocFunc = allocFunc
	p.allocFuncWs = allocFuncWs
	for i := 0; i < p.init; i++ {
		cli := p.allocFunc()
		cliWs := p.allocFuncWs()
		p.freeList = append(p.freeList, cli)
		p.freeListWs = append(p.freeListWs, cliWs)
		p.cnt++
	}
}

func (p *Solana_ClientPool) Alloc() (interface{}, interface{}) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var ret interface{}
	if len(p.freeList) == 0 && len(p.freeListWs) == 0 {
		cli := p.allocFunc()
		cliWs := p.allocFuncWs()
		//ret = cli
		p.cnt++
		return cli, cliWs
	} else {
		ret = p.freeList[0]
		p.freeList = p.freeList[1:]
		retWs := p.freeListWs[0]
		p.freeListWs = p.freeListWs[1:]

		return ret, retWs
	}
}

func (p *Solana_ClientPool) Free(v interface{}, w interface{}) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.freeList = append(p.freeList, v)
	p.freeListWs = append(p.freeListWs, w)
}
