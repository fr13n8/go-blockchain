package trxpool

import (
	"fmt"
	"github.com/fr13n8/go-blockchain/pkg/services/transaction"
	"sync"
)

type TransactionPool struct {
	pool map[string]*transaction.Transaction
	l    sync.RWMutex
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		pool: make(map[string]*transaction.Transaction, 1024),
	}
}

func (tp *TransactionPool) Add(tx *transaction.Transaction) {
	tp.l.Lock()
	Id, err := tx.Hash()
	if err != nil {
		return
	}
	tx.Id = Id
	hash, err := tx.Hash()
	if err != nil {
		return
	}
	tp.pool[fmt.Sprintf("%x", hash)] = tx
	tp.l.Unlock()
}

func (tp *TransactionPool) Clean(trxs []*transaction.Transaction) {
	for _, t := range trxs {
		h, err := t.Hash()
		if err != nil {
			continue
		}
		delete(tp.pool, fmt.Sprintf("%x", h))
	}
}

func (tp *TransactionPool) GetAndClean(n int) []*transaction.Transaction {
	foundTXs := make([]*transaction.Transaction, 0, n)
	tp.l.Lock()

	defer func() {
		tp.Clean(foundTXs)
		tp.l.Unlock()
	}()

	for _, t := range tp.pool {
		if len(foundTXs) >= n {
			return foundTXs
		}
		foundTXs = append(foundTXs, t)
	}
	return foundTXs
}

func (tp *TransactionPool) Read(n int) []*transaction.Transaction {
	tp.l.RLock()
	defer tp.l.RUnlock()

	txs := make([]*transaction.Transaction, 0, n)
	for _, t := range tp.pool {
		txs = append(txs, t)
	}
	return txs
}

func (tp *TransactionPool) Size() int {
	return len(tp.pool)
}
